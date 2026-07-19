package semantic_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
)

func TestAnalyzeBuildsTypedHIRForScalarFunction(t *testing.T) {
	t.Parallel()
	text := "module main;\n\nfn choose(flag: bool) -> i32 {\n    let value: i32 = 1;\n    if flag { value } else { 2 }\n}\n"
	result := analyzeText(t, map[string]string{"main.fy": text})
	assertCleanTypeResult(t, result)
	if len(result.HIR.Modules) != 1 || len(result.HIR.Modules[0].Functions) != 1 {
		t.Fatalf("HIR = %+v", result.HIR)
	}
	function := result.HIR.Modules[0].Functions[0]
	if function.Name != "choose" || function.ResultType != hir.TypeI32 || len(function.Parameters) != 1 || len(function.Locals) != 1 {
		t.Fatalf("function = %+v", function)
	}
	if function.Parameters[0].Type != hir.TypeBool || function.Locals[0].Type != hir.TypeI32 {
		t.Fatalf("parameters/locals = %+v / %+v", function.Parameters, function.Locals)
	}
	assertTypeFact(t, result, "main.fy", strings.Index(text, "flag: bool"), strings.Index(text, "flag: bool")+4, hir.TypeBool)
	ifStart := strings.Index(text, "if flag")
	ifEnd := strings.Index(text[ifStart:], "}\n}") + ifStart + 1
	assertTypeFact(t, result, "main.fy", ifStart, ifEnd, hir.TypeI32)
}

func TestAnalyzeReportsUnresolvedNameWithoutTypeCascade(t *testing.T) {
	t.Parallel()
	text := "module main;\n\nfn answer() -> i32 {\n    missing\n}\n"
	result := analyzeText(t, map[string]string{"main.fy": text})
	if result.FailedPhase != semantic.PhaseResolve || len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "E0101" {
		t.Fatalf("result = %+v", result)
	}
	start := strings.Index(text, "missing")
	assertSpan(t, result.Diagnostics[0].Primary, "main.fy", start, start+len("missing"))
	if len(result.Types) != 0 || len(result.HIR.Modules) != 0 {
		t.Fatalf("resolve failure manufactured type/HIR facts: %+v", result)
	}
}

func TestAnalyzeReportsDuplicateFunctionWithPreviousDeclaration(t *testing.T) {
	t.Parallel()
	text := "module main;\n\nfn value() -> i32 { 1 }\n\nfn value() -> i32 { 2 }\n"
	result := analyzeText(t, map[string]string{"main.fy": text})
	if result.FailedPhase != semantic.PhaseResolve || len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "E0102" {
		t.Fatalf("result = %+v", result)
	}
	first := strings.Index(text, "value")
	second := strings.LastIndex(text, "value")
	assertSpan(t, result.Diagnostics[0].Primary, "main.fy", second, second+5)
	if len(result.Diagnostics[0].Related) != 1 || result.Diagnostics[0].Related[0].Role != "previous-declaration" {
		t.Fatalf("related = %+v", result.Diagnostics[0].Related)
	}
	assertSpan(t, result.Diagnostics[0].Related[0].Span, "main.fy", first, first+5)
}

func TestAnalyzeReportsCallableDuplicateInSourceOrder(t *testing.T) {
	t.Parallel()
	text := "module main;\n\nunsafe extern C fn value() -> i32;\n\nfn value() -> i32 { 2 }\n"
	result := analyzeText(t, map[string]string{"main.fy": text})
	if result.FailedPhase != semantic.PhaseResolve || len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "E0102" {
		t.Fatalf("result = %+v", result)
	}
	first := strings.Index(text, "value")
	second := strings.LastIndex(text, "value")
	assertSpan(t, result.Diagnostics[0].Primary, "main.fy", second, second+5)
	if len(result.Diagnostics[0].Related) != 1 || result.Diagnostics[0].Related[0].Role != "previous-declaration" {
		t.Fatalf("related = %+v", result.Diagnostics[0].Related)
	}
	assertSpan(t, result.Diagnostics[0].Related[0].Span, "main.fy", first, first+5)
}

