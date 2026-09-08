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

func TestAnalyzeRejectsResourceFieldOutsideOwnerModule(t *testing.T) {
	for _, receiver := range []string{"token", "borrowed"} {
		t.Run(receiver, func(t *testing.T) {
			result := analyzeText(t, map[string]string{
				"lib.fy": `module lib;
pub resource Token { i32 value; cleanup {} }
pub fn Token token() { return Token { .value = 42 }; }
pub fn i32 read(ref Token value) { return value.value; }
`,
				"main.fy": `module main; import lib as lib;
fn i32 main() {
    let lib::Token token = lib::token();
    let ref lib::Token borrowed = ref token;
    return ` + receiver + `.value;
}`,
			})
			if !hasDiagnosticCode(result, "E0103") {
				t.Fatalf("external resource field: %+v %+v", result.Diagnostics, result.Unsupported)
			}
		})
	}
}
