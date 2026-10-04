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

func TestArrayElementBorrowRoundTrip(t *testing.T) {
	src, e := os.ReadFile("testdata/array_element_borrow.fy")
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
	outcome, e := interpreter.Execute(p, "main::main")
	if e != nil || outcome.Kind != interpreter.OutcomeReturn || outcome.Value != 0 {
		t.Fatalf("execute: %+v %v", outcome, e)
	}
	for _, profile := range []cbackend.Profile{cbackend.Stage0WindowsMSVCProfile(), cbackend.Stage0Profile()} {
		if _, e := cbackend.Emit(p, "main::main", profile); e != nil {
			t.Fatal(e)
		}
	}
	counts := map[hir.Type]int{}
	for _, fn := range p.Functions {
		for _, b := range fn.Blocks {
			for _, op := range b.Operations {
				if op.Kind == mir.OperationArrayBorrowView {
					element, _, _ := hir.FixedArrayParts(op.OwnerType)
					counts[element]++
				}
			}
		}
	}
	for _, typ := range []hir.Type{hir.TypeU8, hir.TypeI32, hir.TypeI64, hir.TypeU32, hir.TypeUSize} {
		if counts[typ] == 0 {
			t.Fatalf("no original array descriptor for %s", typ)
		}
	}
	for _, name := range []string{"bounds", "array-owner", "array-width", "shared-to-mut", "element-type"} {
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
						if found {
							continue
						}
						op := &b.Operations[oi]
						if name == "bounds" && op.Kind == mir.OperationViewIndexCheck {
							b.Operations = append(b.Operations[:oi], b.Operations[oi+1:]...)
							found = true
							break
						}
						if name == "array-owner" && op.Kind == mir.OperationArrayBorrowView && op.Type == hir.MutableBorrow(hir.TypeByteSlice) {
							op.Operands[0].Local = op.DestinationLocal
							found = true
						}
						if name == "array-width" && op.Kind == mir.OperationArrayBorrowView && op.Type == hir.MutableBorrow(hir.TypeByteSlice) {
							op.OwnerType = hir.FixedArray(hir.TypeI64, 3)
							found = true
						}
						if op.Kind == mir.OperationViewElementBorrow && name == "shared-to-mut" && !hir.MutableIndexedView(op.OwnerType) {
							element, _, _ := hir.BorrowElement(op.OwnerType)
							op.OwnerType = hir.MutableBorrow(element)
							item, _, _ := hir.BorrowElement(op.Type)
							op.Type = hir.MutableBorrow(item)
							found = true
						}
						if op.Kind == mir.OperationViewElementBorrow && name == "element-type" {
							op.Type = hir.MutableBorrow(hir.TypeBool)
							found = true
						}
					}
				}
			}
			if !found || len(mir.Verify(bad)) == 0 {
				t.Fatal("invalid array element reference verified")
			}
			if _, e := mircodec.Encode(bad, 1<<20); e == nil {
				t.Fatal("invalid array reference crossed codec")
			}
			for _, profile := range []cbackend.Profile{cbackend.Stage0WindowsMSVCProfile(), cbackend.Stage0Profile()} {
				if _, e := cbackend.Emit(bad, "main::main", profile); e == nil {
					t.Fatal("invalid array reference emitted")
				}
			}
		})
	}
}
