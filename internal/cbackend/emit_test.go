package cbackend_test

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
	"github.com/0disoft/fairyc/internal/target"
)

func TestEmitScalarControlUnit(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, "module main;\nfn add(left: i32, right: i32) -> i32 { left + right }\nfn main() -> i32 { var count: i32 = 0; while count < 3 { count = count + 1; } add(count, 39) }\n")
	outcome, err := interpreter.Execute(program, "main::main")
	if err != nil || outcome.Kind != interpreter.OutcomeReturn || outcome.Value != 42 {
		t.Fatalf("interpreter = %+v, %v", outcome, err)
	}
	unit, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	if err != nil {
		t.Fatalf("Emit() error = %v", err)
	}
	if unit.Target != string(target.X8664LinuxGNU) || unit.Dialect != cbackend.DialectC11 || unit.Entry != "main::main" {
		t.Fatalf("Emit() identity = %+v", unit)
	}
	if !slices.Equal(unit.Functions, []string{"main::add", "main::main"}) {
		t.Fatalf("Emit() functions = %v", unit.Functions)
	}
	wantFeatures := []string{"checked-arithmetic", "scalar-call", "scalar-comparison", "scalar-control-flow", "trap"}
	if !slices.Equal(unit.Features, wantFeatures) {
		t.Fatalf("Emit() features = %v, want %v", unit.Features, wantFeatures)
	}
	for _, required := range []string{
		"#include <stdint.h>",
		"#define FY_TRAP_INTEGER_OVERFLOW UINT32_C(2)",
		"_Noreturn void fy_runtime_trap(uint32_t category);",
		"static int32_t fy_fn_0(int32_t",
		"int main(void)",
	} {
		if !strings.Contains(unit.Source, required) {
			t.Errorf("generated C is missing %q", required)
		}
	}
	if strings.Contains(unit.Source, "abort(") || strings.Contains(unit.Source, "#include <stdlib.h>") {
		t.Fatalf("generated C owns trap termination instead of delegating to the runtime:\n%s", unit.Source)
	}
	second, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	if err != nil || second.Source != unit.Source {
		t.Fatalf("second Emit() differs: error=%v", err)
	}
}

func TestEmitScalarForeignCall(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, "module main;\nunsafe extern C fn fairyc_fixture_add_one(value: i32) -> i32;\nfn main() -> i32 { unsafe { fairyc_fixture_add_one(41) } }\n")
	_, interpreterErr := interpreter.Execute(program, "main::main")
	var interpreterCapability interpreter.CapabilityError
	if !errors.As(interpreterErr, &interpreterCapability) || interpreterCapability.Kind != "foreign-call" {
		t.Fatalf("interpreter error = %v", interpreterErr)
	}
	unit, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	if err != nil {
		t.Fatalf("Emit() error = %v", err)
	}
	if !slices.Equal(unit.Features, []string{"foreign-c-call", "scalar-call"}) {
		t.Fatalf("Emit() features = %v", unit.Features)
	}
	for _, required := range []string{
		"extern int32_t fairyc_fixture_add_one(int32_t fy_a_0);",
		"fairyc_fixture_add_one(INT32_C(41));",
	} {
		if !strings.Contains(unit.Source, required) {
			t.Errorf("generated C is missing %q:\n%s", required, unit.Source)
		}
	}
	if strings.Contains(unit.Source, "static int32_t fairyc_fixture_add_one") {
		t.Fatalf("generated C turned an imported symbol into an internal definition:\n%s", unit.Source)
	}
	if strings.Contains(unit.Source, "int32_t fy_l_0;") {
		t.Fatalf("generated C declared a phantom local for the foreign symbol:\n%s", unit.Source)
	}
}

func TestEmitRawPointerAndTargetScalarForeignCall(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, "module main;\nunsafe extern C fn foreign_probe(data: *u8, output: *mut u32, length: usize, marker: u8) -> u32;\nfn main() -> i32 { if unsafe { foreign_probe(null, null, 0, 7) } == 42 { 0 } else { 1 } }\n")
	unit, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	if err != nil {
		t.Fatalf("Emit() error = %v", err)
	}
	for _, required := range []string{
		"extern uint32_t foreign_probe(const uint8_t * fy_a_0, uint32_t * fy_a_1, uintptr_t fy_a_2, uint8_t fy_a_3);",
		"foreign_probe(((void *)0), ((void *)0), ((uintptr_t)UINT64_C(0)), UINT8_C(7));",
		"== (UINT32_C(42))",
	} {
		if !strings.Contains(unit.Source, required) {
			t.Errorf("generated C is missing %q:\n%s", required, unit.Source)
		}
	}
}

