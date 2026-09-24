package compiler_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/cruntime"
	"github.com/0disoft/fairyc/internal/cstdlib"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

func TestPrivateVectorSDKCandidateRealScannerNativeParity(t *testing.T) {
	const before = `    var usize index = 0;
    var usize position = 0;
    while (index < v::len<Token>(ref tokens)) {
        let ref Token token = v::at<Token>(ref tokens, index);`
	const after = `    var usize position = 0;
    for (let ref Token token in ref tokens) {`
	const endBefore = `        position = token.end;
        index += 1;
    }
    assert(position == input.len);`
	const endAfter = `        position = token.end;
    }
    assert(position == input.len);`

	example := filepath.Join("..", "..", "examples", "lexer_candidate")
	scannerBytes, err := os.ReadFile(filepath.Join(example, "src", "scanner.fy"))
	if err != nil {
		t.Fatal(err)
	}
	scanner := string(scannerBytes)
	if strings.Count(scanner, before) != 1 || strings.Count(scanner, endBefore) != 1 {
		t.Fatal("real scanner loop changed; refresh the private candidate comparison")
	}
	outputs := make(map[string]map[string]string)
	for _, variant := range []struct {
		name      string
		candidate bool
		scanner   string
	}{
		{name: "explicit index", scanner: scanner},
		{name: "vector for", candidate: true, scanner: strings.Replace(strings.Replace(scanner, before, after, 1), endBefore, endAfter, 1)},
	} {
		t.Run(variant.name, func(t *testing.T) {
			root := t.TempDir()
			for _, name := range []string{"fairy.toml", "src/numbers.fy", "src/diagnostics.fy", "src/utf8.fy", "src/strings.fy", "src/budget.fy"} {
				data, err := os.ReadFile(filepath.Join(example, filepath.FromSlash(name)))
				if err != nil {
					t.Fatal(err)
				}
				writeVectorCacheFile(t, root, name, string(data))
			}
			writeVectorCacheFile(t, root, "src/scanner.fy", variant.scanner)
			writeVectorCacheFile(t, root, "src/main.fy", `module main;
import scanner as scanner;
@link_name("fairyc_fixture_vector_clean") unsafe extern C fn i32 clean();
@effects(console) fn i32 main() {
    let i32 empty = scanner::report_work(b""[..], false, 32768, 1048576, 256, 16777216);
    if (empty != 0) { return empty; }
    let i32 sample = scanner::report_work(b"let n=12345; /*open"[..], false, 32768, 1048576, 256, 16777216);
    assert(unsafe { clean() } == 1);
    return sample;
}`)
			workspace, err := workspaceloader.LoadVectorCandidate(root)
			if err != nil {
				t.Fatal(err)
			}
			input, err := workspace.SemanticInput()
			if err != nil {
				t.Fatal(err)
			}
			coordinates, err := workspace.PackageCoordinates()
			if err != nil {
				t.Fatal(err)
			}
			owners, err := workspace.VectorCandidateStorageOwners()
			if err != nil || len(owners) != 1 {
				t.Fatalf("candidate SDK authority: %v owners=%v", err, owners)
			}
			if variant.candidate {
				stable, err := compiler.AnalyzeSDKStoragePackages(input, coordinates, genericplan.DefaultPlanLimits(), owners)
				if err != nil || len(stable.Analysis.Semantic.Unsupported) == 0 {
					t.Fatalf("stable source gate changed: %v %+v", err, stable.Analysis.Semantic)
				}
			}
			var analysis compiler.GenericAnalysis
			if variant.candidate {
				analysis, err = compiler.AnalyzeSDKVectorCandidatePackages(input, coordinates, genericplan.DefaultPlanLimits(), owners)
			} else {
				analysis, err = compiler.AnalyzeSDKStoragePackages(input, coordinates, genericplan.DefaultPlanLimits(), owners)
			}
			if err != nil || analysis.Analysis.Semantic.CompletedThrough != semantic.PhaseFlow ||
				len(analysis.Analysis.Semantic.Diagnostics) != 0 || len(analysis.Analysis.Semantic.Unsupported) != 0 {
				t.Fatalf("real scanner source and Flow: %v diagnostics=%+v unsupported=%+v", err,
					analysis.Analysis.Semantic.Diagnostics, analysis.Analysis.Semantic.Unsupported)
			}
			result, err := compiler.LowerAnalyzedStorageTransfers(analysis.Analysis)
			if err != nil || !result.Verified {
				t.Fatalf("real scanner verified MIR: %v %+v", err, result.Semantic.Diagnostics)
			}
			if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
				return
			}
			rootID, err := workspace.RootPackageID()
			if err != nil {
				t.Fatal(err)
			}
			outputs[variant.name] = make(map[string]string)
			for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
				t.Run(profile.Name, func(t *testing.T) {
					outputs[variant.name][profile.Name] = runRealScannerVectorNative(t, result.MIR,
						hir.QualifiedModulePath(rootID, "main")+"::main", profile)
				})
			}
		})
	}
	if runtime.GOOS == "windows" && runtime.GOARCH == "amd64" {
		for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
			baseline, candidate := outputs["explicit index"][profile.Name], outputs["vector for"][profile.Name]
			if baseline == "" || candidate != baseline {
				t.Fatalf("%s scanner output differs: explicit=%q candidate=%q", profile.Name, baseline, candidate)
			}
		}
	}
}

func runRealScannerVectorNative(t *testing.T, program mir.Program, entry string, profile cdriver.Profile) string {
	t.Helper()
	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	unit, err := cbackend.EmitStorageTransfers(program, entry, cbackend.Stage0WindowsMSVCProfile())
	if err != nil {
		t.Fatal(err)
	}
	scalar, err := cruntime.ScalarSource(192)
	if err != nil {
		t.Fatal(err)
	}
	storage, err := cstdlib.WindowsMSVCVectorStorageSource()
	if err != nil {
		t.Fatal(err)
	}
	symbols := []string{"fy_std_system_heap_allocate_v0"}
	for _, symbol := range unit.ForeignSymbols {
		if strings.HasPrefix(symbol, "fy_std_") {
			symbols = append(symbols, symbol)
		}
	}
	sdk, err := cstdlib.WindowsMSVCSourceForSymbols(symbols)
	if err != nil {
		t.Fatal(err)
	}
	heap, err := cstdlib.WindowsMSVCSourceForSymbols(symbols[:1])
	if err != nil || !strings.HasPrefix(storage, heap) {
		t.Fatalf("private scanner storage heap boundary: %v", err)
	}
	artifact, err := cdriver.BuildWindowsHost(toolchain, profile, unit.Source,
		scalar+vectorStorageAllocator+sdk+strings.TrimPrefix(storage, heap)+vectorForNativeProbe)
	if err != nil {
		t.Fatal(err)
	}
	outcome, err := cdriver.ExecuteWindowsHost(artifact)
	if err != nil || outcome.ExitCode != 0 || outcome.Stderr != "" || outcome.TrapControl != "" {
		t.Fatalf("native scanner: %+v %v", outcome, err)
	}
	return outcome.Stdout
}
