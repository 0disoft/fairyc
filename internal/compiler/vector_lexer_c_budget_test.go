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
	"github.com/0disoft/fairyc/internal/pecoff"
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

// One execution per lane proves the independent images can be compared.
// Timing from this smoke is diagnostic only; it cannot decide the 8% budget.
func TestPrivateVectorSDKCandidateIndependentCScannerBudgetSmoke(t *testing.T) {
	if os.Getenv("FAIRYC_VECTOR_LEXER_C_BUDGET_SMOKE") != "1" {
		t.Skip("opt-in three-way scanner budget smoke")
	}
	storageMode := os.Getenv("FAIRYC_VECTOR_LEXER_C_BUDGET_BOTH_STORAGE")
	if storageMode != "" && storageMode != "1" {
		t.Fatalf("unknown C scanner storage mode %q", storageMode)
	}
	bothStorage := storageMode == "1"
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
	if bothStorage {
		geometricSource := scannerCGeometricBenchmarkSource(t)
		geometricArtifact, buildErr := cdriver.BuildWindowsMSVCProbe(toolchain, cdriver.OptimizedProfile, geometricSource)
		if buildErr != nil {
			t.Fatal(buildErr)
		}
		images = append(images, scannerBudgetImage{
			name: "geometric C", artifact: geometricArtifact, sourceBytes: len(geometricSource),
		})
	}
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
	if os.Getenv("FAIRYC_VECTOR_LEXER_C_BUDGET_SECTIONS") == "1" {
		sectionRows := make([]map[string]any, 0, len(images))
		for _, image := range images {
			pe, inspectErr := pecoff.InspectExecutable(image.artifact.Bytes)
			if inspectErr != nil {
				t.Fatalf("%s PE inspection: %v", image.name, inspectErr)
			}
			sections := make([]map[string]any, 0, len(pe.Sections))
			accounted := uint64(pe.SizeOfHeaders)
			for _, section := range pe.Sections {
				sections = append(sections, map[string]any{
					"name": section.Name, "raw_bytes": section.RawSize,
					"virtual_bytes": section.VirtualSize, "raw_offset": section.RawOffset,
				})
				accounted += uint64(section.RawSize)
			}
			if accounted > uint64(len(image.artifact.Bytes)) {
				t.Fatalf("%s PE section bytes exceed image: %d > %d", image.name, accounted, len(image.artifact.Bytes))
			}
			sectionRows = append(sectionRows, map[string]any{
				"lane": image.name, "exe_bytes": len(image.artifact.Bytes),
				"exe_sha256":   fmt.Sprintf("%x", sha256.Sum256(image.artifact.Bytes)),
				"header_bytes": pe.SizeOfHeaders, "file_alignment": pe.FileAlignment,
				"sections": sections, "unattributed_bytes": uint64(len(image.artifact.Bytes)) - accounted,
			})
		}
		sectionReport, marshalErr := json.Marshal(map[string]any{
			"schema": "fairyc.vector-lexer-c-budget-sections/v1", "images": sectionRows,
		})
		if marshalErr != nil {
			t.Fatal(marshalErr)
		}
		t.Log("VECTOR_LEXER_C_BUDGET_SECTIONS " + string(sectionReport))
	}
	if os.Getenv("FAIRYC_VECTOR_LEXER_C_BUDGET_OBJECT_SECTIONS") == "1" {
		objectRows := make([]map[string]any, 0, len(images))
		for _, image := range images {
			objects := make([]map[string]any, 0, len(image.artifact.Objects))
			for _, object := range image.artifact.Objects {
				coff, inspectErr := pecoff.InspectObject(object.Bytes)
				if inspectErr != nil {
					t.Fatalf("%s/%s COFF inspection: %v", image.name, object.Name, inspectErr)
				}
				var codeBytes, otherBytes uint64
				for _, section := range coff.Sections {
					if section.Characteristics&0x20 != 0 { // IMAGE_SCN_CNT_CODE
						codeBytes += uint64(section.RawSize)
					} else {
						otherBytes += uint64(section.RawSize)
					}
				}
				objects = append(objects, map[string]any{
					"name": object.Name, "object_bytes": len(object.Bytes),
					"code_raw_bytes": codeBytes, "other_raw_bytes": otherBytes,
					"sections": len(coff.Sections), "relocations": coff.Relocations,
				})
			}
			objectRows = append(objectRows, map[string]any{
				"lane": image.name, "exe_bytes": len(image.artifact.Bytes),
				"exe_sha256": fmt.Sprintf("%x", sha256.Sum256(image.artifact.Bytes)),
				"objects":    objects,
			})
		}
		objectReport, marshalErr := json.Marshal(map[string]any{
			"schema": "fairyc.vector-lexer-c-budget-object-sections/v1", "images": objectRows,
		})
		if marshalErr != nil {
			t.Fatal(marshalErr)
		}
		t.Log("VECTOR_LEXER_C_BUDGET_OBJECT_SECTIONS " + string(objectReport))
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
		if image.name == "independent C" || image.name == "geometric C" {
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
	smokeSchema := "fairyc.vector-lexer-c-budget-smoke/v1"
	scope := "Three standalone images, one diagnostic execution per lane; no 8% runtime verdict or timing-stability claim."
	if bothStorage {
		smokeSchema = "fairyc.vector-lexer-c-budget-smoke/v2"
		scope = "Four standalone images including both C storage modes; one diagnostic execution per lane, no 8% verdict."
	}
	report, err := json.Marshal(map[string]any{
		"schema": smokeSchema, "sample_sha256": fmt.Sprintf("%x", sha256.Sum256([]byte(sample))),
		"msvc": toolchain.Version, "profile": cdriver.OptimizedProfile.Name,
		"rounds": rounds, "per_round_result": perRound, "lanes": rows,
		"scope": scope,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Log("VECTOR_LEXER_C_BUDGET_SMOKE " + string(report))
	if os.Getenv("FAIRYC_VECTOR_LEXER_C_BUDGET_MEASURE") == "1" {
		measureScannerBudgetRotations(t, images, sample, uint64(perRound))
	}
}

// Twelve fixed, rotating image sets follow one warmup. Raw QPC samples are
// evidence for a later stability and 8% assessment, not an automatic verdict.
func measureScannerBudgetRotations(t *testing.T, images []scannerBudgetImage, sample string, perRound uint64) {
	t.Helper()
	if len(images) != 3 && len(images) != 4 {
		t.Fatalf("budget lane count=%d", len(images))
	}
	const rounds = 250000
	const measured = 12
	const seed = 12345
	prefixes := make([]string, len(images))
	rows := make([]map[string]any, 0, measured+1)
	for triplet := 0; triplet <= measured; triplet++ {
		order := make([]string, len(images))
		lanes := make([]map[string]any, 0, len(images))
		for position := range images {
			index := (triplet + position) % len(images)
			image := images[index]
			order[position] = image.name
			outcome, err := cdriver.ExecuteWindowsMSVCTargetWithInput(image.artifact,
				strings.NewReader(fmt.Sprintf("%d %d\n", seed, rounds)))
			if err != nil || outcome.ExitCode != 0 || outcome.Stderr != "" || outcome.TrapControl != "" {
				t.Fatalf("triplet %d lane %s: %+v %v", triplet, image.name, outcome, err)
			}
			fields := strings.Fields(outcome.Stdout)
			if len(fields) < 5 {
				t.Fatalf("triplet %d lane %s missing timing: %q", triplet, image.name, outcome.Stdout)
			}
			prefix := strings.Join(fields[:len(fields)-4], " ")
			if prefixes[index] == "" {
				prefixes[index] = prefix
			} else if prefixes[index] != prefix {
				t.Fatalf("triplet %d lane %s reporter changed", triplet, image.name)
			}
			last := fields[len(fields)-4:]
			completed, completedErr := strconv.ParseUint(last[0], 10, 32)
			ticks, ticksErr := strconv.ParseUint(last[1], 10, 64)
			frequency, frequencyErr := strconv.ParseUint(last[2], 10, 64)
			total, totalErr := strconv.ParseUint(last[3], 10, 64)
			if completedErr != nil || ticksErr != nil || frequencyErr != nil || totalErr != nil ||
				completed != rounds || ticks == 0 || frequency == 0 || total != perRound*rounds {
				t.Fatalf("triplet %d lane %s timing=%v errors=%v,%v,%v,%v", triplet, image.name,
					last, completedErr, ticksErr, frequencyErr, totalErr)
			}
			lanes = append(lanes, map[string]any{
				"lane": image.name, "qpc_ticks": ticks, "qpc_frequency": frequency,
				"total": total, "completed": completed,
			})
		}
		row := map[string]any{"warmup": triplet == 0, "order": order, "lanes": lanes}
		if len(images) == 4 {
			row["rotation"] = triplet
		} else {
			row["triplet"] = triplet
		}
		rows = append(rows, row)
	}
	imageRows := make([]map[string]any, 0, len(images))
	for _, image := range images {
		imageRows = append(imageRows, map[string]any{
			"lane": image.name, "exe_bytes": len(image.artifact.Bytes),
			"exe_sha256":   fmt.Sprintf("%x", sha256.Sum256(image.artifact.Bytes)),
			"source_bytes": image.sourceBytes, "specializations": image.specializations,
		})
	}
	reportValues := map[string]any{
		"sample_sha256": fmt.Sprintf("%x", sha256.Sum256([]byte(sample))),
		"go":            runtime.Version(), "msvc": images[0].artifact.Toolchain.Version,
		"gomaxprocs": runtime.GOMAXPROCS(0), "seed": seed,
		"profile":          cdriver.OptimizedProfile.Name,
		"rounds":           rounds,
		"per_round_result": perRound, "images": imageRows, "samples": rows,
		"scope": "Standalone optimized images, one fixed warmup and twelve rotating measured sets; each execution verifies reporter, completed rounds and checksum. QPC excludes process startup and final output. No retries, exclusions or adaptive stopping. Token reporting is outside the timed loop; this sample does not prove a full-language 8% budget.",
	}
	if len(images) == 4 {
		reportValues["schema"] = "fairyc.vector-lexer-c-budget-rotations/v2"
		reportValues["warmup_rotations"] = 1
		reportValues["measured_rotations"] = measured
	} else {
		reportValues["schema"] = "fairyc.vector-lexer-c-budget-triplets/v1"
		reportValues["warmup_triplets"] = 1
		reportValues["measured_triplets"] = measured
		reportValues["scope"] = "Three standalone optimized images, one fixed warmup and twelve rotating measured triplets; each execution verifies reporter, completed rounds and checksum. QPC excludes process startup and final output. No retries, exclusions or adaptive stopping. Token reporting is outside the timed loop; this sample does not prove a full-language 8% budget."
	}
	report, err := json.Marshal(reportValues)
	if err != nil {
		t.Fatal(err)
	}
	if len(images) == 4 {
		t.Log("VECTOR_LEXER_C_BUDGET_ROTATIONS " + string(report))
	} else {
		t.Log("VECTOR_LEXER_C_BUDGET_TRIPLETS " + string(report))
	}
}

func scannerBudgetFairyProgram(t *testing.T, variant realScannerVectorVariant) (mir.Program, string, int) {
	return scannerBudgetFairyProgramWithMain(t, variant, realScannerExecutionMain)
}

func scannerBudgetFairyProgramWithMain(t *testing.T, variant realScannerVectorVariant, mainSource string) (mir.Program, string, int) {
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
	writeVectorCacheFile(t, root, "src/main.fy", mainSource)
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
