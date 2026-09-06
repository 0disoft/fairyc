package compiler_test

import (
	"runtime"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/cruntime"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/hir"
)

func TestGenericSourceRunsLowAndOptimizedMSVC(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("native proof requires Windows amd64")
	}
	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatalf("native proof requires local x64 MSVC: %v", err)
	}
	for _, fixture := range genericFixtures(t) {
		if fixture.Outcome != "return" {
			continue
		}
		t.Run(fixture.ID, func(t *testing.T) {
			file := parseCompilerFile(t, hir.DefaultPackage, "main.fy", genericFixtureSource(t, fixture.Source))
			result, _, err := compiler.CheckGenericFunctions(file, genericplan.DefaultPlanLimits())
			if err != nil || !result.Verified {
				t.Fatalf("compile: %v %+v", err, result.Semantic)
			}
			unit, err := cbackend.Emit(result.MIR, "main::main", cbackend.Stage0WindowsMSVCProfile())
			if err != nil {
				t.Fatal(err)
			}
			scalarRuntime, err := cruntime.ScalarSource(192)
			if err != nil {
				t.Fatal(err)
			}
			for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
				t.Run(profile.Name, func(t *testing.T) {
					artifact, err := cdriver.BuildWindowsHost(toolchain, profile, unit.Source, scalarRuntime)
					if err != nil {
						t.Fatal(err)
					}
					outcome, err := cdriver.ExecuteWindowsHost(artifact)
					if err != nil || outcome.ExitCode != fixture.Value || outcome.Stdout != "" || outcome.Stderr != "" {
						t.Fatalf("execute: %+v %v", outcome, err)
					}
				})
			}
		})
	}
}
