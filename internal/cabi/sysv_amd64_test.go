package cabi

import (
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/layout"
	"github.com/0disoft/fairyc/internal/target"
)

func TestClassifyClosedStage0Types(t *testing.T) {
	t.Parallel()
	engine := testEngine(t)
	tests := []struct {
		name      string
		value     layout.Type
		size      uint64
		alignment uint64
		classes   []Class
	}{
		{name: "integer", value: scalar("i64"), size: 8, alignment: 8, classes: []Class{ClassInteger}},
		{name: "float", value: scalar("f64"), size: 8, alignment: 8, classes: []Class{ClassSSE}},
		{name: "pointer", value: layout.Type{Kind: layout.KindPointer}, size: 8, alignment: 8, classes: []Class{ClassInteger}},
		{name: "mixed aggregate", value: structure(field("left", scalar("f64")), field("right", scalar("i32"))), size: 16, alignment: 8, classes: []Class{ClassSSE, ClassInteger}},
		{name: "integer dominates one eightbyte", value: structure(field("left", scalar("f32")), field("right", scalar("u32"))), size: 8, alignment: 4, classes: []Class{ClassInteger}},
		{name: "float array", value: array(scalar("f32"), 3), size: 12, alignment: 4, classes: []Class{ClassSSE, ClassSSE}},
		{name: "large aggregate", value: array(scalar("i64"), 3), size: 24, alignment: 8, classes: []Class{ClassMemory}},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, err := engine.Classify(test.value)
			if err != nil {
				t.Fatal(err)
			}
			if got.SizeBytes != test.size || got.AlignmentBytes != test.alignment || !slices.Equal(got.Classes, test.classes) {
				t.Fatalf("Classify() = %+v", got)
			}
		})
	}
}

func TestPlanAssignsMixedReturnAndHiddenResultPointer(t *testing.T) {
	t.Parallel()
	engine := testEngine(t)
	mixed := structure(field("left", scalar("f64")), field("right", scalar("i32")))
	plan, err := engine.Plan(Signature{Result: &mixed})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Return.Kind != LocationRegisters || !slices.Equal(plan.Return.Registers, []Register{"xmm0", "rax"}) {
		t.Fatalf("mixed return = %+v", plan.Return)
	}
	large := array(scalar("i64"), 3)
	plan, err = engine.Plan(Signature{Result: &large, Parameters: []layout.Type{scalar("i64"), scalar("f64")}})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Return.Kind != LocationIndirect || plan.Return.HiddenPointerRegister != "rdi" || plan.Return.PointerResultRegister != "rax" ||
		plan.Arguments[0].Kind != LocationRegisters || !slices.Equal(plan.Arguments[0].Registers, []Register{"rsi"}) ||
		plan.Arguments[1].Kind != LocationRegisters || !slices.Equal(plan.Arguments[1].Registers, []Register{"xmm0"}) {
		t.Fatalf("indirect plan = %+v", plan)
	}
}

func TestPlanRollsBackWholeAggregateOnRegisterExhaustion(t *testing.T) {
	t.Parallel()
	engine := testEngine(t)
	pair := structure(field("first", scalar("i64")), field("second", scalar("i64")))
	parameters := []layout.Type{scalar("i64"), scalar("i64"), scalar("i64"), scalar("i64"), scalar("i64"), pair, scalar("i64")}
	plan, err := engine.Plan(Signature{Parameters: parameters})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Arguments[5].Kind != LocationStack || len(plan.Arguments[5].Registers) != 0 ||
		plan.Arguments[6].Kind != LocationRegisters || !slices.Equal(plan.Arguments[6].Registers, []Register{"r9"}) {
		t.Fatalf("integer rollback plan = %+v", plan.Arguments)
	}

	mixed := structure(field("integer", scalar("i64")), field("vector", scalar("f64")))
	parameters = parameters[:0]
	for index := 0; index < 6; index++ {
		parameters = append(parameters, scalar("i64"))
	}
	for index := 0; index < 7; index++ {
		parameters = append(parameters, scalar("f64"))
	}
	parameters = append(parameters, mixed, scalar("f64"))
	plan, err = engine.Plan(Signature{Parameters: parameters})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Arguments[13].Kind != LocationStack || plan.Arguments[14].Kind != LocationRegisters || !slices.Equal(plan.Arguments[14].Registers, []Register{"xmm7"}) {
		t.Fatalf("mixed rollback plan = %+v", plan.Arguments)
	}
}

