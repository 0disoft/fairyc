package compiler_test

import (
	"bytes"
	"os"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/mircodec"
	"github.com/0disoft/fairyc/internal/semantic"
)

func TestDerefCompoundRoundTrip(t *testing.T) {
	src, e := os.ReadFile("testdata/deref_compound.fy")
	if e != nil {
		t.Fatal(e)
	}
	files := []semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", string(src))}
	checked, e := compiler.Check(files)
	if e != nil || !checked.Verified {
		t.Fatalf("check: %v %+v %+v", e, checked.Semantic.Diagnostics, checked.Semantic.Unsupported)
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
}

func TestDerefCompoundCheckedStore(t *testing.T) {
	source := `module main;fn void add(ref mut i64 p){*p+=2;}fn i32 main(){var i64 x=10;add(ref mut x);assert(x==12);return 0;}`
	r, e := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", source)})
	if e != nil || !r.Verified {
		t.Fatalf("check %v", e)
	}
	reads, stores := 0, 0
	for _, fn := range r.MIR.Functions {
		if fn.Name != "main::add" {
			continue
		}
		for _, block := range fn.Blocks {
			for _, op := range block.Operations {
				if op.Kind == mir.OperationUnary && op.Operator == "*" {
					reads++
				}
				if op.Kind == mir.OperationDerefStore {
					stores++
				}
			}
		}
	}
	if reads != 1 || stores != 1 {
		t.Fatalf("want one load and store, got %d %d", reads, stores)
	}
	data, e := mircodec.Encode(r.MIR, 1<<20)
	if e != nil {
		t.Fatal(e)
	}
	for _, kind := range []string{"unchecked", "shared-store"} {
		t.Run(kind, func(t *testing.T) {
			p, e := mircodec.Decode(data, 1<<20)
			if e != nil {
				t.Fatal(e)
			}
			changed := false
			for fi := range p.Functions {
				for bi := range p.Functions[fi].Blocks {
					b := &p.Functions[fi].Blocks[bi]
					for oi := range b.Operations {
						op := &b.Operations[oi]
						if changed {
							continue
						}
						if kind == "unchecked" && op.Kind == mir.OperationArithmeticCheck {
							b.Operations = append(b.Operations[:oi], b.Operations[oi+1:]...)
							changed = true
							break
						}
						if kind == "shared-store" && op.Kind == mir.OperationDerefStore {
							op.OwnerType = hir.SharedBorrow(op.Type)
							changed = true
						}
					}
				}
			}
			if !changed || len(mir.Verify(p)) == 0 {
				t.Fatal("invalid compound store verified")
			}
			if _, e := mircodec.Encode(p, 1<<20); e == nil {
				t.Fatal("invalid compound crossed codec")
			}
			if _, e := cbackend.Emit(p, "main::main", cbackend.Stage0WindowsMSVCProfile()); e == nil {
				t.Fatal("invalid compound emitted")
			}
		})
	}
}
