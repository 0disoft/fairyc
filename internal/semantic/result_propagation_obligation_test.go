package semantic_test

import "testing"

func TestAnalyzeResultPropagationChecksOtherPendingObligations(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		body string
	}{
		{"local", "let Result<i32,bool> other=.ok(1);let n=r?;discard other;return .ok(n);"},
		{"branch", "let Result<i32,bool> other=.ok(1);let n=if(b){r?}else{discard r;0};discard other;return .ok(n);"},
		{"loop", "let Result<i32,bool> other=.ok(1);var i32 n=0;while(b){n=r?;break;}discard r;discard other;return .ok(n);"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := analyzeText(t, map[string]string{"main.fy": "module main;fn Result<i32,bool> f(Result<i32,bool> r,bool b){" + tc.body + "}"})
			if len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "E4101" {
				t.Fatalf("diagnostics = %+v, unsupported = %+v", result.Diagnostics, result.Unsupported)
			}
		})
	}
}

func TestAnalyzeResultPropagationPreservesHandledAndUnreachablePaths(t *testing.T) {
	t.Parallel()
	for _, body := range []string{
		"let Result<i32,bool> other=.ok(1);discard other;let n=r?;return .ok(n);",
		"let n=r?;return .ok(n);",
		"discard r;let n={return .err(true);r}?;return .ok(n);",
	} {
		result := analyzeText(t, map[string]string{"main.fy": "module main;fn Result<i32,bool> f(Result<i32,bool> r){" + body + "}"})
		assertCleanTypeResult(t, result)
	}
}
