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
	text := "module main;\n\nfn i32 choose(bool flag) {\n    let i32 value = 1;\n    return if (flag) { value } else { 2 };\n}\n"
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
	flagStart := strings.Index(text, "bool flag") + len("bool ")
	assertTypeFact(t, result, "main.fy", flagStart, flagStart+len("flag"), hir.TypeBool)
	ifStart := strings.Index(text, "if (flag)")
	ifEnd := strings.Index(text[ifStart:], ";") + ifStart
	assertTypeFact(t, result, "main.fy", ifStart, ifEnd, hir.TypeI32)
}

func TestAnalyzeCarriesExactTopLevelDocumentationSpansIntoHIR(t *testing.T) {
	t.Parallel()
	text := strings.Join([]string{
		"module main;",
		"/// Structure docs.",
		"struct Packet { i32 value; }",
		"/// Enum docs.",
		"enum Error { Failed; }",
		"/// Resource docs.",
		"resource Handle { i32 value; cleanup {} }",
		"/// Foreign docs.",
		"unsafe extern C fn i32 raw(i32 value);",
		"/// Function docs.",
		"fn i32 run() { return 0; }",
		"",
	}, "\n")
	result := analyzeText(t, map[string]string{"main.fy": text})
	assertCleanTypeResult(t, result)
	if len(result.HIR.Modules) != 1 {
		t.Fatalf("HIR modules = %+v", result.HIR.Modules)
	}
	module := result.HIR.Modules[0]
	if len(module.Structs) != 1 || len(module.Enums) != 1 || len(module.Resources) != 1 ||
		len(module.ForeignFunctions) != 1 {
		t.Fatalf("documented HIR declarations = %+v", module)
	}
	spans := []struct {
		name string
		span hir.Span
		want string
	}{
		{name: "struct", span: module.Structs[0].DocumentationSpan, want: "/// Structure docs."},
		{name: "enum", span: module.Enums[0].DocumentationSpan, want: "/// Enum docs."},
		{name: "resource", span: module.Resources[0].DocumentationSpan, want: "/// Resource docs."},
		{name: "foreign", span: module.ForeignFunctions[0].DocumentationSpan, want: "/// Foreign docs."},
	}
	for _, function := range module.Functions {
		if function.Name == "run" && function.DropGlueFor == "" {
			spans = append(spans, struct {
				name string
				span hir.Span
				want string
			}{name: "function", span: function.DocumentationSpan, want: "/// Function docs."})
		}
		if function.DropGlueFor != "" && function.DocumentationSpan != (hir.Span{}) {
			t.Fatalf("compiler-private drop glue inherited documentation: %+v", function)
		}
	}
	if len(spans) != 5 {
		t.Fatalf("documentation observations = %+v", spans)
	}
	for _, item := range spans {
		if item.span.Source != "main.fy" || item.span.EndByte <= item.span.StartByte ||
			text[item.span.StartByte:item.span.EndByte] != item.want {
			t.Fatalf("%s documentation = %+v (%q), want %q", item.name, item.span, text[item.span.StartByte:item.span.EndByte], item.want)
		}
	}
}

func TestAnalyzeCanonicalizesCShapedReferenceAndSliceTypes(t *testing.T) {
	t.Parallel()
	text := "module main;\n\nfn usize length(ref u8[] bytes) { return bytes.len; }\nfn usize main() { return length(b\"fairy\"); }\n"
	result := analyzeText(t, map[string]string{"main.fy": text})
	assertCleanTypeResult(t, result)
	if len(result.HIR.Modules) != 1 || len(result.HIR.Modules[0].Functions) != 2 {
		t.Fatalf("HIR = %+v", result.HIR)
	}
	function := result.HIR.Modules[0].Functions[0]
	if len(function.Parameters) != 1 || function.Parameters[0].Type != hir.SharedBorrow(hir.TypeByteSlice) || function.ResultType != hir.TypeUSize {
		t.Fatalf("function = %+v", function)
	}
}

func TestAnalyzeCShapedDeclarationsAndInferredBindings(t *testing.T) {
	t.Parallel()
	text := "module main;\n\nfn usize length(ref u8[] bytes) { return bytes.len; }\nfn usize main() { let explicit = length(b\"fairy\"); let usize typed = explicit; var usize copy = typed; return copy; }\n"
	result := analyzeText(t, map[string]string{"main.fy": text})
	assertCleanTypeResult(t, result)
	if len(result.HIR.Modules) != 1 || len(result.HIR.Modules[0].Functions) != 2 {
		t.Fatalf("HIR = %+v", result.HIR)
	}
	main := result.HIR.Modules[0].Functions[1]
	if main.ResultType != hir.TypeUSize || len(main.Locals) != 3 || main.Locals[0].Type != hir.TypeUSize || main.Locals[1].Type != hir.TypeUSize || main.Locals[2].Type != hir.TypeUSize {
		t.Fatalf("main = %+v", main)
	}
}

func TestAnalyzeCShapedBorrowAndRecordDesignators(t *testing.T) {
	t.Parallel()
	text := "module main;\n\nstruct Pair { i32 left; i32 right; }\nfn usize length(ref u8[] bytes) { return bytes.len; }\nfn usize main() { let u8[3] values = [1, 2, 3]; let Pair pair = Pair { .left = 20, .right = 22 }; if (pair.left == 20) { return length(ref values); } else { return 0; } }\n"
	result := analyzeText(t, map[string]string{"main.fy": text})
	assertCleanTypeResult(t, result)
	if len(result.HIR.Modules) != 1 || len(result.HIR.Modules[0].Functions) != 2 {
		t.Fatalf("HIR = %+v", result.HIR)
	}
	main := result.HIR.Modules[0].Functions[1]
	if main.ResultType != hir.TypeUSize || len(main.Locals) != 2 || main.Locals[0].Type != hir.FixedArray(hir.TypeU8, 3) {
		t.Fatalf("main = %+v", main)
	}
}

func TestAnalyzeReportsUnresolvedNameWithoutTypeCascade(t *testing.T) {
	t.Parallel()
	text := "module main;\n\nfn i32 answer() {\n    return missing;\n}\n"
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
	text := "module main;\n\nfn i32 value() { return 1; }\n\nfn i32 value() { return 2; }\n"
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
	text := "module main;\n\nunsafe extern C fn i32 value() ;\n\nfn i32 value() { return 2; }\n"
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
	text := "module main;\n\nfn i32 answer() {\n    let bool value = 1;\n    return 0;\n}\n"
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
fn i32 main() {
    let u8 byte = 0xFF;
    let u32 word = 0xDEAD_BEEF;
    let usize size = 0b1111_1111;
    return if (byte == 255 && word == 3735928559 && size == 255) { 42 } else { 1 };
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
			text := "module main;\nfn i32 main() { let " + test.typeRef + " value = " + test.value + "; return 0; }\n"
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
				"main.fy": "module main;\nfn i32 main() { return " + value + "; }\n",
			})
			assertCleanTypeResult(t, result)
			expression := soleReturnExpression(t, result.HIR.Modules[0].Functions[0])
			if expression.Kind != "literal" || expression.Type != hir.TypeI32 || expression.Value != "-2147483648" || len(expression.Children) != 0 {
				t.Fatalf("folded minimum = %+v", expression)
			}
		})
	}
}

