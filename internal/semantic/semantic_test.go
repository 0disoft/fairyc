package semantic_test

import (
	"fmt"
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

func TestAnalyzeTypesRadixAndSeparatedIntegerLiterals(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": `module main;
fn main() -> i32 {
    let byte: u8 = 0xFF;
    let word: u32 = 0xDEAD_BEEF;
    let size: usize = 0b1111_1111;
    if byte == 255 && word == 3735928559 && size == 255 { 42 } else { 1 }
}
`,
	})
	assertCleanTypeResult(t, result)
}

func TestAnalyzeRejectsIntegerLiteralOutsideSelectedType(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name    string
		typeRef string
		value   string
	}{
		{name: "i32", typeRef: "i32", value: "2147483648"},
		{name: "u8", typeRef: "u8", value: "0x100"},
		{name: "u32", typeRef: "u32", value: "4_294_967_296"},
		{name: "usize", typeRef: "usize", value: "18_446_744_073_709_551_616"},
	} {
		test := test
		t.Run(test.name, func(t *testing.T) {
			text := "module main;\nfn main() -> i32 { let value: " + test.typeRef + " = " + test.value + "; 0 }\n"
			result := analyzeText(t, map[string]string{"main.fy": text})
			if result.FailedPhase != semantic.PhaseType || len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "E1013" {
				t.Fatalf("result = %+v", result)
			}
			start := strings.Index(text, test.value)
			assertSpan(t, result.Diagnostics[0].Primary, "main.fy", start, start+len(test.value))
		})
	}
}

func TestAnalyzeFoldsDirectI32MinimumLiteral(t *testing.T) {
	t.Parallel()
	for _, value := range []string{"-2147483648", "-0x8000_0000", "-0b10000000000000000000000000000000"} {
		value := value
		t.Run(value, func(t *testing.T) {
			result := analyzeText(t, map[string]string{
				"main.fy": "module main;\nfn main() -> i32 { " + value + " }\n",
			})
			assertCleanTypeResult(t, result)
			tail := result.HIR.Modules[0].Functions[0].Body.Tail
			if tail == nil || tail.Kind != "literal" || tail.Type != hir.TypeI32 || tail.Value != "-2147483648" || len(tail.Children) != 0 {
				t.Fatalf("folded minimum = %+v", tail)
			}
		})
	}
}

func TestAnalyzeRejectsIntegerMagnitudeBeyondDirectI32Minimum(t *testing.T) {
	t.Parallel()
	for _, value := range []string{"-2147483649", "-0x8000_0001"} {
		value := value
		t.Run(value, func(t *testing.T) {
			text := "module main;\nfn main() -> i32 { " + value + " }\n"
			result := analyzeText(t, map[string]string{"main.fy": text})
			if result.FailedPhase != semantic.PhaseType || len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "E1013" {
				t.Fatalf("result = %+v", result)
			}
			start := strings.Index(text, value) + 1
			assertSpan(t, result.Diagnostics[0].Primary, "main.fy", start, start+len(value)-1)
		})
	}
}

func TestAnalyzeRejectsVeryLongIntegerWithoutHostOverflow(t *testing.T) {
	t.Parallel()
	value := strings.Repeat("9", 64<<10)
	text := "module main;\nfn main() -> i32 { " + value + " }\n"
	result := analyzeText(t, map[string]string{"main.fy": text})
	if result.FailedPhase != semantic.PhaseType || len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "E1013" {
		t.Fatalf("diagnostics = %+v, unsupported = %+v", result.Diagnostics, result.Unsupported)
	}
	start := strings.Index(text, value)
	assertSpan(t, result.Diagnostics[0].Primary, "main.fy", start, start+len(value))
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
	text := "module main;\n\nfn text() -> i32 {\n    1.5\n}\n"
	result := analyzeText(t, map[string]string{"main.fy": text})
	if len(result.Unsupported) != 1 || result.Unsupported[0].Feature != "literal float" {
		t.Fatalf("unsupported = %+v", result.Unsupported)
	}
	if result.CompletedThrough != semantic.PhaseResolve {
		t.Fatalf("completed through = %q", result.CompletedThrough)
	}
}

func TestAnalyzeTypesCStringLiteralWithStaticBorrowOrigin(t *testing.T) {
	t.Parallel()
	text := "module main;\n\nfn main() -> i32 {\n    let name: &CStr = c\"fairy\";\n    42\n}\n"
	result := analyzeText(t, map[string]string{"main.fy": text})
	assertCleanTypeResult(t, result)

	if len(result.HIR.Modules) != 1 || len(result.HIR.Modules[0].Functions) != 1 {
		t.Fatalf("HIR modules = %+v", result.HIR.Modules)
	}
	function := result.HIR.Modules[0].Functions[0]
	wantType := hir.SharedBorrow(hir.TypeCStr)
	if len(function.Locals) != 1 || function.Locals[0].Type != wantType {
		t.Fatalf("locals = %+v, want one %q local", function.Locals, wantType)
	}
	if len(function.Body.Statements) != 1 || function.Body.Statements[0].Expression == nil {
		t.Fatalf("body = %+v", function.Body)
	}
	literal := function.Body.Statements[0].Expression
	if literal.Kind != "literal" || literal.Type != wantType || literal.Value != "c\"fairy\"" || literal.Cooked != "666169727900" {
		t.Fatalf("literal = %+v", literal)
	}

	start := strings.Index(text, "c\"fairy\"")
	if len(result.Borrows) != 1 || result.Borrows[0].Kind != "shared" || result.Borrows[0].Origin != "Static" {
		t.Fatalf("borrows = %+v", result.Borrows)
	}
	assertSpan(t, result.Borrows[0].Borrow, "main.fy", start, start+len("c\"fairy\""))
}

func TestAnalyzeTypesByteStringLiteralWithStaticBorrowOrigin(t *testing.T) {
	t.Parallel()
	text := "module main;\n\nfn main() -> i32 {\n    let bytes: &[u8] = b\"\\x00A\";\n    42\n}\n"
	result := analyzeText(t, map[string]string{"main.fy": text})
	assertCleanTypeResult(t, result)

	if len(result.HIR.Modules) != 1 || len(result.HIR.Modules[0].Functions) != 1 {
		t.Fatalf("HIR modules = %+v", result.HIR.Modules)
	}
	function := result.HIR.Modules[0].Functions[0]
	wantType := hir.SharedBorrow(hir.TypeByteSlice)
	if len(function.Locals) != 1 || function.Locals[0].Type != wantType {
		t.Fatalf("locals = %+v, want one %q local", function.Locals, wantType)
	}
	if len(function.Body.Statements) != 1 || function.Body.Statements[0].Expression == nil {
		t.Fatalf("body = %+v", function.Body)
	}
	literal := function.Body.Statements[0].Expression
	if literal.Kind != "literal" || literal.Type != wantType || literal.Value != `b"\x00A"` || literal.Cooked != "0041" {
		t.Fatalf("literal = %+v", literal)
	}

	start := strings.Index(text, `b"\x00A"`)
	if len(result.Borrows) != 1 || result.Borrows[0].Kind != "shared" || result.Borrows[0].Origin != "Static" {
		t.Fatalf("borrows = %+v", result.Borrows)
	}
	assertSpan(t, result.Borrows[0].Borrow, "main.fy", start, start+len(`b"\x00A"`))
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

func TestAnalyzeBuildsRestrictedEnumConstructorsAndExhaustiveMatch(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": `module main;
enum Message {
    Empty;
    Value { amount: i32; };
    Pair { left: i32; right: i32; };
}
fn score(message: Message) -> i32 {
    match message {
        Message::Empty => 0,
        Message::Value { amount: value } => value,
        Message::Pair { left: first, right: second } => first + second,
    }
}
fn main() -> i32 { score(Message::Pair { left: 20, right: 22 }) }
`,
	})
	assertCleanTypeResult(t, result)
	if len(result.HIR.Modules) != 1 || len(result.HIR.Modules[0].Functions) != 2 {
		t.Fatalf("HIR modules = %+v", result.HIR.Modules)
	}
	score := result.HIR.Modules[0].Functions[0]
	if score.Body.Tail == nil || score.Body.Tail.Kind != "match" || len(score.Body.Tail.Arms) != 3 {
		t.Fatalf("score = %+v", score)
	}
	if score.Body.Tail.Arms[0].Variant != "Empty" || len(score.Body.Tail.Arms[0].Bindings) != 0 {
		t.Fatalf("empty arm = %+v", score.Body.Tail.Arms[0])
	}
	if bindings := score.Body.Tail.Arms[2].Bindings; len(bindings) != 2 || bindings[0].Field != "left" || bindings[0].Type != hir.TypeI32 || bindings[1].Field != "right" || bindings[1].Type != hir.TypeI32 {
		t.Fatalf("pair bindings = %+v", bindings)
	}
	main := result.HIR.Modules[0].Functions[1]
	if main.Body.Tail == nil || main.Body.Tail.Kind != "call" || len(main.Body.Tail.Children) != 2 || main.Body.Tail.Children[1].Kind != "enum-variant" || main.Body.Tail.Children[1].Variant != "Pair" {
		t.Fatalf("main = %+v", main)
	}
}

func TestAnalyzeResolvesPublicEnumVariantAcrossImportAlias(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"api.fy":  "module api;\npub enum Status { Ready; Code { value: i32; }; }\n",
		"main.fy": "module main;\nimport api as a;\nfn main() -> i32 { match (a::Status::Code { value: 42 }) { a::Status::Ready => 0, a::Status::Code { value: code } => code, } }\n",
	})
	assertCleanTypeResult(t, result)
	if len(result.Resolutions) < 3 {
		t.Fatalf("resolutions = %+v", result.Resolutions)
	}
}

func TestAnalyzeRejectsNonExhaustiveRestrictedEnumMatch(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": "module main;\nenum Status { Ready; Failed; }\nfn main() -> i32 { match Status::Ready { Status::Ready => 0, } }\n",
	})
	if len(result.Unsupported) != 1 || result.Unsupported[0].Feature != "non-exhaustive enum match" {
		t.Fatalf("diagnostics = %+v, unsupported = %+v", result.Diagnostics, result.Unsupported)
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
		"main.fy": "module main;\n@link_name(\"fairyc_fixture_add_one\")\nunsafe extern C fn add_one(value: i32) -> i32;\nfn main() -> i32 { unsafe { add_one(41) } }\n",
	})
	assertCleanTypeResult(t, result)
	if len(result.HIR.Modules) != 1 || len(result.HIR.Modules[0].ForeignFunctions) != 1 {
		t.Fatalf("HIR = %+v", result.HIR)
	}
	foreign := result.HIR.Modules[0].ForeignFunctions[0]
	if foreign.Name != "add_one" || foreign.LinkName != "fairyc_fixture_add_one" || foreign.ABI != "C" || foreign.ResultType != hir.TypeI32 || len(foreign.Parameters) != 1 || foreign.Parameters[0].Type != hir.TypeI32 {
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

func TestAnalyzePreservesDistinctCIntForeignTransport(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": `module main;
@link_name("fairyc_fixture_make_c_int")
unsafe extern C fn make_c_int() -> c::int;
@link_name("fairyc_fixture_consume_c_int")
unsafe extern C fn consume_c_int(value: c::int) -> i32;
fn main() -> i32 { unsafe { consume_c_int(make_c_int()) } }
`,
	})
	assertCleanTypeResult(t, result)
	if len(result.HIR.Modules) != 1 || len(result.HIR.Modules[0].ForeignFunctions) != 2 {
		t.Fatalf("HIR = %+v", result.HIR)
	}
	producer := result.HIR.Modules[0].ForeignFunctions[0]
	consumer := result.HIR.Modules[0].ForeignFunctions[1]
	if producer.Name != "make_c_int" || producer.ResultType != hir.TypeCInt || producer.ResultType == hir.TypeI32 || len(producer.Parameters) != 0 {
		t.Fatalf("producer = %+v", producer)
	}
	if consumer.Name != "consume_c_int" || consumer.ResultType != hir.TypeI32 || len(consumer.Parameters) != 1 || consumer.Parameters[0].Type != hir.TypeCInt {
		t.Fatalf("consumer = %+v", consumer)
	}
	tail := result.HIR.Modules[0].Functions[0].Body.Tail
	if tail == nil || tail.Kind != "unsafe" || len(tail.Children) != 1 || tail.Children[0].Block == nil || tail.Children[0].Block.Tail == nil {
		t.Fatalf("main tail = %+v", tail)
	}
	outer := tail.Children[0].Block.Tail
	if outer.Kind != "call" || outer.Type != hir.TypeI32 || len(outer.Children) != 2 || outer.Children[1].Kind != "call" || outer.Children[1].Type != hir.TypeCInt {
		t.Fatalf("nested foreign calls = %+v", outer)
	}
}

func TestAnalyzePreservesDistinctCSizeTForeignTransport(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": `module main;
@link_name("fairyc_fixture_make_c_size_t")
unsafe extern C fn make_c_size_t() -> c::size_t;
@link_name("fairyc_fixture_consume_c_size_t")
unsafe extern C fn consume_c_size_t(value: c::size_t) -> i32;
fn main() -> i32 { unsafe { consume_c_size_t(make_c_size_t()) } }
`,
	})
	assertCleanTypeResult(t, result)
	if len(result.HIR.Modules) != 1 || len(result.HIR.Modules[0].ForeignFunctions) != 2 {
		t.Fatalf("HIR = %+v", result.HIR)
	}
	producer := result.HIR.Modules[0].ForeignFunctions[0]
	consumer := result.HIR.Modules[0].ForeignFunctions[1]
	if producer.Name != "make_c_size_t" || producer.ResultType != hir.TypeCSizeT || producer.ResultType == hir.TypeUSize || len(producer.Parameters) != 0 {
		t.Fatalf("producer = %+v", producer)
	}
	if consumer.Name != "consume_c_size_t" || consumer.ResultType != hir.TypeI32 || len(consumer.Parameters) != 1 || consumer.Parameters[0].Type != hir.TypeCSizeT {
		t.Fatalf("consumer = %+v", consumer)
	}
}

func TestAnalyzePreservesDistinctCLongForeignTransport(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": `module main;
@link_name("fairyc_fixture_make_c_long")
unsafe extern C fn make_c_long() -> c::long;
@link_name("fairyc_fixture_consume_c_long")
unsafe extern C fn consume_c_long(value: c::long) -> i32;
fn main() -> i32 { unsafe { consume_c_long(make_c_long()) } }
`,
	})
	assertCleanTypeResult(t, result)
	if len(result.HIR.Modules) != 1 || len(result.HIR.Modules[0].ForeignFunctions) != 2 {
		t.Fatalf("HIR = %+v", result.HIR)
	}
	producer := result.HIR.Modules[0].ForeignFunctions[0]
	consumer := result.HIR.Modules[0].ForeignFunctions[1]
	if producer.Name != "make_c_long" || producer.ResultType != hir.TypeCLong || len(producer.Parameters) != 0 {
		t.Fatalf("producer = %+v", producer)
	}
	if consumer.Name != "consume_c_long" || consumer.ResultType != hir.TypeI32 || len(consumer.Parameters) != 1 || consumer.Parameters[0].Type != hir.TypeCLong {
		t.Fatalf("consumer = %+v", consumer)
	}
}

func TestAnalyzePreservesDistinctCSSizeTForeignTransport(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": `module main;
@link_name("fairyc_fixture_make_c_ssize_t")
unsafe extern C fn make_c_ssize_t() -> c::ssize_t;
@link_name("fairyc_fixture_consume_c_ssize_t")
unsafe extern C fn consume_c_ssize_t(value: c::ssize_t) -> i32;
fn main() -> i32 { unsafe { consume_c_ssize_t(make_c_ssize_t()) } }
`,
	})
	assertCleanTypeResult(t, result)
	if len(result.HIR.Modules) != 1 || len(result.HIR.Modules[0].ForeignFunctions) != 2 {
		t.Fatalf("HIR = %+v", result.HIR)
	}
	producer := result.HIR.Modules[0].ForeignFunctions[0]
	consumer := result.HIR.Modules[0].ForeignFunctions[1]
	if producer.Name != "make_c_ssize_t" || producer.ResultType != hir.TypeCSSizeT || producer.ResultType == hir.TypeUSize || len(producer.Parameters) != 0 {
		t.Fatalf("producer = %+v", producer)
	}
	if consumer.Name != "consume_c_ssize_t" || consumer.ResultType != hir.TypeI32 || len(consumer.Parameters) != 1 || consumer.Parameters[0].Type != hir.TypeCSSizeT {
		t.Fatalf("consumer = %+v", consumer)
	}
}

func TestAnalyzePreservesRemainingCScalarForeignTransport(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		typeRef  string
		wantType hir.Type
	}{
		{name: "bool", typeRef: "c::bool", wantType: hir.TypeCBool},
		{name: "char", typeRef: "c::char", wantType: hir.TypeCChar},
		{name: "double", typeRef: "c::double", wantType: hir.TypeCDouble},
		{name: "float", typeRef: "c::float", wantType: hir.TypeCFloat},
		{name: "longlong", typeRef: "c::longlong", wantType: hir.TypeCLongLong},
		{name: "schar", typeRef: "c::schar", wantType: hir.TypeCSChar},
		{name: "short", typeRef: "c::short", wantType: hir.TypeCShort},
		{name: "uchar", typeRef: "c::uchar", wantType: hir.TypeCUChar},
		{name: "uint", typeRef: "c::uint", wantType: hir.TypeCUInt},
		{name: "ulong", typeRef: "c::ulong", wantType: hir.TypeCULong},
		{name: "ulonglong", typeRef: "c::ulonglong", wantType: hir.TypeCULongLong},
		{name: "ushort", typeRef: "c::ushort", wantType: hir.TypeCUShort},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			text := fmt.Sprintf(`module main;
unsafe extern C fn make_%[1]s() -> %[2]s;
unsafe extern C fn consume_%[1]s(value: %[2]s) -> i32;
fn main() -> i32 { unsafe { consume_%[1]s(make_%[1]s()) } }
`, test.name, test.typeRef)
			result := analyzeText(t, map[string]string{"main.fy": text})
			assertCleanTypeResult(t, result)
			if len(result.HIR.Modules) != 1 || len(result.HIR.Modules[0].ForeignFunctions) != 2 {
				t.Fatalf("HIR = %+v", result.HIR)
			}
			producer := result.HIR.Modules[0].ForeignFunctions[0]
			consumer := result.HIR.Modules[0].ForeignFunctions[1]
			if producer.ResultType != test.wantType || len(producer.Parameters) != 0 ||
				consumer.ResultType != hir.TypeI32 || len(consumer.Parameters) != 1 || consumer.Parameters[0].Type != test.wantType {
				t.Fatalf("foreign functions = %+v", result.HIR.Modules[0].ForeignFunctions)
			}
		})
	}
}

func TestAnalyzeKeepsCScalarImmediatesDeferred(t *testing.T) {
	t.Parallel()
	for _, cScalar := range hir.CScalarTypes() {
		typeRef := string(cScalar)
		t.Run(typeRef+" immediate", func(t *testing.T) {
			text := "module main;\nfn deferred(value: " + typeRef + ") -> i32 { let copy: " + typeRef + " = 1; 0 }\nfn main() -> i32 { 0 }\n"
			result := analyzeText(t, map[string]string{"main.fy": text})
			if result.FailedPhase != semantic.PhaseType || len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "E1006" || len(result.Unsupported) != 0 {
				t.Fatalf("result = %+v", result)
			}
		})
	}
}

func TestAnalyzeRejectsMalformedOrDuplicateLinkName(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name string
		text string
		last bool
	}{
		{name: "no argument", text: "module main;\n@link_name\nunsafe extern C fn add_one(value: i32) -> i32;\nfn main() -> i32 { 0 }\n"},
		{name: "empty string", text: "module main;\n@link_name(\"\")\nunsafe extern C fn add_one(value: i32) -> i32;\nfn main() -> i32 { 0 }\n"},
		{name: "path argument", text: "module main;\n@link_name(C)\nunsafe extern C fn add_one(value: i32) -> i32;\nfn main() -> i32 { 0 }\n"},
		{name: "two arguments", text: "module main;\n@link_name(\"first\", \"second\")\nunsafe extern C fn add_one(value: i32) -> i32;\nfn main() -> i32 { 0 }\n"},
		{name: "byte string", text: "module main;\n@link_name(b\"symbol\")\nunsafe extern C fn add_one(value: i32) -> i32;\nfn main() -> i32 { 0 }\n"},
		{name: "C string", text: "module main;\n@link_name(c\"symbol\")\nunsafe extern C fn add_one(value: i32) -> i32;\nfn main() -> i32 { 0 }\n"},
		{name: "duplicate", text: "module main;\n@link_name(\"first\")\n@link_name(\"second\")\nunsafe extern C fn add_one(value: i32) -> i32;\nfn main() -> i32 { 0 }\n", last: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			result := analyzeText(t, map[string]string{"main.fy": test.text})
			if result.FailedPhase != semantic.PhaseType || len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "E5106" {
				t.Fatalf("result = %+v", result)
			}
			start := strings.Index(test.text, "@link_name")
			if test.last {
				start = strings.LastIndex(test.text, "@link_name")
			}
			end := strings.Index(test.text[start:], "\n")
			if end < 0 {
				end = len(test.text) - start
			}
			assertSpan(t, result.Diagnostics[0].Primary, "main.fy", start, start+end)
		})
	}
}

func TestAnalyzeDoesNotSilentlyAcceptUnknownForeignAttribute(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": "module main;\n@calling_convention(C)\nunsafe extern C fn add_one(value: i32) -> i32;\nfn main() -> i32 { 0 }\n",
	})
	if len(result.Diagnostics) != 0 || len(result.Unsupported) != 1 || result.Unsupported[0].Feature != "foreign function attribute @calling_convention" {
		t.Fatalf("result = %+v", result)
	}
}

func TestAnalyzeDoesNotSilentlyAcceptLinkNameOnOrdinaryFunction(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": "module main;\n@link_name(\"external_main\")\nfn main() -> i32 { 0 }\n",
	})
	if len(result.Diagnostics) != 0 || len(result.Unsupported) != 1 || result.Unsupported[0].Feature != "function attribute @link_name" {
		t.Fatalf("result = %+v", result)
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

func TestAnalyzeTypesRawVoidPointerForeignCall(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": "module main;\nunsafe extern C fn foreign_probe(input: *void, output: *mut void) -> u32;\nfn main() -> i32 { if unsafe { foreign_probe(null, null) } == 0 { 0 } else { 1 } }\n",
	})
	assertCleanTypeResult(t, result)
	foreign := result.HIR.Modules[0].ForeignFunctions[0]
	wantParameters := []hir.Type{
		hir.RawPointer(hir.TypeVoid, false),
		hir.RawPointer(hir.TypeVoid, true),
	}
	actualParameters := make([]hir.Type, len(foreign.Parameters))
	for index := range foreign.Parameters {
		actualParameters[index] = foreign.Parameters[index].Type
	}
	if foreign.ResultType != hir.TypeU32 || !slices.Equal(actualParameters, wantParameters) {
		t.Fatalf("foreign signature = %+v", foreign)
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

func TestAnalyzeTypesValuePreservingIntegerCasts(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": `module main;
fn main() -> i32 {
    let byte: u8 = 42;
    let signed: i32 = byte as i32;
    let word: u32 = byte as u32;
    let byte_size: usize = byte as usize;
    let word_size: usize = word as usize;
    signed
}
`,
	})
	assertCleanTypeResult(t, result)
	statements := result.HIR.Modules[0].Functions[0].Body.Statements
	want := []struct {
		statement int
		source    hir.Type
		target    hir.Type
	}{
		{statement: 1, source: hir.TypeU8, target: hir.TypeI32},
		{statement: 2, source: hir.TypeU8, target: hir.TypeU32},
		{statement: 3, source: hir.TypeU8, target: hir.TypeUSize},
		{statement: 4, source: hir.TypeU32, target: hir.TypeUSize},
	}
	for _, expectation := range want {
		expression := statements[expectation.statement].Expression
		if expression == nil || expression.Kind != "scalar-cast" || expression.Type != expectation.target || expression.TargetType != expectation.target ||
			expression.Operator != "as" || expression.UnsafeScope != nil || len(expression.Children) != 1 || expression.Children[0].Type != expectation.source {
			t.Fatalf("cast statement %d = %+v", expectation.statement, expression)
		}
	}
}

func TestAnalyzeRejectsNonValuePreservingIntegerCast(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": "module main;\nfn main() -> i32 { let word: u32 = 42; word as i32 }\n",
	})
	if len(result.Unsupported) != 1 || result.Unsupported[0].Feature != "cast from u32 to i32" {
		t.Fatalf("result = %+v", result)
	}
}

func TestAnalyzeTypesScalarComparisons(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": `module main;
fn main() -> i32 {
    let byte: u8 = 41;
    let word: u32 = 42;
    let size: usize = 43;
    let pointer: *u8 = null;
    let bool_equal: bool = true != false;
    let signed_ordered: bool = -1 < 0;
    let byte_ordered: bool = byte < 42;
    let word_ordered: bool = word >= 42;
    let size_ordered: bool = size <= 43;
    let pointer_equal: bool = pointer == null;
    if bool_equal && signed_ordered && byte_ordered && word_ordered && size_ordered && pointer_equal { 42 } else { 1 }
}
`,
	})
	assertCleanTypeResult(t, result)
	statements := result.HIR.Modules[0].Functions[0].Body.Statements
	wantOwners := []hir.Type{hir.TypeBool, hir.TypeI32, hir.TypeU8, hir.TypeU32, hir.TypeUSize}
	for index, owner := range wantOwners {
		expression := statements[index+4].Expression
		if expression == nil || expression.Kind != "binary" || expression.Type != hir.TypeBool ||
			len(expression.Children) != 2 || expression.Children[0].Type != owner || expression.Children[1].Type != owner {
			t.Fatalf("comparison statement %d = %+v", index+4, expression)
		}
	}
	pointerComparison := statements[9].Expression
	if pointerComparison == nil || pointerComparison.Type != hir.TypeBool || len(pointerComparison.Children) != 2 ||
		pointerComparison.Children[0].Type != hir.RawPointer(hir.TypeU8, false) ||
		pointerComparison.Children[1].Type != hir.RawPointer(hir.TypeU8, false) {
		t.Fatalf("pointer comparison = %+v", pointerComparison)
	}
}

func TestAnalyzeRejectsNonScalarEqualityAndBooleanOrdering(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": `module main;
struct Pair { value: i32; }
fn main() -> i32 {
    let left: Pair = Pair { value: 1 };
    let right: Pair = Pair { value: 1 };
    let same: bool = left == right;
    let ordered: bool = true < false;
    42
}
`,
	})
	if len(result.Unsupported) != 2 || result.Unsupported[0].Feature != "equality comparison on main::Pair" ||
		result.Unsupported[1].Feature != "ordered comparison on bool" {
		t.Fatalf("result = %+v", result)
	}
}

func TestAnalyzeTypesCheckedUnsignedArithmetic(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": `module main;
fn main() -> i32 {
    let byte: u8 = 6;
    let byte_product: u8 = byte * 7;
    let word: u32 = 50;
    let word_difference: u32 = word - 8;
    let size: usize = 41;
    let size_sum: usize = size + 1;
    if byte_product == 42 && word_difference == 42 && size_sum == 42 { 42 } else { 1 }
}
`,
	})
	assertCleanTypeResult(t, result)
	statements := result.HIR.Modules[0].Functions[0].Body.Statements
	want := []struct {
		statement int
		operator  string
		typ       hir.Type
	}{
		{statement: 1, operator: "*", typ: hir.TypeU8},
		{statement: 3, operator: "-", typ: hir.TypeU32},
		{statement: 5, operator: "+", typ: hir.TypeUSize},
	}
	for _, expectation := range want {
		expression := statements[expectation.statement].Expression
		if expression == nil || expression.Kind != "binary" || expression.Operator != expectation.operator ||
			expression.Type != expectation.typ || len(expression.Children) != 2 ||
			expression.Children[0].Type != expectation.typ || expression.Children[1].Type != expectation.typ {
			t.Fatalf("checked arithmetic statement %d = %+v", expectation.statement, expression)
		}
	}
}

func TestAnalyzeRejectsCheckedArithmeticOnBool(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": "module main;\nfn main() -> i32 { let invalid: bool = true + false; 42 }\n",
	})
	if len(result.Unsupported) != 1 || result.Unsupported[0].Feature != "checked arithmetic on bool" {
		t.Fatalf("result = %+v", result)
	}
}

func TestAnalyzeTypesCheckedUnsignedDivision(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": `module main;
fn main() -> i32 {
    let byte_input: u8 = 84;
    let byte: u8 = byte_input / 2;
    let word_input: u32 = 85;
    let word: u32 = word_input % 43;
    let size_input: usize = 84;
    let size: usize = size_input / 2;
    if byte == 42 && word == 42 && size == 42 { 42 } else { 1 }
}
`,
	})
	assertCleanTypeResult(t, result)
	statements := result.HIR.Modules[0].Functions[0].Body.Statements
	for _, expectation := range []struct {
		statement int
		operator  string
		typ       hir.Type
	}{{1, "/", hir.TypeU8}, {3, "%", hir.TypeU32}, {5, "/", hir.TypeUSize}} {
		expression := statements[expectation.statement].Expression
		if expression == nil || expression.Operator != expectation.operator || expression.Type != expectation.typ ||
			len(expression.Children) != 2 || expression.Children[0].Type != expectation.typ || expression.Children[1].Type != expectation.typ {
			t.Fatalf("checked division statement %d = %+v", expectation.statement, expression)
		}
	}
}

func TestAnalyzeTypesUnsignedBitwiseAndCheckedRightShift(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": `module main;
fn main() -> i32 {
    let byte: u8 = 84;
    let byte_shifted: u8 = byte >> 1;
    let word: u32 = 43;
    let word_masked: u32 = word & 63;
    let size: usize = 43;
    let size_mixed: usize = (size ^ 1) | 0;
    if byte_shifted == 42 && word_masked == 43 && size_mixed == 42 { 42 } else { 1 }
}
`,
	})
	assertCleanTypeResult(t, result)
	statements := result.HIR.Modules[0].Functions[0].Body.Statements
	for _, expectation := range []struct {
		statement int
		operator  string
		typ       hir.Type
	}{{1, ">>", hir.TypeU8}, {3, "&", hir.TypeU32}, {5, "|", hir.TypeUSize}} {
		expression := statements[expectation.statement].Expression
		if expression == nil || expression.Operator != expectation.operator || expression.Type != expectation.typ ||
			len(expression.Children) != 2 || expression.Children[0].Type != expectation.typ || expression.Children[1].Type != expectation.typ {
			t.Fatalf("typed integer operation statement %d = %+v", expectation.statement, expression)
		}
	}
}

func TestAnalyzeTypesCheckedLeftShift(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": `module main;
fn main() -> i32 {
    let signed_input: i32 = 21;
    let signed: i32 = signed_input << 1;
    let byte_input: u8 = 21;
    let byte: u8 = byte_input << 1;
    let word_input: u32 = 21;
    let word: u32 = word_input << 1;
    let size_input: usize = 21;
    let size: usize = size_input << 1;
    if signed == 42 && byte == 42 && word == 42 && size == 42 { 42 } else { 1 }
}
`,
	})
	assertCleanTypeResult(t, result)
	statements := result.HIR.Modules[0].Functions[0].Body.Statements
	for _, expectation := range []struct {
		statement int
		typ       hir.Type
	}{{1, hir.TypeI32}, {3, hir.TypeU8}, {5, hir.TypeU32}, {7, hir.TypeUSize}} {
		expression := statements[expectation.statement].Expression
		if expression == nil || expression.Operator != "<<" || expression.Type != expectation.typ ||
			len(expression.Children) != 2 || expression.Children[0].Type != expectation.typ || expression.Children[1].Type != expectation.typ {
			t.Fatalf("checked left shift statement %d = %+v", expectation.statement, expression)
		}
	}
}

func TestAnalyzeRejectsUnsupportedOrMixedBitwiseAndShiftTypes(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": `module main;
fn main() -> i32 {
    let invalid_bitwise: bool = true | false;
    let invalid_left_shift: bool = true << false;
    let byte: u8 = 42;
    let word: u32 = 1;
    let invalid_shift: u8 = byte >> word;
    let invalid_left_mixed: u8 = byte << word;
    42
}
`,
	})
	if len(result.Unsupported) != 2 || result.Unsupported[0].Feature != "bitwise operation on bool" ||
		result.Unsupported[1].Feature != "checked left shift on bool" {
		t.Fatalf("unsupported = %+v", result.Unsupported)
	}
	if len(result.Diagnostics) != 2 || result.Diagnostics[0].Code != "E1006" || result.Diagnostics[1].Code != "E1006" {
		t.Fatalf("diagnostics = %+v", result.Diagnostics)
	}
}

func TestAnalyzeTypesUnsignedBitwiseNot(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": `module main;
fn main() -> i32 {
    let byte_input: u8 = 213;
    let byte: u8 = ~byte_input;
    let word_input: u32 = 4294967253;
    let word: u32 = ~word_input;
    let size_input: usize = 18446744073709551573;
    let size: usize = ~size_input;
    if byte == 42 && word == 42 && size == 42 { 42 } else { 1 }
}
`,
	})
	assertCleanTypeResult(t, result)
	statements := result.HIR.Modules[0].Functions[0].Body.Statements
	for _, expectation := range []struct {
		statement int
		typ       hir.Type
	}{{1, hir.TypeU8}, {3, hir.TypeU32}, {5, hir.TypeUSize}} {
		expression := statements[expectation.statement].Expression
		if expression == nil || expression.Kind != "prefix" || expression.Operator != "~" ||
			expression.Type != expectation.typ || len(expression.Children) != 1 || expression.Children[0].Type != expectation.typ {
			t.Fatalf("bitwise-not statement %d = %+v", expectation.statement, expression)
		}
	}
}

func TestAnalyzeRejectsBitwiseNotOnBool(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": "module main;\nfn main() -> i32 { let invalid: bool = ~true; 42 }\n",
	})
	if len(result.Unsupported) != 1 || result.Unsupported[0].Feature != "bitwise not on bool" {
		t.Fatalf("result = %+v", result)
	}
}

func TestAnalyzeTypesCStringLiteralToImmutableBytePointerCast(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": `module main;
fn main() -> i32 {
    let pointer: *u8 = unsafe { c"fairy" as *u8 };
    42
}
`,
	})
	assertCleanTypeResult(t, result)
	main := result.HIR.Modules[0].Functions[0]
	if len(main.Locals) != 1 || main.Locals[0].Type != hir.RawPointer(hir.TypeU8, false) {
		t.Fatalf("locals = %+v", main.Locals)
	}
	unsafeExpression := main.Body.Statements[0].Expression
	if unsafeExpression == nil || len(unsafeExpression.Children) != 1 || unsafeExpression.Children[0].Block == nil {
		t.Fatalf("unsafe expression = %+v", unsafeExpression)
	}
	cast := unsafeExpression.Children[0].Block.Tail
	wantType := hir.RawPointer(hir.TypeU8, false)
	if cast == nil || cast.Kind != "cast" || cast.Type != wantType || cast.TargetType != wantType || cast.UnsafeScope == nil || len(cast.Children) != 1 || cast.Children[0].Type != hir.SharedBorrow(hir.TypeCStr) {
		t.Fatalf("cast = %+v", cast)
	}
}

func TestAnalyzeTypesByteStringLiteralToImmutableBytePointerCast(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": `module main;
fn main() -> i32 {
    let pointer: *u8 = unsafe { b"fairy" as *u8 };
    42
}
`,
	})
	assertCleanTypeResult(t, result)
	main := result.HIR.Modules[0].Functions[0]
	if len(main.Locals) != 1 || main.Locals[0].Type != hir.RawPointer(hir.TypeU8, false) {
		t.Fatalf("locals = %+v", main.Locals)
	}
	unsafeExpression := main.Body.Statements[0].Expression
	if unsafeExpression == nil || len(unsafeExpression.Children) != 1 || unsafeExpression.Children[0].Block == nil {
		t.Fatalf("unsafe expression = %+v", unsafeExpression)
	}
	cast := unsafeExpression.Children[0].Block.Tail
	wantType := hir.RawPointer(hir.TypeU8, false)
	if cast == nil || cast.Kind != "cast" || cast.Type != wantType || cast.TargetType != wantType || cast.UnsafeScope == nil || len(cast.Children) != 1 || cast.Children[0].Type != hir.SharedBorrow(hir.TypeByteSlice) {
		t.Fatalf("cast = %+v", cast)
	}
}

func TestAnalyzeRejectsMutableOrUnwrappedByteStringLiteralTypes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		source  string
		feature string
	}{
		{
			name:    "bare unsized byte slice",
			source:  "module main;\nfn main() -> i32 { let value: [u8] = b\"fairy\"; 0 }\n",
			feature: "type [u8]",
		},
		{
			name:    "mutable byte slice borrow",
			source:  "module main;\nfn main() -> i32 { let value: &mut [u8] = b\"fairy\"; 0 }\n",
			feature: "type &mut [u8]",
		},
		{
			name:    "mutable byte pointer extraction",
			source:  "module main;\nfn main() -> i32 { let value: *mut u8 = unsafe { b\"fairy\" as *mut u8 }; 0 }\n",
			feature: "cast from &[u8] to *mut u8",
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

func TestAnalyzeRejectsMutableOrUnwrappedCStringLiteralTypes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		source  string
		feature string
	}{
		{
			name:    "bare unsized CStr",
			source:  "module main;\nfn main() -> i32 { let value: CStr = c\"fairy\"; 0 }\n",
			feature: "type CStr",
		},
		{
			name:    "mutable CStr borrow",
			source:  "module main;\nfn main() -> i32 { let value: &mut CStr = c\"fairy\"; 0 }\n",
			feature: "type &mut CStr",
		},
		{
			name:    "mutable byte pointer extraction",
			source:  "module main;\nfn main() -> i32 { let value: *mut u8 = unsafe { c\"fairy\" as *mut u8 }; 0 }\n",
			feature: "cast from &CStr to *mut u8",
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

func TestAnalyzeTypesUTF8StringLiteralWithStaticBorrowOrigin(t *testing.T) {
	t.Parallel()
	text := "module main;\n\nfn main() -> i32 {\n    let text: &str = \"λ\";\n    42\n}\n"
	result := analyzeText(t, map[string]string{"main.fy": text})
	assertCleanTypeResult(t, result)

	if len(result.HIR.Modules) != 1 || len(result.HIR.Modules[0].Functions) != 1 {
		t.Fatalf("HIR modules = %+v", result.HIR.Modules)
	}
	function := result.HIR.Modules[0].Functions[0]
	wantType := hir.SharedBorrow(hir.TypeStr)
	if len(function.Locals) != 1 || function.Locals[0].Type != wantType {
		t.Fatalf("locals = %+v, want one %q local", function.Locals, wantType)
	}
	if len(function.Body.Statements) != 1 || function.Body.Statements[0].Expression == nil {
		t.Fatalf("body = %+v", function.Body)
	}
	literal := function.Body.Statements[0].Expression
	if literal.Kind != "literal" || literal.Type != wantType || literal.Value != `"λ"` || literal.Cooked != "cebb" {
		t.Fatalf("literal = %+v", literal)
	}

	start := strings.Index(text, `"λ"`)
	if len(result.Borrows) != 1 || result.Borrows[0].Kind != "shared" || result.Borrows[0].Origin != "Static" {
		t.Fatalf("borrows = %+v", result.Borrows)
	}
	assertSpan(t, result.Borrows[0].Borrow, "main.fy", start, start+len(`"λ"`))
}

func TestAnalyzeTypesUTF8StringLocalToImmutableBytePointerCast(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": `module main;
fn main() -> i32 {
    let text: &str = "fairy";
    let pointer: *u8 = unsafe { text as *u8 };
    42
}
`,
	})
	assertCleanTypeResult(t, result)
	main := result.HIR.Modules[0].Functions[0]
	if len(main.Locals) != 2 || main.Locals[0].Type != hir.SharedBorrow(hir.TypeStr) || main.Locals[1].Type != hir.RawPointer(hir.TypeU8, false) {
		t.Fatalf("locals = %+v", main.Locals)
	}
	unsafeExpression := main.Body.Statements[1].Expression
	if unsafeExpression == nil || len(unsafeExpression.Children) != 1 || unsafeExpression.Children[0].Block == nil {
		t.Fatalf("unsafe expression = %+v", unsafeExpression)
	}
	cast := unsafeExpression.Children[0].Block.Tail
	wantType := hir.RawPointer(hir.TypeU8, false)
	if cast == nil || cast.Kind != "cast" || cast.Type != wantType || cast.TargetType != wantType || cast.UnsafeScope == nil || len(cast.Children) != 1 || cast.Children[0].Type != hir.SharedBorrow(hir.TypeStr) {
		t.Fatalf("cast = %+v", cast)
	}
}

func TestAnalyzeTypesBorrowedViewLengthAsUSize(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": `module main;
fn main() -> i32 {
    let text: &str = "λx";
    let bytes: &[u8] = b"\x00ABC";
    let text_length: usize = text.len;
    let byte_length: usize = bytes.len;
    let text_empty: bool = text.is_empty;
    let bytes_empty: bool = b"".is_empty;
    if text_length == 3 && byte_length == 4 && !text_empty && bytes_empty { 42 } else { 1 }
}
`,
	})
	assertCleanTypeResult(t, result)
	main := result.HIR.Modules[0].Functions[0]
	if len(main.Body.Statements) != 6 {
		t.Fatalf("body statements = %+v", main.Body.Statements)
	}
	for index, wantOwner := range []hir.Type{hir.SharedBorrow(hir.TypeStr), hir.SharedBorrow(hir.TypeByteSlice)} {
		expression := main.Body.Statements[index+2].Expression
		if expression == nil || expression.Kind != "view-length" || expression.Type != hir.TypeUSize || expression.Operator != "len" ||
			len(expression.Children) != 1 || expression.Children[0].Type != wantOwner {
			t.Fatalf("length expression %d = %+v, want owner %q", index, expression, wantOwner)
		}
	}
	for index, wantOwner := range []hir.Type{hir.SharedBorrow(hir.TypeStr), hir.SharedBorrow(hir.TypeByteSlice)} {
		expression := main.Body.Statements[index+4].Expression
		if expression == nil || expression.Kind != "view-empty" || expression.Type != hir.TypeBool || expression.Operator != "is_empty" ||
			len(expression.Children) != 1 || expression.Children[0].Type != wantOwner {
			t.Fatalf("emptiness expression %d = %+v, want owner %q", index, expression, wantOwner)
		}
	}
}

func TestAnalyzeTypesImmutableByteViewIndexAsU8(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": `module main;
fn main() -> i32 {
    let value: u8 = b"\x00A"[1];
    42
}
`,
	})
	assertCleanTypeResult(t, result)
	expression := result.HIR.Modules[0].Functions[0].Body.Statements[0].Expression
	if expression == nil || expression.Kind != "view-index" || expression.Type != hir.TypeU8 || expression.Operator != "[]" ||
		len(expression.Children) != 2 || expression.Children[0].Type != hir.SharedBorrow(hir.TypeByteSlice) || expression.Children[1].Type != hir.TypeUSize {
		t.Fatalf("index expression = %+v", expression)
	}
}

func TestAnalyzeTypesScalarFixedArrayLiteralAndIndex(t *testing.T) {
	t.Parallel()
	text := "module main;\nfn pick(values: [u8; 3], index: usize) -> u8 { values[index] }\nfn main() -> i32 { let values: [u8; 3] = [40, 41, 42]; pick(values, 2) as i32 }\n"
	result := analyzeText(t, map[string]string{"main.fy": text})
	assertCleanTypeResult(t, result)
	main := result.HIR.Modules[0].Functions[1]
	array := main.Body.Statements[0].Expression
	if array == nil || array.Kind != "array" || array.Type != hir.FixedArray(hir.TypeU8, 3) || array.Operator != "list" || len(array.Children) != 3 {
		t.Fatalf("array expression = %+v", array)
	}
	pick := result.HIR.Modules[0].Functions[0]
	index := pick.Body.Tail
	if index == nil || index.Kind != "array-index" || index.Type != hir.TypeU8 || index.Operator != "[]" || len(index.Children) != 2 {
		t.Fatalf("array index = %+v", index)
	}
}

func TestAnalyzeTypesScalarFixedArrayLength(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": "module main;\nfn length(values: [u32; 3]) -> usize { values.len }\n",
	})
	assertCleanTypeResult(t, result)
	expression := result.HIR.Modules[0].Functions[0].Body.Tail
	if expression == nil || expression.Kind != "array-length" || expression.Type != hir.TypeUSize || expression.Operator != "len" ||
		len(expression.Children) != 1 || expression.Children[0].Type != hir.FixedArray(hir.TypeU32, 3) {
		t.Fatalf("array length expression = %+v", expression)
	}
}

func TestAnalyzeKeepsOtherFixedArrayFieldsClosed(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": "module main;\nfn main() -> i32 { let values: [i32; 1] = [42]; if values.is_empty { 1 } else { 42 } }\n",
	})
	if len(result.Unsupported) != 1 || result.Unsupported[0].Feature != "unknown fixed-array field is_empty" {
		t.Fatalf("unsupported = %+v", result.Unsupported)
	}
}

func TestAnalyzeRejectsFixedArrayLengthAndElementMismatch(t *testing.T) {
	t.Parallel()
	for _, text := range []string{
		"module main;\nfn main() -> i32 { let values: [i32; 2] = [1, 2, 3]; values[0] }\n",
		"module main;\nfn main() -> i32 { let values = [1, true]; values[0] }\n",
	} {
		result := analyzeText(t, map[string]string{"main.fy": text})
		if result.FailedPhase != semantic.PhaseType || len(result.Diagnostics) == 0 || result.Diagnostics[0].Code != "E1006" {
			t.Fatalf("result = %+v", result)
		}
	}
}

func TestAnalyzeKeepsAmplifyingFixedArrayFormsClosed(t *testing.T) {
	t.Parallel()
	for _, literal := range []string{"[]", "[1; 3]"} {
		text := "module main;\nfn main() -> i32 { let values: [i32; 3] = " + literal + "; 0 }\n"
		result := analyzeText(t, map[string]string{"main.fy": text})
		if len(result.Unsupported) != 1 {
			t.Fatalf("literal %s unsupported = %+v", literal, result.Unsupported)
		}
	}
}

func TestAnalyzeRejectsFixedArrayBeyondExecutableElementLimit(t *testing.T) {
	t.Parallel()
	length := hir.MaxExecutableFixedArrayElements + 1
	t.Run("type", func(t *testing.T) {
		text := fmt.Sprintf("module main;\nfn main(values: [u8; %d]) -> i32 { 0 }\n", length)
		result := analyzeText(t, map[string]string{"main.fy": text})
		if len(result.Unsupported) != 1 || result.Unsupported[0].Feature != fmt.Sprintf("type [u8; %d]", length) {
			t.Fatalf("unsupported = %+v", result.Unsupported)
		}
	})
	t.Run("inferred literal", func(t *testing.T) {
		literal := "[" + strings.Repeat("0,", int(length-1)) + "0]"
		text := "module main;\nfn main() -> i32 { let values = " + literal + "; 0 }\n"
		result := analyzeText(t, map[string]string{"main.fy": text})
		if len(result.Unsupported) != 1 || result.Unsupported[0].Feature != fmt.Sprintf("fixed-array literal with %d elements", length) {
			t.Fatalf("unsupported = %+v", result.Unsupported)
		}
	})
}

func TestAnalyzeRejectsUnsupportedViewIndexShapes(t *testing.T) {
	t.Parallel()
	t.Run("text receiver", func(t *testing.T) {
		result := analyzeText(t, map[string]string{
			"main.fy": "module main;\nfn main() -> i32 { let value: u8 = \"fairy\"[0]; 42 }\n",
		})
		if len(result.Unsupported) != 1 || result.Unsupported[0].Feature != "indexing outside immutable byte views" {
			t.Fatalf("result = %+v", result)
		}
	})
	t.Run("non-usize offset", func(t *testing.T) {
		result := analyzeText(t, map[string]string{
			"main.fy": "module main;\nfn main() -> i32 { let offset: u32 = 0; let value: u8 = b\"fairy\"[offset]; 42 }\n",
		})
		if len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "E1006" || len(result.Unsupported) != 0 {
			t.Fatalf("result = %+v", result)
		}
	})
}

func TestAnalyzeTypesImmutableByteViewSlice(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": `module main;
fn trim(bytes: &[u8]) -> &[u8] { bytes[1..2] }
`,
	})
	assertCleanTypeResult(t, result)
	expression := result.HIR.Modules[0].Functions[0].Body.Tail
	wantView := hir.SharedBorrow(hir.TypeByteSlice)
	if expression == nil || expression.Kind != "view-slice" || expression.Type != wantView || expression.Operator != ".." ||
		len(expression.Children) != 3 || expression.Children[0].Type != wantView ||
		expression.Children[1].Type != hir.TypeUSize || expression.Children[2].Type != hir.TypeUSize {
		t.Fatalf("slice expression = %+v", expression)
	}
}

func TestAnalyzeTypesImmutableByteViewSliceWithOmittedBounds(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name         string
		bounds       string
		omittedStart bool
		omittedEnd   bool
	}{
		{name: "start", bounds: "..4", omittedStart: true},
		{name: "end", bounds: "1..", omittedEnd: true},
		{name: "both", bounds: "..", omittedStart: true, omittedEnd: true},
	} {
		test := test
		t.Run(test.name, func(t *testing.T) {
			result := analyzeText(t, map[string]string{
				"main.fy": "module main;\nfn trim(bytes: &[u8]) -> &[u8] { bytes[" + test.bounds + "] }\n",
			})
			assertCleanTypeResult(t, result)
			expression := result.HIR.Modules[0].Functions[0].Body.Tail
			if expression == nil || expression.Kind != "view-slice" || len(expression.Children) != 3 {
				t.Fatalf("slice expression = %+v", expression)
			}
			if got := expression.Children[1].Kind == "omitted-range-bound"; got != test.omittedStart {
				t.Fatalf("start kind = %q, omitted = %v", expression.Children[1].Kind, got)
			}
			if got := expression.Children[2].Kind == "omitted-range-bound"; got != test.omittedEnd {
				t.Fatalf("end kind = %q, omitted = %v", expression.Children[2].Kind, got)
			}
		})
	}
}

func TestAnalyzeRejectsUnsupportedViewSliceShapes(t *testing.T) {
	t.Parallel()
	t.Run("text receiver", func(t *testing.T) {
		result := analyzeText(t, map[string]string{
			"main.fy": "module main;\nfn main() -> i32 { let value: &str = \"fairy\"[0..1]; 42 }\n",
		})
		if len(result.Unsupported) != 1 || result.Unsupported[0].Feature != "slicing outside immutable byte views" {
			t.Fatalf("result = %+v", result)
		}
	})
	t.Run("non-usize bounds", func(t *testing.T) {
		result := analyzeText(t, map[string]string{
			"main.fy": "module main;\nfn main() -> i32 { let start: u32 = 0; let value: &[u8] = b\"fairy\"[start..1]; 42 }\n",
		})
		if len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "E1006" || len(result.Unsupported) != 0 {
			t.Fatalf("result = %+v", result)
		}
	})
}

func TestAnalyzeRejectsUnknownBorrowedViewField(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": "module main;\nfn main() -> i32 { let text: &str = \"fairy\"; let value: usize = text.capacity; 42 }\n",
	})
	if len(result.Unsupported) != 1 || result.Unsupported[0].Feature != "unknown borrowed-view field capacity" {
		t.Fatalf("result = %+v", result)
	}
}

func TestAnalyzeRejectsMutableOrUnwrappedUTF8StringLiteralTypes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		source  string
		feature string
	}{
		{
			name:    "bare unsized str",
			source:  "module main;\nfn main() -> i32 { let value: str = \"fairy\"; 0 }\n",
			feature: "type str",
		},
		{
			name:    "mutable str borrow",
			source:  "module main;\nfn main() -> i32 { let value: &mut str = \"fairy\"; 0 }\n",
			feature: "type &mut str",
		},
		{
			name:    "mutable byte pointer extraction",
			source:  "module main;\nfn main() -> i32 { let value: *mut u8 = unsafe { \"fairy\" as *mut u8 }; 0 }\n",
			feature: "cast from &str to *mut u8",
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

func TestAnalyzeTypesAlwaysCheckedAssertCondition(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": "module main;\nfn main() -> i32 { let ready: bool = true; assert(ready); 42 }\n",
	})
	assertCleanTypeResult(t, result)
	statement := result.HIR.Modules[0].Functions[0].Body.Statements[1]
	if statement.Kind != "assert" || statement.Expression == nil || statement.Expression.Type != hir.TypeBool {
		t.Fatalf("assert statement = %+v", statement)
	}
}

func TestAnalyzeRejectsNonBooleanAssertCondition(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": "module main;\nfn main() -> i32 { assert(1); 42 }\n",
	})
	if result.FailedPhase != semantic.PhaseType || len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "E1007" || len(result.Unsupported) != 0 {
		t.Fatalf("result = %+v", result)
	}
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
