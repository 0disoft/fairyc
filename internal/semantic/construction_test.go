package semantic_test

import "testing"

func TestAnalyzeKeepsResourceLiteralConstructionInsideOwnerModule(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"lib.fy": `module lib;
pub struct Pair { i32 value; }
pub resource Token { i32 value; cleanup {} }
pub fn Token token(i32 value) { return Token { .value = value }; }
`,
		"main.fy": `module main;
import lib as lib;
fn i32 main() {
    let lib::Pair pair = lib::Pair { .value = 40 };
    let lib::Token token = lib::token(2);
    return pair.value + 2;
}
`,
	})
	assertCleanTypeResult(t, result)
}

func TestAnalyzeRejectsResourceLiteralOutsideOwnerModule(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{
		"lib.fy": `module lib;
pub resource Token { i32 value; cleanup {} }
`,
		"main.fy": `module main;
import lib as lib;
fn i32 main() {
    let lib::Token token = lib::Token { .value = 42 };
    return 0;
}
`,
	})
	if !hasDiagnosticCode(result, "E0103") {
		t.Fatalf("diagnostics = %+v, unsupported = %+v, want E0103", result.Diagnostics, result.Unsupported)
	}
}