func TestAnalyzeReportsBindingTypeMismatch(t *testing.T) {
	t.Parallel()
	text := "module main;\n\nfn answer() -> i32 {\n    let value: bool = 1;\n    0\n}\n"
	result := analyzeText(t, map[string]string{"main.fy": text})
	if result.FailedPhase != semantic.PhaseType || len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "E1006" {
		t.Fatalf("result = %+v", result)
	}
	primary := strings.Index(text, "= 1") + 2
	expected := strings.Index(text, "bool")
	assertSpan(t, result.Diagnostics[0].Primary, "main.fy", primary, primary+1)
	assertSpan(t, result.Diagnostics[0].Related[0].Span, "main.fy", expected, expected+4)
}

func TestAnalyzeReportsNonBooleanCondition(t *testing.T) {
	t.Parallel()
	text := "module main;\n\nfn answer() -> i32 {\n    if 1 { 2 } else { 3 }\n}\n"
	result := analyzeText(t, map[string]string{"main.fy": text})
	if result.FailedPhase != semantic.PhaseType || len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "E1007" {
		t.Fatalf("result = %+v", result)
	}
	start := strings.Index(text, "if 1") + 3
	assertSpan(t, result.Diagnostics[0].Primary, "main.fy", start, start+1)
}

func TestAnalyzeResolvesQualifiedPublicFunction(t *testing.T) {
	t.Parallel()
	mainText := "module main;\n\nimport math as m;\n\nfn answer() -> i32 {\n    m::forty_two()\n}\n"
	mathText := "module math;\n\npub fn forty_two() -> i32 {\n    42\n}\n"
	result := analyzeText(t, map[string]string{"main.fy": mainText, "math.fy": mathText})
	assertCleanTypeResult(t, result)
	useStart := strings.Index(mainText, "m::forty_two")
	declarationStart := strings.Index(mathText, "forty_two")
	found := false
	for _, resolution := range result.Resolutions {
		if resolution.Use.Source == "main.fy" && resolution.Use.StartByte == useStart {
			found = true
			if resolution.Target.Module != "math" || resolution.Target.Name != "forty_two" || resolution.Target.Kind != "function" {
				t.Fatalf("resolution = %+v", resolution)
			}
			assertSpan(t, resolution.Target.Declaration, "math.fy", declarationStart, declarationStart+len("forty_two"))
		}
	}
	if !found {
		t.Fatalf("qualified resolution missing: %+v", result.Resolutions)
	}
}

func TestAnalyzeRejectsUnsupportedLiteralWithoutInventingType(t *testing.T) {
	t.Parallel()
	text := "module main;\n\nfn text() -> i32 {\n    \"not-an-i32\"\n}\n"
	result := analyzeText(t, map[string]string{"main.fy": text})
	if len(result.Unsupported) != 1 || result.Unsupported[0].Feature != "literal string" {
		t.Fatalf("unsupported = %+v", result.Unsupported)
	}
	if result.CompletedThrough != semantic.PhaseResolve {
		t.Fatalf("completed through = %q", result.CompletedThrough)
	}
}

