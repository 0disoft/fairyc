package semantic_test

import "testing"

func TestAnalyzeResourceLoopConditionChecksOnlyReentryEdges(t *testing.T) {
	t.Parallel()
	for _, body := range []string{"", "continue;"} {
		result := analyzeText(t, map[string]string{"main.fy": "module main;resource R{cleanup{}}fn bool inspect(R r){return true;}fn void f(R r){while(inspect(r)){" + body + "}}"})
		found := false
		for _, diagnostic := range result.Diagnostics {
			found = found || diagnostic.Code == "E1101"
		}
		if !found || len(result.Unsupported) != 0 {
			t.Fatalf("body=%q diagnostics=%+v unsupported=%+v", body, result.Diagnostics, result.Unsupported)
		}
	}
	for _, body := range []string{"break;", "return;"} {
		result := analyzeText(t, map[string]string{"main.fy": "module main;resource R{cleanup{}}fn bool inspect(R r){return true;}fn void f(R r){while(inspect(r)){" + body + "}}"})
		assertCleanTypeResult(t, result)
	}
	result := analyzeText(t, map[string]string{"main.fy": "module main;resource R{cleanup{}}fn void take(R r){}fn R f(bool b,R input){var R r=input;take(r);while({r=R{};b}){let R next=r;r=next;continue;}return r;}"})
	assertCleanTypeResult(t, result)
}
