package semantic_test

import (
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/semantic"
)

func TestAnalyzeEvaluatesPrivateScalarConstantsAndFoldsHIRReferences(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": `module main;
@const_eval
fn i32 square(i32 value) { return value * value; }
const i32 ANSWER = square(6) + 6;
const bool READY = ANSWER == 42;
static_assert(READY);
fn i32 main() { return ANSWER; }
`,
	})
	assertCleanTypeResult(t, result)
	if len(result.Constants) != 2 || result.Constants[0].Name != "ANSWER" || result.Constants[0].Type != hir.TypeI32 || result.Constants[0].Value != "42" ||
		result.Constants[1].Name != "READY" || result.Constants[1].Type != hir.TypeBool || result.Constants[1].Value != "true" {
		t.Fatalf("constants = %+v", result.Constants)
	}
	if len(result.HIR.Modules) != 1 || len(result.HIR.Modules[0].Functions) != 2 {
		t.Fatalf("HIR = %+v", result.HIR)
	}
	mainFunction := result.HIR.Modules[0].Functions[1]
	returned := mainFunction.Body.Statements[0].Expression
	if returned == nil || returned.Kind != "literal" || returned.Type != hir.TypeI32 || returned.Value != "42" || returned.Symbol != hir.NoSymbol {
		t.Fatalf("folded return = %+v", returned)
	}
}

func TestAnalyzeCompileTimeEvaluationShortCircuitsTrappingBranches(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"main.fy": `module main;
const bool LEFT = false && (1 / 0 == 0);
const i32 SELECTED = if (true) { 42 } else { 1 / 0 };
static_assert(!LEFT && SELECTED == 42);
fn i32 main() { return SELECTED; }
`,
	})
	assertCleanTypeResult(t, result)
}

func TestAnalyzeRejectsCompileTimeTrapCycleAndFalseAssertion(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		body string
		code string
	}{
		{name: "trap", body: "const i32 BAD = 1 / 0;", code: "E4303"},
		{name: "cycle", body: "const i32 A = B;\nconst i32 B = A;", code: "E4304"},
		{name: "false assertion", body: "static_assert(false);", code: "E4305"},
		{name: "excluded expression", body: "fn i32 ordinary() { return 1; }\nconst i32 BAD = ordinary();", code: "E4302"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			result := analyzeText(t, map[string]string{"main.fy": "module main;\n" + test.body + "\nfn i32 main() { return 0; }\n"})
			if !hasDiagnosticCode(result, test.code) {
				t.Fatalf("diagnostics = %+v, want %s", result.Diagnostics, test.code)
			}
		})
	}
}

func hasDiagnosticCode(result semantic.Result, code string) bool {
	for _, diagnostic := range result.Diagnostics {
		if diagnostic.Code == code {
			return true
		}
	}
	return false
}