func TestEmitRejectsReservedExternalSymbol(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, "module main;\nunsafe extern C fn add_one(value: i32) -> i32;\nfn main() -> i32 { unsafe { add_one(41) } }\n")
	program.ForeignFunctions[0].LinkName = "main"
	_, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	var capability cbackend.CapabilityError
	if !errors.As(err, &capability) || capability.Function != "main::add_one" || capability.Feature != "external C symbol \"main\"" {
		t.Fatalf("Emit() error = %v", err)
	}
}

func TestEmitRejectsUnrecognizedStandardLibrarySymbol(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, "module main;\nunsafe extern C fn add_one(value: i32) -> i32;\nfn main() -> i32 { unsafe { add_one(41) } }\n")
	program.ForeignFunctions[0].LinkName = "fy_std_unrecognized_v0"
	_, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	var capability cbackend.CapabilityError
	if !errors.As(err, &capability) || capability.Function != "main::add_one" || capability.Feature != "external C symbol \"fy_std_unrecognized_v0\"" {
		t.Fatalf("Emit() error = %v", err)
	}
}

func TestEmitGuardsExternalSymbolAgainstHeaderMacro(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, "module main;\nunsafe extern C fn INT64_MIN(value: i32) -> i32;\nfn main() -> i32 { unsafe { INT64_MIN(41) } }\n")
	unit, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	if err != nil {
		t.Fatalf("Emit() error = %v", err)
	}
	for _, required := range []string{
		"#ifdef INT64_MIN\n#undef INT64_MIN\n#endif",
		"extern int32_t INT64_MIN(int32_t fy_a_0);",
		"INT64_MIN(INT32_C(41));",
	} {
		if !strings.Contains(unit.Source, required) {
			t.Errorf("generated C is missing %q:\n%s", required, unit.Source)
		}
	}
}

func TestEmitCoalescesCompatibleExternalDeclarations(t *testing.T) {
	t.Parallel()
	program := lowerSources(t, map[string]string{
		"bindingsa.fy": "module bindingsa;\npub unsafe extern C fn shared_add_one(value: i32) -> i32;\n",
		"bindingsb.fy": "module bindingsb;\npub unsafe extern C fn shared_add_one(value: i32) -> i32;\n",
		"main.fy":      "module main;\nimport bindingsb as b;\nfn main() -> i32 { unsafe { b::shared_add_one(41) } }\n",
	})
	unit, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	if err != nil {
		t.Fatalf("Emit() error = %v", err)
	}
	prototype := "extern int32_t shared_add_one(int32_t fy_a_0);"
	if count := strings.Count(unit.Source, prototype); count != 1 {
		t.Fatalf("generated C contains %d compatible prototypes, want 1:\n%s", count, unit.Source)
	}
}

func TestEmitRejectsConflictingExternalDeclarations(t *testing.T) {
	t.Parallel()
	program := lowerSources(t, map[string]string{
		"bindingsa.fy": "module bindingsa;\npub unsafe extern C fn shared_symbol(value: i32) -> i32;\n",
		"bindingsb.fy": "module bindingsb;\npub unsafe extern C fn shared_symbol() -> i32;\n",
		"main.fy":      "module main;\nfn main() -> i32 { 0 }\n",
	})
	_, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	if err == nil || !strings.Contains(err.Error(), "conflicting external C declarations") || !strings.Contains(err.Error(), "shared_symbol") {
		t.Fatalf("Emit() error = %v", err)
	}
}

func TestTrapCodesAreStable(t *testing.T) {
	t.Parallel()
	want := map[string]uint32{
		"bounds":                   1,
		"integer-overflow":         2,
		"division-by-zero":         3,
		"signed-division-overflow": 4,
		"invalid-shift":            5,
		"assertion-failed":         6,
		"unreachable":              7,
	}
	for category, expected := range want {
		actual, ok := cbackend.TrapCode(category)
		if !ok || actual != expected {
			t.Errorf("TrapCode(%q) = %d, %t; want %d, true", category, actual, ok, expected)
		}
	}
	if code, ok := cbackend.TrapCode("unknown"); ok || code != 0 {
		t.Fatalf("TrapCode(unknown) = %d, %t", code, ok)
	}
}

