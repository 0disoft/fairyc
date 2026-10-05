package compiler_test

import (
	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/mircodec"
	"github.com/0disoft/fairyc/internal/semantic"
	"os"
	"testing"
)

func TestArrayFieldViewRoundTrip(t *testing.T) {
	source, e := os.ReadFile("testdata/array_field_views.fy")
	if e != nil {
		t.Fatal(e)
	}
	r, e := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", string(source))})
	if e != nil || !r.Verified {
		t.Fatalf("check: %v %+v %+v", e, r.Semantic.Diagnostics, r.Semantic.Unsupported)
	}
	data, e := mircodec.Encode(r.MIR, 1<<20)
	if e != nil {
		t.Fatal(e)
	}
	p, e := mircodec.Decode(data, 1<<20)
	if e != nil {
		t.Fatal(e)
	}
	out, e := interpreter.Execute(p, "main::main")
	if e != nil || out.Kind != interpreter.OutcomeReturn || out.Value != 0 {
		t.Fatalf("execute: %+v %v", out, e)
	}
	for _, profile := range []cbackend.Profile{cbackend.Stage0WindowsMSVCProfile(), cbackend.Stage0Profile()} {
		if _, e := cbackend.Emit(p, "main::main", profile); e != nil {
			t.Fatal(e)
		}
	}
	for _, tc := range []struct {
		name   string
		kind   mir.OperationKind
		mutate func(*mir.Operation)
	}{
		{"lost-bounds", mir.OperationViewSliceCheck, func(op *mir.Operation) { op.Kind = mir.OperationUse; op.Symbol = hir.SymbolID(op.Operands[0].Local) }},
		{"wrong-owner", mir.OperationArrayBorrowView, func(op *mir.Operation) { op.OwnerType = hir.MutableBorrow(hir.FixedArray(hir.TypeI32, 4)) }},
		{"mutable-laundering", mir.OperationArrayBorrowView, func(op *mir.Operation) {
			element, _, _ := hir.BorrowElement(op.Type)
			op.Type = hir.MutableBorrow(element)
			op.Operator = "&mut"
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bad, e := mircodec.Decode(data, 1<<20)
			if e != nil {
				t.Fatal(e)
			}
			found := false
			for fi := range bad.Functions {
				for bi := range bad.Functions[fi].Blocks {
					for oi := range bad.Functions[fi].Blocks[bi].Operations {
						op := &bad.Functions[fi].Blocks[bi].Operations[oi]
						if tc.name == "mutable-laundering" {
							_, mutable, _ := hir.BorrowElement(op.Type)
							if mutable {
								continue
							}
						}
						if !found && op.Kind == tc.kind {
							tc.mutate(op)
							found = true
						}
					}
				}
			}
			if !found || len(mir.Verify(bad)) == 0 {
				t.Fatal("invalid field view accepted")
			}
			if _, e := mircodec.Encode(bad, 1<<20); e == nil {
				t.Fatal("invalid field view encoded")
			}
		})
	}
}
func TestArrayFieldViewIndexReturn(t *testing.T) {
	source := `module main;struct P{u8[2] bytes;}fn i32 main(){var P p=P{.bytes=[1,2]};p.bytes[{return 0;0}]={assert(false);3};assert(false);return 1;}`
	r, e := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", source)})
	if e != nil || !r.Verified {
		t.Fatalf("check: %v %+v %+v", e, r.Semantic.Diagnostics, r.Semantic.Unsupported)
	}
	out, e := interpreter.Execute(r.MIR, "main::main")
	if e != nil || out.Kind != interpreter.OutcomeReturn || out.Value != 0 {
		t.Fatalf("execute: %+v %v", out, e)
	}
}
