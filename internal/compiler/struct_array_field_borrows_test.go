package compiler_test

import (
	"os"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/mircodec"
	"github.com/0disoft/fairyc/internal/semantic"
)

func TestStructArrayFieldBorrowRoundTrip(t *testing.T) {
	source, err := os.ReadFile("testdata/struct_array_field_borrows.fy")
	if err != nil {
		t.Fatal(err)
	}
	checked, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", string(source))})
	if err != nil || !checked.Verified {
		t.Fatalf("check: %v %+v %+v", err, checked.Semantic.Diagnostics, checked.Semantic.Unsupported)
	}
	data, err := mircodec.Encode(checked.MIR, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	program, err := mircodec.Decode(data, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	for name, p := range map[string]mir.Program{"fresh": checked.MIR, "decoded": program} {
		t.Run(name, func(t *testing.T) {
			out, err := interpreter.Execute(p, "main::main")
			if err != nil || out.Kind != interpreter.OutcomeReturn || out.Value != 0 {
				t.Fatalf("execute: %+v %v", out, err)
			}
		})
	}
	for _, profile := range []cbackend.Profile{cbackend.Stage0WindowsMSVCProfile(), cbackend.Stage0Profile()} {
		if _, err := cbackend.Emit(program, "main::main", profile); err != nil {
			t.Fatal(err)
		}
	}
	for _, tc := range []struct {
		name   string
		kind   mir.OperationKind
		mutate func(*mir.Operation)
	}{
		{"missing-bounds", mir.OperationViewIndexCheck, func(op *mir.Operation) { op.Kind = mir.OperationUse; op.Symbol = hir.SymbolID(op.Operands[0].Local) }},
		{"wrong-array", mir.OperationArrayBorrowView, func(op *mir.Operation) { op.OwnerType = hir.FixedArray(hir.TypeI32, 2) }},
		{"wrong-field", mir.OperationFieldBorrow, func(op *mir.Operation) { op.Field = "missing" }},
		{"mutable-laundering", mir.OperationFieldBorrow, func(op *mir.Operation) {
			element, _, _ := hir.BorrowElement(op.Type)
			op.Type = hir.MutableBorrow(element)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bad, err := mircodec.Decode(data, 1<<20)
			if err != nil {
				t.Fatal(err)
			}
			found := false
			for fi := range bad.Functions {
				for bi := range bad.Functions[fi].Blocks {
					for oi := range bad.Functions[fi].Blocks[bi].Operations {
						op := &bad.Functions[fi].Blocks[bi].Operations[oi]
						if !found && op.Kind == tc.kind {
							tc.mutate(op)
							found = true
						}
					}
				}
			}
			if !found || len(mir.Verify(bad)) == 0 {
				t.Fatal("forged array field borrow accepted or missing mutation target")
			}
			if _, err := mircodec.Encode(bad, 1<<20); err == nil {
				t.Fatal("forged array field borrow encoded")
			}
		})
	}
}

func TestStructArrayFieldBorrowIndexReturn(t *testing.T) {
	source := `module main;struct Pos{i64 x;}struct Point{Pos pos;}
fn void use(ref mut i64 x,i32 later){assert(false);}
fn i32 main(){var Point[1] a=[Point{.pos=Pos{.x=1}}];
use(ref mut a[{return 0;0}].pos.x,{assert(false);1});assert(false);return 1;}`
	checked, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", source)})
	if err != nil || !checked.Verified {
		t.Fatalf("check: %v %+v %+v", err, checked.Semantic.Diagnostics, checked.Semantic.Unsupported)
	}
	out, err := interpreter.Execute(checked.MIR, "main::main")
	if err != nil || out.Kind != interpreter.OutcomeReturn || out.Value != 0 {
		t.Fatalf("execute: %+v %v", out, err)
	}
	for _, profile := range []cbackend.Profile{cbackend.Stage0WindowsMSVCProfile(), cbackend.Stage0Profile()} {
		if _, err := cbackend.Emit(checked.MIR, "main::main", profile); err != nil {
			t.Fatal(err)
		}
	}
}
