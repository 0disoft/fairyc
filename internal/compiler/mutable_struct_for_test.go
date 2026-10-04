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

func TestMutableStructForMIR(t *testing.T) {
	source := `module main;struct Token{i64 value;}resource Owner{ptr mut void data;usize length;cleanup{}}
 fn void edit(ref mut Token[] items){for(let ref mut Token item in items){item.value+=3;if(item.value==7){continue;}break;}}
 fn i32 main(){var Owner owner=Owner{.data=null,.length=0};let ref mut Token[] items=unsafe{storage_borrow_mut<Token[]>(ref mut owner,owner.data,owner.length)};edit(items);return 0;}`
	a, e := analyzeStorageBorrow(t, source)
	if e != nil {
		t.Fatal(e)
	}
	checked, e := compiler.LowerAnalyzedStorageTransfers(a.Analysis)
	if e != nil || !checked.Verified {
		t.Fatalf("lower: %v", e)
	}
	data, e := mircodec.EncodeStorageTransfers(checked.MIR, 1<<20)
	if e != nil {
		t.Fatal(e)
	}
	decoded, e := mircodec.DecodeStorageTransfers(data, 1<<20)
	if e != nil {
		t.Fatal(e)
	}
	borrows := 0
	for _, fn := range decoded.Functions {
		for _, b := range fn.Blocks {
			for _, op := range b.Operations {
				if op.Kind == mir.OperationViewElementBorrow {
					borrows++
					_, mut, _ := hir.BorrowElement(op.Type)
					if !mut || !hir.MutableIndexedView(op.OwnerType) {
						t.Fatal("copied or shared item")
					}
				}
			}
		}
	}
	if borrows != 1 {
		t.Fatalf("borrows=%d", borrows)
	}
	for _, profile := range []cbackend.Profile{cbackend.Stage0WindowsMSVCProfile(), cbackend.Stage0Profile()} {
		out, e := cbackend.EmitStorageTransfers(decoded, hir.QualifiedModulePath(1, "main")+"::main", profile)
		if e != nil {
			t.Fatal(e)
		}
		if !strings.Contains(out.Source, "&((") {
			t.Fatal("missing stored address")
		}
	}
	removed := false
	for fi := range decoded.Functions {
		for bi := range decoded.Functions[fi].Blocks {
			b := &decoded.Functions[fi].Blocks[bi]
			for oi, op := range b.Operations {
				if op.Kind == mir.OperationViewIndexCheck {
					b.Operations = append(b.Operations[:oi], b.Operations[oi+1:]...)
					removed = true
					break
				}
			}
		}
	}
	if !removed || len(mir.VerifyStorageTransfers(decoded)) == 0 {
		t.Fatal("unguarded loop borrow accepted")
	}
	if _, e := mircodec.EncodeStorageTransfers(decoded, 1<<20); e == nil {
		t.Fatal("unguarded borrow crossed codec")
	}
}
