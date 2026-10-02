package cbackend_test

import (
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
)

func TestEmitLiteralBooleanOmitsImpossiblePaths(t *testing.T) {
	for _, sample := range []struct{ name, body, absent string }{
		{"assert-false", "assert(false); return 42;", "return INT32_C(42);"},
		{"while-true", "while (true) {} return 42;", "return INT32_C(42);"},
		{"while-false", "while (false) { return 7; } return 42;", "return INT32_C(7);"},
	} {
		t.Run(sample.name, func(t *testing.T) {
			program := lowerSource(t, "module main;\nfn i32 main() { "+sample.body+" }\n")
			for _, profile := range []cbackend.Profile{cbackend.Stage0Profile(), cbackend.Stage0WindowsMSVCProfile()} {
				unit, err := cbackend.Emit(program, "main::main", profile)
				if err != nil {
					t.Fatal(err)
				}
				if strings.Contains(unit.Source, sample.absent) {
					t.Fatalf("impossible path emitted: %s\n%s", sample.absent, unit.Source)
				}
				if sample.name == "assert-false" && !strings.Contains(unit.Source, "fy_runtime_trap(FY_TRAP_ASSERTION_FAILED);") {
					t.Fatal("assertion trap removed")
				}
			}
		})
	}
}

func TestEmitNonReturningValueCallOmitsAssignmentAndContinuation(t *testing.T) {
	program := lowerSource(t, "module main;\nfn i32 fails() { assert(false); return 7; }\nfn i32 main() { let i32 value = fails(); return value; }\n")
	for _, profile := range []cbackend.Profile{cbackend.Stage0Profile(), cbackend.Stage0WindowsMSVCProfile()} {
		unit, err := cbackend.Emit(program, "main::main", profile)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(unit.Source, "static _Noreturn void fy_fn_") || strings.Contains(unit.Source, "_Noreturn int32_t") || strings.Contains(unit.Source, " = fy_fn_") || strings.Contains(unit.Source, "return fy_l_") {
			t.Fatalf("invalid non-returning call transport:\n%s", unit.Source)
		}
	}
}
