package cli

import (
	"bytes"
	"encoding/json"
	"reflect"
	"runtime"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/nominalhir"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/target"
)

func TestTokenIndexNominalScannerAnalyzed(t *testing.T) {
	root, _ := tokenIndexWorkspace(t, false)
	workspace, _, err := loadWorkspaceWithLock(root, false)
	if err != nil {
		t.Fatal(err)
	}
	input, err := workspace.SemanticInput()
	if err != nil {
		t.Fatal(err)
	}
	owners, err := workspace.SDKStorageOwners()
	if err != nil || len(owners) == 0 {
		t.Fatalf("SDK owners: %v", err)
	}
	coordinates, err := workspace.PackageCoordinates()
	if err != nil {
		t.Fatal(err)
	}
	analysis, err := compiler.AnalyzeSDKStoragePackages(input, coordinates, genericplan.DefaultPlanLimits(), owners)
	if err != nil {
		t.Fatal(err)
	}
	before, err := json.Marshal(analysis)
	if err != nil {
		t.Fatal(err)
	}
	// This helper reads HIR only. No verified baseline or MIR exists at annotation.
	patch := nominalTokenScannerPatch(t, compiler.Result{Semantic: analysis.Analysis.Semantic})
	candidate, err := nominalhir.LowerAnalyzedStoragePatch(analysis.Analysis, patch)
	if err != nil || !candidate.Verified {
		t.Fatalf("pre-MIR patch: %v", err)
	}
	baseline, err := compiler.LowerAnalyzedStorageTransfers(analysis.Analysis)
	if err != nil || !baseline.Verified {
		t.Fatalf("control: %v", err)
	}
	if !reflect.DeepEqual(candidate, baseline) {
		t.Fatal("pre-MIR path changed compiler result or provenance")
	}
	after, err := json.Marshal(analysis)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("analysis mutated")
	}
	for _, phase := range []semantic.Phase{semantic.PhaseNone, semantic.PhaseResolve} {
		invalid := analysis.Analysis
		invalid.Semantic.CompletedThrough = phase
		got, err := nominalhir.LowerAnalyzedStoragePatch(invalid, patch)
		if err == nil || !reflect.DeepEqual(got, compiler.Result{}) {
			t.Fatal("incomplete analysis published MIR")
		}
	}
	invalid := analysis.Analysis
	invalid.Semantic.FailedPhase = semantic.PhaseFlow
	if got, err := nominalhir.LowerAnalyzedStoragePatch(invalid, patch); err == nil || !reflect.DeepEqual(got, compiler.Result{}) {
		t.Fatal("failed analysis published MIR")
	}
	entry := tokenScannerEntry(t, candidate)
	var sources [2]string
	for i, checked := range []compiler.Result{baseline, candidate} {
		unit, err := cbackend.EmitStorageTransfers(checked.MIR, entry, cbackend.Stage0WindowsMSVCProfile())
		if err != nil {
			t.Fatal(err)
		}
		sources[i] = unit.Source
	}
	if sources[0] != sources[1] {
		t.Fatal("generated C changed")
	}
	t.Logf("generated C bytes=%d sha256=%s", len(sources[1]), tokenIndexHash([]byte(sources[1])))
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Log("native checks require Windows AMD64")
		return
	}
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		var sizes [2]int
		for i, checked := range []compiler.Result{baseline, candidate} {
			var diagnostics bytes.Buffer
			artifact, code := compileWindowsProgramPolicy(checked.MIR, entry, target.X8664WindowsMSVC, profile, false, true, &diagnostics)
			if code != ExitSuccess || diagnostics.Len() != 0 {
				t.Fatalf("native: %d %s", code, diagnostics.String())
			}
			outcome, err := cdriver.ExecuteWindowsMSVCTarget(artifact)
			if err != nil || outcome.ExitCode != 0 || outcome.Stdout != "" || outcome.Stderr != "" {
				t.Fatalf("execute: %+v %v", outcome, err)
			}
			sizes[i] = len(artifact.Bytes)
		}
		if sizes[0] != sizes[1] {
			t.Fatalf("%s EXE grew: %v", profile.Name, sizes)
		}
		t.Logf("%s baseline/candidate EXE bytes=%d", profile.Name, sizes[0])
	}
}
