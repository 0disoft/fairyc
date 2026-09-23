package semantic_test

import (
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
)

func TestByteViewForFailsClosedUntilSemanticLowering(t *testing.T) {
	result := analyzeText(t, map[string]string{
		"main.fy": "module main; fn void show(ref u8[] view) { for (let u8 byte in view) { if (byte == 7) { continue; } break; } }",
	})
	if len(result.Diagnostics) != 0 || len(result.Unsupported) != 1 || result.Unsupported[0].Feature != "byte-view iteration awaits semantic and MIR lowering" {
		t.Fatalf("diagnostics = %+v, unsupported = %+v", result.Diagnostics, result.Unsupported)
	}
	if len(result.HIR.Modules) != 1 || len(result.HIR.Modules[0].Functions) != 1 {
		t.Fatalf("prepared HIR = %+v", result.HIR)
	}
	statements := result.HIR.Modules[0].Functions[0].Body.Statements
	if len(statements) != 1 || statements[0].Expression == nil || statements[0].Expression.Block == nil {
		t.Fatalf("prepared statement = %+v", statements)
	}
	prepared := statements[0].Expression.Block.Statements
	if len(prepared) != 2 || prepared[0].Expression == nil || prepared[0].Expression.Type != hir.TypeUSize || prepared[1].For == nil || prepared[1].For.Post == nil {
		t.Fatalf("prepared length and loop = %+v", prepared)
	}
}
