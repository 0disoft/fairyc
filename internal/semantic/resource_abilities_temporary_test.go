package semantic_test

import (
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/semantic"
)

func TestAnalyzeRejectsDiscardedLinearTemporary(t *testing.T) {
	t.Parallel()
	text := `module main;
@abilities(store, key)
resource Coin {
    i32 amount;
    cleanup {}
}
fn Coin mint() { return Coin { .amount = 1 }; }
fn void lose() { discard mint(); }
`
	result := analyzeText(t, map[string]string{"main.fy": text})
	if result.FailedPhase != semantic.PhaseFlow || len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "E1108" || len(result.Unsupported) != 0 {
		t.Fatalf("result = %+v", result)
	}
	start := strings.Index(text, "@abilities(store, key)")
	if len(result.Diagnostics[0].Related) != 1 || result.Diagnostics[0].Related[0].Role != "ability-declaration" {
		t.Fatalf("related = %+v", result.Diagnostics[0].Related)
	}
	assertSpan(t, result.Diagnostics[0].Related[0].Span, "main.fy", start, start+len("@abilities(store, key)"))
}