func TestEmitFlatI32Resource(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, `module main;
resource Pair {
    left: i32;
    right: i32;
    drop fn(self: &mut Self) {}
}
fn transfer(pair: Pair) -> Pair { pair }
fn main() -> i32 {
    let pair: Pair = Pair { left: 1, right: 42 };
    let moved: Pair = transfer(pair);
    moved.right
}
`)
	outcome, err := interpreter.Execute(program, "main::main")
	if err != nil || outcome.Kind != interpreter.OutcomeReturn || outcome.Value != 42 {
		t.Fatalf("interpreter = %+v, %v", outcome, err)
	}
	unit, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	if err != nil {
		t.Fatalf("Emit() error = %v", err)
	}
	wantFeatures := []string{"flat-i32-resource", "scalar-call"}
	if !slices.Equal(unit.Features, wantFeatures) {
		t.Fatalf("Emit() features = %v, want %v", unit.Features, wantFeatures)
	}
	for _, required := range []string{
		"#include <stddef.h>",
		"} fy_resource_0;",
		"_Static_assert(sizeof(fy_resource_0) == UINT64_C(8)",
		"_Static_assert(_Alignof(fy_resource_0) == UINT64_C(4)",
		"offsetof(fy_resource_0, f_0) == UINT64_C(0)",
		"offsetof(fy_resource_0, f_1) == UINT64_C(4)",
		"(fy_resource_0){ .f_0 = INT32_C(1), .f_1 = INT32_C(42) }",
		".f_1;",
	} {
		if !strings.Contains(unit.Source, required) {
			t.Errorf("generated C is missing %q", required)
		}
	}
	for _, leaked := range []string{"Pair", "left", "right"} {
		if strings.Contains(unit.Source, leaked) {
			t.Errorf("generated C leaks source resource identifier %q", leaked)
		}
	}
}

func TestEmitRejectsNonI32ResourceField(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, "module main;\nresource Flag { value: bool; drop fn(self: &mut Self) {} }\nfn main() -> i32 { 0 }\n")
	_, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	var capability cbackend.CapabilityError
	if !errors.As(err, &capability) || !strings.Contains(capability.Feature, "field \"value\" of type \"bool\"") || capability.Function != "" {
		t.Fatalf("Emit() error = %v", err)
	}
}

func TestEmitReprCStructPreservesPublicTagFieldsAndLayout(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, `module main;
@repr(C)
struct fy_std_text_view_v0 { data: *u8; length: usize; }
unsafe extern C fn foreign_take(output: *mut fy_std_text_view_v0) -> u32;
fn main() -> i32 {
    let view: fy_std_text_view_v0 = fy_std_text_view_v0 { data: null, length: 0 };
    if view.length == 0 { 0 } else { 1 }
}
`)
	unit, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	if err != nil {
		t.Fatalf("Emit() error = %v", err)
	}
	for _, required := range []string{
		"struct fy_std_text_view_v0 {",
		"const uint8_t * data;",
		"uintptr_t length;",
		"_Static_assert(sizeof(struct fy_std_text_view_v0) == UINT64_C(16)",
		"offsetof(struct fy_std_text_view_v0, data) == UINT64_C(0)",
		"offsetof(struct fy_std_text_view_v0, length) == UINT64_C(8)",
		"extern uint32_t foreign_take(struct fy_std_text_view_v0 *",
		"(struct fy_std_text_view_v0){ .data = ((void *)0)",
		".length = ((uintptr_t)UINT64_C(0)) }",
		".length;",
	} {
		if !strings.Contains(unit.Source, required) {
			t.Errorf("generated C is missing %q:\n%s", required, unit.Source)
		}
	}
	if !slices.Contains(unit.Features, "inert-struct") || !slices.Contains(unit.Features, "repr-c-struct") {
		t.Fatalf("features = %v", unit.Features)
	}
}

