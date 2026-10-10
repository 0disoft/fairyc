package semantic_test

import (
	"github.com/0disoft/fairyc/internal/semantic"
	"strings"
	"testing"
)

func TestResourceTemporaryRequiresCleanupEffects(t *testing.T) {
	for _, body := range []string{"fn void f(){discard make();}", "fn void f(){make();}", "resource Outer{i32 n;cleanup{discard make();}}"} {
		text := `module main;@effects(console)resource R{i32 n;cleanup{discard 1;}}fn R make(){return R{.n=1};}` + body
		result := analyzeText(t, map[string]string{"main.fy": text})
		if result.FailedPhase != semantic.PhaseFlow || len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "E7102" || len(result.Unsupported) != 0 {
			t.Fatalf("body=%q result=%+v", body, result)
		}
		start := strings.LastIndex(text, "make()")
		assertSpan(t, result.Diagnostics[0].Primary, "main.fy", start, start+len("make()"))
		if len(result.Diagnostics[0].Related) != 1 || result.Diagnostics[0].Related[0].Role != "effect-declaration" {
			t.Fatalf("related=%+v", result.Diagnostics[0].Related)
		}
		start = strings.Index(text, "@effects(console)")
		assertSpan(t, result.Diagnostics[0].Related[0].Span, "main.fy", start, start+len("@effects(console)"))
	}
}

func TestResourceTemporaryCleanupAuthorityAndWholeMove(t *testing.T) {
	text := `module main;@effects(console)resource R{i32 n;cleanup{discard 1;}}fn R make(){return R{.n=1};}@effects(console)fn void f(){discard make();}fn R transfer(){return make();}`
	assertCleanTypeResult(t, analyzeText(t, map[string]string{"main.fy": text}))
}

func TestResourceLinearTemporaryInsideCleanupCannotBeDiscarded(t *testing.T) {
	text := `module main;@abilities(store)resource R{i32 n;cleanup{}}fn R make(){return R{.n=1};}resource Outer{i32 n;cleanup{discard make();}}`
	result := analyzeText(t, map[string]string{"main.fy": text})
	if result.FailedPhase != semantic.PhaseFlow || len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "E1108" || len(result.Unsupported) != 0 {
		t.Fatalf("result=%+v", result)
	}
	start := strings.LastIndex(text, "make()")
	assertSpan(t, result.Diagnostics[0].Primary, "main.fy", start, start+len("make()"))
}
