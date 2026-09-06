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
			result, _, err := compiler.CheckGenericFunctionsForPackage(file, genericCoordinate(t), genericplan.DefaultPlanLimits())
			if err != nil || !result.Verified {
				t.Fatalf("compile: %v %+v", err, result.Semantic)
			}
			runGenericNative(t, toolchain, result, "main::main", fixture.Value)
		})
	}
	t.Run("package-import-chain", func(t *testing.T) {
		input, coordinates := genericPackageInput(t)
		result, _, err := compiler.CheckGenericPackages(input, coordinates, genericplan.DefaultPlanLimits())
		if err != nil || !result.Verified {
			t.Fatalf("compile: %v %+v", err, result.Semantic)
		}
		runGenericNative(t, toolchain, result, hir.QualifiedModulePath(1, "main")+"::main", 42)
	})
}

func runGenericNative(t *testing.T, toolchain cdriver.WindowsHostToolchain, result compiler.Result, entry string, expected int32) {
	t.Helper()
	unit, err := cbackend.Emit(result.MIR, entry, cbackend.Stage0WindowsMSVCProfile())
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
			if err != nil || outcome.ExitCode != expected || outcome.Stdout != "" || outcome.Stderr != "" {
				t.Fatalf("execute: %+v %v", outcome, err)
			}
		})
	}
}