func TestEmitRejectsInvalidReprCStructName(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, "module main;\n@repr(C)\nstruct Packet { value: u32; }\nfn main() -> i32 { 0 }\n")
	program.StructTypes[0].Name = "bad-name"
	_, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	var capability cbackend.CapabilityError
	if !errors.As(err, &capability) || capability.Feature != "repr(C) struct name \"bad-name\"" {
		t.Fatalf("Emit() error = %v", err)
	}
}

func TestEmitRejectsNonScalarParameterType(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, "module main;\nfn inspect(value: &i32) -> i32 { 0 }\nfn main() -> i32 { 0 }\n")
	_, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	var capability cbackend.CapabilityError
	if !errors.As(err, &capability) || capability.Feature != "parameter type \"&i32\"" || capability.Function != "main::inspect" {
		t.Fatalf("Emit() error = %v", err)
	}
}

func TestEmitAcceptsBoolFunctionResult(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, "module main;\nfn positive(value: i32) -> bool { value > 0 }\nfn main() -> i32 { if positive(1) { 42 } else { 0 } }\n")
	unit, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	if err != nil {
		t.Fatalf("Emit() error = %v", err)
	}
	if !slices.Equal(unit.Functions, []string{"main::main", "main::positive"}) {
		t.Fatalf("Emit() functions = %v", unit.Functions)
	}
}

func TestEmitScalarOptionAndResult(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, `module main;
fn choose(present: bool) -> Option<i32> {
    if present { Option::Some(41) } else { Option::None }
}
fn increment(input: Result<i32, i32>) -> Result<i32, i32> {
    let value: i32 = input?;
    Result::Ok(value + 1)
}
fn main() -> i32 {
    let base: i32 = match choose(true) {
        Option::Some(value) => value,
        Option::None => 0,
    };
    match increment(Result::Ok(base)) {
        Result::Ok(value) => value,
        Result::Err(error) => -error,
    }
}
`)
	outcome, err := interpreter.Execute(program, "main::main")
	if err != nil || outcome.Kind != interpreter.OutcomeReturn || outcome.Value != 42 {
		t.Fatalf("interpreter = %+v, %v", outcome, err)
	}
	unit, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	if err != nil {
		t.Fatalf("Emit() error = %v", err)
	}
	wantFeatures := []string{"checked-arithmetic", "scalar-call", "scalar-control-flow", "scalar-option", "scalar-result", "trap"}
	if !slices.Equal(unit.Features, wantFeatures) {
		t.Fatalf("Emit() features = %v, want %v", unit.Features, wantFeatures)
	}
	for _, required := range []string{
		"typedef struct { uint32_t tag; int32_t payload; } fy_option_i32;",
		"typedef struct { uint32_t tag; int32_t payload; } fy_result_i32_i32;",
		"static fy_option_i32",
		"static fy_result_i32_i32",
		"(fy_option_i32){ .tag = FY_OPTION_SOME, .payload = INT32_C(41) }",
		"(fy_result_i32_i32){ .tag = FY_RESULT_OK",
		".tag == FY_OPTION_SOME",
		".tag == FY_RESULT_OK",
		"fy_runtime_trap(FY_TRAP_UNREACHABLE)",
	} {
		if !strings.Contains(unit.Source, required) {
			t.Errorf("generated C is missing %q", required)
		}
	}
}

func TestEmitRejectsNestedScalarTag(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, `module main;
fn main() -> i32 {
    let nested: Option<Option<i32>> = Option::Some(Option::Some(42));
    match nested {
        Option::Some(value) => match value {
            Option::Some(inner) => inner,
            Option::None => 0,
        },
        Option::None => 0,
    }
}
`)
	_, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	var capability cbackend.CapabilityError
	if !errors.As(err, &capability) || !strings.Contains(capability.Feature, "Option construction type") {
		t.Fatalf("Emit() error = %v", err)
	}
}

