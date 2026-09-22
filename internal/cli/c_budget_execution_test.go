package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/0disoft/fairyc/internal/buildidentity"
	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/sdkstdlib"
	"github.com/0disoft/fairyc/internal/workspacecompiler"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

// Build all four images before executing any timed kernel. No retries or
// outlier removal: unstable runs remain in the report and defer the verdict.
func TestCBudgetExecutionOnly(t *testing.T) {
	if os.Getenv("FAIRYC_C_EXECUTION_ONLY") != "1" {
		t.Skip("opt-in execution-only C comparison")
	}
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Fatal("Windows AMD64 MSVC required")
	}
	revision := os.Getenv("FAIRYC_C_BUDGET_SOURCE")
	if revision == "" {
		t.Fatal("source revision required")
	}
	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	identity, err := buildidentity.Current()
	if err != nil {
		t.Fatal(err)
	}
	read := func(path string) []byte {
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	fixtures := filepath.Join("..", "..", "tools", "benchmarks", "c-budget")
	sources := map[string]string{}
	for _, name := range []string{"checksum.c", "checksum.fy", "classifier.c", "classifier.fy", "support.c"} {
		sources[name] = tokenIndexHash(read(filepath.Join(fixtures, name)))
	}
	harnesses := map[string]string{}
	for _, name := range []string{"c_budget_execution_test.go", "c_budget_execution_stats_test.go", "c_budget_baseline_test.go"} {
		harnesses[name] = tokenIndexHash(read(name))
	}
	type kernel struct {
		name      string
		expected  uint32
		artifacts [2]cdriver.WindowsHostArtifact
		hashes    [2]string
	}
	kernels := []kernel{{name: "checksum"}, {name: "classifier"}}
	builds := []map[string]any{}
	lanes := []string{"c", "fairy"}
	// There is no timer or execution in this entire build phase.
	for index := range kernels {
		k := &kernels[index]
		k.expected = cBudgetReference(k.name, 12345, 50000000)
		for lane, name := range lanes {
			support := string(read(filepath.Join(fixtures, "support.c")))
			unit := string(read(filepath.Join(fixtures, k.name+".c")))
			if name == "fairy" {
				root := t.TempDir()
				writeTestSource(t, root, "fairy.toml", "schema = \"fairyc.manifest/v1\"\n[package]\nuid = \"test.execution-budget\"\nname = \"execution-budget\"\nversion = \"1.0.0\"\nlanguage = \""+sdkstdlib.LanguageRevision+"\"\nsource = \"src\"\n")
				writeTestSource(t, root, "src/main.fy", string(read(filepath.Join(fixtures, k.name+".fy"))))
				w, err := workspaceloader.Load(root)
				if err != nil {
					t.Fatal(err)
				}
				checked, err := workspacecompiler.Check(w)
				if err != nil || !checked.Verified {
					t.Fatalf("check: %v %+v %+v", err, checked.Semantic.Diagnostics, checked.Semantic.Unsupported)
				}
				emitted, err := cbackend.Emit(checked.MIR, tokenScannerEntry(t, checked), cbackend.Stage0WindowsMSVCProfile())
				if err != nil {
					t.Fatal(err)
				}
				for _, symbol := range emitted.ForeignSymbols {
					if symbol != "fc_budget_seed" && symbol != "fc_budget_rounds" && symbol != "fc_budget_begin" && symbol != "fc_budget_finish" {
						t.Fatalf("unexpected symbol %s", symbol)
					}
				}
				product, err := windowsTargetSupportSource(nil)
				if err != nil {
					t.Fatal(err)
				}
				support = product + "\n" + support
				unit = emitted.Source
			}
			artifact, err := cdriver.BuildWindowsMSVCTarget(toolchain, cdriver.OptimizedProfile, unit, support)
			if err != nil {
				t.Fatal(err)
			}
			k.artifacts[lane] = artifact
			k.hashes[lane] = tokenIndexHash(artifact.Bytes)
			builds = append(builds, map[string]any{"workload": k.name, "lane": name, "exe_sha256": k.hashes[lane], "exe_bytes": len(artifact.Bytes), "unit_sha256": tokenIndexHash([]byte(unit)), "unit_bytes": len(unit), "support_sha256": tokenIndexHash([]byte(support))})
		}
	}
	boundary := [][2]uint32{{0, 0}, {65520, 0}, {0, 1}, {65520, 1}, {12345, 257}}
	rows := []map[string]any{}
	summaries := map[string]executionBudgetSummary{}
	for _, k := range kernels {
		for _, input := range boundary {
			for lane := range lanes {
				cBudgetExecute(t, k.artifacts[lane], input[0], input[1], cBudgetReference(k.name, input[0], input[1]))
			}
		}
		var measured [2][]float64
		for repetition := -3; repetition < executionPairs; repetition++ {
			order := []int{0, 1}
			if repetition%2 == 0 {
				order = []int{1, 0}
			}
			for position, lane := range order {
				ns := cBudgetExecute(t, k.artifacts[lane], 12345, 50000000, k.expected)
				if ns <= 0 {
					t.Fatal("nonpositive kernel duration")
				}
				rows = append(rows, map[string]any{"workload": k.name, "lane": lanes[lane], "repetition": repetition, "warmup": repetition < 0, "position": position, "kernel_ns": ns, "result": k.expected, "exe_sha256": k.hashes[lane]})
				if repetition >= 0 {
					measured[lane] = append(measured[lane], float64(ns))
				}
			}
		}
		summaries[k.name] = summarizeExecutionBudget(measured[0], measured[1])
		for lane := range lanes {
			if tokenIndexHash(k.artifacts[lane].Bytes) != k.hashes[lane] {
				t.Fatal("artifact changed during measurement")
			}
		}
	}
	report := map[string]any{"schema": "fairyc.c-execution-budget/v1", "compiler_source": revision, "test_executable_identity": identity.String(), "go": runtime.Version(), "msvc": toolchain.Version, "gomaxprocs": runtime.GOMAXPROCS(0), "logical_cpus": runtime.NumCPU(), "flags": []string{"/std:c11", "/TC", "/W4", "/WX", "/MT", "/O2", "/INCREMENTAL:NO"}, "source_sha256": sources, "harness_sha256": harnesses, "seed": 12345, "rounds": 50000000, "builds": builds, "samples": rows, "summary": summaries, "warmup_pairs": 3, "measured_pairs": executionPairs, "boundary_cases": boundary, "boundary_checks": 20, "native_executions": 80, "runtime_budget_ratio": executionBudget, "stability_span_limit": executionSpanLimit,
		"method": "Build all four images exactly once before execution. Reuse identical artifact bytes for 3 warmup pairs and 12 measured pairs per kernel, alternating first lane. The native driver materializes the same bytes in a fresh temporary executable path per invocation; QPC kernel time excludes that write, process startup, input parsing and output. Every result is checked against Go. No rebuilds, sample removal, retries, affinity/power changes or adaptive stopping. Build-time comparisons are intentionally absent. Ambient host load and frequency are uncontrolled. Report medians, paired ratios, min/max, median absolute deviation, relative full span and first-lane groups. Defer if either lane or paired ratios exceed 10% relative full span, or median estimators/order groups disagree about 1.08. Otherwise report only observed-within/exceeds-budget for these samples; never a language-wide or statistical certification."}
	encoded, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("C_EXECUTION_REPORT " + string(encoded))
}
