package semantic_test

import (
	"github.com/0disoft/fairyc/internal/hir"
	"testing"

	"github.com/0disoft/fairyc/internal/semantic"
)

func TestAnalyzeBurnsDeclarationOwnedNoDropResource(t *testing.T) {
	t.Parallel()
	result := analyzeText(t, map[string]string{"asset.fy": `module asset;
@abilities(store, key)
resource Coin { i32 amount; cleanup {} }
fn i32 burn_coin(Coin coin) {
    burn coin;
    return 42;
}
`})
	assertCleanTypeResult(t, result)
	if len(result.HIR.Modules) != 1 || len(result.HIR.Modules[0].Functions) != 1 ||
		len(result.HIR.Modules[0].Functions[0].Body.Statements) != 2 ||
		result.HIR.Modules[0].Functions[0].Body.Statements[0].Kind != hir.StatementBurn {
		t.Fatalf("burn HIR = %+v", result.HIR)
	}
}

func TestAnalyzeRejectsUnauthorizedResourceBurns(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		files map[string]string
	}{
		{
			name: "droppable resource",
			files: map[string]string{"main.fy": `module main;
resource Handle { i32 raw; cleanup {} }
fn void burn_handle(Handle handle) { burn handle; }
`},
		},
		{
			name: "temporary",
			files: map[string]string{"main.fy": `module main;
@abilities(store, key)
resource Coin { i32 amount; cleanup {} }
fn Coin mint() { return Coin { .amount = 1 }; }
fn void burn_coin() { burn mint(); }
`},
		},
		{
			name: "projection",
			files: map[string]string{"main.fy": `module main;
@abilities(store, key)
resource Coin { i32 amount; cleanup {} }
fn void burn_amount(Coin coin) { burn coin.amount; }
`},
		},
		{
			name: "resource wrapper",
			files: map[string]string{"main.fy": `module main;
@abilities(store, key)
resource Coin { i32 amount; cleanup {} }
fn void burn_optional(Option<Coin> coin) { burn coin; }
`},
		},
		{
			name: "resource bearing field",
			files: map[string]string{"main.fy": `module main;
@abilities(store, key)
resource Coin { i32 amount; cleanup {} }
@abilities(store, key)
resource Vault { Coin coin; cleanup {} }
fn void burn_vault(Vault vault) { burn vault; }
`},
		},
		{
			name: "foreign declaration module",
			files: map[string]string{
				"asset.fy": `module asset;
@abilities(store, key)
pub resource Coin { i32 amount; cleanup {} }
`,
				"main.fy": `module main;
import asset as asset;
fn void burn_coin(asset::Coin coin) { burn coin; }
`,
			},
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			result := analyzeText(t, test.files)
			if result.FailedPhase != semantic.PhaseType || len(result.Diagnostics) != 1 ||
				result.Diagnostics[0].Code != "E1109" || len(result.Unsupported) != 0 {
				t.Fatalf("result = %+v", result)
			}
		})
	}
}
