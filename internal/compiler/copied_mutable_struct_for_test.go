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

func TestCopiedMutableStructForMIR(t *testing.T) {
	const source = `module main;struct Span{usize start;}struct Token{i64 value;Span position;}struct Other{i64 value;Span position;}
 resource Owner{ptr mut void data;usize length;cleanup{}}
 fn Option<Token> first(ref mut Token[] items){for(let Token item in items){items[0]=Token{.value=9,.position=Span{.start=1}};assert(item.value>0);return .some(item);}return .none;}
 fn i32 main(){var Owner owner=Owner{.data=null,.length=0};let ref mut Token[] items=unsafe{storage_borrow_mut<Token[]>(ref mut owner,owner.data,owner.length)};discard first(items);return 0;}`
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
					count++
					if !hir.MutableIndexedView(op.OwnerType) || op.Type != hir.ParseType(hir.QualifiedModulePath(1, "main")+"::Token") {
						t.Fatal("loop did not load a matching independent Token value")
					}
				}
				if op.Kind == mir.OperationViewElementBorrow {
					t.Fatal("copied loop yielded an element reference")
				}
			}
		}
	}
	if count != 1 {
		t.Fatalf("copy loads=%d", count)
	}
	for _, profile := range []cbackend.Profile{cbackend.Stage0WindowsMSVCProfile(), cbackend.Stage0Profile()} {
		if _, err := cbackend.EmitStorageTransfers(p, hir.QualifiedModulePath(1, "main")+"::main", profile); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"bounds", "wrong-nominal", "receiver-definition"} {
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
						if op.Kind != mir.OperationViewIndex {
							continue
						}
						found = true
						if name == "wrong-nominal" {
							op.Type = hir.ParseType(hir.QualifiedModulePath(1, "main") + "::Other")
						} else {
							op.OwnerType = hir.SharedBorrow(hir.ParseType("[" + hir.QualifiedModulePath(1, "main") + "::Token]"))
						}
					}
				}
			}
			if !found || len(mir.VerifyStorageTransfers(bad)) == 0 {
				t.Fatal("invalid struct loop load verified")
			}
			if _, err := mircodec.EncodeStorageTransfers(bad, 1<<20); err == nil {
				t.Fatal("invalid load crossed codec")
			}
			if _, err := cbackend.EmitStorageTransfers(bad, hir.QualifiedModulePath(1, "main")+"::main", cbackend.Stage0WindowsMSVCProfile()); err == nil {
				t.Fatal("invalid load emitted")
			}
		})
	}
}
