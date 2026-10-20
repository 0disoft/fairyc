package cbackend_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/interpreter"
)

func TestEmitZeroEndSlice(t *testing.T) {
	for _, start := range []string{"0", "1", "18446744073709551615"} {
		t.Run(start, func(t *testing.T) {
			program := lowerSource(t, fmt.Sprintf(`module main;
fn i32 probe(usize start) {
    let ref u8[] empty = b"x"[start..0];
    assert(empty.is_empty);
    return 0;
}
fn i32 main() { return probe(%s); }
`, start))
			outcome, err := interpreter.Execute(program, "main::main")
			if err != nil || start == "0" && outcome.Kind != interpreter.OutcomeReturn || start != "0" && outcome.Kind != interpreter.OutcomeTrap {
				t.Fatalf("slice outcome: %+v %v", outcome, err)
			}
			for _, profile := range []cbackend.Profile{cbackend.Stage0Profile(), cbackend.Stage0WindowsMSVCProfile()} {
				unit, err := cbackend.Emit(program, "main::main", profile)
				if err != nil {
					t.Fatal(err)
				}
				if strings.Contains(unit.Source, "((uintptr_t)UINT64_C(0))) >") || !strings.Contains(unit.Source, "!= ((uintptr_t)UINT64_C(0))") {
					t.Fatal("zero end must check only whether start is nonzero")
				}
			}
		})
	}
}