func TestAnalyzeRejectsIntegerMagnitudeBeyondDirectI32Minimum(t *testing.T) {
	t.Parallel()
	for _, value := range []string{"-2147483649", "-0x8000_0001"} {
		value := value
		t.Run(value, func(t *testing.T) {
			text := "module main;\nfn i32 main() { return " + value + "; }\n"
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
	text := "module main;\nfn i32 main() { return " + value + "; }\n"
	result := analyzeText(t, map[string]string{"main.fy": text})
	if result.FailedPhase != semantic.PhaseType || len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "E1013" {
		t.Fatalf("diagnostics = %+v, unsupported = %+v", result.Diagnostics, result.Unsupported)
	}
	start := strings.Index(text, value)
	assertSpan(t, result.Diagnostics[0].Primary, "main.fy", start, start+len(value))
}

func TestAnalyzeReportsNonBooleanCondition(t *testing.T) {
	t.Parallel()
	text := "module main;\n\nfn i32 answer() {\n    return if (1) { 2 } else { 3 };\n}\n"
	result := analyzeText(t, map[string]string{"main.fy": text})
	if result.FailedPhase != semantic.PhaseType || len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "E1007" {
		t.Fatalf("result = %+v", result)
	}
	start := strings.Index(text, "if (1)") + len("if (")
	assertSpan(t, result.Diagnostics[0].Primary, "main.fy", start, start+1)
}

func TestAnalyzeResolvesQualifiedPublicFunction(t *testing.T) {
	t.Parallel()
	mainText := "module main;\n\nimport math as m;\n\nfn i32 answer() {\n    return m::forty_two();\n}\n"
	mathText := "module math;\n\npub fn i32 forty_two() {\n    return 42;\n}\n"
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
	text := "module main;\n\nfn i32 text() {\n    return 1.5;\n}\n"
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
	text := "module main;\n\nfn i32 main() {\n    let ref CStr name = c\"fairy\";\n    return 42;\n}\n"
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
	if len(function.Body.Statements) != 2 || function.Body.Statements[0].Expression == nil {
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
	text := "module main;\n\nfn i32 main() {\n    let ref u8[] bytes = b\"\\x00A\";\n    return 42;\n}\n"
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
	if len(function.Body.Statements) != 2 || function.Body.Statements[0].Expression == nil {
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

func TestAnalyzeBuildsContextualResultConstructorsAndSwitch(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": `module main;
fn Result<i32, i32> classify(i32 value) {
    return if (value >= 0) { .ok(value) } else { .err(-value) };
}
fn i32 unwrap(Result<i32, i32> input) {
    return switch (input) {
        case .ok(value) : { value }
        case .err(error) : { -error }
    };
}
`,
	})
	assertCleanTypeResult(t, result)
	if len(result.HIR.Modules) != 1 || len(result.HIR.Modules[0].Functions) != 2 {
		t.Fatalf("HIR modules = %+v", result.HIR.Modules)
	}
	classify := result.HIR.Modules[0].Functions[0]
	classifyReturn := soleReturnExpression(t, classify)
	if classifyReturn.Kind != "if" {
		t.Fatalf("classify return = %+v", classifyReturn)
	}
	unwrap := result.HIR.Modules[0].Functions[1]
	unwrapReturn := soleReturnExpression(t, unwrap)
	if unwrapReturn.Kind != "match" || len(unwrapReturn.Arms) != 2 {
		t.Fatalf("unwrap return = %+v", unwrapReturn)
	}
	if unwrapReturn.Arms[0].Variant != "ok" || unwrapReturn.Arms[1].Variant != "err" {
		t.Fatalf("match arms = %+v", unwrapReturn.Arms)
	}
}

func TestAnalyzeBuildsContextualOptionConstructorsAndSwitch(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": `module main;
fn Option<i32> choose(bool present) {
    return if (present) { .some(42) } else { .none };
}
fn i32 unwrap(Option<i32> input) {
    return switch (input) {
        case .some(value) : { value }
        case .none : { 0 }
    };
}
`,
	})
	assertCleanTypeResult(t, result)
	if len(result.HIR.Modules) != 1 || len(result.HIR.Modules[0].Functions) != 2 {
		t.Fatalf("HIR modules = %+v", result.HIR.Modules)
	}
	choose := result.HIR.Modules[0].Functions[0]
	chooseReturn := soleReturnExpression(t, choose)
	if chooseReturn.Kind != "if" || len(chooseReturn.Children) != 3 ||
		chooseReturn.Children[1].Block == nil || chooseReturn.Children[1].Block.Tail == nil || chooseReturn.Children[1].Block.Tail.Kind != "option-variant" ||
		chooseReturn.Children[2].Block == nil || chooseReturn.Children[2].Block.Tail == nil || chooseReturn.Children[2].Block.Tail.Variant != "none" {
		t.Fatalf("choose return = %+v", chooseReturn)
	}
	unwrap := result.HIR.Modules[0].Functions[1]
	unwrapReturn := soleReturnExpression(t, unwrap)
	if unwrapReturn.Kind != "match" || len(unwrapReturn.Arms) != 2 {
		t.Fatalf("unwrap return = %+v", unwrapReturn)
	}
	if unwrapReturn.Arms[0].Variant != "some" || unwrapReturn.Arms[1].Variant != "none" || unwrapReturn.Arms[1].Binding != hir.NoSymbol {
		t.Fatalf("Option match arms = %+v", unwrapReturn.Arms)
	}
}

func TestAnalyzeRejectsPreviousRevisionQualifiedVariants(t *testing.T) {
	t.Parallel()
	tests := map[string]string{
		"built-in Result": "fn Result<i32, i32> main() { return Result::Ok(42); }\n",
		"user enum":       "enum Status { Ready; }\nfn Status main() { return Status::Ready; }\n",
	}
	for name, body := range tests {
		name, body := name, body
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			input := "module main;\n" + body
			result := analyzeText(t, map[string]string{"main.fy": input})
			if result.FailedPhase != semantic.PhaseResolve || len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "E0101" {
				t.Fatalf("result = %+v", result)
			}
		})
	}
}

func TestAnalyzeLowersCShapedSwitchAndLeadingDotVariants(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": `module main;
fn Result<i32, i32> classify(i32 value) {
    if (value >= 0) { return .ok(value); } else { return .err(-value); }
}
fn i32 unwrap(Result<i32, i32> input) {
    return switch (input) {
        case .ok(value): { value }
        case .err(error): { -error }
    };
}
`,
	})
	assertCleanTypeResult(t, result)
	if len(result.HIR.Modules) != 1 || len(result.HIR.Modules[0].Functions) != 2 {
		t.Fatalf("HIR modules = %+v", result.HIR.Modules)
	}
	unwrap := result.HIR.Modules[0].Functions[1]
	if len(unwrap.Body.Statements) != 1 || unwrap.Body.Statements[0].Expression == nil {
		t.Fatalf("unwrap body = %+v", unwrap.Body)
	}
	lowered := unwrap.Body.Statements[0].Expression
	if lowered.Kind != "match" || len(lowered.Arms) != 2 || lowered.Arms[0].Variant != "ok" || lowered.Arms[1].Variant != "err" {
		t.Fatalf("lowered switch = %+v", lowered)
	}
}

func TestAnalyzePreservesOrderedResourceDefinitionWithEmptyDrop(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": "module main;\nresource Pair { i32 left; bool right; cleanup {} }\nfn i32 main() { return 0; }\n",
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

func TestAnalyzeLowersCShapedCleanupWithContextualThis(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": `module main;
resource Token {
    i32 value;
    cleanup { let i32 marker = this.value; }
}
fn i32 main() { return 0; }
`,
	})
	assertCleanTypeResult(t, result)
	resource := result.HIR.Modules[0].Resources[0]
	if resource.DropKind != hir.ResourceDropNonEmpty || resource.DropSymbol == hir.NoSymbol {
		t.Fatalf("resource = %+v", resource)
	}
	for _, function := range result.HIR.Modules[0].Functions {
		if function.Symbol == resource.DropSymbol {
			if len(function.Parameters) != 1 || function.Parameters[0].Name != "this" {
				t.Fatalf("cleanup glue parameters = %+v", function.Parameters)
			}
			return
		}
	}
	t.Fatal("cleanup glue function not found")
}

func TestAnalyzeOwnerQualifiedAssociatedFunction(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": `module main;
struct Counter { i32 value; }
fn i32 Counter::double(i32 value) { return value * 2; }
fn i32 main() { return Counter::double(21); }
`,
	})
	assertCleanTypeResult(t, result)
	if len(result.HIR.Modules) != 1 || len(result.HIR.Modules[0].Functions) != 2 {
		t.Fatalf("HIR modules = %+v", result.HIR.Modules)
	}
	associated := result.HIR.Modules[0].Functions[0]
	if associated.Name != "Counter::double" || associated.OwnerType != hir.Type("main::Counter") {
		t.Fatalf("associated function = %+v", associated)
	}
	main := result.HIR.Modules[0].Functions[1]
	if len(main.Body.Statements) != 1 || main.Body.Statements[0].Expression == nil || main.Body.Statements[0].Expression.Symbol != associated.Symbol {
		t.Fatalf("main = %+v", main)
	}
}

func TestAnalyzeOwnerQualifiedMethodCallSynthesizesReceiverBorrow(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": `module main;
struct Counter { i32 value; }
fn i32 Counter::get(ref Counter this) { return this.value; }
fn i32 main() {
    let Counter counter = Counter { .value = 42 };
    return counter.get();
}
`,
	})
	assertCleanTypeResult(t, result)
	method := result.HIR.Modules[0].Functions[0]
	if method.Name != "Counter::get" || method.Receiver != "shared" || method.OwnerType != hir.Type("main::Counter") {
		t.Fatalf("method = %+v", method)
	}
	main := result.HIR.Modules[0].Functions[1]
	call := main.Body.Statements[1].Expression
	if call == nil || call.Kind != "call" || call.Symbol != method.Symbol || len(call.Children) != 2 || call.Children[1].Kind != "prefix" || call.Children[1].Operator != "&" {
		t.Fatalf("method call = %+v", call)
	}
}

func TestAnalyzeOwnerQualifiedExclusiveMethodRequiresMutableReceiver(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": `module main;
struct Counter { i32 value; }
fn void Counter::set(ref mut Counter this, i32 value) { this.value = value; }
fn i32 main() {
    var Counter counter = Counter { .value = 0 };
    counter.set(42);
    return counter.value;
}
`,
	})
	assertCleanTypeResult(t, result)
	method := result.HIR.Modules[0].Functions[0]
	if method.Receiver != "exclusive" {
		t.Fatalf("method = %+v", method)
	}
	call := result.HIR.Modules[0].Functions[1].Body.Statements[1].Expression
	if call == nil || len(call.Children) != 3 || call.Children[1].Operator != "&mut" {
		t.Fatalf("method call = %+v", call)
	}
}

func TestAnalyzeOwnerQualifiedConsumingMethodMovesReceiver(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": `module main;
resource Token { i32 value; cleanup {} }
fn i32 Token::take(Token this) { return this.value; }
fn i32 main() {
    let Token token = Token { .value = 42 };
    let i32 value = token.take();
    return value + token.value;
}
`,
	})
	if len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "E1101" {
		t.Fatalf("diagnostics = %+v, unsupported = %+v", result.Diagnostics, result.Unsupported)
	}
}

func TestAnalyzeContextualUserEnumConstructionAndSwitch(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": `module main;
enum Message {
    Empty;
    Pair { i32 left; i32 right; };
}
fn Message make() { return .Pair { .left = 20, .right = 22 }; }
fn i32 score(Message value) {
    return switch (value) {
        case .Empty: { 0 }
        case .Pair { .left = first, .right = second }: { first + second }
    };
}
fn i32 main() { return score(make()); }
`,
	})
	assertCleanTypeResult(t, result)
	makeFunction := result.HIR.Modules[0].Functions[0]
	constructed := makeFunction.Body.Statements[0].Expression
	if constructed == nil || constructed.Kind != "enum-variant" || constructed.Variant != "Pair" || !slices.Equal(constructed.Fields, []string{"left", "right"}) {
		t.Fatalf("constructed enum = %+v", constructed)
	}
	score := result.HIR.Modules[0].Functions[1]
	lowered := score.Body.Statements[0].Expression
	if lowered == nil || lowered.Kind != "match" || len(lowered.Arms) != 2 || lowered.Arms[0].Variant != "Empty" || lowered.Arms[1].Variant != "Pair" {
		t.Fatalf("lowered switch = %+v", lowered)
	}
}

func TestAnalyzePreservesReprCStructAndTypesAggregateExpressions(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": `module main;
@repr(C)
struct Packet { u32 code; usize length; }
fn i32 main() {
    let Packet packet = Packet { .code = 42, .length = 0 };
    return if (packet.code == 42) { 0 } else { 1 };
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
	if len(main.Locals) != 1 || main.Locals[0].Type != hir.Type("main::Packet") || soleReturnExpression(t, main).Kind != "if" {
		t.Fatalf("main = %+v", main)
	}
}

func TestAnalyzePreservesRestrictedInertEnumDefinitions(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": `module main;
struct Position { u32 line; u32 column; }
enum ParseError {
    Empty;
    Unexpected { Position position; u8 byte; };
}
fn i32 main() { return 0; }
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
    Value { i32 amount; };
    Pair { i32 left; i32 right; };
}
fn i32 score(Message message) {
    return switch (message) {
        case .Empty : { 0 }
        case .Value { .amount = value } : { value }
        case .Pair { .left = first, .right = second } : { first + second }
    };
}
fn i32 main() { return score(.Pair { .left = 20, .right = 22 }); }
`,
	})
	assertCleanTypeResult(t, result)
	if len(result.HIR.Modules) != 1 || len(result.HIR.Modules[0].Functions) != 2 {
		t.Fatalf("HIR modules = %+v", result.HIR.Modules)
	}
	score := result.HIR.Modules[0].Functions[0]
	scoreReturn := soleReturnExpression(t, score)
	if scoreReturn.Kind != "match" || len(scoreReturn.Arms) != 3 {
		t.Fatalf("score = %+v", score)
	}
	if scoreReturn.Arms[0].Variant != "Empty" || len(scoreReturn.Arms[0].Bindings) != 0 {
		t.Fatalf("empty arm = %+v", scoreReturn.Arms[0])
	}
	if bindings := scoreReturn.Arms[2].Bindings; len(bindings) != 2 || bindings[0].Field != "left" || bindings[0].Type != hir.TypeI32 || bindings[1].Field != "right" || bindings[1].Type != hir.TypeI32 {
		t.Fatalf("pair bindings = %+v", bindings)
	}
	main := result.HIR.Modules[0].Functions[1]
	mainReturn := soleReturnExpression(t, main)
	if mainReturn.Kind != "call" || len(mainReturn.Children) != 2 || mainReturn.Children[1].Kind != "enum-variant" || mainReturn.Children[1].Variant != "Pair" {
		t.Fatalf("main = %+v", main)
	}
}

func TestAnalyzeResolvesPublicEnumVariantAcrossImportAlias(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"api.fy":  "module api;\npub enum Status { Ready; Code { i32 value; }; }\n",
		"main.fy": "module main;\nimport api as a;\nfn i32 main() { let a::Status status = .Code { .value = 42 }; return switch (status) { case .Ready: { 0 } case .Code { .value = code }: { code } }; }\n",
	})
	assertCleanTypeResult(t, result)
	main := result.HIR.Modules[1].Functions[0]
	if len(main.Locals) != 2 || main.Locals[0].Name != "status" || main.Locals[0].Type != hir.Type("api::Status") ||
		len(main.Body.Statements) != 2 || main.Body.Statements[0].Expression == nil ||
		main.Body.Statements[0].Expression.Kind != "enum-variant" || main.Body.Statements[0].Expression.Variant != "Code" {
		t.Fatalf("imported enum construction = %+v", main)
	}
	switched := soleReturnExpression(t, main)
	if switched.Kind != "match" || len(switched.Arms) != 2 || switched.Arms[0].Variant != "Ready" || switched.Arms[1].Variant != "Code" {
		t.Fatalf("imported enum switch = %+v", switched)
	}
}

func TestAnalyzeRejectsNonExhaustiveRestrictedEnumMatch(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": "module main;\nenum Status { Ready; Failed; }\nfn i32 main() { let Status status = .Ready; return switch (status) { case .Ready : { 0 } }; }\n",
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
		{name: "borrow", text: "module main;\nenum Bad { Value { ref i32 payload; }; }\nfn i32 main() { return 0; }\n", code: "E1004"},
		{name: "region", text: "module main;\nenum Bad { Value { Region payload; }; }\nfn i32 main() { return 0; }\n", code: "E3103"},
		{name: "resource", text: "module main;\nresource Token { i32 value; cleanup {} }\nenum Bad { Value { Token payload; }; }\nfn i32 main() { return 0; }\n", code: "E1004"},
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
		"main.fy": "module main;\nenum Bad { Same; Same; Payload { i32 value; u32 value; }; }\nfn i32 main() { return 0; }\n",
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
		{name: "borrow", text: "module main;\nstruct Holder { ref i32 value; }\nfn i32 main() { return 0; }\n", code: "E1003"},
		{name: "region", text: "module main;\nstruct Holder { Region value; }\nfn i32 main() { return 0; }\n", code: "E3103"},
		{name: "resource", text: "module main;\nresource Token { i32 value; cleanup {} }\nstruct Holder { Token value; }\nfn i32 main() { return 0; }\n", code: "E1002"},
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
		"main.fy": "module main;\n@repr(packed)\nstruct Packet { u32 value; }\nfn i32 main() { return 0; }\n",
	})
	if len(result.Unsupported) != 1 || result.Unsupported[0].Feature != "struct attribute @repr" {
		t.Fatalf("result = %+v", result)
	}
}

func TestAnalyzeLowersNonEmptyResourceDropBodyToHiddenGlue(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": "module main;\nresource Token { i32 value; cleanup { let i32 marker = this.value; } }\nfn i32 main() { return 0; }\n",
	})
	assertCleanTypeResult(t, result)
	if len(result.HIR.Modules) != 1 || len(result.HIR.Modules[0].Resources) != 1 || len(result.HIR.Modules[0].Functions) != 2 {
		t.Fatalf("result = %+v", result)
	}
	resource := result.HIR.Modules[0].Resources[0]
	if resource.DropKind != hir.ResourceDropNonEmpty || resource.DropSymbol == hir.NoSymbol {
		t.Fatalf("resource = %+v", resource)
	}
	var glue *hir.Function
	for index := range result.HIR.Modules[0].Functions {
		function := &result.HIR.Modules[0].Functions[index]
		if function.Symbol == resource.DropSymbol {
			glue = function
		}
	}
	if glue == nil || glue.DropGlueFor != resource.Type || len(glue.Parameters) != 1 || glue.Parameters[0].Type != resource.Type || glue.ResultType != hir.TypeVoid {
		t.Fatalf("drop glue = %+v", glue)
	}
}

func TestAnalyzePropagatesDeclaredResourceDropEffects(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": `module main;
@effects(filesystem)
resource Token {
    i32 value;
    cleanup { let i32 marker = touch(); }
}
@effects(filesystem)
fn i32 touch() { return 1; }
@effects(filesystem)
fn i32 main() { let Token token = Token { .value = 42 }; return 0; }
`,
	})
	assertCleanTypeResult(t, result)
	resource := result.HIR.Modules[0].Resources[0]
	if !slices.Equal(resource.DropEffects, []hir.Effect{hir.EffectFilesystem}) {
		t.Fatalf("resource drop effects = %v", resource.DropEffects)
	}
	var glue *hir.Function
	for index := range result.HIR.Modules[0].Functions {
		function := &result.HIR.Modules[0].Functions[index]
		if function.Symbol == resource.DropSymbol {
			glue = function
		}
	}
	if glue == nil || !slices.Equal(glue.Effects, resource.DropEffects) {
		t.Fatalf("drop glue = %+v", glue)
	}
	if len(result.Effects) != 3 || !slices.Equal(result.Effects[0].Effects, resource.DropEffects) {
		t.Fatalf("effect facts = %+v", result.Effects)
	}
}

func TestAnalyzeRequiresDeclaredEffectsAtImplicitResourceDropSites(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name string
		body string
	}{
		{name: "normal function exit", body: "fn i32 main() { let Token token = Token { .value = 1 }; return 0; }"},
		{name: "early return", body: "fn i32 main(Token token) { return 0; }"},
		{name: "nested block exit", body: "fn i32 main() { { let Token token = Token { .value = 1 }; }; return 0; }"},
		{name: "loop break", body: "fn i32 main() { while (true) { let Token token = Token { .value = 1 }; break; } return 0; }"},
		{name: "live reassignment", body: "fn Token replace() { var Token token = Token { .value = 1 }; token = Token { .value = 2 }; return token; }\nfn i32 main() { return 0; }"},
	} {
		t.Run(test.name, func(t *testing.T) {
			text := `module main;
@effects(filesystem)
resource Token {
    i32 value;
    cleanup { let i32 marker = this.value; }
}
` + test.body + "\n"
			result := analyzeText(t, map[string]string{"main.fy": text})
			if result.FailedPhase != semantic.PhaseFlow || len(result.Diagnostics) == 0 || result.Diagnostics[0].Code != "E7102" || len(result.Unsupported) != 0 {
				t.Fatalf("result = %+v", result)
			}
			if len(result.Diagnostics[0].Related) != 1 || result.Diagnostics[0].Related[0].Role != "effect-declaration" {
				t.Fatalf("related = %+v", result.Diagnostics[0].Related)
			}
			attributeStart := strings.Index(text, "@effects")
			assertSpan(t, result.Diagnostics[0].Related[0].Span, "main.fy", attributeStart, attributeStart+len("@effects(filesystem)"))
		})
	}
}

func TestAnalyzeDoesNotRequireDropEffectsAfterWholeResourceMove(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": `module main;
@effects(filesystem)
resource Token {
    i32 value;
    cleanup { let i32 marker = this.value; }
}
fn Token pass(Token token) { return token; }
fn i32 main() { return 0; }
`,
	})
	assertCleanTypeResult(t, result)
}

func TestAnalyzeRejectsEffectsOnEmptyResourceDrop(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": "module main;\n@effects(filesystem)\nresource Token { i32 value; cleanup {} }\nfn i32 main() { return 0; }\n",
	})
	if result.FailedPhase != semantic.PhaseType || len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "E7101" {
		t.Fatalf("result = %+v", result)
	}
}

func TestAnalyzeRejectsMovingDropReceiver(t *testing.T) {
	t.Parallel()
	text := `module main;
resource Token {
    i32 value;
    cleanup {
        let i32 consumed = consume(this);
    }
}
fn i32 consume(Token token) { return token.value; }
fn i32 main() { return 0; }
`
	result := analyzeText(t, map[string]string{"main.fy": text})
	if result.FailedPhase != semantic.PhaseFlow || len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "E1102" {
		t.Fatalf("result = %+v", result)
	}
	start := strings.Index(text, "this);")
	assertSpan(t, result.Diagnostics[0].Primary, "main.fy", start, start+len("this"))
}

func TestAnalyzeTypesForeignCallInsideUnsafeBlock(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": "module main;\n@link_name(\"fairyc_fixture_add_one\")\nunsafe extern C fn i32 add_one(i32 value) ;\nfn i32 main() { return unsafe { add_one(41) }; }\n",
	})
	assertCleanTypeResult(t, result)
	if len(result.HIR.Modules) != 1 || len(result.HIR.Modules[0].ForeignFunctions) != 1 {
		t.Fatalf("HIR = %+v", result.HIR)
	}
	foreign := result.HIR.Modules[0].ForeignFunctions[0]
	if foreign.Name != "add_one" || foreign.LinkName != "fairyc_fixture_add_one" || foreign.ABI != "C" || foreign.ResultType != hir.TypeI32 || len(foreign.Parameters) != 1 || foreign.Parameters[0].Type != hir.TypeI32 {
		t.Fatalf("foreign function = %+v", foreign)
	}
	returned := soleReturnExpression(t, result.HIR.Modules[0].Functions[0])
	if returned.Kind != "unsafe" || len(returned.Children) != 1 || returned.Children[0].Block == nil || returned.Children[0].Block.Tail == nil {
		t.Fatalf("main return = %+v", returned)
	}
	call := returned.Children[0].Block.Tail
	if call.Kind != "call" || call.ForeignABI != "C" || call.UnsafeScope == nil || call.Symbol != foreign.Symbol {
		t.Fatalf("foreign call = %+v", call)
	}
}

func TestAnalyzePreservesDistinctCIntForeignTransport(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": `module main;
@link_name("fairyc_fixture_make_c_int")
unsafe extern C fn c::int make_c_int() ;
@link_name("fairyc_fixture_consume_c_int")
unsafe extern C fn i32 consume_c_int(c::int value) ;
fn i32 main() { return unsafe { consume_c_int(make_c_int()) }; }
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
	returned := soleReturnExpression(t, result.HIR.Modules[0].Functions[0])
	if returned.Kind != "unsafe" || len(returned.Children) != 1 || returned.Children[0].Block == nil || returned.Children[0].Block.Tail == nil {
		t.Fatalf("main return = %+v", returned)
	}
	outer := returned.Children[0].Block.Tail
	if outer.Kind != "call" || outer.Type != hir.TypeI32 || len(outer.Children) != 2 || outer.Children[1].Kind != "call" || outer.Children[1].Type != hir.TypeCInt {
		t.Fatalf("nested foreign calls = %+v", outer)
	}
}

func TestAnalyzePreservesDistinctCSizeTForeignTransport(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": `module main;
@link_name("fairyc_fixture_make_c_size_t")
unsafe extern C fn c::size_t make_c_size_t() ;
@link_name("fairyc_fixture_consume_c_size_t")
unsafe extern C fn i32 consume_c_size_t(c::size_t value) ;
fn i32 main() { return unsafe { consume_c_size_t(make_c_size_t()) }; }
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
unsafe extern C fn c::long make_c_long() ;
@link_name("fairyc_fixture_consume_c_long")
unsafe extern C fn i32 consume_c_long(c::long value) ;
fn i32 main() { return unsafe { consume_c_long(make_c_long()) }; }
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
unsafe extern C fn c::ssize_t make_c_ssize_t() ;
@link_name("fairyc_fixture_consume_c_ssize_t")
unsafe extern C fn i32 consume_c_ssize_t(c::ssize_t value) ;
fn i32 main() { return unsafe { consume_c_ssize_t(make_c_ssize_t()) }; }
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
unsafe extern C fn %[2]s make_%[1]s();
unsafe extern C fn i32 consume_%[1]s(%[2]s value);
fn i32 main() { return unsafe { consume_%[1]s(make_%[1]s()) }; }
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
			text := "module main;\nfn i32 deferred(" + typeRef + " value) { let " + typeRef + " copy = 1; return 0; }\nfn i32 main() { return 0; }\n"
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
		{name: "no argument", text: "module main;\n@link_name\nunsafe extern C fn i32 add_one(i32 value) ;\nfn i32 main() { return 0; }\n"},
		{name: "empty string", text: "module main;\n@link_name(\"\")\nunsafe extern C fn i32 add_one(i32 value) ;\nfn i32 main() { return 0; }\n"},
		{name: "path argument", text: "module main;\n@link_name(C)\nunsafe extern C fn i32 add_one(i32 value) ;\nfn i32 main() { return 0; }\n"},
		{name: "two arguments", text: "module main;\n@link_name(\"first\", \"second\")\nunsafe extern C fn i32 add_one(i32 value) ;\nfn i32 main() { return 0; }\n"},
		{name: "byte string", text: "module main;\n@link_name(b\"symbol\")\nunsafe extern C fn i32 add_one(i32 value) ;\nfn i32 main() { return 0; }\n"},
		{name: "C string", text: "module main;\n@link_name(c\"symbol\")\nunsafe extern C fn i32 add_one(i32 value) ;\nfn i32 main() { return 0; }\n"},
		{name: "duplicate", text: "module main;\n@link_name(\"first\")\n@link_name(\"second\")\nunsafe extern C fn i32 add_one(i32 value) ;\nfn i32 main() { return 0; }\n", last: true},
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
		"main.fy": "module main;\n@calling_convention(C)\nunsafe extern C fn i32 add_one(i32 value) ;\nfn i32 main() { return 0; }\n",
	})
	if len(result.Diagnostics) != 0 || len(result.Unsupported) != 1 || result.Unsupported[0].Feature != "foreign function attribute @calling_convention" {
		t.Fatalf("result = %+v", result)
	}
}

func TestAnalyzeDoesNotSilentlyAcceptLinkNameOnOrdinaryFunction(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": "module main;\n@link_name(\"external_main\")\nfn i32 main() { return 0; }\n",
	})
	if len(result.Diagnostics) != 0 || len(result.Unsupported) != 1 || result.Unsupported[0].Feature != "function attribute @link_name" {
		t.Fatalf("result = %+v", result)
	}
}

func TestAnalyzePreservesCanonicalDeclaredEffectsOnFunctionsAndCalls(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": `module main;
@effects(filesystem)
fn i32 load() { return 42; }
@effects(filesystem, console)
fn i32 main() { return load(); }
`,
	})
	assertCleanTypeResult(t, result)
	functions := result.HIR.Modules[0].Functions
	if len(functions) != 2 {
		t.Fatalf("functions = %+v", functions)
	}
	if !slices.Equal(functions[0].Effects, []hir.Effect{hir.EffectFilesystem}) {
		t.Fatalf("load effects = %v", functions[0].Effects)
	}
	wantMain := []hir.Effect{hir.EffectConsole, hir.EffectFilesystem}
	if !slices.Equal(functions[1].Effects, wantMain) {
		t.Fatalf("main effects = %v, want %v", functions[1].Effects, wantMain)
	}
	call := soleReturnExpression(t, functions[1])
	if !slices.Equal(call.Effects, functions[0].Effects) {
		t.Fatalf("call effects = %+v", call)
	}
	if len(result.Effects) != 2 || !slices.Equal(result.Effects[0].Effects, functions[0].Effects) || !slices.Equal(result.Effects[1].Effects, functions[1].Effects) {
		t.Fatalf("effect facts = %+v", result.Effects)
	}
}

func TestAnalyzeChecksForeignEffectsIndependentlyFromUnsafe(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": `module main;
@effects(filesystem)
unsafe extern C fn i32 raw_sync() ;
@effects(filesystem)
fn i32 main() { return unsafe { raw_sync() }; }
`,
	})
	assertCleanTypeResult(t, result)
	foreign := result.HIR.Modules[0].ForeignFunctions[0]
	if !slices.Equal(foreign.Effects, []hir.Effect{hir.EffectFilesystem}) {
		t.Fatalf("foreign effects = %v", foreign.Effects)
	}
	body := result.HIR.Modules[0].Functions[0].Body
	if len(body.Statements) != 1 || body.Statements[0].Kind != "return" || body.Statements[0].Expression == nil {
		t.Fatalf("main body = %+v", body)
	}
	call := body.Statements[0].Expression.Children[0].Block.Tail
	if call.UnsafeScope == nil || !slices.Equal(call.Effects, foreign.Effects) {
		t.Fatalf("foreign call = %+v", call)
	}
}

func TestAnalyzeRejectsCallWithoutRequiredEffects(t *testing.T) {
	t.Parallel()
	text := `module main;
@effects(filesystem)
fn i32 load() { return 42; }
fn i32 main() { return load(); }
`
	result := analyzeText(t, map[string]string{"main.fy": text})
	if result.FailedPhase != semantic.PhaseType || len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "E7102" || len(result.Unsupported) != 0 {
		t.Fatalf("result = %+v", result)
	}
	callStart := strings.LastIndex(text, "load()")
	assertSpan(t, result.Diagnostics[0].Primary, "main.fy", callStart, callStart+len("load"))
	attributeStart := strings.Index(text, "@effects")
	if len(result.Diagnostics[0].Related) != 1 || result.Diagnostics[0].Related[0].Role != "effect-declaration" {
		t.Fatalf("related = %+v", result.Diagnostics[0].Related)
	}
	assertSpan(t, result.Diagnostics[0].Related[0].Span, "main.fy", attributeStart, attributeStart+len("@effects(filesystem)"))
}

func TestAnalyzeRejectsMalformedUnknownOrDuplicateEffects(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name string
		attr string
		last bool
	}{
		{name: "empty", attr: "@effects"},
		{name: "empty arguments", attr: "@effects()"},
		{name: "unknown", attr: "@effects(teleport)"},
		{name: "string", attr: "@effects(\"filesystem\")"},
		{name: "duplicate argument", attr: "@effects(console, console)"},
		{name: "duplicate attribute", attr: "@effects(console)\n@effects(filesystem)", last: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			text := "module main;\n" + test.attr + "\nfn i32 main() { return 0; }\n"
			result := analyzeText(t, map[string]string{"main.fy": text})
			if result.FailedPhase != semantic.PhaseType || len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "E7101" || len(result.Unsupported) != 0 {
				t.Fatalf("result = %+v", result)
			}
			if test.last {
				start := strings.LastIndex(text, "@effects")
				assertSpan(t, result.Diagnostics[0].Primary, "main.fy", start, start+len("@effects(filesystem)"))
			}
		})
	}
}

func TestAnalyzeChecksEffectsAcrossImportedModule(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"api.fy": `module api;
@effects(network)
pub fn i32 fetch() { return 42; }
`,
		"main.fy": `module main;
import api as remote;
@effects(network, console)
fn i32 main() { return remote::fetch(); }
`,
	})
	assertCleanTypeResult(t, result)
	if len(result.HIR.Modules) != 2 || result.HIR.Modules[1].Path != "main" {
		t.Fatalf("HIR modules = %+v", result.HIR.Modules)
	}
	body := result.HIR.Modules[1].Functions[0].Body
	if len(body.Statements) != 1 || body.Statements[0].Kind != "return" {
		t.Fatalf("main body = %+v", body)
	}
	call := body.Statements[0].Expression
	if call == nil || !slices.Equal(call.Effects, []hir.Effect{hir.EffectNetwork}) {
		t.Fatalf("imported call = %+v", call)
	}
}

func TestAnalyzeSuppressesMissingEffectCascadeAfterInvalidDeclaration(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": `module main;
@effects(filesystem)
fn i32 load() { return 42; }
@effects(teleport)
fn i32 main() { return load(); }
`,
	})
	if result.FailedPhase != semantic.PhaseType || len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "E7101" {
		t.Fatalf("result = %+v", result)
	}
}

func TestAnalyzeTypesRawPointerAndTargetScalarForeignCall(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": "module main;\nunsafe extern C fn u32 foreign_probe(ptr u8 data, ptr mut u32 output, usize length, u8 marker) ;\nfn i32 main() { return if (unsafe { foreign_probe(null, null, 0, 7) } == 42) { 0 } else { 1 }; }\n",
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
	body := result.HIR.Modules[0].Functions[0].Body
	if len(body.Statements) != 1 || body.Statements[0].Kind != "return" {
		t.Fatalf("main body = %+v", body)
	}
	tail := body.Statements[0].Expression
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
		"main.fy": "module main;\nunsafe extern C fn u32 foreign_probe(ptr void input, ptr mut void output) ;\nfn i32 main() { return if (unsafe { foreign_probe(null, null) } == 0) { 0 } else { 1 }; }\n",
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
	text := "module main;\nunsafe extern C fn i32 add_one(i32 value) ;\nfn i32 main() { return add_one(41); }\n"
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
struct Packet { u32 code; }
unsafe extern C fn u32 sink(ptr mut Packet output) ;
fn i32 main() {
    var Packet output = Packet { .code = 0 };
    return if (unsafe { sink(ref mut output as ptr mut Packet) } == 0) { 0 } else { 1 };
}
`,
	})
	assertCleanTypeResult(t, result)
	main := result.HIR.Modules[0].Functions[0]
	if len(main.Body.Statements) != 2 || main.Body.Statements[1].Kind != "return" || main.Body.Statements[1].Expression == nil {
		t.Fatalf("main body = %+v", main.Body)
	}
	condition := &main.Body.Statements[1].Expression.Children[0]
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

func TestAnalyzeAllowsMigratedMutableParameterLocalAssignment(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": "module main;\nfn i32 update(i32 input) {\n    var i32 value = input; value = 42; return value; }\nfn i32 main() { return update(0); }\n",
	})
	assertCleanTypeResult(t, result)
	function := result.HIR.Modules[0].Functions[0]
	if function.Name != "update" || len(function.Parameters) != 1 || function.Parameters[0].Mutable || function.Parameters[0].Type != hir.TypeI32 ||
		len(function.Locals) != 1 || function.Locals[0].Name != "value" || !function.Locals[0].Mutable || function.Locals[0].Type != hir.TypeI32 {
		t.Fatalf("function parameters = %+v", function.Parameters)
	}
}

