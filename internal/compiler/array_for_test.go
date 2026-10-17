package compiler_test

import (
	"bytes"
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

func TestArrayForRoundTrip(t *testing.T) {
	src, e := os.ReadFile("testdata/array_for.fy")
	if e != nil {
		t.Fatal(e)
	}
	files := []semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", string(src))}
	checked, e := compiler.Check(files)
	if e != nil || !checked.Verified {
		t.Fatalf("check: %v %+v", e, checked.Semantic)
	}
	again, e := compiler.Check(files)
	if e != nil || !again.Verified {
		t.Fatalf("repeated source check: %v %+v %+v", e, again.Semantic.Diagnostics, again.Semantic.Unsupported)
	}
	data, e := mircodec.Encode(checked.MIR, 1<<20)
	if e != nil {
		t.Fatal(e)
	}
	againData, e := mircodec.Encode(again.MIR, 1<<20)
	if e != nil || !bytes.Equal(data, againData) {
		t.Fatalf("repeated analysis changed MIR: %v", e)
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
	types := map[hir.Type]int{}
	for _, fn := range p.Functions {
		for _, b := range fn.Blocks {
			for _, op := range b.Operations {
				if op.Kind == mir.OperationViewElementBorrow {
					element, mut, ok := hir.BorrowElement(op.Type)
					if !ok || !mut || !hir.MutableIndexedView(op.OwnerType) {
						t.Fatal("loop yielded value or shared ref")
					}
					types[element]++
				}
			}
		}
	}
	for _, typ := range []hir.Type{hir.TypeU8, hir.TypeI32, hir.TypeI64, hir.TypeU32, hir.TypeUSize} {
		if types[typ] == 0 {
			t.Fatalf("missing stored reference for %s", typ)
		}
	}
	for _, name := range []string{"bounds", "shared-receiver", "wrong-width", "array-owner"} {
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
						if name == "array-owner" {
							if op.Kind == mir.OperationArrayBorrowView {
								op.OwnerType = hir.ParseType("[bool; 1]")
								found = true
							}
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
						found = true
						if name == "shared-receiver" {
							element, _, _ := hir.BorrowElement(op.OwnerType)
							op.OwnerType = hir.SharedBorrow(element)
						} else {
							op.Type = hir.MutableBorrow(hir.TypeBool)
						}
					}
				}
			}
			if !found || len(mir.Verify(bad)) == 0 {
				t.Fatal("invalid loop ref verified")
			}
			if _, e := mircodec.Encode(bad, 1<<20); e == nil {
				t.Fatal("invalid loop ref crossed codec")
			}
			if _, e := cbackend.Emit(bad, "main::main", cbackend.Stage0WindowsMSVCProfile()); e == nil {
				t.Fatal("invalid loop ref emitted")
			}
		})
	}
}
