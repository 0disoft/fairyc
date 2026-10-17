package compiler_test

import (
	"bytes"
	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/mircodec"
	"testing"
)

func TestSharedStructForMIR(t *testing.T) {
	const source = `module main;struct Span{usize start;usize end;}struct Token{i64 value;Span position;}struct Other{i64 value;Span position;}
 resource Owner{ptr mut void data;usize length;cleanup{}}
 fn Option<ref Token> first(ref Token[] items){for(let ref Token item in items){assert(item.position.end>=item.position.start);return .some(item);}return .none;}
 fn Option<ref Token> first_mut(ref mut Token[] items){for(let ref Token item in items){return .some(item);}return .none;}
 fn i32 main(){var Owner owner=Owner{.data=null,.length=0};let ref mut Token[] items=unsafe{storage_borrow_mut<Token[]>(ref mut owner,owner.data,owner.length)};discard first_mut(items);
 let ref Token[] shared=unsafe{storage_borrow<Token[]>(ref owner,owner.data,owner.length)};discard first(shared);return 0;}`
	encode := func() []byte {
		a, err := analyzeStorageBorrow(t, source)
		if err != nil {
			t.Fatal(err)
		}
		r, err := compiler.LowerAnalyzedStorageTransfers(a.Analysis)
		if err != nil || !r.Verified {
			t.Fatalf("lower: %v %+v", err, r.Semantic)
		}
		data, err := mircodec.EncodeStorageTransfers(r.MIR, 1<<20)
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	data := encode()
	if !bytes.Equal(data, encode()) {
		t.Fatal("repeated source analysis changed MIR")
	}
	p, err := mircodec.DecodeStorageTransfers(data, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, fn := range p.Functions {
		for _, b := range fn.Blocks {
			for _, op := range b.Operations {
				if op.Kind == mir.OperationViewIndex {
					t.Fatal("reference iteration copied a struct")
				}
				if op.Kind == mir.OperationViewElementBorrow {
					count++
					_, mut, ok := hir.BorrowElement(op.Type)
					if !ok || mut {
						t.Fatal("shared loop yielded mutable/value item")
					}
				}
			}
		}
	}
	if count != 2 {
		t.Fatalf("shared borrows=%d", count)
	}
	for _, profile := range []cbackend.Profile{cbackend.Stage0WindowsMSVCProfile(), cbackend.Stage0Profile()} {
		if _, err := cbackend.EmitStorageTransfers(p, hir.QualifiedModulePath(1, "main")+"::main", profile); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"bounds", "mutable-result", "wrong-nominal"} {
		t.Run(name, func(t *testing.T) {
			bad, err := mircodec.DecodeStorageTransfers(data, 1<<20)
			if err != nil {
				t.Fatal(err)
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
						if name == "bounds" {
							if op.Kind == mir.OperationViewIndexCheck {
								b.Operations = append(b.Operations[:oi], b.Operations[oi+1:]...)
								found = true
								break
							}
							continue
						}
						if op.Kind != mir.OperationViewElementBorrow {
							continue
						}
						found = true
						if name == "mutable-result" {
							element, _, _ := hir.BorrowElement(op.Type)
							op.Type = hir.MutableBorrow(element)
						} else {
							op.Type = hir.SharedBorrow(hir.ParseType(hir.QualifiedModulePath(1, "main") + "::Other"))
						}
					}
				}
			}
			if !found || len(mir.VerifyStorageTransfers(bad)) == 0 {
				t.Fatal("invalid struct loop reference verified")
			}
			if _, err := mircodec.EncodeStorageTransfers(bad, 1<<20); err == nil {
				t.Fatal("invalid loop crossed codec")
			}
			if _, err := cbackend.EmitStorageTransfers(bad, hir.QualifiedModulePath(1, "main")+"::main", cbackend.Stage0WindowsMSVCProfile()); err == nil {
				t.Fatal("invalid loop emitted")
			}
		})
	}
}
