package compiler_test

import (
	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/mircodec"
	"strings"
	"testing"
)

func TestIndexedStructBorrowMIR(t *testing.T) {
	source := `module main;struct Token{i64 value;}resource Owner{ptr mut void data;usize length;cleanup{}}
 fn ref Token selected(ref Token[] items,usize index){return ref (items[index]);}
 fn ref mut Token selected_mut(ref mut Token[] items,usize index){return ref mut items[index];}
 fn i32 main(){var Owner owner=Owner{.data=null,.length=0};let ref mut Token[] items=unsafe{storage_borrow_mut<Token[]>(ref mut owner,owner.data,owner.length)};
 if(items.len>0){let ref Token shared=selected(items,0);assert(shared.value>=0);let ref mut Token item=selected_mut(items,0);item.value+=1;}return 0;}`
	analysis, e := analyzeStorageBorrow(t, source)
	if e != nil {
		t.Fatal(e)
	}
	checked, e := compiler.LowerAnalyzedStorageTransfers(analysis.Analysis)
	if e != nil || !checked.Verified {
		t.Fatalf("lower: %v", e)
	}
	encoded, e := mircodec.EncodeStorageTransfers(checked.MIR, 1<<20)
	if e != nil {
		t.Fatal(e)
	}
	program, e := mircodec.DecodeStorageTransfers(encoded, 1<<20)
	if e != nil {
		t.Fatal(e)
	}
	shared, mutable := 0, 0
	for _, fn := range program.Functions {
		for _, b := range fn.Blocks {
			for _, op := range b.Operations {
				if op.Kind == mir.OperationViewIndex {
					t.Fatal("whole-element borrow copied its referent")
				}
				if op.Kind == mir.OperationViewElementBorrow {
					_, mut, _ := hir.BorrowElement(op.Type)
					if mut {
						mutable++
					} else {
						shared++
					}
				}
			}
		}
	}
	if shared != 1 || mutable != 1 {
		t.Fatalf("shared=%d mutable=%d", shared, mutable)
	}
	for _, profile := range []cbackend.Profile{cbackend.Stage0WindowsMSVCProfile(), cbackend.Stage0Profile()} {
		out, e := cbackend.EmitStorageTransfers(program, hir.QualifiedModulePath(1, "main")+"::main", profile)
		if e != nil {
			t.Fatal(e)
		}
		if !strings.Contains(out.Source, "&((") {
			t.Fatal("missing stored address")
		}
	}
}