func TestPlanSupportsVoidAndRejectsInvalidTypes(t *testing.T) {
	t.Parallel()
	engine := testEngine(t)
	plan, err := engine.Plan(Signature{})
	if err != nil || plan.Return.Kind != LocationVoid || len(plan.Return.Registers) != 0 || len(plan.Return.Classification.Classes) != 0 {
		t.Fatalf("void plan = %+v, %v", plan, err)
	}
	for _, test := range []struct {
		name  string
		value layout.Type
		code  ErrorCode
	}{
		{name: "floating enum", value: layout.Type{Kind: layout.KindCEnum, Name: "c.float"}, code: ErrorInvalidCEnumBase},
		{name: "empty aggregate", value: layout.Type{Kind: layout.KindStruct}, code: ErrorLayoutRejected},
	} {
		_, err := engine.Classify(test.value)
		code, ok := Code(err)
		if !ok || code != test.code {
			t.Errorf("%s error = %v, code %q", test.name, err, code)
		}
	}
}

func TestNewRejectsClassifierIdentityDrift(t *testing.T) {
	t.Parallel()
	spec, _ := target.Lookup(target.X8664LinuxGNU)
	spec.ReturnClassifier = "other"
	_, err := New(spec)
	code, ok := Code(err)
	if !ok || code != ErrorUnsupportedTarget || !strings.Contains(err.Error(), "does not declare") {
		t.Fatalf("New() error = %v, code %q", err, code)
	}
}

func TestPlansAreIsolatedFromCallerMutation(t *testing.T) {
	t.Parallel()
	engine := testEngine(t)
	value := structure(field("left", scalar("i64")), field("right", scalar("f64")))
	first, err := engine.Plan(Signature{Result: &value, Parameters: []layout.Type{value}})
	if err != nil {
		t.Fatal(err)
	}
	second, err := engine.Plan(Signature{Result: &value, Parameters: []layout.Type{value}})
	if err != nil {
		t.Fatal(err)
	}
	first.Return.Classification.Classes[0] = ClassMemory
	first.Return.Registers[0] = "r9"
	first.Arguments[0].Classification.Classes[0] = ClassMemory
	if reflect.DeepEqual(first, second) || second.Return.Classification.Classes[0] != ClassInteger || second.Return.Registers[0] != "rax" || second.Arguments[0].Classification.Classes[0] != ClassInteger {
		t.Fatalf("plan isolation failed: first=%+v second=%+v", first, second)
	}
}

func testEngine(t *testing.T) *Engine {
	t.Helper()
	spec, found := target.Lookup(target.X8664LinuxGNU)
	if !found {
		t.Fatal("target is missing")
	}
	engine, err := New(spec)
	if err != nil {
		t.Fatal(err)
	}
	if engine.Target() != target.X8664LinuxGNU {
		t.Fatalf("engine target = %q", engine.Target())
	}
	return engine
}

func scalar(name string) layout.Type {
	return layout.Type{Kind: layout.KindLanguageScalar, Name: name}
}

func array(element layout.Type, length uint64) layout.Type {
	return layout.Type{Kind: layout.KindArray, Element: &element, Length: length}
}

func structure(fields ...layout.Field) layout.Type {
	return layout.Type{Kind: layout.KindStruct, Fields: fields}
}

func field(name string, value layout.Type) layout.Field {
	return layout.Field{Name: name, Type: value}
}