func TestAnalyzeRejectsImmutableOrdinaryParameterAssignment(t *testing.T) {
	t.Parallel()
	text := "module main;\nfn i32 update(i32 value) { value = 42; return value; }\n"
	result := analyzeText(t, map[string]string{"main.fy": text})
	if result.FailedPhase != semantic.PhaseType || len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "E1008" {
		t.Fatalf("result = %+v", result)
	}
	start := strings.Index(text, "value =")
	assertSpan(t, result.Diagnostics[0].Primary, "main.fy", start, start+len("value"))
}

func TestAnalyzeAllowsMigratedMutableParameterLocalBorrowToOpaquePointer(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": `module main;
@repr(C)
struct Packet { u32 code; }
unsafe extern C fn u32 sink(ptr mut void output) ;
fn u32 call(Packet input) {
    var Packet packet = input; return unsafe { sink(ref mut packet as ptr mut void) }; }
fn i32 main() { return if (call(Packet { .code = 0 }) == 0) { 42 } else { 1 }; }
`,
	})
	assertCleanTypeResult(t, result)
	callFunction := result.HIR.Modules[0].Functions[0]
	if callFunction.Name != "call" || len(callFunction.Parameters) != 1 || callFunction.Parameters[0].Mutable || callFunction.Parameters[0].Name != "input" ||
		len(callFunction.Locals) != 1 || callFunction.Locals[0].Name != "packet" || !callFunction.Locals[0].Mutable {
		t.Fatalf("call function = %+v", callFunction)
	}
	unsafeExpression := soleReturnExpression(t, callFunction)
	if len(unsafeExpression.Children) != 1 || unsafeExpression.Children[0].Block == nil {
		t.Fatalf("unsafe expression = %+v", unsafeExpression)
	}
	call := unsafeExpression.Children[0].Block.Tail
	if call == nil || len(call.Children) != 2 {
		t.Fatalf("foreign call = %+v", call)
	}
	cast := &call.Children[1]
	if cast.Type != hir.RawPointer(hir.TypeVoid, true) || cast.TargetType != cast.Type ||
		len(cast.Children) != 1 || cast.Children[0].Type != hir.MutableBorrow(hir.Type("main::Packet")) || cast.UnsafeScope == nil {
		t.Fatalf("opaque cast = %+v", cast)
	}
}

func TestAnalyzeAllowsBorrowValueToOpaquePointer(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": `module main;
resource Token { u32 value; cleanup {} }
unsafe extern C fn u32 sink(ptr mut void output) ;
fn u32 inspect(ref mut Token token) { return unsafe { sink(token as ptr mut void) }; }
fn i32 main() {
    var Token token = Token { .value = 42 };
    return if (inspect(ref mut token) == 0) { 42 } else { 1 };
}
`,
	})
	assertCleanTypeResult(t, result)
}

func TestAnalyzeTypesValuePreservingIntegerCasts(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": `module main;
fn i32 main() {
    let u8 byte = 42;
    let i32 signed = byte as i32;
    let u32 word = byte as u32;
    let usize byte_size = byte as usize;
    let usize word_size = word as usize;
    return signed;
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
		"main.fy": "module main;\nfn i32 main() { let u32 word = 42; return word as i32; }\n",
	})
	if len(result.Unsupported) != 1 || result.Unsupported[0].Feature != "cast from u32 to i32" {
		t.Fatalf("result = %+v", result)
	}
}

