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

func TestIntegerElementBorrowRoundTrip(t *testing.T) {
	src, e := os.ReadFile("testdata/integer_element_borrow.fy")
	if e != nil {
		t.Fatal(e)
	}
	c, e := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", string(src))})
	if e != nil || !c.Verified {
		t.Fatalf("check: %v %+v", e, c.Semantic)
	}
	data, e := mircodec.Encode(c.MIR, 1<<20)
	if e != nil {
		t.Fatal(e)
	}
	p, e := mircodec.Decode(data, 1<<20)
	if e != nil {
		t.Fatal(e)
	}
	out, e := interpreter.Execute(p, "main::main")
	if e != nil || out.Kind != interpreter.OutcomeReturn || out.Value != 0 {
		t.Fatalf("run: %+v %v", out, e)
	}
	for _, profile := range []cbackend.Profile{cbackend.Stage0WindowsMSVCProfile(), cbackend.Stage0Profile()} {
		if _, e := cbackend.Emit(p, "main::main", profile); e != nil {
			t.Fatal(e)
		}
	}
	borrows := 0
	for _, fn := range p.Functions {
		for _, b := range fn.Blocks {
			for _, op := range b.Operations {
				if op.Kind == mir.OperationViewElementBorrow {
					borrows++
				}
			}
		}
	}
	if borrows < 20 {
		t.Fatalf("borrow path count=%d", borrows)
	}
	for _, name := range []string{"bounds", "shared-to-mut", "wrong-element", "wrong-receiver", "wrong-index", "metadata"} {
		t.Run(name, func(t *testing.T) {
			bad, e := mircodec.Decode(data, 1<<20)
			if e != nil {
				t.Fatal(e)
			}
			found := false
			for fi := range bad.Functions {
				for bi := range bad.Functions[fi].Blocks {
					b := &bad.Functions[fi].Blocks[bi]
					for oi := range b.Operations {
						op := &b.Operations[oi]
						if found {
							continue
						}
						if name == "bounds" && op.Kind == mir.OperationViewIndexCheck {
							b.Operations = append(b.Operations[:oi], b.Operations[oi+1:]...)
							found = true
							break
						}
						if name == "bounds" || op.Kind != mir.OperationViewElementBorrow {
							continue
						}
						if name == "shared-to-mut" && hir.MutableIndexedView(op.OwnerType) {
							continue
						}
						found = true
						switch name {
						case "shared-to-mut":
							slice, _, _ := hir.BorrowElement(op.OwnerType)
							element, _ := hir.IndexedViewElement(op.OwnerType)
							op.OwnerType = hir.MutableBorrow(slice)
							op.Type = hir.MutableBorrow(element)
						case "wrong-element":
							op.Type = hir.SharedBorrow(hir.TypeI64)
						case "wrong-receiver":
							receiver := op.Operands[0].Local
							for bj := range bad.Functions[fi].Blocks {
								for oj := range bad.Functions[fi].Blocks[bj].Operations {
									producer := &bad.Functions[fi].Blocks[bj].Operations[oj]
									if producer.Kind == mir.OperationAssign && producer.DestinationLocal == receiver {
										producer.Operands = []mir.Operand{{Kind: mir.OperandUSize, U64: 0}}
									}
								}
							}
						case "wrong-index":
							op.Operands[1] = mir.Operand{Kind: mir.OperandLocal, Local: op.Operands[0].Local, Type: hir.TypeUSize}
						case "metadata":
							op.MovePayload = true
						}
					}
				}
			}
			if !found || len(mir.Verify(bad)) == 0 {
				t.Fatal("invalid element ref verified")
			}
			if _, e := mircodec.Encode(bad, 1<<20); e == nil {
				t.Fatal("invalid element ref crossed codec")
			}
			if _, e := cbackend.Emit(bad, "main::main", cbackend.Stage0WindowsMSVCProfile()); e == nil {
				t.Fatal("invalid element ref emitted")
			}
		})
	}
}
