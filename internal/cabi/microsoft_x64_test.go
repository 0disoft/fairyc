package cabi

import (
	"slices"
	"testing"

	"github.com/0disoft/fairyc/internal/layout"
	"github.com/0disoft/fairyc/internal/target"
)

func TestMicrosoftX64ClassificationUsesScalarAndAggregateRules(t *testing.T) {
	t.Parallel()
	engine := microsoftEngine(t)
	for _, test := range []struct {
		name    string
		value   layout.Type
		classes []Class
	}{
		{name: "integer scalar", value: scalar("i64"), classes: []Class{ClassInteger}},
		{name: "floating scalar", value: scalar("f64"), classes: []Class{ClassSSE}},
		{name: "pointer", value: layout.Type{Kind: layout.KindPointer}, classes: []Class{ClassInteger}},
		{name: "four byte aggregate", value: structure(field("value", scalar("f32"))), classes: []Class{ClassInteger}},
		{name: "eight byte aggregate", value: structure(field("left", scalar("u32")), field("right", scalar("u32"))), classes: []Class{ClassInteger}},
		{name: "three byte aggregate", value: structure(field("a", scalar("u8")), field("b", scalar("u8")), field("c", scalar("u8"))), classes: []Class{ClassMemory}},
		{name: "sixteen byte aggregate", value: structure(field("left", scalar("i64")), field("right", scalar("i64"))), classes: []Class{ClassMemory}},
	} {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			classification, err := engine.Classify(test.value)
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(classification.Classes, test.classes) {
				t.Fatalf("Classify() = %+v", classification)
			}
		})
	}
}

func TestMicrosoftX64PlanUsesSharedOrdinalRegisterSlots(t *testing.T) {
	t.Parallel()
	engine := microsoftEngine(t)
	plan, err := engine.Plan(Signature{Parameters: []layout.Type{
		scalar("f64"), scalar("i64"), scalar("f32"), {Kind: layout.KindPointer}, scalar("f64"),
	}})
	if err != nil {
		t.Fatal(err)
	}
	want := [][]Register{{"xmm0"}, {"rdx"}, {"xmm2"}, {"r9"}, {}}
	for index, argument := range plan.Arguments {
		if !slices.Equal(argument.Registers, want[index]) {
			t.Fatalf("argument %d = %+v, want registers %v", index, argument, want[index])
		}
	}
	if plan.Arguments[4].Kind != LocationStack {
		t.Fatalf("fifth argument = %+v", plan.Arguments[4])
	}
}

func TestMicrosoftX64PlanUsesIndirectAggregatesAndHiddenReturnSlot(t *testing.T) {
	t.Parallel()
	engine := microsoftEngine(t)
	large := structure(field("left", scalar("i64")), field("right", scalar("i64")))
	plan, err := engine.Plan(Signature{Result: &large, Parameters: []layout.Type{
		scalar("i64"), scalar("f64"), {Kind: layout.KindPointer}, scalar("i32"), large,
	}})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Return.Kind != LocationIndirect || plan.Return.HiddenPointerRegister != "rcx" || plan.Return.PointerResultRegister != "rax" {
		t.Fatalf("return = %+v", plan.Return)
	}
	wantKinds := []LocationKind{LocationRegisters, LocationRegisters, LocationRegisters, LocationStack, LocationIndirect}
	wantRegisters := [][]Register{{"rdx"}, {"xmm2"}, {"r9"}, {}, {}}
	for index, argument := range plan.Arguments {
		if argument.Kind != wantKinds[index] || !slices.Equal(argument.Registers, wantRegisters[index]) {
			t.Fatalf("argument %d = %+v, want kind %q registers %v", index, argument, wantKinds[index], wantRegisters[index])
		}
	}

	plan, err = engine.Plan(Signature{Parameters: []layout.Type{large}})
	if err != nil || plan.Arguments[0].Kind != LocationIndirect || !slices.Equal(plan.Arguments[0].Registers, []Register{"rcx"}) {
		t.Fatalf("indirect first argument = %+v, %v", plan, err)
	}
}

func TestMicrosoftX64SmallFloatRecordUsesIntegerTransport(t *testing.T) {
	t.Parallel()
	engine := microsoftEngine(t)
	record := structure(field("value", scalar("f32")))
	plan, err := engine.Plan(Signature{Result: &record, Parameters: []layout.Type{record}})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(plan.Return.Registers, []Register{"rax"}) || !slices.Equal(plan.Arguments[0].Registers, []Register{"rcx"}) {
		t.Fatalf("small float record plan = %+v", plan)
	}
}

func microsoftEngine(t *testing.T) *Engine {
	t.Helper()
	spec, found := target.Lookup(target.X8664WindowsMSVC)
	if !found {
		t.Fatal("Windows target is missing")
	}
	engine, err := New(spec)
	if err != nil {
		t.Fatal(err)
	}
	if engine.Target() != target.X8664WindowsMSVC {
		t.Fatalf("engine target = %q", engine.Target())
	}
	return engine
}
