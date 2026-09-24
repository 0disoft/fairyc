package compiler_test

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/lexer"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

type scannerBudgetImage struct {
	name            string
	artifact        cdriver.WindowsHostArtifact
	sourceBytes     int
	specializations int
}

// One execution per lane proves the three independent images can be compared.
// Timing from this smoke is diagnostic only; it cannot decide the 8% budget.
func TestPrivateVectorSDKCandidateIndependentCScannerBudgetSmoke(t *testing.T) {
	if os.Getenv("FAIRYC_VECTOR_LEXER_C_BUDGET_SMOKE") != "1" {
		t.Skip("opt-in three-way scanner budget smoke")
	}
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Fatal("requires Windows AMD64 MSVC")
	}
	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	const sample = "let n=12345; /*open"
	const rounds = 1000
	lexed := lexer.Lex(source.NewFile("input.fy", []byte(sample)), source.NewBudget(source.DefaultLimits(), len(sample)))
	perRound := len(sample) + len(lexed.Tokens) + len(lexed.Diagnostics)
	for _, token := range lexed.Tokens {
		perRound += len(token.Cooked)
	}
	for _, diagnostic := range lexed.Diagnostics {
		perRound += len(diagnostic.Token)
	}
	if perRound <= len(sample) || len(lexed.Diagnostics) == 0 {
		t.Fatal("scanner sample lost cooked or diagnostic coverage")
	}
	cSource := scannerCBenchmarkSource(t)
	cArtifact, err := cdriver.BuildWindowsMSVCProbe(toolchain, cdriver.OptimizedProfile, cSource)
	if err != nil {
		t.Fatal(err)
	}
	images := []scannerBudgetImage{{name: "independent C", artifact: cArtifact, sourceBytes: len(cSource)}}
	variants := realScannerVectorVariants(t)
	if len(variants) != 5 || variants[0].name != "explicit index" || variants[4].name != "all vector for" {
		t.Fatal("real scanner variants changed")
	}
	support, err := os.ReadFile(filepath.Join("..", "..", "tools", "benchmarks", "c-budget", "support.c"))
	if err != nil {
		t.Fatal(err)
	}
	for _, variant := range []realScannerVectorVariant{variants[0], variants[4]} {
		program, entry, specializations := scannerBudgetFairyProgram(t, variant)
		artifact, unit := buildRealScannerVectorNative(t, program, entry, cdriver.OptimizedProfile,
			"\n"+string(support)+"\nvoid fc_budget_flush(void) { fflush(stdout); }\n")
		images = append(images, scannerBudgetImage{
			name: variant.name, artifact: artifact, sourceBytes: len(unit.Source), specializations: specializations,
		})
	}
	reporter := ""
	rows := make([]map[string]any, 0, len(images))
	for _, image := range images {
		outcome, err := cdriver.ExecuteWindowsMSVCTargetWithInput(image.artifact, strings.NewReader("12345 1000\n"))
		if err != nil || outcome.ExitCode != 0 || outcome.Stderr != "" || outcome.TrapControl != "" {
			t.Fatalf("%s execution: %+v %v", image.name, outcome, err)
		}
		fields := strings.Fields(outcome.Stdout)
		if len(fields) < 5 {
			t.Fatalf("%s missing timing fields: %q", image.name, outcome.Stdout)
		}
		prefix := strings.Join(fields[:len(fields)-4], " ")
		if image.name == "independent C" {
			if !strings.HasPrefix(outcome.Stdout, scannerGoObservation([]byte(sample))) {
				t.Fatalf("independent C reporter differs: %q", outcome.Stdout)
			}
		} else if reporter == "" {
			reporter = prefix
		} else if prefix != reporter {
			t.Fatalf("%s reporter differs from explicit-index Fairy", image.name)
		}
		last := fields[len(fields)-4:]
		completed, completedErr := strconv.ParseUint(last[0], 10, 32)
		ticks, ticksErr := strconv.ParseUint(last[1], 10, 64)
		frequency, frequencyErr := strconv.ParseUint(last[2], 10, 64)
		total, totalErr := strconv.ParseUint(last[3], 10, 64)
		if completedErr != nil || ticksErr != nil || frequencyErr != nil || totalErr != nil ||
			completed != rounds || ticks == 0 || frequency == 0 || total != uint64(perRound*rounds) {
			t.Fatalf("%s timing fields=%v errors=%v,%v,%v,%v", image.name, last,
				completedErr, ticksErr, frequencyErr, totalErr)
		}
		rows = append(rows, map[string]any{
			"lane": image.name, "exe_bytes": len(image.artifact.Bytes), "source_bytes": image.sourceBytes,
			"specializations": image.specializations, "qpc_ticks": ticks, "qpc_frequency": frequency,
			"exe_sha256": fmt.Sprintf("%x", sha256.Sum256(image.artifact.Bytes)),
		})
	}
	report, err := json.Marshal(map[string]any{
		"schema": "fairyc.vector-lexer-c-budget-smoke/v1", "sample_sha256": fmt.Sprintf("%x", sha256.Sum256([]byte(sample))),
		"msvc": toolchain.Version, "profile": cdriver.OptimizedProfile.Name,
		"rounds": rounds, "per_round_result": perRound, "lanes": rows,
		"scope": "Three standalone images, one diagnostic execution per lane; no 8% runtime verdict or timing-stability claim.",
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Log("VECTOR_LEXER_C_BUDGET_SMOKE " + string(report))
}

func scannerBudgetFairyProgram(t *testing.T, variant realScannerVectorVariant) (mir.Program, string, int) {
	t.Helper()
	root := t.TempDir()
	example := filepath.Join("..", "..", "examples", "lexer_candidate")
	for _, name := range []string{"fairy.toml", "src/numbers.fy", "src/diagnostics.fy", "src/utf8.fy", "src/strings.fy", "src/budget.fy"} {
		data, err := os.ReadFile(filepath.Join(example, filepath.FromSlash(name)))
		if err != nil {
			t.Fatal(err)
		}
		writeVectorCacheFile(t, root, name, string(data))
	}
	writeVectorCacheFile(t, root, "src/scanner.fy", variant.scanner)
	writeVectorCacheFile(t, root, "src/main.fy", realScannerExecutionMain)
	workspace, err := workspaceloader.LoadVectorCandidate(root)
	if err != nil {
		t.Fatal(err)
	}
	input, inputErr := workspace.SemanticInput()
	coordinates, coordinateErr := workspace.PackageCoordinates()
	owners, ownerErr := workspace.VectorCandidateStorageOwners()
	if inputErr != nil || coordinateErr != nil || ownerErr != nil || len(owners) != 1 {
		t.Fatalf("scanner package authority: %v %v %v owners=%v", inputErr, coordinateErr, ownerErr, owners)
	}
	var analysis compiler.GenericAnalysis
	if variant.candidate {
		analysis, err = compiler.AnalyzeSDKVectorCandidatePackages(input, coordinates, genericplan.DefaultPlanLimits(), owners)
	} else {
		analysis, err = compiler.AnalyzeSDKStoragePackages(input, coordinates, genericplan.DefaultPlanLimits(), owners)
	}
	if err != nil || analysis.Analysis.Semantic.CompletedThrough != semantic.PhaseFlow ||
		len(analysis.Analysis.Semantic.Diagnostics) != 0 || len(analysis.Analysis.Semantic.Unsupported) != 0 {
		t.Fatalf("%s scanner Flow: %v diagnostics=%+v unsupported=%+v", variant.name, err,
			analysis.Analysis.Semantic.Diagnostics, analysis.Analysis.Semantic.Unsupported)
	}
	checked, err := compiler.LowerAnalyzedStorageTransfers(analysis.Analysis)
	if err != nil || !checked.Verified {
		t.Fatalf("%s verified MIR: %v diagnostics=%+v", variant.name, err, checked.Semantic.Diagnostics)
	}
	rootID, err := workspace.RootPackageID()
	if err != nil {
		t.Fatal(err)
	}
	return checked.MIR, hir.QualifiedModulePath(rootID, "main") + "::main", len(analysis.Plan.Specializations)
}
