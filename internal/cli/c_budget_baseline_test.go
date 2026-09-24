package cli

import (
	"bytes"
	"context"
	"debug/pe"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/sdkstdlib"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/workspacecompiler"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

// This opt-in pilot compares separately authored source, never generated C
// against itself. It does not establish a language-wide 8% performance claim.
func TestCBudgetBaseline(t *testing.T) {
	if os.Getenv("FAIRYC_C_BUDGET") != "1" {
		t.Skip("opt-in C baseline")
	}
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Fatal("Windows AMD64 MSVC required")
	}
	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	fixtures := filepath.Join("..", "..", "tools", "benchmarks", "c-budget")
	read := func(name string) string {
		b, err := os.ReadFile(filepath.Join(fixtures, name))
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	workloads := []string{"checksum", "classifier"}
	if os.Getenv("FAIRYC_C_BUDGET_RESULT_RETURN_ONLY") == "1" {
		workloads = []string{"result_return"}
	} else if os.Getenv("FAIRYC_C_BUDGET_BYTE_VIEW_ONLY") == "1" {
		workloads = []string{"byte_view"}
	} else if os.Getenv("FAIRYC_C_BUDGET_BYTE_INDEX_ONLY") == "1" {
		workloads = []string{"byte_index"}
	} else if os.Getenv("FAIRYC_C_BUDGET_DEREF_ONLY") == "1" {
		workloads = []string{"deref"}
	} else if os.Getenv("FAIRYC_C_BUDGET_VECTOR_ONLY") == "1" {
		workloads = []string{"vector_slot"}
	} else if os.Getenv("FAIRYC_C_BUDGET_VECTOR_READ_ONLY") == "1" {
		workloads = []string{"vector_read_index"}
	} else if os.Getenv("FAIRYC_C_BUDGET_VECTOR_READ_CANDIDATE_ONLY") == "1" {
		workloads = []string{"vector_read_candidate"}
	} else if os.Getenv("FAIRYC_C_BUDGET_INCLUDE_DEREF") == "1" {
		workloads = append(workloads, "deref")
	}
	repetitions := 5
	if os.Getenv("FAIRYC_C_BUDGET_SMOKE") == "1" {
		repetitions = 1
	}
	oneTU := os.Getenv("FAIRYC_C_BUDGET_ONE_TU") == "1"
	measurePhases := os.Getenv("FAIRYC_C_BUDGET_PHASES") == "1"
	profile := cdriver.OptimizedProfile
	if os.Getenv("FAIRYC_C_BUDGET_LOW_PROFILE") == "1" {
		profile = cdriver.LowProfile
	}
	flags := []string{"/std:c11", "/TC", "/W4", "/WX", "/MT", "/O2", "/INCREMENTAL:NO"}
	if profile == cdriver.LowProfile {
		flags[5] = "/Od"
	}
	if oneTU && (len(workloads) != 1 || workloads[0] != "vector_slot") {
		t.Fatal("one-translation-unit diagnostic requires vector_slot only")
	}
	if oneTU && profile == cdriver.LowProfile {
		t.Fatal("one-translation-unit diagnostic requires optimized profile")
	}
	fixtureNames := []string{"support.c"}
	for _, workload := range workloads {
		fixtureNames = append(fixtureNames, workload+".fy", cBudgetCSource(workload))
	}
	hashes := map[string]string{}
	for _, name := range fixtureNames {
		hashes[name] = tokenIndexHash([]byte(read(name)))
	}
	samples := []map[string]any{}
	boundary := [][2]uint32{{0, 0}, {65520, 0}, {0, 1}, {65520, 1}, {12345, 257}}
	for _, workload := range workloads {
		root := t.TempDir()
		writeTestSource(t, root, "fairy.toml", "schema = \"fairyc.manifest/v1\"\n[package]\nuid = \"test.c-budget\"\nname = \"c-budget\"\nversion = \"1.0.0\"\nlanguage = \""+sdkstdlib.LanguageRevision+"\"\nsource = \"src\"\n")
		writeTestSource(t, root, "src/main.fy", read(workload+".fy"))
		if workload == "vector_read_candidate" {
			candidateWorkspace, err := workspaceloader.LoadVectorCandidate(root)
			if err != nil {
				t.Fatal(err)
			}
			stable, err := workspacecompiler.Check(candidateWorkspace)
			if err != nil || stable.Verified || len(stable.Semantic.Unsupported) == 0 {
				t.Fatalf("stable vector source gate changed: %v %+v", err, stable.Semantic)
			}
		}
		expected := cBudgetReference(workload, 12345, 50000000)
		for repetition := -1; repetition < repetitions; repetition++ {
			lanes := []string{"c", "fairy"}
			if repetition%2 == 0 {
				lanes = []string{"fairy", "c"}
			}
			for _, lane := range lanes {
				// Warm tool discovery is common and outside the measurement. Source
				// read/load, complete translation, C compile/link, artifact read/write
				// are inside; process startup and workspace scaffolding are outside.
				start := time.Now()
				support := read("support.c")
				var unit string
				var phaseNS map[string]int64
				if measurePhases {
					phaseNS = make(map[string]int64, 5)
				}
				if lane == "c" {
					unit = read(cBudgetCSource(workload))
					if measurePhases {
						phaseNS["load_ns"] = time.Since(start).Nanoseconds()
					}
				} else {
					candidate := workload == "vector_read_candidate"
					var w workspaceloader.Workspace
					var err error
					if candidate {
						w, err = workspaceloader.LoadVectorCandidate(root)
					} else {
						w, err = workspaceloader.Load(root)
					}
					if err != nil {
						t.Fatal(err)
					}
					var phaseStart time.Time
					if measurePhases {
						phaseNS["load_ns"] = time.Since(start).Nanoseconds()
						phaseStart = time.Now()
					}
					var checked compiler.Result
					if candidate {
						input, inputErr := w.SemanticInput()
						if inputErr != nil {
							t.Fatal(inputErr)
						}
						coordinates, coordinateErr := w.PackageCoordinates()
						if coordinateErr != nil {
							t.Fatal(coordinateErr)
						}
						owners, ownerErr := w.VectorCandidateStorageOwners()
						if ownerErr != nil || len(owners) != 1 {
							t.Fatalf("vector candidate SDK authority: %v owners=%v", ownerErr, owners)
						}
						analysis, analysisErr := compiler.AnalyzeSDKVectorCandidatePackages(input, coordinates, genericplan.DefaultPlanLimits(), owners)
						if analysisErr != nil || analysis.Analysis.Semantic.CompletedThrough != semantic.PhaseFlow || len(analysis.Analysis.Semantic.Diagnostics) != 0 || len(analysis.Analysis.Semantic.Unsupported) != 0 {
							t.Fatalf("vector candidate source and Flow: %v diagnostics=%+v unsupported=%+v", analysisErr, analysis.Analysis.Semantic.Diagnostics, analysis.Analysis.Semantic.Unsupported)
						}
						checked, err = compiler.LowerAnalyzedStorageTransfers(analysis.Analysis)
					} else {
						checked, err = workspacecompiler.Check(w)
					}
					if err != nil || !checked.Verified {
						t.Fatalf("check: %v %+v %+v", err, checked.Semantic.Diagnostics, checked.Semantic.Unsupported)
					}
					if measurePhases {
						phaseNS["check_ns"] = time.Since(phaseStart).Nanoseconds()
						phaseStart = time.Now()
					}
					emit := cbackend.Emit
					if workload == "vector_slot" || workload == "vector_read_index" || candidate {
						storage, err := workspaceSDKStorage(w)
						if err != nil || !storage {
							t.Fatalf("vector SDK storage authority: %v, %v", storage, err)
						}
						emit = cbackend.EmitStorageTransfers
					}
					emitted, err := emit(checked.MIR, tokenScannerEntry(t, checked), cbackend.Stage0WindowsMSVCProfile())
					if err != nil {
						t.Fatal(err)
					}
					for _, symbol := range emitted.ForeignSymbols {
						if symbol != "fc_budget_seed" && symbol != "fc_budget_rounds" && symbol != "fc_budget_begin" && symbol != "fc_budget_finish" && !((workload == "vector_slot" || workload == "vector_read_index" || candidate) && strings.HasPrefix(symbol, "fy_private_vector_")) {
							t.Fatalf("unexpected foreign symbol %s", symbol)
						}
					}
					var product string
					if workload == "vector_slot" || workload == "vector_read_index" || candidate {
						var vectorSymbols []string
						for _, symbol := range emitted.ForeignSymbols {
							if strings.HasPrefix(symbol, "fy_private_vector_") {
								vectorSymbols = append(vectorSymbols, symbol)
							}
						}
						product, err = windowsVectorSupportSource(vectorSymbols, false)
					} else {
						product, err = windowsTargetSupportSource(nil)
					}
					if err != nil {
						t.Fatal(err)
					}
					support = product + "\n" + support
					unit = emitted.Source
					if measurePhases {
						phaseNS["emit_support_ns"] = time.Since(phaseStart).Nanoseconds()
					}
				}
				translationNS := time.Since(start).Nanoseconds()
				var nativeStart time.Time
				if measurePhases {
					nativeStart = time.Now()
				}
				var artifact cdriver.WindowsHostArtifact
				if oneTU && lane == "fairy" {
					artifact, err = cdriver.BuildWindowsMSVCProbe(toolchain, cdriver.OptimizedProfile, support+"\n"+unit)
				} else {
					artifact, err = cdriver.BuildWindowsMSVCTarget(toolchain, profile, unit, support)
				}
				if err != nil {
					t.Fatal(err)
				}
				if measurePhases {
					phaseNS["msvc_build_ns"] = time.Since(nativeStart).Nanoseconds()
					nativeStart = time.Now()
				}
				if err = os.WriteFile(filepath.Join(root, lane+".exe"), artifact.Bytes, 0600); err != nil {
					t.Fatal(err)
				}
				if measurePhases {
					phaseNS["exe_write_ns"] = time.Since(nativeStart).Nanoseconds()
				}
				buildNS := time.Since(start).Nanoseconds()
				if repetition == -1 {
					cBudgetInspect(t, toolchain, workload, lane, unit, artifact)
					for _, input := range boundary {
						cBudgetExecute(t, artifact, input[0], input[1], cBudgetReference(workload, input[0], input[1]))
					}
				}
				kernelNS := cBudgetExecute(t, artifact, 12345, 50000000, expected)
				image, err := pe.NewFile(bytes.NewReader(artifact.Bytes))
				if err != nil {
					t.Fatal(err)
				}
				sections := map[string]uint32{}
				for _, section := range image.Sections {
					sections[section.Name] = section.VirtualSize
				}
				image.Close()
				objects := map[string]int{}
				for _, object := range artifact.Objects {
					objects[object.Name] = len(object.Bytes)
				}
				if repetition >= 0 {
					sample := map[string]any{"workload": workload, "lane": lane, "repetition": repetition, "build_ns": buildNS, "translation_ns": translationNS, "kernel_ns": kernelNS, "exe_bytes": len(artifact.Bytes), "object_bytes": objects, "pe_virtual_section_bytes": sections, "unit_bytes": len(unit), "unit_sha256": tokenIndexHash([]byte(unit)), "support_sha256": tokenIndexHash([]byte(support)), "result": expected}
					if measurePhases {
						sample["phase_ns"] = phaseNS
					}
					samples = append(samples, sample)
				}
			}
		}
	}
	report := map[string]any{"schema": "fairyc.c-budget-baseline/v1", "compiler_source": os.Getenv("FAIRYC_C_BUDGET_SOURCE"), "go": runtime.Version(), "msvc": toolchain.Version, "gomaxprocs": runtime.GOMAXPROCS(0), "flags": flags, "profile": profile.Name, "one_translation_unit": oneTU, "source_sha256": hashes, "boundary_cases": boundary, "boundary_checks": len(workloads) * len(boundary) * 2, "seed": 12345, "rounds": 50000000, "samples": samples,
		"method": fmt.Sprintf("Independently authored scalar kernels; optional deref, vector_slot, vector_read_index, vector_read_candidate, result_return and byte-loop pairs each compare the same algorithm with C. Vector_read_candidate uses the compiler-private authenticated SDK and source/Flow/MIR proof lane, not stable source admission. Byte_view and byte_index share one independent C loop and differ only in the Fairy inner-loop spelling; vector_read_index and vector_read_candidate likewise share one independent C workload. Same stdin input and QPC/output support. All workloads guard seed<65521 and rounds<=50000000; arithmetic is proven in-range over admitted inputs. Fairy retains ordinary checked arithmetic and full required product support; C does not link Fairy support. %d alternating paired samples after one build/run warmup per workload/lane. Kernel time excludes input parsing, process startup and printing; result is consumed and independently checked in Go. Build time covers source read/load, frontend analysis/verified MIR, C emission, C compile/link, artifact reads and final EXE write; common warmed toolchain discovery, harness process startup, workspace setup and execution excluded. No persistent package cache; ambient OS caches uncontrolled. No PDB, runtime-speed generalization, Linux, real parser/container workload or cold CLI startup claim.", repetitions), "budget": map[string]any{"runtime_ratio": 1.08, "exe_ratio": 1.08, "build_ratio_observation": 1.08, "scope": "cumulative against equivalent C per workload; never a per-feature allowance; metadata tracked separately"}}
	if measurePhases {
		report["phase_method"] = "Opt-in wall-clock subdivisions: load includes support-file read and workspace load or C source read; check covers the verified workspace compile; emit_support covers authority check, C emission and support assembly; msvc_build covers C compile, link and artifact read; exe_write covers final file write. Phases exclude harness startup, workspace setup, warm toolchain discovery and program execution."
	}
	encoded, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("C_BUDGET_REPORT " + string(encoded))
}

func cBudgetExecute(t *testing.T, artifact cdriver.WindowsHostArtifact, seed, rounds, expected uint32) int64 {
	t.Helper()
	outcome, err := cdriver.ExecuteWindowsMSVCTargetWithInput(artifact, strings.NewReader(fmt.Sprintf("%d %d\n", seed, rounds)))
	if err != nil || outcome.ExitCode != 0 || outcome.Stderr != "" {
		t.Fatalf("execution: %+v %v", outcome, err)
	}
	fields := strings.Fields(outcome.Stdout)
	if len(fields) != 3 {
		t.Fatalf("invalid timing output %q", outcome.Stdout)
	}
	value, err := strconv.ParseUint(fields[0], 10, 32)
	if err != nil || uint32(value) != expected {
		t.Fatalf("result=%s expected=%d", fields[0], expected)
	}
	ticks, err := strconv.ParseInt(fields[1], 10, 64)
	if err != nil || ticks < 0 {
		t.Fatal("invalid ticks")
	}
	frequency, err := strconv.ParseInt(fields[2], 10, 64)
	if err != nil || frequency <= 0 {
		t.Fatal("invalid frequency")
	}
	return ticks * 1000000000 / frequency
}

func cBudgetReference(workload string, seed, rounds uint32) uint32 {
	value := seed
	if workload == "classifier" || workload == "result_return" {
		value = 0
	}
	for i := uint32(0); i < rounds; i++ {
		if workload == "byte_view" || workload == "byte_index" || workload == "vector_read_index" || workload == "vector_read_candidate" {
			for _, b := range [...]uint32{7, 11, 13, 17} {
				value = (value*33 + b + i%251) % 65521
			}
		} else if workload == "checksum" || workload == "deref" || workload == "vector_slot" {
			value = (value*33 + i%251) % 65521
		} else if workload == "result_return" {
			byteValue := (i + seed) % 128
			decoded := uint32(3)
			if byteValue%8 != 0 {
				decoded = byteValue + 1
			}
			value = (value + decoded) % 65521
		} else {
			b := (i + seed) % 128
			kind := uint32(4)
			if b == 32 || b == 10 {
				kind = 1
			} else if b >= 48 && b <= 57 {
				kind = 2
			} else if b >= 65 && b <= 90 {
				kind = 3
			}
			value = (value + kind) % 65521
		}
	}
	return value
}

func cBudgetCSource(workload string) string {
	if workload == "byte_view" || workload == "byte_index" {
		return "byte_loop.c"
	}
	if workload == "vector_read_index" || workload == "vector_read_candidate" {
		return "vector_read.c"
	}
	return workload + ".c"
}

// Inspection is outside timed builds and only requested by the bounded runner.
func cBudgetInspect(t *testing.T, toolchain cdriver.WindowsHostToolchain, workload, lane, unit string, artifact cdriver.WindowsHostArtifact) {
	t.Helper()
	phase := os.Getenv("FAIRYC_C_BUDGET_PHASE")
	if phase == "" {
		return
	}
	if phase != "before" && phase != "after" {
		t.Fatal("invalid inspection phase")
	}
	root := filepath.Join("..", "..", ".cache", "constant-divisors", phase, workload, lane)
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "unit.c"), []byte(unit), 0600); err != nil {
		t.Fatal(err)
	}
	for _, object := range artifact.Objects {
		if object.Name != "unit.obj" {
			continue
		}
		path := filepath.Join(root, "unit.obj")
		if err := os.WriteFile(path, object.Bytes, 0600); err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		command := exec.CommandContext(ctx, filepath.Join(filepath.Dir(toolchain.Path), "dumpbin.exe"), "/DISASM", path)
		command.Env = toolchain.Environment
		out, err := command.CombinedOutput()
		cancel()
		if err != nil {
			t.Fatalf("dumpbin: %v %s", err, out)
		}
		if err := os.WriteFile(filepath.Join(root, "unit.asm"), out, 0600); err != nil {
			t.Fatal(err)
		}
		return
	}
	t.Fatal("unit.obj missing")
}
