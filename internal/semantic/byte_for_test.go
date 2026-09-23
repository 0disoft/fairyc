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

func TestByteViewForRejectsNonAdmittedSourceShapes(t *testing.T) {
	for _, sample := range []struct {
		name string
		text string
		want string
	}{
		{
			name: "wrong item type",
			text: "module main; fn void show(ref u8[] view) { for (let i32 byte in view) {} }",
			want: "byte-view iteration requires an exact u8 item binding",
		},
		{
			name: "mutable view",
			text: "module main; fn void show(ref mut u8[] view) { for (let u8 byte in view) {} }",
			want: "byte-view iteration requires a named immutable ref u8[] view",
		},
		{
			name: "nonpath view",
			text: "module main; fn void show(ref u8[] view) { for (let u8 byte in view[0..1]) {} }",
			want: "byte-view iteration requires a named immutable ref u8[] view",
		},
		{
			name: "mutable local view binding",
			text: "module main; fn void show(ref u8[] view) { var ref u8[] local = view; for (let u8 byte in local) {} }",
			want: "byte-view iteration requires a named immutable ref u8[] view",
		},
	} {
		t.Run(sample.name, func(t *testing.T) {
			result := analyzeText(t, map[string]string{"main.fy": sample.text})
			if len(result.Diagnostics) != 0 || len(result.Unsupported) != 1 || result.Unsupported[0].Feature != sample.want {
				t.Fatalf("diagnostics = %+v, unsupported = %+v", result.Diagnostics, result.Unsupported)
			}
		})
	}
}

func TestByteViewForAcceptsImmutableLocalForPreparedAnalysis(t *testing.T) {
	result := analyzeText(t, map[string]string{
		"main.fy": "module main; fn void show(ref u8[] view) { let ref u8[] local = view; for (let u8 byte in local) { assert(byte == 7); } }",
	})
	if len(result.Diagnostics) != 0 || len(result.Unsupported) != 1 || result.Unsupported[0].Feature != "byte-view iteration awaits semantic and MIR lowering" {
		t.Fatalf("diagnostics = %+v, unsupported = %+v", result.Diagnostics, result.Unsupported)
	}
}

func TestByteViewForBindingCannotEscapeLoop(t *testing.T) {
	result := analyzeText(t, map[string]string{
		"main.fy": "module main; fn u8 show(ref u8[] view) { for (let u8 byte in view) {} return byte; }",
	})
	if len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "E0101" {
		t.Fatalf("diagnostics = %+v", result.Diagnostics)
	}
}
