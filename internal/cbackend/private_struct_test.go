package cbackend_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/interpreter"
)

func TestEmitPrivateNestedStructLayoutAndOrder(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, `module main;
struct Outer { u8 prefix; Middle middle; u8 suffix; }
struct Middle { u8 prefix; Leaf leaf; }
struct Leaf { u32 value; }
fn i32 score(Outer value) {
    return if (value.prefix == 1 && value.middle.prefix == 2 && value.middle.leaf.value == 41 && value.suffix == 3) { 42 } else { 0 };
}
fn i32 main() {
    return score(Outer { .prefix = 1, .middle = Middle { .prefix = 2, .leaf = Leaf { .value = 41 } }, .suffix = 3 });
}`)
	outcome, err := interpreter.Execute(program, "main::main")
	if err != nil || outcome.Kind != interpreter.OutcomeReturn || outcome.Value != 42 {
		t.Fatalf("interpreter: %+v %v", outcome, err)
	}
	for _, profile := range []cbackend.Profile{cbackend.Stage0Profile(), cbackend.Stage0WindowsMSVCProfile()} {
		unit, err := cbackend.Emit(program, "main::main", profile)
		if err != nil {
			t.Fatal(err)
		}
		for _, required := range []string{
			"sizeof(fy_struct_0) == UINT64_C(4)",
			"sizeof(fy_struct_1) == UINT64_C(8)",
			"sizeof(fy_struct_2) == UINT64_C(16)",
			"_Alignof(fy_struct_2) == UINT64_C(4)",
			"offsetof(fy_struct_1, f_1) == UINT64_C(4)",
			"offsetof(fy_struct_2, f_1) == UINT64_C(4)",
			"offsetof(fy_struct_2, f_2) == UINT64_C(12)",
		} {
			if !strings.Contains(unit.Source, required) {
				t.Errorf("missing layout assertion %q", required)
			}
		}
		for _, edge := range [][2]string{{"} fy_struct_0;", "fy_struct_0 f_1;"}, {"} fy_struct_1;", "fy_struct_1 f_1;"}} {
			definition, use := strings.Index(unit.Source, edge[0]), strings.Index(unit.Source, edge[1])
			if definition < 0 || use <= definition {
				t.Fatalf("nested definition does not precede use: %v", edge)
			}
		}
		for _, public := range []string{"struct Outer {", "struct Middle {", "struct Leaf {"} {
			if strings.Contains(unit.Source, public) {
				t.Fatalf("private type exposed as public C tag: %s", public)
			}
		}
	}
}

func TestEmitPrivateNestedStructRejectsCycles(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, `module main;
struct First { Second next; }
struct Second { First next; }
fn i32 main() { return 0; }
`)
	_, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	var capability cbackend.CapabilityError
	if !errors.As(err, &capability) || !strings.Contains(capability.Feature, "recursive struct dependency involving") {
		t.Fatalf("cycle: %v", err)
	}
}
