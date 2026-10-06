package semantic_test

import "testing"

func TestAnalyzeResourcePropagationConsumesTheResultRoot(t *testing.T) {
	t.Parallel()
	for _, body := range []string{
		"let R r=x?;return x;",
		"let R r=x?;return .ok(x?);",
	} {
		result := analyzeText(t, map[string]string{"main.fy": "module main;resource R{cleanup{}}fn Result<R,i32> bad(Result<R,i32> x){" + body + "}"})
		if len(result.Unsupported) != 0 || len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "E1101" {
			t.Fatalf("diagnostics=%+v unsupported=%+v", result.Diagnostics, result.Unsupported)
		}
		hasMove := false
		for _, related := range result.Diagnostics[0].Related {
			if related.Role == "move" {
				hasMove = true
			}
		}
		if !hasMove {
			t.Fatalf("missing propagation move provenance: %+v", result.Diagnostics)
		}
	}
}

func TestAnalyzeResourcePropagationKeepsExtractedAndUnreachableValues(t *testing.T) {
	t.Parallel()
	for _, body := range []string{
		"let R r=x?;return .ok(r);",
		"return .ok(x?);",
		"return x;x?;",
	} {
		result := analyzeText(t, map[string]string{"main.fy": "module main;resource R{cleanup{}}fn Result<R,i32> good(Result<R,i32> x){" + body + "}"})
		assertCleanTypeResult(t, result)
	}
}