func TestEmitResourceBearingOption(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, `module main;
resource Token {
    value: i32;
    drop fn(self: &mut Self) {}
}
fn main() -> i32 {
    let wrapped: Option<Token> = Option::Some(Token { value: 42 });
    match wrapped {
        Option::Some(token) => token.value,
        Option::None => 0,
    }
}
`)
	outcome, err := interpreter.Execute(program, "main::main")
	if err != nil || outcome.Kind != interpreter.OutcomeReturn || outcome.Value != 42 {
		t.Fatalf("interpreter = %+v, %v", outcome, err)
	}
	unit, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	if err != nil {
		t.Fatalf("Emit() error = %v", err)
	}
	wantFeatures := []string{"flat-i32-resource", "resource-option", "scalar-control-flow", "trap"}
	if !slices.Equal(unit.Features, wantFeatures) {
		t.Fatalf("Emit() features = %v, want %v", unit.Features, wantFeatures)
	}
	for _, required := range []string{
		"} fy_tagged_0;",
		"_Static_assert(sizeof(fy_tagged_0) == UINT64_C(8)",
		"offsetof(fy_tagged_0, p_0) == UINT64_C(4)",
		"(fy_tagged_0){ .tag = FY_OPTION_SOME, .p_0 =",
		".p_0;",
	} {
		if !strings.Contains(unit.Source, required) {
			t.Errorf("generated C is missing %q", required)
		}
	}
}

func TestEmitRejectsNestedResourceTag(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, `module main;
resource Token {
    value: i32;
    drop fn(self: &mut Self) {}
}
fn main() -> i32 {
    let wrapped: Option<Option<Token>> = Option::Some(Option::Some(Token { value: 42 }));
    42
}
`)
	_, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	var capability cbackend.CapabilityError
	if !errors.As(err, &capability) || !strings.Contains(capability.Feature, "resource local type \"Option<Option<main::Token>>\"") {
		t.Fatalf("Emit() error = %v", err)
	}
}

func TestEmitResourceBearingResultLayout(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, `module main;
resource Token {
    value: i32;
    drop fn(self: &mut Self) {}
}
fn main() -> i32 {
    let wrapped: Result<Token, i32> = Result::Ok(Token { value: 42 });
    match wrapped {
        Result::Ok(token) => token.value,
        Result::Err(error) => -error,
    }
}
`)
	unit, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	if err != nil {
		t.Fatalf("Emit() error = %v", err)
	}
	for _, required := range []string{
		"_Static_assert(sizeof(fy_tagged_0) == UINT64_C(12)",
		"offsetof(fy_tagged_0, p_0) == UINT64_C(4)",
		"offsetof(fy_tagged_0, p_1) == UINT64_C(8)",
		"(fy_tagged_0){ .tag = FY_RESULT_OK, .p_0 =",
		".p_0;",
		".p_1;",
	} {
		if !strings.Contains(unit.Source, required) {
			t.Errorf("generated C is missing %q", required)
		}
	}
}

func TestEmitDropTraceProfileKeepsInstrumentationClosed(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, `module main;
resource Token {
    value: i32;
    drop fn(self: &mut Self) {}
}
fn main() -> i32 {
    let direct: Token = Token { value: 1 };
    discard direct;
    let wrapped: Option<Token> = Option::Some(Token { value: 2 });
    discard wrapped;
    42
}
`)
	production, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	if err != nil {
		t.Fatalf("production Emit() error = %v", err)
	}
	if production.Instrumentation != "" || len(production.DropSites) != 0 || strings.Contains(production.Source, "fy_runtime_trace_drop") {
		t.Fatalf("production unit contains conformance instrumentation: %+v", production)
	}
	instrumented, err := cbackend.Emit(program, "main::main", cbackend.Stage0DropTraceProfile())
	if err != nil {
		t.Fatalf("instrumented Emit() error = %v", err)
	}
	if instrumented.Instrumentation != cbackend.InstrumentationDropTraceV0 || len(instrumented.DropSites) != 2 {
		t.Fatalf("instrumented unit = %+v", instrumented)
	}
	for index, site := range instrumented.DropSites {
		if site.ID != uint32(index) || site.ResourceType != "main::Token" || site.Source != "main.fy" || site.StartByte >= site.EndByte {
			t.Errorf("drop site %d = %+v", index, site)
		}
	}
	for _, required := range []string{
		"void fy_runtime_trace_drop(uint32_t site_id);",
		"fy_runtime_trace_drop(UINT32_C(0));",
		"if (fy_l_",
		".tag == FY_OPTION_SOME) { fy_runtime_trace_drop(UINT32_C(1)); }",
	} {
		if !strings.Contains(instrumented.Source, required) {
			t.Errorf("instrumented C is missing %q", required)
		}
	}
}

