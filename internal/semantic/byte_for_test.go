package semantic_test

import (
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/semantic"
)

func TestByteViewForLowersToVerifiedMIR(t *testing.T) {
	result := analyzeText(t, map[string]string{
		"main.fy": "module main; fn void show(ref u8[] view) { for (let u8 byte in view) { if (byte == 7) { continue; } break; } }",
	})
	if len(result.Diagnostics) != 0 || len(result.Unsupported) != 0 || result.CompletedThrough != semantic.PhaseFlow {
		t.Fatalf("diagnostics = %+v, unsupported = %+v", result.Diagnostics, result.Unsupported)
	}
	if len(result.HIR.Modules) != 1 || len(result.HIR.Modules[0].Functions) != 1 {
		t.Fatalf("prepared HIR = %+v", result.HIR)
	}
	statements := result.HIR.Modules[0].Functions[0].Body.Statements
	if len(statements) != 1 || statements[0].Kind != "expression-statement" || statements[0].Expression == nil || statements[0].Expression.Block == nil {
		t.Fatalf("prepared statement = %+v", statements)
	}
	prepared := statements[0].Expression.Block.Statements
	if len(prepared) != 2 || prepared[0].Expression == nil || prepared[0].Expression.Type != hir.TypeUSize || prepared[1].For == nil || prepared[1].For.Post == nil {
		t.Fatalf("prepared length and loop = %+v", prepared)
	}
	program, err := mir.Lower(result.HIR)
	if err != nil {
		t.Fatalf("lower prepared HIR: %v", err)
	}
	if failures := mir.Verify(program); len(failures) != 0 {
		t.Fatalf("verify prepared MIR: %+v", failures)
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
			want: "i32-view iteration requires a named immutable ref i32[] or ref mut i32[] view",
		},
		{
			name: "wrong view element",
			text: "module main; fn void show(ref mut i64[] view) { for (let u8 byte in view) {} }",
			want: "byte-view iteration requires a named immutable ref u8[] or ref mut u8[] view",
		},
		{
			name: "nonpath view",
			text: "module main; fn void show(ref u8[] view) { for (let u8 byte in view[0..1]) {} }",
			want: "byte-view iteration requires a named immutable ref u8[] or ref mut u8[] view",
		},
		{
			name: "mutable local view binding",
			text: "module main; fn void show(ref u8[] view) { var ref u8[] local = view; for (let u8 byte in local) {} }",
			want: "byte-view iteration requires a named immutable ref u8[] or ref mut u8[] view",
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

func TestPreparedVectorForDoesNotAdmitSourceBeforeSemanticGate(t *testing.T) {
	result := analyzeText(t, map[string]string{
		"main.fy":    "module main; import std::vec as items; fn void show(ref u8[] view) { for (let ref i32 item in view) {} }",
		"std/vec.fy": "module std::vec; pub struct Placeholder { i32 value; }",
	})
	if len(result.Diagnostics) != 0 || len(result.Unsupported) != 1 ||
		result.Unsupported[0].Feature != "view iteration requires an exact integer or scalar-field struct item binding" {
		t.Fatalf("candidate source admission changed: diagnostics = %+v, unsupported = %+v", result.Diagnostics, result.Unsupported)
	}
}

func TestByteViewForAcceptsImmutableLocal(t *testing.T) {
	result := analyzeText(t, map[string]string{
		"main.fy": "module main; fn void show(ref u8[] view) { let ref u8[] local = view; for (let u8 byte in local) { assert(byte == 7); } }",
	})
	if len(result.Diagnostics) != 0 || len(result.Unsupported) != 0 || result.CompletedThrough != semantic.PhaseFlow {
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

func TestByteViewForExecutesVerifiedMIR(t *testing.T) {
	for _, sample := range []struct {
		name string
		view string
		loop string
		want int32
	}{
		{name: "empty", view: `b"abc"[0..0]`, loop: "for (let u8 byte in view) { total += byte as i32; }", want: 0},
		{name: "ordered", view: `b"abc"[..]`, loop: "for (let u8 byte in view) { total += byte as i32; }", want: 294},
		{name: "continue and break", view: `b"abc"[..]`, loop: "for (let u8 byte in view) { if (byte == 98) { continue; } total += byte as i32; if (byte == 99) { break; } }", want: 196},
	} {
		t.Run(sample.name, func(t *testing.T) {
			text := "module main; fn i32 main() { let ref u8[] view = " + sample.view + "; var i32 total = 0; " + sample.loop + " return total; }"
			result := analyzeText(t, map[string]string{"main.fy": text})
			if len(result.Diagnostics) != 0 || len(result.Unsupported) != 0 || result.CompletedThrough != semantic.PhaseFlow {
				t.Fatalf("public boundary: diagnostics = %+v, unsupported = %+v", result.Diagnostics, result.Unsupported)
			}
			program, err := mir.Lower(result.HIR)
			if err != nil {
				t.Fatalf("lower prepared HIR: %v", err)
			}
			outcome, err := interpreter.Execute(program, "main::main")
			if err != nil || outcome.Kind != interpreter.OutcomeReturn || outcome.Value != sample.want {
				t.Fatalf("execute prepared MIR: outcome = %+v, error = %v", outcome, err)
			}
		})
	}
}
