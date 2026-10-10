package semantic_test

import (
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/semantic"
)

func TestResourceMatchPayloadRequiresDropOnLoopExit(t *testing.T) {
	for _, exit := range []string{"break", "continue"} {
		t.Run(exit, func(t *testing.T) {
			text := `module main;
@abilities(store) resource R { i32 n; cleanup {} }
fn R make() { return R { .n = 1 }; }
fn void f(bool c) {
    while (c) {
        let Option<R> item = .some(make());
        switch (item) {
            case .some(value): { ` + exit + `; }
            case .none: { break; }
        };
    }
}`
			result := analyzeText(t, map[string]string{"main.fy": text})
			if result.FailedPhase != semantic.PhaseFlow || len(result.Diagnostics) != 1 || len(result.Unsupported) != 0 || result.Diagnostics[0].Code != "E1108" {
				t.Fatalf("result = %+v", result)
			}
			start := strings.Index(text, exit+";")
			assertSpan(t, result.Diagnostics[0].Primary, "main.fy", start, start+len(exit)+1)
			if len(result.Diagnostics[0].Related) != 1 || result.Diagnostics[0].Related[0].Role != "ability-declaration" {
				t.Fatalf("related = %+v", result.Diagnostics[0].Related)
			}
			start = strings.Index(text, "@abilities(store)")
			assertSpan(t, result.Diagnostics[0].Related[0].Span, "main.fy", start, start+len("@abilities(store)"))
		})
	}
}

func TestResourceMatchPayloadSurvivesNestedLoopExit(t *testing.T) {
	text := `module main;
@abilities(store) resource R { i32 n; cleanup {} }
fn R make() { return R { .n = 1 }; }
fn R f(bool c) {
    while (c) {
        let Option<R> item = .some(make());
        switch (item) {
            case .some(value): { while (c) { break; } return value; }
            case .none: { break; }
        };
    }
    return make();
}`
	result := analyzeText(t, map[string]string{"main.fy": text})
	if result.FailedPhase != "" || len(result.Diagnostics) != 0 || len(result.Unsupported) != 0 {
		t.Fatalf("result = %+v", result)
	}
}
