package cli

import (
	"bytes"
	"debug/pe"
	"encoding/json"
	"os"
	"reflect"
	"runtime"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/buildidentity"
	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/nominalhir"
	"github.com/0disoft/fairyc/internal/target"
	"github.com/0disoft/fairyc/internal/workspacecompiler"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

// Mirror only Check's authenticated SDK lane, with annotation before lowering.
// The measured scalar and struct controls still use workspacecompiler.Check.
func checkNominalScannerWorkspace(t *testing.T, workspace workspaceloader.Workspace) (compiler.Result, error) {
	t.Helper()
	input, err := workspace.SemanticInput()
	if err != nil {
		return compiler.Result{}, err
	}
	owners, err := workspace.SDKStorageOwners()
	if err != nil {
		return compiler.Result{}, err
	}
	if len(owners) == 0 {
		t.Fatal("measurement requires the SDK storage lane")
	}
	coordinates, err := workspace.PackageCoordinates()
	if err != nil {
		return compiler.Result{}, err
	}
	analysis, err := compiler.AnalyzeSDKStoragePackages(input, coordinates, genericplan.DefaultPlanLimits(), owners)
	if err != nil {
		return compiler.Result{}, err
	}
	patch := nominalTokenScannerPatch(t, compiler.Result{Semantic: analysis.Analysis.Semantic})
	return nominalhir.LowerAnalyzedStoragePatch(analysis.Analysis, patch)
}

type nominalTotalSample struct {
	Repetition   int               `json:"repetition"`
	Variant      string            `json:"variant"`
	Profile      string            `json:"profile"`
	CheckNS      int64             `json:"check_ns"`
	NativeNS     int64             `json:"native_ns"`
	TotalNS      int64             `json:"total_ns"`
	GoAllocBytes uint64            `json:"go_alloc_bytes"`
	GoMallocs    uint64            `json:"go_mallocs"`
	GoGCs        uint32            `json:"go_gc_cycles"`
	CBytes       int               `json:"c_bytes"`
	CHash        string            `json:"c_sha256"`
	ExeBytes     int               `json:"exe_bytes"`
	Sections     map[string]uint32 `json:"pe_virtual_section_bytes"`
	Objects      map[string]int    `json:"object_bytes"`
	PDBBytes     int               `json:"pdb_bytes"`
	ExitCode     int               `json:"exit_code"`
}

func TestTokenIndexNominalTotalMSVC(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows AMD64 MSVC measurement")
	}
	variants := []string{"scalar", "nominal", "struct"}
	profiles := []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile}
	var samples []nominalTotalSample
	sources := map[string]map[string]string{}
	var msvc string
	// Warm each lane/profile once, then run two complete rotations. Force GC
	// outside each timed window to avoid inheriting the preceding lane's heap.
	for repetition := -1; repetition < 6; repetition++ {
		for turn := 0; turn < len(variants); turn++ {
			variant := variants[(turn+repetition+len(variants))%len(variants)]
			for _, profile := range profiles {
				root, hashes := tokenIndexWorkspace(t, variant == "struct")
				if prior := sources[variant]; prior != nil && !reflect.DeepEqual(prior, hashes) {
					t.Fatal("source corpus changed during measurement")
				}
				sources[variant] = hashes
				runtime.GC()
				var before, after runtime.MemStats
				runtime.ReadMemStats(&before)
				start := time.Now()
				workspace, _, err := loadWorkspaceWithLock(root, false)
				if err != nil {
					t.Fatal(err)
				}
				var checked compiler.Result
				if variant == "nominal" {
					checked, err = checkNominalScannerWorkspace(t, workspace)
				} else {
					checked, err = workspacecompiler.Check(workspace)
				}
				if err != nil || !checked.Verified {
					t.Fatalf("check %s: %v", variant, err)
				}
				checkNS := time.Since(start).Nanoseconds()
				entry := tokenScannerEntry(t, checked)
				var diagnostics bytes.Buffer
				nativeStart := time.Now()
				artifact, code := compileWindowsProgramPolicy(checked.MIR, entry, target.X8664WindowsMSVC, profile, false, true, &diagnostics)
				nativeNS := time.Since(nativeStart).Nanoseconds()
				totalNS := time.Since(start).Nanoseconds()
				runtime.ReadMemStats(&after)
				if code != ExitSuccess || diagnostics.Len() != 0 {
					t.Fatalf("native: %d %s", code, diagnostics.String())
				}
				if msvc != "" && msvc != artifact.Toolchain.Version {
					t.Fatal("MSVC changed during measurement")
				}
				msvc = artifact.Toolchain.Version
				// Inspection emission and execution are outside both time and allocation
				// windows. The native helper already emitted C inside the timed window.
				unit, err := cbackend.EmitStorageTransfers(checked.MIR, entry, cbackend.Stage0WindowsMSVCProfile())
				if err != nil {
					t.Fatal(err)
				}
				outcome, err := cdriver.ExecuteWindowsMSVCTarget(artifact)
				if err != nil || outcome.ExitCode != 0 || outcome.Stdout != "" || outcome.Stderr != "" {
					t.Fatalf("execute: %+v %v", outcome, err)
				}
				image, err := pe.NewFile(bytes.NewReader(artifact.Bytes))
				if err != nil {
					t.Fatal(err)
				}
				sections := map[string]uint32{}
				for _, section := range image.Sections {
					sections[section.Name] = section.VirtualSize
				}
				if err = image.Close(); err != nil {
					t.Fatal(err)
				}
				objects := map[string]int{}
				for _, object := range artifact.Objects {
					objects[object.Name] = len(object.Bytes)
				}
				if repetition >= 0 {
					samples = append(samples, nominalTotalSample{
						Repetition: repetition, Variant: variant, Profile: profile.Name,
						CheckNS: checkNS, NativeNS: nativeNS, TotalNS: totalNS,
						GoAllocBytes: after.TotalAlloc - before.TotalAlloc, GoMallocs: after.Mallocs - before.Mallocs, GoGCs: after.NumGC - before.NumGC,
						CBytes: len(unit.Source), CHash: tokenIndexHash([]byte(unit.Source)), ExeBytes: len(artifact.Bytes), Sections: sections, Objects: objects,
						PDBBytes: len(artifact.ProgramDatabase), ExitCode: int(outcome.ExitCode),
					})
				}
			}
		}
		// C equality is a correctness invariant. Artifact size is a measured
		// outcome: retain increases instead of discarding unfavorable samples.
		for _, candidate := range samples {
			if candidate.Repetition != repetition || candidate.Variant != "nominal" {
				continue
			}
			for _, baseline := range samples {
				if baseline.Variant != "scalar" || baseline.Repetition != candidate.Repetition || baseline.Profile != candidate.Profile {
					continue
				}
				if baseline.CHash != candidate.CHash || baseline.CBytes != candidate.CBytes {
					t.Fatalf("nominal C mismatch: scalar=%+v nominal=%+v", baseline, candidate)
				}
				if baseline.ExeBytes != candidate.ExeBytes || !reflect.DeepEqual(baseline.Sections, candidate.Sections) || baseline.Objects["unit.obj"] != candidate.Objects["unit.obj"] {
					t.Logf("paired artifact size difference retained: repetition=%d profile=%s scalar_exe=%d nominal_exe=%d scalar_sections=%v nominal_sections=%v scalar_objects=%v nominal_objects=%v", repetition, candidate.Profile, baseline.ExeBytes, candidate.ExeBytes, baseline.Sections, candidate.Sections, baseline.Objects, candidate.Objects)
				}
			}
		}
	}
	if !reflect.DeepEqual(sources["scalar"], sources["nominal"]) {
		t.Fatal("nominal and scalar source differ")
	}
	identity, err := buildidentity.Current()
	if err != nil {
		t.Fatal(err)
	}
	hashes := map[string]string{}
	for _, path := range []string{"token_index_nominal_total_test.go", "token_index_nominal_test.go", "token_index_experiment_test.go", "target_build.go", "../workspacecompiler/check.go", "../compiler/check.go", "../compiler/storage_source.go", "../nominalhir/analysis.go", "../nominalhir/patch.go", "../nominalhir/boundary.go"} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		hashes[path] = tokenIndexHash(data)
	}
	report := struct {
		Schema     string                       `json:"schema"`
		Baseline   string                       `json:"compiler_source"`
		Identity   string                       `json:"test_executable_identity"`
		Go         string                       `json:"go_version"`
		MSVC       string                       `json:"msvc_version"`
		GOMAXPROCS int                          `json:"gomaxprocs"`
		Method     string                       `json:"method"`
		Sources    map[string]map[string]string `json:"source_sha256"`
		Hashes     map[string]string            `json:"harness_source_sha256"`
		Samples    []nominalTotalSample         `json:"samples"`
	}{"fairyc.nominal-scanner-total/v1", "0fa4ea36a650a5f84c51689eeff9ffc6674d9db0", identity.String(), runtime.Version(), msvc, runtime.GOMAXPROCS(0),
		"One warmup per variant/profile, then 6 repetitions x 3 rotating variants x 2 profiles (36 samples, 42 builds/executions). Fresh scalar/nominal/struct workspaces each time; identical scalar/nominal source. Explicit GC and MemStats snapshots outside time window; GC remains enabled inside. check_ns covers workspace load/lock plus analysis and one verified MIR lowering; nominal additionally includes test-owned annotation and validation. native_ns covers normal C emission/tool discovery/MSVC compile/link/artifact reading. total_ns continuously spans both phases plus entry selection, excluding workspace setup, final user EXE write and execution. go_alloc_bytes and go_mallocs are cumulative Go process allocations over that window, not peak/RSS or MSVC/linker child allocations. Ambient OS/tool caches; in-process harness, no compiler-process startup. Extra C inspection and correctness execution outside measurement. No nominal source parser, runtime benchmark, Linux, cache or debug-size claim; no PDB requested.", sources, hashes, samples}
	encoded, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("NOMINAL_TOTAL_REPORT " + string(encoded))
}
