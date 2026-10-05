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

func TestNominalArrayFieldViewRoundTrip(t *testing.T) {
	source, e := os.ReadFile("testdata/nominal_array_field_views.fy")
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
				t.Fatal("invalid nominal field view accepted")
			}
			if _, e := mircodec.Encode(bad, 1<<20); e == nil {
				t.Fatal("invalid nominal field view encoded")
			}
		})
	}
}

// Direct accesses select the original field address; whole-array copies belong
// only to explicitly written value reads, not descriptor or element lowering.
func TestNominalArrayFieldViewStorageIdentity(t *testing.T) {
	source := `module main;struct T{i64 x;}struct P{T[2] data;}fn void edit(ref mut P p){p.data[0].x=7;let ref T[] v=ref p.data;assert(v[0].x==7);for(let ref T t in p.data){assert(t.x>0);}}`
	r, e := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", source)})
	if e != nil || !r.Verified {
		t.Fatalf("check: %v %+v %+v", e, r.Semantic.Diagnostics, r.Semantic.Unsupported)
	}
	views, elements := 0, 0
	for _, f := range r.MIR.Functions {
		for _, b := range f.Blocks {
			for _, op := range b.Operations {
				if op.Kind == mir.OperationFieldRead {
					if _, _, array := hir.FixedArrayParts(op.Type); array {
						t.Fatal("field array copied while selecting storage")
					}
				}
				if op.Kind == mir.OperationArrayIndex {
					t.Fatal("direct field element used copied array path")
				}
				if op.Kind == mir.OperationArrayBorrowView {
					if _, _, borrowed := hir.BorrowElement(op.OwnerType); !borrowed {
						t.Fatal("field descriptor lacks original storage address")
					}
					views++
				}
				if op.Kind == mir.OperationViewElementBorrow {
					elements++
				}
			}
		}
	}
	if views == 0 || elements == 0 {
		t.Fatal("storage address/view element path not exercised")
	}
}