func TestAnalyzeTypesScalarComparisons(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": `module main;
fn i32 main() {
    let u8 byte = 41;
    let u32 word = 42;
    let usize size = 43;
    let ptr u8 pointer = null;
    let bool bool_equal = true != false;
    let bool signed_ordered = -1 < 0;
    let bool byte_ordered = byte < 42;
    let bool word_ordered = word >= 42;
    let bool size_ordered = size <= 43;
    let bool pointer_equal = pointer == null;
    return if (bool_equal && signed_ordered && byte_ordered && word_ordered && size_ordered && pointer_equal) { 42 } else { 1 };
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
struct Pair { i32 value; }
fn i32 main() {
    let Pair left = Pair { .value = 1 };
    let Pair right = Pair { .value = 1 };
    let bool same = left == right;
    let bool ordered = true < false;
    return 42;
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
fn i32 main() {
    let u8 byte = 6;
    let u8 byte_product = byte * 7;
    let u32 word = 50;
    let u32 word_difference = word - 8;
    let usize size = 41;
    let usize size_sum = size + 1;
    return if (byte_product == 42 && word_difference == 42 && size_sum == 42) { 42 } else { 1 };
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
		"main.fy": "module main;\nfn i32 main() { let bool invalid = true + false; return 42; }\n",
	})
	if len(result.Unsupported) != 1 || result.Unsupported[0].Feature != "checked arithmetic on bool" {
		t.Fatalf("result = %+v", result)
	}
}

func TestAnalyzeTypesCheckedUnsignedDivision(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": `module main;
fn i32 main() {
    let u8 byte_input = 84;
    let u8 byte = byte_input / 2;
    let u32 word_input = 85;
    let u32 word = word_input % 43;
    let usize size_input = 84;
    let usize size = size_input / 2;
    return if (byte == 42 && word == 42 && size == 42) { 42 } else { 1 };
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
fn i32 main() {
    let u8 byte = 84;
    let u8 byte_shifted = byte >> 1;
    let u32 word = 43;
    let u32 word_masked = word & 63;
    let usize size = 43;
    let usize size_mixed = (size ^ 1) | 0;
    return if (byte_shifted == 42 && word_masked == 43 && size_mixed == 42) { 42 } else { 1 };
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
fn i32 main() {
    let i32 signed_input = 21;
    let i32 signed = signed_input << 1;
    let u8 byte_input = 21;
    let u8 byte = byte_input << 1;
    let u32 word_input = 21;
    let u32 word = word_input << 1;
    let usize size_input = 21;
    let usize size = size_input << 1;
    return if (signed == 42 && byte == 42 && word == 42 && size == 42) { 42 } else { 1 };
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
fn i32 main() {
    let bool invalid_bitwise = true | false;
    let bool invalid_left_shift = true << false;
    let u8 byte = 42;
    let u32 word = 1;
    let u8 invalid_shift = byte >> word;
    let u8 invalid_left_mixed = byte << word;
    return 42;
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
fn i32 main() {
    let u8 byte_input = 213;
    let u8 byte = ~byte_input;
    let u32 word_input = 4294967253;
    let u32 word = ~word_input;
    let usize size_input = 18446744073709551573;
    let usize size = ~size_input;
    return if (byte == 42 && word == 42 && size == 42) { 42 } else { 1 };
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
		"main.fy": "module main;\nfn i32 main() { let bool invalid = ~true; return 42; }\n",
	})
	if len(result.Unsupported) != 1 || result.Unsupported[0].Feature != "bitwise not on bool" {
		t.Fatalf("result = %+v", result)
	}
}

func TestAnalyzeTypesCStringLiteralToImmutableBytePointerCast(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": `module main;
fn i32 main() {
    let ptr u8 pointer = unsafe { c"fairy" as ptr u8 };
    return 42;
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
fn i32 main() {
    let ptr u8 pointer = unsafe { b"fairy" as ptr u8 };
    return 42;
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

func TestAnalyzeRejectsUnwrappedOrMutablePointerByteStringLiteralTypes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		source  string
		feature string
	}{
		{
			name:    "bare unsized byte slice",
			source:  "module main;\nfn i32 main() { let u8[] value = b\"fairy\"; return 0; }\n",
			feature: "type u8[]",
		},
		{
			name:    "mutable byte pointer extraction",
			source:  "module main;\nfn i32 main() { let ptr mut u8 value = unsafe { b\"fairy\" as ptr mut u8 }; return 0; }\n",
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

func TestAnalyzeRejectsImmutableByteStringAsMutableByteView(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": "module main;\nfn i32 main() { let ref mut u8[] value = b\"fairy\"; return 0; }\n",
	})
	if len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "E1006" || len(result.Unsupported) != 0 {
		t.Fatalf("result = %+v", result)
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
			source:  "module main;\nfn i32 main() { let CStr value = c\"fairy\"; return 0; }\n",
			feature: "type CStr",
		},
		{
			name:    "mutable CStr borrow",
			source:  "module main;\nfn i32 main() { let ref mut CStr value = c\"fairy\"; return 0; }\n",
			feature: "type ref mut CStr",
		},
		{
			name:    "mutable byte pointer extraction",
			source:  "module main;\nfn i32 main() { let ptr mut u8 value = unsafe { c\"fairy\" as ptr mut u8 }; return 0; }\n",
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
	text := "module main;\nfn i32 main() { var i32 value = 0; let ptr mut i32 pointer = ref mut value as ptr mut i32; return 0; }\n"
	result := analyzeText(t, map[string]string{"main.fy": text})
	if result.FailedPhase != semantic.PhaseType || len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "E5101" {
		t.Fatalf("result = %+v", result)
	}
	start := strings.Index(text, "ref mut value")
	assertSpan(t, result.Diagnostics[0].Primary, "main.fy", start, start+len("ref mut value as ptr mut i32"))
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
			source:  "module main;\nfn i32 main() { let i32 value = 0; let ptr mut i32 pointer = unsafe { ref value as ptr mut i32 }; return 0; }\n",
			feature: "cast from &i32 to *mut i32",
		},
		{
			name:    "opaque mutability strengthening",
			source:  "module main;\nfn i32 main() { let i32 value = 0; let ptr mut void pointer = unsafe { ref value as ptr mut void }; return 0; }\n",
			feature: "cast from &i32 to *mut void",
		},
		{
			name:    "pointee change",
			source:  "module main;\nfn i32 main() { var i32 value = 0; let ptr mut u32 pointer = unsafe { ref mut value as ptr mut u32 }; return 0; }\n",
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
	text := "module main;\nunsafe extern C fn i32 inspect(ref i32 value) ;\nfn i32 main() { return 0; }\n"
	result := analyzeText(t, map[string]string{"main.fy": text})
	if result.FailedPhase != semantic.PhaseType || len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "E5102" {
		t.Fatalf("result = %+v", result)
	}
	start := strings.Index(text, "ref i32")
	assertSpan(t, result.Diagnostics[0].Primary, "main.fy", start, start+len("ref i32"))
}

func TestAnalyzeRejectsDuplicateForeignParameterAtResolution(t *testing.T) {
	t.Parallel()
	text := "module main;\nunsafe extern C fn i32 combine(i32 value, i32 value) ;\nfn i32 main() { return 0; }\n"
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
	text := "module main;\n\nfn i32 main() {\n    let ref str text = \"λ\";\n    return 42;\n}\n"
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
	if len(function.Body.Statements) != 2 || function.Body.Statements[0].Expression == nil {
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
fn i32 main() {
    let ref str text = "fairy";
    let ptr u8 pointer = unsafe { text as ptr u8 };
    return 42;
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
fn i32 main() {
    let ref str text = "λx";
    let ref u8[] bytes = b"\x00ABC";
    let usize text_length = text.len;
    let usize byte_length = bytes.len;
    let bool text_empty = text.is_empty;
    let bool bytes_empty = b"".is_empty;
    return if (text_length == 3 && byte_length == 4 && !text_empty && bytes_empty) { 42 } else { 1 };
}
`,
	})
	assertCleanTypeResult(t, result)
	main := result.HIR.Modules[0].Functions[0]
	if len(main.Body.Statements) != 7 {
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
fn i32 main() {
    let u8 value = b"\x00A"[1];
    return 42;
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
	text := "module main;\nfn u8 pick(u8[3] values, usize index) { return values[index]; }\nfn i32 main() { let u8[3] values = [40, 41, 42]; return pick(values, 2) as i32; }\n"
	result := analyzeText(t, map[string]string{"main.fy": text})
	assertCleanTypeResult(t, result)
	main := result.HIR.Modules[0].Functions[1]
	array := main.Body.Statements[0].Expression
	if array == nil || array.Kind != "array" || array.Type != hir.FixedArray(hir.TypeU8, 3) || array.Operator != "list" || len(array.Children) != 3 {
		t.Fatalf("array expression = %+v", array)
	}
	pick := result.HIR.Modules[0].Functions[0]
	index := soleReturnExpression(t, pick)
	if index.Kind != "array-index" || index.Type != hir.TypeU8 || index.Operator != "[]" || len(index.Children) != 2 {
		t.Fatalf("array index = %+v", index)
	}
}

func TestAnalyzeTypesScalarFixedArrayLength(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": "module main;\nfn usize length(u32[3] values) { return values.len; }\n",
	})
	assertCleanTypeResult(t, result)
	body := result.HIR.Modules[0].Functions[0].Body
	if len(body.Statements) != 1 || body.Statements[0].Kind != "return" {
		t.Fatalf("trim body = %+v", body)
	}
	expression := body.Statements[0].Expression
	if expression == nil || expression.Kind != "array-length" || expression.Type != hir.TypeUSize || expression.Operator != "len" ||
		len(expression.Children) != 1 || expression.Children[0].Type != hir.FixedArray(hir.TypeU32, 3) {
		t.Fatalf("array length expression = %+v", expression)
	}
}

func TestAnalyzeTypesContextualU8ArrayBorrowAsByteView(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": `module main;
fn u8 first(ref u8[] bytes) { return bytes[0]; }
fn i32 main() {
    let u8[3] values = [40, 41, 42];
    let ref u8[] bytes = ref values;
    return first(bytes) as i32;
}
`,
	})
	assertCleanTypeResult(t, result)
	main := result.HIR.Modules[0].Functions[1]
	if len(main.Body.Statements) != 3 || main.Body.Statements[1].Expression == nil {
		t.Fatalf("main statements = %+v", main.Body.Statements)
	}
	borrow := main.Body.Statements[1].Expression
	if borrow.Kind != "array-view-borrow" || borrow.Operator != "&" || borrow.Type != hir.SharedBorrow(hir.TypeByteSlice) ||
		len(borrow.Children) != 1 || borrow.Children[0].Type != hir.FixedArray(hir.TypeU8, 3) {
		t.Fatalf("array byte-view borrow = %+v", borrow)
	}
}

func TestAnalyzeTypesContextualMutableU8ArrayBorrowAsByteView(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": `module main;
fn usize length(ref mut u8[] bytes) { return bytes.len; }
fn i32 main() {
    var u8[3] values = [40, 41, 42];
    return if (length(ref mut values) == 3) { 42 } else { 1 };
}

`,
	})
	assertCleanTypeResult(t, result)
	main := result.HIR.Modules[0].Functions[1]
	condition := soleReturnExpression(t, main)
	if condition.Kind != "if" || len(condition.Children) == 0 {
		t.Fatalf("main return = %+v", condition)
	}
	call := condition.Children[0].Children[0]
	if call.Kind != "call" || len(call.Children) != 2 {
		t.Fatalf("mutable byte-view call = %+v", call)
	}
	borrow := call.Children[1]
	if borrow.Kind != "array-view-borrow" || borrow.Operator != "&mut" || borrow.Type != hir.MutableBorrow(hir.TypeByteSlice) ||
		len(borrow.Children) != 1 || borrow.Children[0].Type != hir.FixedArray(hir.TypeU8, 3) {
		t.Fatalf("mutable array byte-view borrow = %+v", borrow)
	}
}

func TestAnalyzeTypesCheckedMutableIndexedAssignment(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": `module main;
fn u8 update_array(u8[3] input) {
    var u8[3] values = input;
    values[1] = 42;
    return values[1];
}
fn u8 update_view(ref mut u8[] bytes) {
    bytes[1] = 42;
    return bytes[1];
}
fn i32 main() { return 0; }
`,
	})
	assertCleanTypeResult(t, result)
	functions := result.HIR.Modules[0].Functions
	if len(functions) != 3 {
		t.Fatalf("functions = %+v", functions)
	}
	arrayTarget := functions[0].Body.Statements[1].Target
	if arrayTarget == nil || arrayTarget.Kind != "array-index" || arrayTarget.Type != hir.TypeU8 ||
		len(arrayTarget.Children) != 2 || arrayTarget.Children[0].Type != hir.FixedArray(hir.TypeU8, 3) {
		t.Fatalf("array assignment target = %+v", arrayTarget)
	}
	viewTarget := functions[1].Body.Statements[0].Target
	if viewTarget == nil || viewTarget.Kind != "view-index" || viewTarget.Type != hir.TypeU8 ||
		len(viewTarget.Children) != 2 || viewTarget.Children[0].Type != hir.MutableBorrow(hir.TypeByteSlice) {
		t.Fatalf("view assignment target = %+v", viewTarget)
	}
	viewRead := soleReturnExpression(t, functions[1])
	if viewRead.Kind != "view-index" || viewRead.Children[0].Type != hir.MutableBorrow(hir.TypeByteSlice) {
		t.Fatalf("mutable view read = %+v", viewRead)
	}
}

func TestAnalyzeRejectsImmutableIndexedAssignmentTargets(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name string
		text string
	}{
		{
			name: "fixed array binding",
			text: "module main;\nfn i32 main() { let u8[1] values = [0]; values[0] = 42; return 0; }\n",
		},
		{
			name: "shared byte view",
			text: "module main;\nfn i32 update(ref u8[] bytes) { bytes[0] = 42; return 0; }\n",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			result := analyzeText(t, map[string]string{"main.fy": test.text})
			if result.FailedPhase != semantic.PhaseType || len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "E1008" {
				t.Fatalf("result = %+v", result)
			}
		})
	}
}

func TestAnalyzeRejectsMutableByteViewOfImmutableFixedArray(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": `module main;
fn usize length(ref mut u8[] bytes) { return bytes.len; }
fn i32 main() {
    let u8[3] values = [40, 41, 42];
    return if (length(ref mut values) == 3) { 42 } else { 1 };
}
`,
	})
	if result.FailedPhase != semantic.PhaseType || len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "E1008" {
		t.Fatalf("result = %+v", result)
	}
}

func TestAnalyzeTracksSequentialAndBranchSafeBorrowReassignment(t *testing.T) {
	t.Parallel()
	for _, fixture := range []struct {
		name   string
		branch string
	}{
		{name: "sequential", branch: "current = right;"},
		{name: "same branch origin", branch: "if (flag) { current = right; 0 } else { current = right; 0 };"},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			result := analyzeText(t, map[string]string{
				"main.fy": `module main;
fn ref i32 select(bool flag, ref i32 left, ref i32 right) {
    var ref i32 current = left;
    ` + fixture.branch + `
    return current;
}
`,
			})
			assertCleanTypeResult(t, result)
			if len(result.Borrows) != 1 || result.Borrows[0].Origin != "Param(2)" || !result.Borrows[0].Escapes {
				t.Fatalf("borrows = %+v", result.Borrows)
			}
		})
	}
}

func TestAnalyzeRejectsBranchMergedSafeBorrowOrigins(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": `module main;
fn ref i32 select(bool flag, ref i32 left, ref i32 right) {
    var ref i32 current = left;
    if (flag) { current = left; 0 } else { current = right; 0 };
    return current;
}
`,
	})
	if len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "E2103" {
		t.Fatalf("diagnostics = %+v", result.Diagnostics)
	}
}

func TestAnalyzeMapsCalleeBorrowReturnParameterToExactArgument(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": `module main;
fn ref i32 relay(ref i32 left, ref i32 right) { return second(right, left); }
fn ref i32 second(ref i32 left, ref i32 right) { return right; }
`,
	})
	assertCleanTypeResult(t, result)
	var escaping []string
	for _, fact := range result.Borrows {
		if fact.Escapes {
			escaping = append(escaping, fact.Origin)
		}
	}
	if !slices.Equal(escaping, []string{"Param(0)", "Param(1)"}) {
		t.Fatalf("escaping origins = %v, borrows = %+v", escaping, result.Borrows)
	}
}

func TestAnalyzeMapsCalleeRegionReturnParameterToExactArgument(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": `module main;
fn ref i32 relay(Region scratch, i32 value) { return store(scratch, value); }
fn ref i32 store(Region scratch, i32 value) { return scratch.place(value); }
`,
	})
	assertCleanTypeResult(t, result)
	var escaping []string
	for _, fact := range result.Borrows {
		if fact.Escapes {
			escaping = append(escaping, fact.Origin)
		}
	}
	if !slices.Equal(escaping, []string{"RegionParam(0)", "RegionParam(0)"}) {
		t.Fatalf("escaping origins = %v, borrows = %+v", escaping, result.Borrows)
	}
}

func TestAnalyzeKeepsUnprovedRecursiveBorrowReturnClosed(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": "module main;\nfn ref i32 recurse(ref i32 value) { return recurse(value); }\n",
	})
	if len(result.Unsupported) != 1 || result.Unsupported[0].Feature != "safe-borrow return with unknown origin" {
		t.Fatalf("unsupported = %+v", result.Unsupported)
	}
}

func TestAnalyzeMapsCalleeStaticBorrowReturnWithoutRetainingArguments(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": `module main;
fn ref u8[] literal(ref u8[] left, ref u8[] right) { return b"z"; }
fn ref u8[] relay(ref u8[] left, ref u8[] right) { return literal(left, right); }
`,
	})
	assertCleanTypeResult(t, result)
	var escaping []string
	for _, fact := range result.Borrows {
		if fact.Escapes {
			escaping = append(escaping, fact.Origin)
		}
	}
	if !slices.Equal(escaping, []string{"Static", "Static"}) {
		t.Fatalf("escaping origins = %v, borrows = %+v", escaping, result.Borrows)
	}
}

func TestAnalyzeCalleeBorrowSummaryReleasesUnselectedOwner(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": `module main;
fn ref u8[] second(ref u8[] left, ref u8[] right) { return right; }
fn i32 main() {
    var u8[1] left = [1];
    let u8[1] right = [42];
    let ref u8[] selected = second(ref left, ref right);
    left = [2];
    return selected[0] as i32;
}
`,
	})
	assertCleanTypeResult(t, result)
}

func TestAnalyzeCalleeBorrowSummaryProtectsSelectedOwner(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": `module main;
fn ref u8[] second(ref u8[] left, ref u8[] right) { return right; }
fn i32 main() {
    let u8[1] left = [1];
    var u8[1] right = [42];
    let ref u8[] selected = second(ref left, ref right);
    right = [0];
    return selected[0] as i32;
}
`,
	})
	if len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "E2101" {
		t.Fatalf("diagnostics = %+v", result.Diagnostics)
	}
}

func TestAnalyzeEndsReassignedArrayViewLoanBeforeNewOrigin(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": `module main;
fn i32 main() {
    var u8[1] left = [41];
    var u8[1] right = [42];
    var ref u8[] bytes = ref left;
    bytes = ref right;
    left = [40];
    return bytes[0] as i32;
}
`,
	})
	assertCleanTypeResult(t, result)
}

func TestAnalyzeProtectsReassignedArrayViewNewOrigin(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": `module main;
fn i32 main() {
    var u8[1] left = [41];
    var u8[1] right = [42];
    var ref u8[] bytes = ref left;
    bytes = ref right;
    right = [43];
    return bytes[0] as i32;
}
`,
	})
	if len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "E2101" {
		t.Fatalf("diagnostics = %+v", result.Diagnostics)
	}
}

func TestAnalyzeBorrowLivenessIsBranchReachable(t *testing.T) {
	t.Parallel()
	t.Run("disjoint branches", func(t *testing.T) {
		result := analyzeText(t, map[string]string{
			"main.fy": `module main;
fn i32 main(bool flag) {
    var i32 value = 1;
    let ref i32 view = ref value;
    if (flag) { value = 2; 0 } else { *view };
    return 42;
}
`,
		})
		assertCleanTypeResult(t, result)
	})
	t.Run("same branch", func(t *testing.T) {
		result := analyzeText(t, map[string]string{
			"main.fy": `module main;
fn i32 main(bool flag) {
    var i32 value = 1;
    let ref i32 view = ref value;
    if (flag) { value = 2; *view } else { 0 };
    return 42;
}
`,
		})
		if len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "E2101" {
			t.Fatalf("diagnostics = %+v", result.Diagnostics)
		}
	})
}

func TestAnalyzeTracksLoopCarriedSafeBorrowReassignment(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": `module main;
fn i32 main() {
    let u8[1] left = [41];
    let u8[1] right = [42];
    var ref u8[] bytes = ref left;
    while (false) { bytes = ref right; }
    return bytes[0] as i32;
}
`,
	})
	assertCleanTypeResult(t, result)
}

func TestAnalyzeLoopBorrowReassignmentReleasesPriorOwnerOnBodyPath(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": `module main;
fn i32 main(bool run) {
    var u8[1] left = [41];
    let u8[1] right = [42];
    var ref u8[] bytes = ref left;
    while (run) {
        bytes = ref right;
        left = [40];
        if (bytes[0] == 42) { break; } else { continue; }
    }
    return 42;
}
`,
	})
	assertCleanTypeResult(t, result)
}

func TestAnalyzeLoopBorrowReassignmentProtectsNewOwner(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": `module main;
fn i32 main(bool run) {
    let u8[1] left = [41];
    var u8[1] right = [42];
    var ref u8[] bytes = ref left;
    while (run) {
        bytes = ref right;
        right = [0];
        if (bytes[0] == 42) { break; } else { continue; }
    }
    return 42;
}
`,
	})
	if len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "E2101" {
		t.Fatalf("diagnostics = %+v", result.Diagnostics)
	}
}

func TestAnalyzeLoopContinueCarriesBorrowToHeader(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": `module main;
fn i32 main(bool run) {
    let u8[1] left = [41];
    var u8[1] right = [42];
    var ref u8[] bytes = ref left;
    var bool first = true;
    while (run) {
        if (first) {
            bytes = ref right;
            first = false;
            continue;
        };
        right = [0];
        if (bytes[0] == 42) { break; } else { break; }
    }
    return 42;
}
`,
	})
	if len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "E2101" {
		t.Fatalf("diagnostics = %+v", result.Diagnostics)
	}
}

func TestAnalyzeLoopBreakJoinsBorrowReturnOrigins(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": `module main;
fn ref i32 select(bool run, ref i32 left, ref i32 right) {
    var ref i32 current = left;
    while (run) {
        current = right;
        break;
    }
    return current;
}
`,
	})
	if len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "E2103" {
		t.Fatalf("diagnostics = %+v", result.Diagnostics)
	}
}

func TestAnalyzeKeepsOtherFixedArrayFieldsClosed(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": "module main;\nfn i32 main() { let i32[1] values = [42]; return if (values.is_empty) { 1 } else { 42 }; }\n",
	})
	if len(result.Unsupported) != 1 || result.Unsupported[0].Feature != "unknown fixed-array field is_empty" {
		t.Fatalf("unsupported = %+v", result.Unsupported)
	}
}

func TestAnalyzeRejectsFixedArrayLengthAndElementMismatch(t *testing.T) {
	t.Parallel()
	for _, text := range []string{
		"module main;\nfn i32 main() { let i32[2] values = [1, 2, 3]; return values[0]; }\n",
		"module main;\nfn i32 main() { let values = [1, true]; return values[0]; }\n",
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
		text := "module main;\nfn i32 main() { let i32[3] values = " + literal + "; return 0; }\n"
		result := analyzeText(t, map[string]string{"main.fy": text})
		if len(result.Unsupported) != 1 {
			t.Fatalf("literal %s unsupported = %+v", literal, result.Unsupported)
		}
	}
}

func TestAnalyzeBindsRawPartsBorrowedViewToOwnerOrigin(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": `module main;
resource String {
    ptr mut void data;
    usize length;
    cleanup {}
}
fn ref str as_str(ref String value) {
    return unsafe { str::from_raw_parts(value, value.data, value.length) };
}
fn ref str relay(ref String value) { return as_str(value); }
`,
	})
	assertCleanTypeResult(t, result)

	var constructor *hir.Expression
	for functionIndex := range result.HIR.Modules[0].Functions {
		function := &result.HIR.Modules[0].Functions[functionIndex]
		if function.Name != "as_str" {
			continue
		}
		returned := soleReturnExpression(t, *function)
		if len(returned.Children) != 1 || returned.Children[0].Block == nil {
			t.Fatalf("as_str return = %+v", returned)
		}
		constructor = returned.Children[0].Block.Tail
	}
	if constructor == nil || constructor.Kind != "borrowed-view-from-raw-parts" || constructor.Type != hir.SharedBorrow(hir.TypeStr) ||
		constructor.Operator != "str::from_raw_parts" || constructor.UnsafeScope == nil || len(constructor.Children) != 3 {
		t.Fatalf("constructor = %+v", constructor)
	}
	var escaping []string
	for _, fact := range result.Borrows {
		if fact.Escapes {
			escaping = append(escaping, fact.Origin)
		}
	}
	if !slices.Equal(escaping, []string{"Param(0)", "Param(0)"}) {
		t.Fatalf("escaping origins = %v, borrows = %+v", escaping, result.Borrows)
	}
}

func TestAnalyzeRejectsRawPartsBorrowedViewOutsideUnsafe(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": `module main;
resource String {
    ptr mut void data;
    usize length;
    cleanup {}
}
fn ref str as_str(ref String value) {
    return str::from_raw_parts(value, value.data, value.length);
}
`,
	})
	if result.FailedPhase != semantic.PhaseType || len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "E5101" {
		t.Fatalf("result = %+v", result)
	}
}

func TestAnalyzeRejectsOwnerMoveBeforeRawPartsViewLastUse(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": `module main;
resource String {
    ptr mut void data;
    usize length;
    cleanup {}
}
fn ref str as_str(ref String value) {
    return unsafe { str::from_raw_parts(value, value.data, value.length) };
}
fn i32 consume_owned(String value) { return 0; }
fn i32 main() {
    let String value = String { .data = null, .length = 0 };
    let ref str view = as_str(ref value);
    let i32 consumed = consume_owned(value);
    return if (view.len == 0) { 0 } else { 1 };
}
`,
	})
	if result.FailedPhase != semantic.PhaseFlow || len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "E2105" {
		t.Fatalf("result = %+v", result)
	}
}

func TestAnalyzeRejectsFixedArrayBeyondExecutableElementLimit(t *testing.T) {
	t.Parallel()
	length := hir.MaxExecutableFixedArrayElements + 1
	t.Run("type", func(t *testing.T) {
		text := fmt.Sprintf("module main;\nfn i32 main(u8[%d] values) { return 0; }\n", length)
		result := analyzeText(t, map[string]string{"main.fy": text})
		if len(result.Unsupported) != 1 || result.Unsupported[0].Feature != fmt.Sprintf("type u8[%d]", length) {
			t.Fatalf("unsupported = %+v", result.Unsupported)
		}
	})
	t.Run("inferred literal", func(t *testing.T) {
		literal := "[" + strings.Repeat("0,", int(length-1)) + "0]"
		text := "module main;\nfn i32 main() { let values = " + literal + "; return 0; }\n"
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
			"main.fy": "module main;\nfn i32 main() { let u8 value = \"fairy\"[0]; return 42; }\n",
		})
		if len(result.Unsupported) != 1 || result.Unsupported[0].Feature != "indexing outside byte views" {
			t.Fatalf("result = %+v", result)
		}
	})
	t.Run("non-usize offset", func(t *testing.T) {
		result := analyzeText(t, map[string]string{
			"main.fy": "module main;\nfn i32 main() { let u32 offset = 0; let u8 value = b\"fairy\"[offset]; return 42; }\n",
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
fn ref u8[] trim(ref u8[] bytes) { return bytes[1..2]; }
`,
	})
	assertCleanTypeResult(t, result)
	expression := soleReturnExpression(t, result.HIR.Modules[0].Functions[0])
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
				"main.fy": "module main;\nfn ref u8[] trim(ref u8[] bytes) { return bytes[" + test.bounds + "]; }\n",
			})
			assertCleanTypeResult(t, result)
			expression := soleReturnExpression(t, result.HIR.Modules[0].Functions[0])
			if expression.Kind != "view-slice" || len(expression.Children) != 3 {
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
			"main.fy": "module main;\nfn i32 main() { let ref str value = \"fairy\"[0..1]; return 42; }\n",
		})
		if len(result.Unsupported) != 1 || result.Unsupported[0].Feature != "slicing outside immutable byte views" {
			t.Fatalf("result = %+v", result)
		}
	})
	t.Run("non-usize bounds", func(t *testing.T) {
		result := analyzeText(t, map[string]string{
			"main.fy": "module main;\nfn i32 main() { let u32 start = 0; let ref u8[] value = b\"fairy\"[start..1]; return 42; }\n",
		})
		if len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "E1006" || len(result.Unsupported) != 0 {
			t.Fatalf("result = %+v", result)
		}
	})
}

func TestAnalyzeRejectsUnknownBorrowedViewField(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": "module main;\nfn i32 main() { let ref str text = \"fairy\"; let usize value = text.capacity; return 42; }\n",
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
			source:  "module main;\nfn i32 main() { let str value = \"fairy\"; return 0; }\n",
			feature: "type str",
		},
		{
			name:    "mutable str borrow",
			source:  "module main;\nfn i32 main() { let ref mut str value = \"fairy\"; return 0; }\n",
			feature: "type ref mut str",
		},
		{
			name:    "mutable byte pointer extraction",
			source:  "module main;\nfn i32 main() { let ptr mut u8 value = unsafe { \"fairy\" as ptr mut u8 }; return 0; }\n",
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
		"main.fy": "module main;\nfn i32 main() { let bool ready = true; assert(ready); return 42; }\n",
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
		"main.fy": "module main;\nfn i32 main() { assert(1); return 42; }\n",
	})
	if result.FailedPhase != semantic.PhaseType || len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "E1007" || len(result.Unsupported) != 0 {
		t.Fatalf("result = %+v", result)
	}
}

func TestAnalyzeTypesClassicScalarForAndScopesInitializer(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": "module main;\nfn i32 main() { var i32 total = 0; for (var i32 index = 0; index < 6; index = index + 1) { total = total + 7; } return total; }\n",
	})
	assertCleanTypeResult(t, result)
	function := result.HIR.Modules[0].Functions[0]
	if len(function.Locals) != 2 || len(function.Body.Statements) != 3 {
		t.Fatalf("function = %+v", function)
	}
	loop := function.Body.Statements[1].For
	if loop == nil || loop.Initializer == nil || loop.Initializer.Expression == nil || loop.Initializer.Expression.Type != hir.TypeI32 ||
		loop.Condition == nil || loop.Condition.Type != hir.TypeBool || loop.Post == nil || loop.Post.Target == nil || loop.Post.Target.Type != hir.TypeI32 ||
		len(loop.Body.Statements) != 1 {
		t.Fatalf("typed for = %+v", loop)
	}

	scopeResult := analyzeText(t, map[string]string{
		"main.fy": "module main;\nfn i32 main() { for (var i32 index = 0; index < 1; index = index + 1) {} return index; }\n",
	})
	if scopeResult.FailedPhase != semantic.PhaseResolve || len(scopeResult.Diagnostics) != 1 || scopeResult.Diagnostics[0].Code != "E0101" {
		t.Fatalf("scope result = %+v", scopeResult)
	}
}

