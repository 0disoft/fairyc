package compiler_test

import (
	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/mircodec"
	"github.com/0disoft/fairyc/internal/semantic"
	"strings"
	"testing"
)

func TestStructDerefStoreRoundTrip(t *testing.T) {
	const src = `module main;struct Span{usize start;}struct T{i64 value;Span span;}
 fn void replace(ref mut T target,T value){*target=value;}
 fn i32 main(){var T original=T{.value=7,.span=Span{.start=1}};var T replacement=T{.value=4000000000,.span=Span{.start=2}};var T old=original;replace(ref mut original,replacement);replacement.span.start=99;assert(original.value==4000000000 && original.span.start==2 && old.value==7);let ref mut Span nested=ref mut original.span;*nested=Span{.start=3};assert(original.span.start==3);return 42;}`
	c, e := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", src)})
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
	if e != nil || out.Kind != interpreter.OutcomeReturn || out.Value != 42 {
		t.Fatalf("execute: %+v %v", out, e)
	}
	for _, profile := range []cbackend.Profile{cbackend.Stage0WindowsMSVCProfile(), cbackend.Stage0Profile()} {
		out, e := cbackend.Emit(p, "main::main", profile)
		if e != nil {
			t.Fatal(e)
		}
		if !strings.Contains(out.Source, "*(") {
			t.Fatal("missing deref store")
		}
	}
	stores := 0
	for _, f := range p.Functions {
		for _, b := range f.Blocks {
			for _, op := range b.Operations {
				if op.Kind == mir.OperationDerefStore {
					stores++
					if mir.ValidDerefStoreShape(op) {
						t.Fatal("struct admitted as scalar")
					}
				}
			}
		}
	}
	if stores != 2 {
		t.Fatalf("stores=%d", stores)
	}
	for _, name := range []string{"shared", "wrong-value", "missing-value", "nonplain", "move", "destination"} {
		t.Run(name, func(t *testing.T) {
			bad, e := mircodec.Decode(data, 1<<20)
			if e != nil {
				t.Fatal(e)
			}
			found := false
			for fi := range bad.Functions {
				for bi := range bad.Functions[fi].Blocks {
					for oi := range bad.Functions[fi].Blocks[bi].Operations {
						op := &bad.Functions[fi].Blocks[bi].Operations[oi]
						if op.Kind != mir.OperationDerefStore || found {
							continue
						}
						found = true
						switch name {
						case "shared":
							op.OwnerType = hir.SharedBorrow(op.Type)
						case "wrong-value":
							op.Operands[1] = op.Operands[0]
						case "missing-value":
							op.Operands = op.Operands[:1]
						case "nonplain":
							for i := range bad.StructTypes {
								if bad.StructTypes[i].Type == op.Type {
									bad.StructTypes[i].Fields[0].Type = hir.Type("*void")
								}
							}
						case "move":
							op.MovePayload = true
						case "destination":
							op.DestinationLocal = 3
						}
					}
				}
			}
			if !found || len(mir.Verify(bad)) == 0 {
				t.Fatal("invalid struct store verified")
			}
			if _, e := mircodec.Encode(bad, 1<<20); e == nil {
				t.Fatal("invalid struct store crossed codec")
			}
			if _, e := cbackend.Emit(bad, "main::main", cbackend.Stage0WindowsMSVCProfile()); e == nil {
				t.Fatal("invalid store emitted")
			}
		})
	}
}

func TestStructDerefStoreTerminatingRHS(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		trap       bool
	}{
		{"return", "return;", false}, {"trap", "assert(false);", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			src := `module main;struct T{i64 value;}fn void edit(ref mut T item){*item={` + tc.body + `T{.value=9}};}fn i32 main(){var T value=T{.value=7};edit(ref mut value);assert(value.value==7);return 42;}`
			c, e := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", src)})
			if e != nil || !c.Verified {
				t.Fatalf("check: %v %+v", e, c.Semantic)
			}
			for _, f := range c.MIR.Functions {
				if f.Name == "main::edit" {
					for _, b := range f.Blocks {
						for _, op := range b.Operations {
							if op.Kind == mir.OperationDerefStore {
								t.Fatal("terminating RHS has store")
							}
						}
					}
				}
			}
			out, e := interpreter.Execute(c.MIR, "main::main")
			if e != nil {
				t.Fatal(e)
			}
			if tc.trap {
				if out.Kind != interpreter.OutcomeTrap {
					t.Fatalf("trap: %+v", out)
				}
			} else if out.Kind != interpreter.OutcomeReturn || out.Value != 42 {
				t.Fatalf("return: %+v", out)
			}
		})
	}
}

func TestStructDerefStoreIndexedMIR(t *testing.T) {
	src := `module main;struct T{i64 value;}resource Owner{ptr mut void data;usize length;cleanup{}}fn void edit(ref mut T[] view){let ref mut T item=ref mut view[0];*item=T{.value=4000000000};}fn i32 main(){var Owner owner=Owner{.data=null,.length=0};let ref mut T[] view=unsafe{storage_borrow_mut<T[]>(ref mut owner,owner.data,owner.length)};edit(view);return 0;}`
	a, e := analyzeStorageBorrow(t, src)
	if e != nil {
		t.Fatal(e)
	}
	c, e := compiler.LowerAnalyzedStorageTransfers(a.Analysis)
	if e != nil || !c.Verified {
		t.Fatalf("lower: %v", e)
	}
	data, e := mircodec.EncodeStorageTransfers(c.MIR, 1<<20)
	if e != nil {
		t.Fatal(e)
	}
	p, e := mircodec.DecodeStorageTransfers(data, 1<<20)
	if e != nil {
		t.Fatal(e)
	}
	for _, profile := range []cbackend.Profile{cbackend.Stage0WindowsMSVCProfile(), cbackend.Stage0Profile()} {
		if _, e := cbackend.EmitStorageTransfers(p, hir.QualifiedModulePath(1, "main")+"::main", profile); e != nil {
			t.Fatal(e)
		}
	}
	removed := false
	for fi := range p.Functions {
		for bi := range p.Functions[fi].Blocks {
			b := &p.Functions[fi].Blocks[bi]
			for oi, op := range b.Operations {
				if op.Kind == mir.OperationViewIndexCheck {
					b.Operations = append(b.Operations[:oi], b.Operations[oi+1:]...)
					removed = true
					break
				}
			}
		}
	}
	if !removed || len(mir.VerifyStorageTransfers(p)) == 0 {
		t.Fatal("unguarded address/store verified")
	}
	if _, e := mircodec.EncodeStorageTransfers(p, 1<<20); e == nil {
		t.Fatal("unguarded store crossed codec")
	}
}
