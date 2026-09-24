package cbackend_test

import (
	"fmt"
	"runtime"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/cruntime"
	"github.com/0disoft/fairyc/internal/interpreter"
)

func sharedScalarBorrowOptionSource(scalar string) string {
	initial := "42"
	if scalar == "bool" {
		initial = "true"
	}
	return fmt.Sprintf(`module main;
fn Option<ref %[1]s> choose(ref %[1]s value, bool include) {
    return if (include) { .some(value) } else { .none };
}
fn i32 main() {
    let %[1]s source = %[2]s;
    let Option<ref %[1]s> present = choose(ref source, true);
    let Option<ref %[1]s> absent = choose(ref source, false);
    let i32 first = switch (present) { case .some(value): { if (*value == source) { 42 } else { 1 } } case .none: { 2 } };
    let i32 second = switch (absent) { case .some(_): { 3 } case .none: { 0 } };
    return first + second;
}
`, scalar, initial)
}

func TestEmitSharedScalarBorrowOptionUsesNullNiche(t *testing.T) {
	t.Parallel()
	for _, scalar := range []string{"bool", "u8", "i32", "u32", "usize"} {
		t.Run(scalar, func(t *testing.T) {
			program := lowerSource(t, sharedScalarBorrowOptionSource(scalar))
			outcome, err := interpreter.Execute(program, "main::main")
			if err != nil || outcome.Kind != interpreter.OutcomeReturn || outcome.Value != 42 {
				t.Fatalf("interpret Option<ref %s>: %+v %v", scalar, outcome, err)
			}
			unit, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
			if err != nil {
				t.Fatalf("Emit() error = %v", err)
			}
			for _, required := range []string{
				"_Static_assert(sizeof(fy_tagged_0) == UINT64_C(8)",
				".p_0 = ((void *)0)",
				".p_0 != ((void *)0)",
			} {
				if !strings.Contains(unit.Source, required) {
					t.Errorf("generated C is missing %q", required)
				}
			}
		})
	}
}

func TestSharedScalarBorrowOptionRunsUnderMSVC(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("native proof requires Windows amd64")
	}
	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatalf("native proof requires local x64 MSVC: %v", err)
	}
	scalarRuntime, err := cruntime.ScalarSource(192)
	if err != nil {
		t.Fatal(err)
	}
	for _, scalar := range []string{"bool", "u8", "i32", "u32", "usize"} {
		t.Run(scalar, func(t *testing.T) {
			program := lowerSource(t, sharedScalarBorrowOptionSource(scalar))
			unit, err := cbackend.Emit(program, "main::main", cbackend.Stage0WindowsMSVCProfile())
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
					if err != nil || outcome.ExitCode != 42 || outcome.Stdout != "" || outcome.Stderr != "" {
						t.Fatalf("execute: %+v %v", outcome, err)
					}
				})
			}
		})
	}
}
