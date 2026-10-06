package semantic_test

import (
	"strings"
	"testing"
)

func TestAnalyzeResultCaptureOnExitedLoopEdges(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ name, body string }{
		{"break", "break;"},
		{"continue", "continue;"},
		{"branch-break", "if(b){break;}discard inner;"},
		{"branch-continue", "if(b){continue;}discard inner;"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := "module main;fn i32 f(bool b){while(b){let Option<Result<i32,bool>> r=.some(.ok(1));switch(r){case .some(inner):{" + tc.body + "}case .none:{break;}};}return 0;}"
			result := analyzeText(t, map[string]string{"main.fy": source})
			if len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "E4101" {
				t.Fatalf("diagnostics = %+v, unsupported = %+v", result.Diagnostics, result.Unsupported)
			}
			start := strings.Index(source, "inner")
			assertSpan(t, result.Diagnostics[0].Primary, "main.fy", start, start+len("inner"))
		})
	}
}

func TestAnalyzeResultCapturePreservesInnerLoopAndHandledEdges(t *testing.T) {
	t.Parallel()
	for _, body := range []string{
		"discard inner;break;",
		"discard inner;continue;",
		"while(b){break;}discard inner;break;",
		"while(b){continue;}discard inner;break;",
		"if(b){discard inner;break;}discard inner;continue;",
	} {
		source := "module main;fn i32 f(bool b){while(b){let Option<Result<i32,bool>> r=.some(.ok(1));switch(r){case .some(inner):{" + body + "}case .none:{break;}};}return 0;}"
		result := analyzeText(t, map[string]string{"main.fy": source})
		assertCleanTypeResult(t, result)
	}
}
