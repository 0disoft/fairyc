package semantic_test

import "testing"

func TestAnalyzeShortCircuitPreservesResourceAndResultBranches(t *testing.T) {
	t.Parallel()
	for _, operator := range []string{"&&", "||"} {
		for _, resource := range []bool{false, true} {
			prefix, typ, want := "", "i32", "E4101"
			if resource {
				prefix, typ, want = "resource R{cleanup{}}", "R", "E1104"
			}
			input := "module main;" + prefix + "fn bool test(bool gate,Result<" + typ + ",i32> r){return gate " + operator + " r.is_ok;}"
			result := analyzeText(t, map[string]string{"main.fy": input})
			found := false
			for _, diagnostic := range result.Diagnostics {
				found = found || diagnostic.Code == want
			}
			if !found || len(result.Unsupported) != 0 {
				t.Fatalf("%s resource=%v diagnostics=%+v unsupported=%+v", operator, resource, result.Diagnostics, result.Unsupported)
			}
		}
	}
}

func TestAnalyzeShortCircuitPreservesHandledAndReadOnlyRoots(t *testing.T) {
	t.Parallel()
	for _, operator := range []string{"&&", "||"} {
		for _, body := range []string{
			"let bool seen=r.is_ok;return gate " + operator + " seen;",
			"let bool seen=gate " + operator + " r.is_ok;discard r;return seen;",
		} {
			result := analyzeText(t, map[string]string{"main.fy": "module main;fn bool test(bool gate,Result<i32,i32> r){" + body + "}"})
			assertCleanTypeResult(t, result)
		}
		result := analyzeText(t, map[string]string{"main.fy": "module main;resource R{i32 n;cleanup{}}fn bool test(bool gate,R r){return gate " + operator + " (r.n==0);}"})
		assertCleanTypeResult(t, result)
	}
}