func TestAnalyzeTypesCheckedCompoundAssignments(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": `module main;
fn i32 main() {
    var i32 value = 21;
    value += 21;
    value -= 2;
    value *= 2;
    value /= 4;
    value %= 30;
    value <<= 1;
    value >>= 1;
    value &= 31;
    value ^= 1;
    value |= 2;
    var u8[1] bytes = [40];
    bytes[0] += 2;
    for (var i32 index = 0; index < 1; index += 1) { value += index; }
    return bytes[0] as i32;
}
`,
	})
	assertCleanTypeResult(t, result)
	function := result.HIR.Modules[0].Functions[0]
	var operators []string
	for _, statement := range function.Body.Statements {
		if statement.Kind == "assignment" {
			operators = append(operators, statement.Operator)
		}
	}
	want := []string{"+=", "-=", "*=", "/=", "%=", "<<=", ">>=", "&=", "^=", "|=", "+="}
	if !slices.Equal(operators, want) || function.Body.Statements[len(function.Body.Statements)-2].For.Post.Operator != "+=" {
		t.Fatalf("compound assignment operators = %v, function = %+v", operators, function)
	}
}

func TestAnalyzeRejectsUnsupportedCompoundAssignmentType(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": "module main;\nfn i32 main() { var bool flag = true; flag += true; return 0; }\n",
	})
	if len(result.Unsupported) != 1 || result.Unsupported[0].Feature != "compound assignment += on bool" {
		t.Fatalf("result = %+v", result)
	}
}

func TestAnalyzeTypesLexicalDeferWithCopyScalarArguments(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": "module main;\nfn void record(i32 value, bool enabled) {}\nfn i32 main() { defer record(42, true); return 42; }\n",
	})
	assertCleanTypeResult(t, result)
	statement := result.HIR.Modules[0].Functions[1].Body.Statements[0]
	if statement.Kind != "defer" || statement.Expression == nil || statement.Expression.Kind != "call" || statement.Expression.Type != hir.TypeVoid {
		t.Fatalf("defer statement = %+v", statement)
	}
}

func TestAnalyzeRejectsDeferredValueAndBorrowCalls(t *testing.T) {
	t.Parallel()
	for _, sourceText := range []string{
		"module main;\nfn i32 value() { return 42; }\nfn i32 main() { defer value(); return 42; }\n",
		"module main;\nfn void inspect(ref i32 value) {}\nfn i32 main() { let i32 value = 42; defer inspect(ref value); return value; }\n",
	} {
		result := analyzeText(t, map[string]string{"main.fy": sourceText})
		if len(result.Unsupported) != 1 || result.Unsupported[0].Feature != "defer outside an internal void call with copy-scalar arguments" {
			t.Fatalf("result = %+v", result)
		}
	}
}

func TestAnalyzeRejectsNonScalarForHeaderBehavior(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": "module main;\nfn i32 next(i32 value) { return value + 1; }\nfn i32 main() { for (var i32 index = next(0); index < 2; index = next(index)) {} return 42; }\n",
	})
	if len(result.Unsupported) != 2 {
		t.Fatalf("result = %+v", result)
	}
	if result.Unsupported[0].Feature != "for initializer outside the scalar declaration or direct-assignment subset" ||
		result.Unsupported[1].Feature != "for post clause outside direct scalar assignment" {
		t.Fatalf("unsupported = %+v", result.Unsupported)
	}
}

func TestAnalyzePreservesAndHandlesMustUseCalls(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": "module main;\n@must_use\nfn i32 status() { return 7; }\nfn i32 accept(i32 value) { return value; }\nfn i32 main() { discard status(); let stored = status(); return accept(stored) + status(); }\n",
	})
	assertCleanTypeResult(t, result)
	if len(result.HIR.Modules) != 1 || len(result.HIR.Modules[0].Functions) != 3 {
		t.Fatalf("HIR = %+v", result.HIR)
	}
	if status := result.HIR.Modules[0].Functions[0]; status.Name != "status" || !status.MustUse {
		t.Fatalf("status = %+v", status)
	}
}

func TestAnalyzeRejectsIgnoredMustUseCallShapes(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": "module main;\n@must_use\nfn i32 status() { return 7; }\nfn i32 main() { status(); (status()); unsafe { status() }; if (true) { status() } else { status() }; return 0; }\n",
	})
	if result.FailedPhase != semantic.PhaseFlow || len(result.Unsupported) != 0 || len(result.Diagnostics) != 5 {
		t.Fatalf("result = %+v", result)
	}
	for _, value := range result.Diagnostics {
		if value.Code != "E4202" || len(value.Related) != 1 || value.Related[0].Role != "must-use-declaration" {
			t.Fatalf("diagnostic = %+v", value)
		}
	}
}

func TestAnalyzePreservesMustUseOnMethodsAndForeignFunctions(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": "module main;\nstruct Meter { i32 value; }\n@must_use\nfn i32 Meter::read(ref Meter this) { return this.value; }\n@must_use\n@link_name(\"fairyc_raw_status\")\nunsafe extern C fn i32 raw_status();\nfn i32 main() { let Meter meter = Meter { .value = 40 }; return meter.read() + unsafe { raw_status() }; }\n",
	})
	assertCleanTypeResult(t, result)
	module := result.HIR.Modules[0]
	if len(module.ForeignFunctions) != 1 || !module.ForeignFunctions[0].MustUse {
		t.Fatalf("foreign functions = %+v", module.ForeignFunctions)
	}
	if len(module.Functions) != 2 || !module.Functions[0].MustUse || module.Functions[0].OwnerType == "" {
		t.Fatalf("functions = %+v", module.Functions)
	}
}

func TestAnalyzeEnforcesMustUseAcrossImportedModule(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"api.fy":  "module api;\n@must_use\npub fn i32 status() { return 7; }\n",
		"main.fy": "module main;\nimport api as api;\nfn i32 main() { api::status(); return 0; }\n",
	})
	if result.FailedPhase != semantic.PhaseFlow || len(result.Unsupported) != 0 || len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "E4202" {
		t.Fatalf("result = %+v", result)
	}
	if len(result.Diagnostics[0].Related) != 1 || result.Diagnostics[0].Related[0].Span.Source != "api.fy" {
		t.Fatalf("diagnostic = %+v", result.Diagnostics[0])
	}
}

func TestAnalyzeRejectsInvalidMustUseDeclarations(t *testing.T) {
	t.Parallel()
	tests := map[string]string{
		"arguments": "module main;\n@must_use(1)\nfn i32 status() { return 1; }\n",
		"duplicate": "module main;\n@must_use\n@must_use\nfn i32 status() { return 1; }\n",
		"void":      "module main;\n@must_use\nfn void notify() {}\n",
		"result":    "module main;\n@must_use\nfn Result<i32, i32> status() { return .ok(1); }\n",
		"borrow":    "module main;\n@must_use\nfn ref i32 choose(ref i32 value) { return value; }\n",
		"struct":    "module main;\n@must_use\nstruct Status { i32 value; }\n",
	}
	for name, text := range tests {
		name, text := name, text
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			result := analyzeText(t, map[string]string{"main.fy": text})
			if len(result.Diagnostics) == 0 || result.Diagnostics[0].Code != "E4201" {
				t.Fatalf("result = %+v", result)
			}
		})
	}
}

func TestAnalyzeFailsClosedForPreservedCompileTimeItemsBeforeADR0220Implementation(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": "module main;\nconst i32 ANSWER = 42;\nstatic_assert(ANSWER == 42);\nfn i32 main() { return 0; }\n",
	})
	if result.CompletedThrough != semantic.PhaseNone || result.FailedPhase != semantic.PhaseNone || len(result.Diagnostics) != 0 || len(result.Unsupported) != 2 {
		t.Fatalf("result = %+v", result)
	}
	if result.Unsupported[0].Feature != "module constant pending ADR 0220 implementation" ||
		result.Unsupported[1].Feature != "static_assert pending ADR 0220 implementation" {
		t.Fatalf("unsupported = %+v", result.Unsupported)
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

func soleReturnExpression(t *testing.T, function hir.Function) *hir.Expression {
	t.Helper()
	var returned *hir.Expression
	for statementIndex := range function.Body.Statements {
		statement := &function.Body.Statements[statementIndex]
		if statement.Kind != "return" {
			continue
		}
		if returned != nil {
			t.Fatalf("function %q has multiple top-level returns: %+v", function.Name, function.Body)
		}
		if statement.Expression == nil {
			t.Fatalf("function %q has a valueless top-level return: %+v", function.Name, statement)
		}
		returned = statement.Expression
	}
	if returned == nil {
		t.Fatalf("function %q has no top-level return: %+v", function.Name, function.Body)
	}
	return returned
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
