package semantic_test

import (
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/semantic"
)

const linearCoinDeclaration = `@abilities(store, key)
resource Coin {
    i32 amount;
    cleanup {}
}
`

func TestAnalyzeRejectsImplicitDropWithoutDropAbility(t *testing.T) {
	t.Parallel()
	text := "module main;\n" + linearCoinDeclaration + `
fn void lose(Coin coin) {}
`
	result := analyzeText(t, map[string]string{"main.fy": text})
	if result.FailedPhase != semantic.PhaseFlow || len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "E1108" || len(result.Unsupported) != 0 {
		t.Fatalf("result = %+v", result)
	}
	if len(result.Diagnostics[0].Related) != 1 || result.Diagnostics[0].Related[0].Role != "ability-declaration" {
		t.Fatalf("related = %+v", result.Diagnostics[0].Related)
	}
	start := strings.Index(text, "@abilities(store, key)")
	assertSpan(t, result.Diagnostics[0].Related[0].Span, "main.fy", start, start+len("@abilities(store, key)"))
}

func TestAnalyzeAllowsWholeRootTransferWithoutDropAbility(t *testing.T) {
	t.Parallel()
	text := "module main;\n" + linearCoinDeclaration + `
fn Coin pass(Coin coin) { return coin; }
`
	result := analyzeText(t, map[string]string{"main.fy": text})
	assertCleanTypeResult(t, result)
	if len(result.HIR.Modules) != 1 || len(result.HIR.Modules[0].Resources) != 1 {
		t.Fatalf("HIR resources = %+v", result.HIR.Modules)
	}
	abilities := result.HIR.Modules[0].Resources[0].Abilities
	if len(abilities) != 2 || abilities[0] != hir.ResourceAbilityStore || abilities[1] != hir.ResourceAbilityKey {
		t.Fatalf("resource abilities = %v", abilities)
	}
}

func TestAnalyzeAllowsImplicitDropWhenExplicitlyDeclared(t *testing.T) {
	t.Parallel()
	text := `module main;
@abilities(drop, store, key)
resource Coin {
    i32 amount;
    cleanup {}
}
fn void dispose(Coin coin) {}
`
	result := analyzeText(t, map[string]string{"main.fy": text})
	assertCleanTypeResult(t, result)
}

func TestAnalyzePreservesLegacyImplicitDropDefault(t *testing.T) {
	t.Parallel()
	text := `module main;
resource Handle {
    i32 raw;
    cleanup {}
}
fn void dispose(Handle handle) {}
`
	result := analyzeText(t, map[string]string{"main.fy": text})
	assertCleanTypeResult(t, result)
	if len(result.HIR.Modules) != 1 || len(result.HIR.Modules[0].Resources) != 1 {
		t.Fatalf("HIR resources = %+v", result.HIR.Modules)
	}
	abilities := result.HIR.Modules[0].Resources[0].Abilities
	if len(abilities) != 1 || abilities[0] != hir.ResourceAbilityDrop {
		t.Fatalf("legacy resource abilities = %v", abilities)
	}
}

func TestAnalyzeRejectsImplicitDropThroughDroppableWrapper(t *testing.T) {
	t.Parallel()
	text := `module main;
@abilities(store, key)
resource Coin {
    i32 amount;
    cleanup {}
}
@abilities(drop)
resource Vault {
    Coin coin;
    cleanup {}
}
fn void lose(Vault vault) {}
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

func TestAnalyzeRejectsMalformedResourceAbilities(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name       string
		attributes string
	}{
		{name: "empty", attributes: "@abilities()"},
		{name: "unknown copy", attributes: "@abilities(copy)"},
		{name: "key without store", attributes: "@abilities(key)"},
		{name: "duplicate ability", attributes: "@abilities(store, store)"},
		{name: "duplicate declaration", attributes: "@abilities(drop)\n@abilities(store)"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			text := "module main;\n" + test.attributes + `
resource Coin {
    i32 amount;
    cleanup {}
}
fn void main() {}
`
			result := analyzeText(t, map[string]string{"main.fy": text})
			if result.FailedPhase != semantic.PhaseType || len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "E1107" || len(result.Unsupported) != 0 {
				t.Fatalf("result = %+v", result)
			}
		})
	}
}
