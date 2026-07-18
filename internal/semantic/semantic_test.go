package semantic_test

import (
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
