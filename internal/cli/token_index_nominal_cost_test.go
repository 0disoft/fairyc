package cli

import (
	"bytes"
	"debug/pe"
	"encoding/json"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/buildidentity"
	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/nominalhir"
	"github.com/0disoft/fairyc/internal/target"
	"github.com/0disoft/fairyc/internal/workspacecompiler"
)

// Three predeclared repetitions rotate variant order; each sample starts with a
// fresh workspace. This is an in-process experiment with ambient OS/tool caches.
func TestTokenIndexNominalCostMSVC(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows AMD64 MSVC measurement")
	}
	type sample struct {
		Repetition   int                     `json:"repetition"`
		Variant      string                  `json:"variant"`
		Profile      string                  `json:"profile"`
		CheckNS      int64                   `json:"check_ns"`
		AnnotationNS int64                   `json:"annotation_ns"`
		BoundaryNS   int64                   `json:"boundary_ns"`
		PatchPhases  nominalhir.PatchTimings `json:"patch_phases"`
		NativeNS     int64                   `json:"native_ns"`
		CBytes       int                     `json:"c_bytes"`
		CHash        string                  `json:"c_sha256"`
		ExeBytes     int                     `json:"exe_bytes"`
		Sections     map[string]uint32       `json:"pe_virtual_section_bytes"`
		Objects      map[string]int          `json:"object_bytes"`
		PDBBytes     int                     `json:"pdb_bytes"`
		Sources      map[string]string       `json:"source_sha256"`
		ExitCode     int                     `json:"exit_code"`
	}
	var samples []sample
	var msvc string
	variants := []string{"scalar", "nominal-patch", "nominal-reuse"}
	for repetition := 0; repetition < 3; repetition++ {
		for turn := 0; turn < len(variants); turn++ {
			variant := variants[(turn+repetition)%len(variants)]
			for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
				root, hashes := tokenIndexWorkspace(t, false)
				start := time.Now()
				workspace, _, err := loadWorkspaceWithLock(root, false)
				if err != nil {
					t.Fatal(err)
				}
				checked, err := workspacecompiler.Check(workspace)
				if err != nil || !checked.Verified {
					t.Fatalf("check %s: %v", variant, err)
				}
				checkNS := time.Since(start).Nanoseconds()
				program := checked.MIR
				var annotationNS, boundaryNS int64
				var phases nominalhir.PatchTimings
				if variant == "nominal-patch" || variant == "nominal-reuse" {
					start = time.Now()
					patch := nominalTokenScannerPatch(t, checked)
					annotationNS = time.Since(start).Nanoseconds()
					start = time.Now()
					program, phases, err = nominalhir.MeasureBoundaryPatch(checked, patch, variant == "nominal-reuse")
					boundaryNS = time.Since(start).Nanoseconds()
					if err != nil {
						t.Fatal(err)
					}
				}
				entry := tokenScannerEntry(t, checked)
				unit, err := cbackend.EmitStorageTransfers(program, entry, cbackend.Stage0WindowsMSVCProfile())
				if err != nil {
					t.Fatal(err)
				}
				var diagnostics bytes.Buffer
				start = time.Now()
				artifact, code := compileWindowsProgramPolicy(program, entry, target.X8664WindowsMSVC, profile, false, true, &diagnostics)
				nativeNS := time.Since(start).Nanoseconds()
				if code != ExitSuccess || diagnostics.Len() != 0 {
					t.Fatalf("native: %d %s", code, diagnostics.String())
				}
				msvc = artifact.Toolchain.Version
				outcome, err := cdriver.ExecuteWindowsMSVCTarget(artifact)
				if err != nil || outcome.ExitCode != 0 || outcome.Stdout != "" || outcome.Stderr != "" {
					t.Fatalf("execute: %+v %v", outcome, err)
				}
				image, err := pe.NewFile(bytes.NewReader(artifact.Bytes))
				if err != nil {
					t.Fatal(err)
				}
				sections := map[string]uint32{}
				for _, s := range image.Sections {
					sections[s.Name] = s.VirtualSize
				}
				if err = image.Close(); err != nil {
					t.Fatal(err)
				}
				objects := map[string]int{}
				for _, o := range artifact.Objects {
					objects[o.Name] = len(o.Bytes)
				}
				samples = append(samples, sample{repetition, variant, profile.Name, checkNS, annotationNS, boundaryNS, phases, nativeNS, len(unit.Source), tokenIndexHash([]byte(unit.Source)), len(artifact.Bytes), sections, objects, len(artifact.ProgramDatabase), hashes, int(outcome.ExitCode)})
			}
		}
	}
	identity, err := buildidentity.Current()
	if err != nil {
		t.Fatal(err)
	}
	hashes := map[string]string{}
	for _, path := range []string{"token_index_nominal_cost_test.go", "token_index_nominal_test.go", "../nominalhir/boundary.go", "../nominalhir/patch.go", "../mir/clone.go"} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		hashes[path] = tokenIndexHash(data)
	}
	report := struct {
		Schema   string            `json:"schema"`
		Baseline string            `json:"compiler_source"`
		Identity string            `json:"test_executable_identity"`
		Go       string            `json:"go_version"`
		MSVC     string            `json:"msvc_version"`
		Method   string            `json:"method"`
		Hashes   map[string]string `json:"harness_source_sha256"`
		Samples  []sample          `json:"samples"`
	}{"fairyc.nominal-scanner-cost/v3", "0465013c0ce5069895cab6a6e39c17611cddda8b", identity.String(), runtime.Version(), msvc,
		"3 variants x 2 profiles x 3 repetitions, rotating variant order, fresh workspaces, ambient caches. check includes workspace load/lock and ordinary scalar HIR+MIR checking. Both nominal variants use identical function-scoped annotations and validation. nominal-patch materializes by relowering HIR; nominal-reuse materializes by independent mir.Clone. patch_phases separately time indexing/caller coverage, edited-function validation, and materialization; boundary_ns includes their clock overhead. Annotation and boundary are extra work, not a nominal source frontend. Native compiles checked MIR after separate C inspection; final EXE write and execution time excluded. No PDB requested. No runtime benchmark or cache/debug size measurement.", hashes, samples}
	encoded, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("NOMINAL_SCANNER_REPORT " + string(encoded))
}
