package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/buildidentity"
	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/sdkstdlib"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/workspacecompiler"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

// Build three images once, then rotate execution order without rebuilding.
// The candidate lane remains compiler-private; no stable source gate changes.
func TestVectorReadThreeWayExecution(t *testing.T) {
	if os.Getenv("FAIRYC_VECTOR_READ_THREE_WAY") != "1" {
		t.Skip("opt-in three-way vector-read comparison")
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
	fixtures := filepath.Join("..", "..", "tools", "benchmarks", "c-budget")
	read := func(name string) []byte {
		data, readErr := os.ReadFile(filepath.Join(fixtures, name))
		if readErr != nil {
			t.Fatal(readErr)
		}
		return data
	}
	sources := map[string]string{}
	for _, name := range []string{"support.c", "vector_read.c", "vector_read_index.fy", "vector_read_candidate.fy"} {
		sources[name] = tokenIndexHash(read(name))
	}
	type image struct {
		name     string
		artifact cdriver.WindowsHostArtifact
		hash     string
	}
	images := []image{{name: "c"}, {name: "index"}, {name: "candidate"}}
	builds := []map[string]any{}
	for lane := range images {
		name := images[lane].name
		support := string(read("support.c"))
		unit := string(read("vector_read.c"))
		if name != "c" {
			root := t.TempDir()
			writeTestSource(t, root, "fairy.toml", "schema = \"fairyc.manifest/v1\"\n[package]\nuid = \"test.vector-read-three-way\"\nname = \"vector-read-three-way\"\nversion = \"1.0.0\"\nlanguage = \""+sdkstdlib.LanguageRevision+"\"\nsource = \"src\"\n")
			fixture := "vector_read_index.fy"
			if name == "candidate" {
				fixture = "vector_read_candidate.fy"
			}
			writeTestSource(t, root, "src/main.fy", string(read(fixture)))
			var workspace workspaceloader.Workspace
			if name == "candidate" {
				workspace, err = workspaceloader.LoadVectorCandidate(root)
			} else {
				workspace, err = workspaceloader.Load(root)
			}
			if err != nil {
				t.Fatal(err)
			}
			var checked compiler.Result
			if name == "candidate" {
				stable, stableErr := workspacecompiler.Check(workspace)
				if stableErr != nil || stable.Verified || len(stable.Semantic.Unsupported) == 0 {
					t.Fatalf("stable vector source gate changed: %v %+v", stableErr, stable.Semantic)
				}
				input, inputErr := workspace.SemanticInput()
				coordinates, coordinateErr := workspace.PackageCoordinates()
				owners, ownerErr := workspace.VectorCandidateStorageOwners()
				if inputErr != nil || coordinateErr != nil || ownerErr != nil || len(owners) != 1 {
					t.Fatalf("candidate input/authority: %v %v %v owners=%v", inputErr, coordinateErr, ownerErr, owners)
				}
				analysis, analysisErr := compiler.AnalyzeSDKVectorCandidatePackages(input, coordinates, genericplan.DefaultPlanLimits(), owners)
				if analysisErr != nil || analysis.Analysis.Semantic.CompletedThrough != semantic.PhaseFlow || len(analysis.Analysis.Semantic.Diagnostics) != 0 || len(analysis.Analysis.Semantic.Unsupported) != 0 {
					t.Fatalf("candidate Flow: %v diagnostics=%+v unsupported=%+v", analysisErr, analysis.Analysis.Semantic.Diagnostics, analysis.Analysis.Semantic.Unsupported)
				}
				checked, err = compiler.LowerAnalyzedStorageTransfers(analysis.Analysis)
			} else {
				checked, err = workspacecompiler.Check(workspace)
			}
			if err != nil || !checked.Verified {
				t.Fatalf("%s verified MIR: %v diagnostics=%+v unsupported=%+v", name, err, checked.Semantic.Diagnostics, checked.Semantic.Unsupported)
			}
			storage, authorityErr := workspaceSDKStorage(workspace)
			if authorityErr != nil || !storage {
				t.Fatalf("%s SDK storage authority: %v %v", name, storage, authorityErr)
			}
			emitted, emitErr := cbackend.EmitStorageTransfers(checked.MIR, tokenScannerEntry(t, checked), cbackend.Stage0WindowsMSVCProfile())
			if emitErr != nil {
				t.Fatal(emitErr)
			}
			vectorSymbols := []string{}
			for _, symbol := range emitted.ForeignSymbols {
				if strings.HasPrefix(symbol, "fy_private_vector_") {
					vectorSymbols = append(vectorSymbols, symbol)
					continue
				}
				if symbol != "fc_budget_seed" && symbol != "fc_budget_rounds" && symbol != "fc_budget_begin" && symbol != "fc_budget_finish" {
					t.Fatalf("unexpected foreign symbol %s", symbol)
				}
			}
			product, supportErr := windowsVectorSupportSource(vectorSymbols, false)
			if supportErr != nil {
				t.Fatal(supportErr)
			}
			support = product + "\n" + support
			unit = emitted.Source
		}
		artifact, buildErr := cdriver.BuildWindowsMSVCTarget(toolchain, cdriver.OptimizedProfile, unit, support)
		if buildErr != nil {
			t.Fatal(buildErr)
		}
		images[lane].artifact = artifact
		images[lane].hash = tokenIndexHash(artifact.Bytes)
		builds = append(builds, map[string]any{"lane": name, "exe_sha256": images[lane].hash, "exe_bytes": len(artifact.Bytes), "unit_sha256": tokenIndexHash([]byte(unit)), "unit_bytes": len(unit), "support_sha256": tokenIndexHash([]byte(support))})
	}
	expected := cBudgetReference("vector_read_index", 12345, 50000000)
	boundary := [][2]uint32{{0, 0}, {65520, 0}, {0, 1}, {65520, 1}, {12345, 257}}
	for _, input := range boundary {
		want := cBudgetReference("vector_read_index", input[0], input[1])
		for _, lane := range images {
			cBudgetExecute(t, lane.artifact, input[0], input[1], want)
		}
	}
	permutations := [6][3]int{{0, 1, 2}, {1, 2, 0}, {2, 0, 1}, {2, 1, 0}, {1, 0, 2}, {0, 2, 1}}
	pairs, warmups := executionPairs, 3
	if os.Getenv("FAIRYC_VECTOR_READ_THREE_WAY_SMOKE") == "1" {
		pairs, warmups = 1, 1
	}
	rows := []map[string]any{}
	measured := [3][]float64{}
	for repetition := -warmups; repetition < pairs; repetition++ {
		order := permutations[((repetition + 6) % 6)]
		for position, lane := range order {
			ns := cBudgetExecute(t, images[lane].artifact, 12345, 50000000, expected)
			if ns <= 0 {
				t.Fatal("nonpositive kernel duration")
			}
			rows = append(rows, map[string]any{"lane": images[lane].name, "repetition": repetition, "warmup": repetition < 0, "position": position, "kernel_ns": ns, "result": expected, "exe_sha256": images[lane].hash})
			if repetition >= 0 {
				measured[lane] = append(measured[lane], float64(ns))
			}
		}
	}
	for _, lane := range images {
		if tokenIndexHash(lane.artifact.Bytes) != lane.hash {
			t.Fatal("artifact changed during measurement")
		}
	}
	summary := map[string]any{}
	for lane, name := range []string{"c", "index", "candidate"} {
		summary[name+"_ns"] = executionDescribe(measured[lane])
	}
	for _, comparison := range []struct {
		name      string
		baseline  int
		candidate int
	}{{"index_over_c", 0, 1}, {"candidate_over_c", 0, 2}, {"candidate_over_index", 1, 2}} {
		ratios := make([]float64, pairs)
		for i := range ratios {
			ratios[i] = measured[comparison.candidate][i] / measured[comparison.baseline][i]
		}
		summary[comparison.name] = map[string]any{"distribution": executionDescribe(ratios), "ratios_in_order": ratios, "ratio_of_medians": executionMedian(measured[comparison.candidate]) / executionMedian(measured[comparison.baseline])}
	}
	harness, err := os.ReadFile("vector_read_three_way_execution_test.go")
	if err != nil {
		t.Fatal(err)
	}
	report := map[string]any{"schema": "fairyc.vector-read-three-way/v1", "compiler_source": revision, "test_executable_identity": identity.String(), "go": runtime.Version(), "msvc": toolchain.Version, "gomaxprocs": runtime.GOMAXPROCS(0), "flags": []string{"/std:c11", "/TC", "/W4", "/WX", "/MT", "/O2", "/INCREMENTAL:NO"}, "source_sha256": sources, "harness_sha256": tokenIndexHash(harness), "seed": 12345, "rounds": 50000000, "builds": builds, "samples": rows, "summary": summary, "warmup_triads": warmups, "measured_triads": pairs, "boundary_cases": boundary, "boundary_checks": len(boundary) * len(images), "native_executions": len(rows) + len(boundary)*len(images), "runtime_budget_ratio": executionBudget, "stability_span_limit": executionSpanLimit,
		"method": fmt.Sprintf("Build independent checked C, stable indexed Fairy and compiler-private vector-iteration Fairy images once each. Check stable rejection of candidate syntax and validate private source through authenticated SDK, Flow and verified MIR. Check five boundary inputs on every image. Reuse identical artifact bytes for %d warmup and %d measured triads, rotating all six lane orders; no rebuild, retry, removal or adaptive stopping. QPC kernel time excludes process startup, stdin parsing and output. Each result is independently checked in Go. Report raw samples and distributions without a promotion verdict; relative full spans above 10%% or inconsistent C-relative estimators defer interpretation. Ambient host load and frequency are uncontrolled.", warmups, pairs)}
	encoded, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("VECTOR_THREE_WAY_REPORT " + string(encoded))
}
