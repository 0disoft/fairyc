package semantic_test

import (
	"github.com/0disoft/fairyc/internal/semantic"
	"strings"
	"testing"
)

func TestResourceCleanupLocalsRequireDropAuthorityOnFallthrough(t *testing.T) {
	for _, test := range []struct{ contract, body, code, role string }{
		{"@effects(console)", "discard 1;", "E7102", "effect-declaration"},
		{"@abilities(store)", "", "E1108", "ability-declaration"},
	} {
		text := `module main;` + test.contract + `resource R{i32 n;cleanup{` + test.body + `}}fn R make(){return R{.n=1};}resource Outer{i32 n;cleanup{let R local=make();}}`
		result := analyzeText(t, map[string]string{"main.fy": text})
		if result.FailedPhase != semantic.PhaseFlow || len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != test.code || len(result.Unsupported) != 0 {
			t.Fatalf("result=%+v", result)
		}
		start := strings.Index(text, "{let R local=")
		assertSpan(t, result.Diagnostics[0].Primary, "main.fy", start, len(text)-1)
		if len(result.Diagnostics[0].Related) != 1 || result.Diagnostics[0].Related[0].Role != test.role {
			t.Fatalf("related=%+v", result.Diagnostics[0].Related)
		}
		assertSpan(t, result.Diagnostics[0].Related[0].Span, "main.fy", len("module main;"), len("module main;")+len(test.contract))
	}
}

func TestResourceCleanupLocalsWithAuthorityRemainAdmitted(t *testing.T) {
	text := `module main;@effects(console)resource R{i32 n;cleanup{discard 1;}}fn R make(){return R{.n=1};}@effects(console)resource Outer{i32 n;cleanup{let R local=make();}}`
	assertCleanTypeResult(t, analyzeText(t, map[string]string{"main.fy": text}))
}