func TestEmitRejectsMalformedScalarDestination(t *testing.T) {
	t.Parallel()
	program := lowerSource(t, "module main;\nfn main() -> i32 { let unused: i32 = 1; 42 }\n")
	mutated := mir.Clone(program)
	changed := false
	for functionIndex := range mutated.Functions {
		for blockIndex := range mutated.Functions[functionIndex].Blocks {
			for operationIndex := range mutated.Functions[functionIndex].Blocks[blockIndex].Operations {
				operation := &mutated.Functions[functionIndex].Blocks[blockIndex].Operations[operationIndex]
				if operation.Kind == mir.OperationAssign {
					operation.DestinationLocal = mir.NoLocal
					changed = true
					break
				}
			}
		}
	}
	if !changed {
		t.Fatal("fixture has no assignment to mutate")
	}
	if failures := mir.Verify(mutated); len(failures) != 0 {
		t.Fatalf("mutation must reach the C backend shape guard, verifier failures = %+v", failures)
	}
	_, err := cbackend.Emit(mutated, "main::main", cbackend.Stage0Profile())
	if err == nil || !strings.Contains(err.Error(), "no destination local") {
		t.Fatalf("Emit(malformed destination) error = %v", err)
	}
}

func TestEmitRejectsUnverifiedMIRAndUnknownProfile(t *testing.T) {
	t.Parallel()
	invalid := mir.Program{Functions: []mir.Function{{
		Name: "main::main", ResultType: hir.TypeI32, Entry: 0,
		Blocks: []mir.Block{{ID: 0, Terminator: mir.Terminator{UnwindSuccessor: mir.NoBlock}}},
	}}}
	if _, err := cbackend.Emit(invalid, "main::main", cbackend.Stage0Profile()); err == nil || !strings.Contains(err.Error(), "verified MIR") {
		t.Fatalf("Emit(unverified) error = %v", err)
	}
	program := lowerSource(t, "module main;\nfn main() -> i32 { 0 }\n")
	if _, err := cbackend.Emit(program, "main::main", cbackend.Profile{Target: "host", Dialect: cbackend.DialectC11}); err == nil || !strings.Contains(err.Error(), "unsupported C emission profile") {
		t.Fatalf("Emit(profile) error = %v", err)
	}
	invalidInstrumentation := cbackend.Stage0Profile()
	invalidInstrumentation.Instrumentation = "unbounded-observer"
	if _, err := cbackend.Emit(program, "main::main", invalidInstrumentation); err == nil || !strings.Contains(err.Error(), "unsupported C emission profile") {
		t.Fatalf("Emit(instrumentation) error = %v", err)
	}
}

func lowerSource(t *testing.T, contents string) mir.Program {
	t.Helper()
	return lowerSources(t, map[string]string{"main.fy": contents})
}

func lowerSources(t *testing.T, contents map[string]string) mir.Program {
	t.Helper()
	paths := make([]string, 0, len(contents))
	for path := range contents {
		paths = append(paths, path)
	}
	slices.Sort(paths)
	files := make([]semantic.File, 0, len(paths))
	for _, path := range paths {
		file := source.NewFile(path, []byte(contents[path]))
		parsed, err := frontend.AnalyzeSyntax(file, source.DefaultLimits())
		if err != nil {
			t.Fatalf("AnalyzeSyntax(%s) error = %v", path, err)
		}
		if len(parsed.Tree.Diagnostics) != 0 || parsed.AST == nil {
			t.Fatalf("AnalyzeSyntax(%s) diagnostics = %+v", path, parsed.Tree.Diagnostics)
		}
		files = append(files, semantic.File{Path: path, Module: parsed.AST})
	}
	semanticResult := semantic.Analyze(files)
	if len(semanticResult.Diagnostics) != 0 || len(semanticResult.Unsupported) != 0 {
		t.Fatalf("Analyze() diagnostics = %+v, unsupported = %+v", semanticResult.Diagnostics, semanticResult.Unsupported)
	}
	program, err := mir.Lower(semanticResult.HIR)
	if err != nil {
		t.Fatalf("Lower() error = %v", err)
	}
	return program
}