func TestAnalyzeBuildsQualifiedResultConstructorsAndMatch(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": `module main;
fn classify(value: i32) -> Result<i32, i32> {
    if value >= 0 { Result::Ok(value) } else { Result::Err(-value) }
}
fn unwrap(input: Result<i32, i32>) -> i32 {
    match input {
        Result::Ok(value) => value,
        Result::Err(error) => -error,
    }
}
`,
	})
	assertCleanTypeResult(t, result)
	if len(result.HIR.Modules) != 1 || len(result.HIR.Modules[0].Functions) != 2 {
		t.Fatalf("HIR modules = %+v", result.HIR.Modules)
	}
	classify := result.HIR.Modules[0].Functions[0]
	if classify.Body.Tail == nil || classify.Body.Tail.Kind != "if" {
		t.Fatalf("classify tail = %+v", classify.Body.Tail)
	}
	unwrap := result.HIR.Modules[0].Functions[1]
	if unwrap.Body.Tail == nil || unwrap.Body.Tail.Kind != "match" || len(unwrap.Body.Tail.Arms) != 2 {
		t.Fatalf("unwrap tail = %+v", unwrap.Body.Tail)
	}
	if unwrap.Body.Tail.Arms[0].Variant != "ok" || unwrap.Body.Tail.Arms[1].Variant != "err" {
		t.Fatalf("match arms = %+v", unwrap.Body.Tail.Arms)
	}
}

func TestAnalyzeBuildsQualifiedOptionConstructorsAndMatch(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": `module main;
fn choose(present: bool) -> Option<i32> {
    if present { Option::Some(42) } else { Option::None }
}
fn unwrap(input: Option<i32>) -> i32 {
    match input {
        Option::Some(value) => value,
        Option::None => 0,
    }
}
`,
	})
	assertCleanTypeResult(t, result)
	if len(result.HIR.Modules) != 1 || len(result.HIR.Modules[0].Functions) != 2 {
		t.Fatalf("HIR modules = %+v", result.HIR.Modules)
	}
	choose := result.HIR.Modules[0].Functions[0]
	if choose.Body.Tail == nil || choose.Body.Tail.Kind != "if" || len(choose.Body.Tail.Children) != 3 ||
		choose.Body.Tail.Children[1].Block == nil || choose.Body.Tail.Children[1].Block.Tail == nil || choose.Body.Tail.Children[1].Block.Tail.Kind != "option-variant" ||
		choose.Body.Tail.Children[2].Block == nil || choose.Body.Tail.Children[2].Block.Tail == nil || choose.Body.Tail.Children[2].Block.Tail.Variant != "none" {
		t.Fatalf("choose tail = %+v", choose.Body.Tail)
	}
	unwrap := result.HIR.Modules[0].Functions[1]
	if unwrap.Body.Tail == nil || unwrap.Body.Tail.Kind != "match" || len(unwrap.Body.Tail.Arms) != 2 {
		t.Fatalf("unwrap tail = %+v", unwrap.Body.Tail)
	}
	if unwrap.Body.Tail.Arms[0].Variant != "some" || unwrap.Body.Tail.Arms[1].Variant != "none" || unwrap.Body.Tail.Arms[1].Binding != hir.NoSymbol {
		t.Fatalf("Option match arms = %+v", unwrap.Body.Tail.Arms)
	}
}

func TestAnalyzePreservesOrderedResourceDefinitionWithEmptyDrop(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": "module main;\nresource Pair { left: i32; right: bool; drop fn(self: &mut Self) {} }\nfn main() -> i32 { 0 }\n",
	})
	assertCleanTypeResult(t, result)
	if len(result.HIR.Modules) != 1 || len(result.HIR.Modules[0].Resources) != 1 {
		t.Fatalf("HIR modules = %+v", result.HIR.Modules)
	}
	resource := result.HIR.Modules[0].Resources[0]
	if resource.Name != "Pair" || resource.Type != hir.Type("main::Pair") || resource.DropKind != hir.ResourceDropEmpty || len(resource.Fields) != 2 {
		t.Fatalf("resource = %+v", resource)
	}
	if resource.Fields[0].Name != "left" || resource.Fields[0].Type != hir.TypeI32 || resource.Fields[1].Name != "right" || resource.Fields[1].Type != hir.TypeBool {
		t.Fatalf("resource fields = %+v", resource.Fields)
	}
}

func TestAnalyzePreservesReprCStructAndTypesAggregateExpressions(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": `module main;
@repr(C)
struct Packet { code: u32; length: usize; }
fn main() -> i32 {
    let packet: Packet = Packet { code: 42, length: 0 };
    if packet.code == 42 { 0 } else { 1 }
}
`,
	})
	assertCleanTypeResult(t, result)
	if len(result.HIR.Modules) != 1 || len(result.HIR.Modules[0].Structs) != 1 {
		t.Fatalf("HIR modules = %+v", result.HIR.Modules)
	}
	structure := result.HIR.Modules[0].Structs[0]
	if structure.Name != "Packet" || structure.Type != hir.Type("main::Packet") || !structure.ReprC || len(structure.Fields) != 2 {
		t.Fatalf("struct = %+v", structure)
	}
	if structure.Fields[0].Name != "code" || structure.Fields[0].Type != hir.TypeU32 || structure.Fields[1].Name != "length" || structure.Fields[1].Type != hir.TypeUSize {
		t.Fatalf("struct fields = %+v", structure.Fields)
	}
	main := result.HIR.Modules[0].Functions[0]
	if len(main.Locals) != 1 || main.Locals[0].Type != hir.Type("main::Packet") || main.Body.Tail == nil || main.Body.Tail.Kind != "if" {
		t.Fatalf("main = %+v", main)
	}
}

func TestAnalyzePreservesRestrictedInertEnumDefinitions(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": `module main;
struct Position { line: u32; column: u32; }
enum ParseError {
    Empty;
    Unexpected { position: Position; byte: u8; };
}
fn main() -> i32 { 0 }
`,
	})
	assertCleanTypeResult(t, result)
	if len(result.HIR.Modules) != 1 || len(result.HIR.Modules[0].Enums) != 1 {
		t.Fatalf("HIR modules = %+v", result.HIR.Modules)
	}
	enumeration := result.HIR.Modules[0].Enums[0]
	if enumeration.Name != "ParseError" || enumeration.Type != hir.Type("main::ParseError") || len(enumeration.Variants) != 2 {
		t.Fatalf("enum = %+v", enumeration)
	}
	if enumeration.Variants[0].Name != "Empty" || len(enumeration.Variants[0].Fields) != 0 {
		t.Fatalf("payloadless variant = %+v", enumeration.Variants[0])
	}
	variant := enumeration.Variants[1]
	if variant.Name != "Unexpected" || len(variant.Fields) != 2 || variant.Fields[0].Name != "position" || variant.Fields[0].Type != hir.Type("main::Position") || variant.Fields[1].Name != "byte" || variant.Fields[1].Type != hir.TypeU8 {
		t.Fatalf("payload variant = %+v", variant)
	}
}

func TestAnalyzeRejectsNonInertEnumPayloads(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		text string
		code string
	}{
		{name: "borrow", text: "module main;\nenum Bad { Value { payload: &i32; }; }\nfn main() -> i32 { 0 }\n", code: "E1004"},
		{name: "region", text: "module main;\nenum Bad { Value { payload: Region; }; }\nfn main() -> i32 { 0 }\n", code: "E3103"},
		{name: "resource", text: "module main;\nresource Token { value: i32; drop fn(self: &mut Self) {} }\nenum Bad { Value { payload: Token; }; }\nfn main() -> i32 { 0 }\n", code: "E1004"},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			result := analyzeText(t, map[string]string{"main.fy": test.text})
			if len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != test.code {
				t.Fatalf("diagnostics = %+v, unsupported = %+v", result.Diagnostics, result.Unsupported)
			}
		})
	}
}

func TestAnalyzeRejectsDuplicateEnumVariantsAndFields(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": "module main;\nenum Bad { Same; Same; Payload { value: i32; value: u32; }; }\nfn main() -> i32 { 0 }\n",
	})
	if len(result.Diagnostics) != 2 || result.Diagnostics[0].Code != "E0102" || result.Diagnostics[1].Code != "E0102" {
		t.Fatalf("diagnostics = %+v", result.Diagnostics)
	}
}

func TestAnalyzeRejectsUnsafeFieldsInInertStructs(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		text string
		code string
	}{
		{name: "borrow", text: "module main;\nstruct Holder { value: &i32; }\nfn main() -> i32 { 0 }\n", code: "E1003"},
		{name: "region", text: "module main;\nstruct Holder { value: Region; }\nfn main() -> i32 { 0 }\n", code: "E3103"},
		{name: "resource", text: "module main;\nresource Token { value: i32; drop fn(self: &mut Self) {} }\nstruct Holder { value: Token; }\nfn main() -> i32 { 0 }\n", code: "E1002"},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			result := analyzeText(t, map[string]string{"main.fy": test.text})
			if result.FailedPhase != semantic.PhaseType || len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != test.code {
				t.Fatalf("result = %+v", result)
			}
		})
	}
}

func TestAnalyzeRejectsUnsupportedStructRepresentation(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": "module main;\n@repr(packed)\nstruct Packet { value: u32; }\nfn main() -> i32 { 0 }\n",
	})
	if len(result.Unsupported) != 1 || result.Unsupported[0].Feature != "struct attribute @repr" {
		t.Fatalf("result = %+v", result)
	}
}

func TestAnalyzeRejectsNonEmptyResourceDropBodyAsUnsupported(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": "module main;\nresource Token { value: i32; drop fn(self: &mut Self) { let marker: i32 = 1; } }\nfn main() -> i32 { 0 }\n",
	})
	if len(result.Unsupported) != 1 || result.Unsupported[0].Feature != "non-empty resource drop body" {
		t.Fatalf("unsupported = %+v", result.Unsupported)
	}
	if result.CompletedThrough != semantic.PhaseResolve || len(result.HIR.Modules) != 1 || len(result.HIR.Modules[0].Resources) != 1 || result.HIR.Modules[0].Resources[0].DropKind != hir.ResourceDropNonEmpty {
		t.Fatalf("result = %+v", result)
	}
}

func TestAnalyzeTypesForeignCallInsideUnsafeBlock(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": "module main;\nunsafe extern C fn add_one(value: i32) -> i32;\nfn main() -> i32 { unsafe { add_one(41) } }\n",
	})
	assertCleanTypeResult(t, result)
	if len(result.HIR.Modules) != 1 || len(result.HIR.Modules[0].ForeignFunctions) != 1 {
		t.Fatalf("HIR = %+v", result.HIR)
	}
	foreign := result.HIR.Modules[0].ForeignFunctions[0]
	if foreign.Name != "add_one" || foreign.LinkName != "add_one" || foreign.ABI != "C" || foreign.ResultType != hir.TypeI32 || len(foreign.Parameters) != 1 || foreign.Parameters[0].Type != hir.TypeI32 {
		t.Fatalf("foreign function = %+v", foreign)
	}
	tail := result.HIR.Modules[0].Functions[0].Body.Tail
	if tail == nil || tail.Kind != "unsafe" || len(tail.Children) != 1 || tail.Children[0].Block == nil || tail.Children[0].Block.Tail == nil {
		t.Fatalf("main tail = %+v", tail)
	}
	call := tail.Children[0].Block.Tail
	if call.Kind != "call" || call.ForeignABI != "C" || call.UnsafeScope == nil || call.Symbol != foreign.Symbol {
		t.Fatalf("foreign call = %+v", call)
	}
}

func TestAnalyzeTypesRawPointerAndTargetScalarForeignCall(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": "module main;\nunsafe extern C fn foreign_probe(data: *u8, output: *mut u32, length: usize, marker: u8) -> u32;\nfn main() -> i32 { if unsafe { foreign_probe(null, null, 0, 7) } == 42 { 0 } else { 1 } }\n",
	})
	assertCleanTypeResult(t, result)
	foreign := result.HIR.Modules[0].ForeignFunctions[0]
	wantParameters := []hir.Type{
		hir.RawPointer(hir.TypeU8, false),
		hir.RawPointer(hir.TypeU32, true),
		hir.TypeUSize,
		hir.TypeU8,
	}
	actualParameters := make([]hir.Type, len(foreign.Parameters))
	for index := range foreign.Parameters {
		actualParameters[index] = foreign.Parameters[index].Type
	}
	if foreign.ResultType != hir.TypeU32 || !slices.Equal(actualParameters, wantParameters) {
		t.Fatalf("foreign signature = %+v", foreign)
	}
	tail := result.HIR.Modules[0].Functions[0].Body.Tail
	if tail == nil || tail.Kind != "if" || len(tail.Children) < 1 {
		t.Fatalf("main tail = %+v", tail)
	}
	condition := &tail.Children[0]
	if condition.Kind != "binary" || len(condition.Children) != 2 || condition.Children[0].Kind != "unsafe" || len(condition.Children[0].Children) != 1 {
		t.Fatalf("if condition = %+v", condition)
	}
	unsafeBlock := &condition.Children[0].Children[0]
	if unsafeBlock.Block == nil || unsafeBlock.Block.Tail == nil {
		t.Fatalf("unsafe block = %+v", unsafeBlock)
	}
	call := unsafeBlock.Block.Tail
	if call.Kind != "call" || call.Type != hir.TypeU32 || len(call.Children) != 5 {
		t.Fatalf("foreign call = %+v", call)
	}
	for index, want := range wantParameters {
		if call.Children[index+1].Type != want {
			t.Errorf("argument %d type = %q, want %q", index, call.Children[index+1].Type, want)
		}
	}
}

func TestAnalyzeRejectsForeignCallOutsideUnsafeBoundary(t *testing.T) {
	t.Parallel()
	text := "module main;\nunsafe extern C fn add_one(value: i32) -> i32;\nfn main() -> i32 { add_one(41) }\n"
	result := analyzeText(t, map[string]string{"main.fy": text})
	if result.FailedPhase != semantic.PhaseType || len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "E5101" {
		t.Fatalf("result = %+v", result)
	}
	start := strings.LastIndex(text, "add_one")
	assertSpan(t, result.Diagnostics[0].Primary, "main.fy", start, start+len("add_one(41)"))
}

func TestAnalyzeTypesBorrowToRawPointerCastInsideUnsafeBoundary(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": `module main;
@repr(C)
struct Packet { code: u32; }
unsafe extern C fn sink(output: *mut Packet) -> u32;
fn main() -> i32 {
    var output: Packet = Packet { code: 0 };
    if unsafe { sink(&mut output as *mut Packet) } == 0 { 0 } else { 1 }
}
`,
	})
	assertCleanTypeResult(t, result)
	main := result.HIR.Modules[0].Functions[0]
	condition := &main.Body.Tail.Children[0]
	unsafeExpression := &condition.Children[0]
	call := unsafeExpression.Children[0].Block.Tail
	if call == nil || len(call.Children) != 2 {
		t.Fatalf("call = %+v", call)
	}
	cast := &call.Children[1]
	wantType := hir.RawPointer(hir.Type("main::Packet"), true)
	if cast.Kind != "cast" || cast.Operator != "as" || cast.Type != wantType || cast.TargetType != wantType || cast.UnsafeScope == nil || len(cast.Children) != 1 || cast.Children[0].Type != hir.MutableBorrow(hir.Type("main::Packet")) {
		t.Fatalf("cast = %+v", cast)
	}
}

func TestAnalyzeRejectsBorrowToRawPointerCastOutsideUnsafeBoundary(t *testing.T) {
	t.Parallel()
	text := "module main;\nfn main() -> i32 { var value: i32 = 0; let pointer: *mut i32 = &mut value as *mut i32; 0 }\n"
	result := analyzeText(t, map[string]string{"main.fy": text})
	if result.FailedPhase != semantic.PhaseType || len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "E5101" {
		t.Fatalf("result = %+v", result)
	}
	start := strings.Index(text, "&mut value")
	assertSpan(t, result.Diagnostics[0].Primary, "main.fy", start, start+len("&mut value as *mut i32"))
}

func TestAnalyzeRejectsRawPointerCastThatChangesPointeeOrStrengthensMutability(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		source  string
		feature string
	}{
		{
			name:    "mutability strengthening",
			source:  "module main;\nfn main() -> i32 { let value: i32 = 0; let pointer: *mut i32 = unsafe { &value as *mut i32 }; 0 }\n",
			feature: "cast from &i32 to *mut i32",
		},
		{
			name:    "pointee change",
			source:  "module main;\nfn main() -> i32 { var value: i32 = 0; let pointer: *mut u32 = unsafe { &mut value as *mut u32 }; 0 }\n",
			feature: "cast from &mut i32 to *mut u32",
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			result := analyzeText(t, map[string]string{"main.fy": test.source})
			if len(result.Unsupported) != 1 || result.Unsupported[0].Feature != test.feature {
				t.Fatalf("result = %+v", result)
			}
		})
	}
}

func TestAnalyzeRejectsSafeBorrowInForeignSignature(t *testing.T) {
	t.Parallel()
	text := "module main;\nunsafe extern C fn inspect(value: &i32) -> i32;\nfn main() -> i32 { 0 }\n"
	result := analyzeText(t, map[string]string{"main.fy": text})
	if result.FailedPhase != semantic.PhaseType || len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "E5102" {
		t.Fatalf("result = %+v", result)
	}
	start := strings.Index(text, "&i32")
	assertSpan(t, result.Diagnostics[0].Primary, "main.fy", start, start+len("&i32"))
}

func TestAnalyzeRejectsDuplicateForeignParameterAtResolution(t *testing.T) {
	t.Parallel()
	text := "module main;\nunsafe extern C fn combine(value: i32, value: i32) -> i32;\nfn main() -> i32 { 0 }\n"
	result := analyzeText(t, map[string]string{"main.fy": text})
	if result.FailedPhase != semantic.PhaseResolve || len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "E0102" {
		t.Fatalf("result = %+v", result)
	}
	first := strings.Index(text, "value")
	second := strings.LastIndex(text, "value")
	assertSpan(t, result.Diagnostics[0].Primary, "main.fy", second, second+len("value"))
	if len(result.Diagnostics[0].Related) != 1 {
		t.Fatalf("related = %+v", result.Diagnostics[0].Related)
	}
	assertSpan(t, result.Diagnostics[0].Related[0].Span, "main.fy", first, first+len("value"))
}

func analyzeText(t *testing.T, files map[string]string) semantic.Result {
	t.Helper()
	inputs := make([]semantic.File, 0, len(files))
	for path, text := range files {
		frontendResult, err := frontend.AnalyzeSyntax(source.NewFile(path, []byte(text)), source.DefaultLimits())
		if err != nil {
			t.Fatalf("AnalyzeSyntax(%s) error = %v", path, err)
		}
		if len(frontendResult.Tree.Diagnostics) != 0 || frontendResult.AST == nil {
			t.Fatalf("AnalyzeSyntax(%s) = diagnostics %+v, AST %+v", path, frontendResult.Tree.Diagnostics, frontendResult.AST)
		}
		inputs = append(inputs, semantic.File{Path: path, Module: frontendResult.AST})
	}
	return semantic.Analyze(inputs)
}

func assertCleanTypeResult(t *testing.T, result semantic.Result) {
	t.Helper()
	if result.CompletedThrough != semantic.PhaseFlow || result.FailedPhase != semantic.PhaseNone || len(result.Diagnostics) != 0 || len(result.Unsupported) != 0 {
		t.Fatalf("semantic result = %+v", result)
	}
}

func assertTypeFact(t *testing.T, result semantic.Result, sourcePath string, start, end int, expected hir.Type) {
	t.Helper()
	for _, fact := range result.Types {
		if fact.Expression.Source == sourcePath && fact.Expression.StartByte == start && fact.Expression.EndByte == end && fact.CanonicalType == expected {
			return
		}
	}
	t.Fatalf("type fact %s:%d..%d = %s missing from %+v", sourcePath, start, end, expected, result.Types)
}

func assertSpan(t *testing.T, actual hir.Span, sourcePath string, start, end int) {
	t.Helper()
	if actual.Source != sourcePath || actual.StartByte != start || actual.EndByte != end {
		t.Fatalf("span = %+v, want %s:%d..%d", actual, sourcePath, start, end)
	}
}
