package cbackend_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/interpreter"
)

func TestClosureValueMSVCCheckedDivisionGuards(t *testing.T) {
	for _, op := range []string{"/", "%"} {
		for _, tc := range []struct{ name, left, divisor, argument, trap string }{
			{"literal-zero", "42", "0", "1", "division-by-zero"},
			{"runtime-zero", "42", "x", "0", "division-by-zero"},
			{"runtime-success", "42", "x", "1", ""},
			{"literal-signed-overflow", "-2147483648", "-1", "1", "signed-division-overflow"},
			{"runtime-signed-overflow", "-2147483648", "x", "-1", "signed-division-overflow"},
			{"signed-boundary-success", "-2147483648", "x", "1", ""},
		} {
			t.Run(op+tc.name, func(t *testing.T) {
				text := fmt.Sprintf(`module main;
fn void exercise(){let fn i32(i32) action=fn [] i32(i32 x){let unused=%s%s%s;return x;};discard action(%s);}
fn i32 main(){exercise();return 0;}`, tc.left, op, tc.divisor, tc.argument)
				program := lowerSource(t, text)
				outcome, err := interpreter.Execute(program, "main::main")
				if err != nil || outcome.TrapCategory != tc.trap || tc.trap == "" && (outcome.Kind != interpreter.OutcomeReturn || outcome.Value != 0) {
					t.Fatalf("original source: %+v %v", outcome, err)
				}
				for _, profile := range []cbackend.Profile{cbackend.Stage0Profile(), cbackend.Stage0WindowsMSVCProfile()} {
					unit, err := cbackend.Emit(program, "main::main", profile)
					if err != nil {
						t.Fatal(err)
					}
					guard := "#if defined(_MSC_VER)\n    fy_checked_divisor_i32 = "
					start := strings.Index(unit.Source, guard)
					if start < 0 {
						t.Fatal("lifted target lost the MSVC checked-edge barrier")
					}
					end := strings.Index(unit.Source[start:], "#else\n")
					if end < 0 {
						t.Fatal("missing feature-free compiler branch")
					}
					branch := unit.Source[start : start+end]
					if !strings.Contains(branch, "if (fy_checked_divisor_i32 == 0)") || !strings.Contains(branch, "&& fy_checked_divisor_i32 == -INT32_C(1))") {
						t.Fatalf("zero and signed-overflow guards must read the same private volatile divisor:\n%s", branch)
					}
				}
			})
		}
	}
	// Ordinary runtime-input division retains its existing guard emission.
	program := lowerSource(t, "module main;fn i32 divide(i32 x){return 42/x;}fn i32 main(){return divide(1);}")
	unit, err := cbackend.Emit(program, "main::main", cbackend.Stage0WindowsMSVCProfile())
	if err != nil || strings.Contains(unit.Source, "#if defined(_MSC_VER)\n    fy_checked_divisor_i32 = ") {
		t.Fatalf("ordinary transport changed: %v", err)
	}
	// A constant trap in an ordinary closure caller needs the same barrier.
	program = lowerSource(t, "module main;fn i32 main(){return 42/0;}")
	unit, err = cbackend.Emit(program, "main::main", cbackend.Stage0WindowsMSVCProfile())
	if err != nil || !strings.Contains(unit.Source, "#if defined(_MSC_VER)\n    fy_checked_divisor_i32 = INT32_C(0);") {
		t.Fatalf("literal divisor lost its MSVC guard: %v", err)
	}
}
