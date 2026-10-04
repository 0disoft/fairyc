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

func TestArraySubviewBorrowRoundTrip(t *testing.T) {
	source, e := os.ReadFile("testdata/array_subview_borrow.fy")
	if e != nil {
		t.Fatal(e)
	}
	checked, e := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", string(source))})
	if e != nil || !checked.Verified {
		t.Fatalf("check: %v %+v %+v", e, checked.Semantic.Diagnostics, checked.Semantic.Unsupported)
	}
	data, e := mircodec.Encode(checked.MIR, 1<<20)
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
	removed := false
	for fi := range p.Functions {
		for bi := range p.Functions[fi].Blocks {
			b := &p.Functions[fi].Blocks[bi]
			for oi, op := range b.Operations {
				if !removed && op.Kind == mir.OperationViewSliceCheck {
					b.Operations = append(b.Operations[:oi], b.Operations[oi+1:]...)
					removed = true
					break
				}
			}
		}
	}
	if !removed || len(mir.Verify(p)) == 0 {
		t.Fatal("unchecked array subview accepted")
	}
	if _, e := mircodec.Encode(p, 1<<20); e == nil {
		t.Fatal("unchecked array subview encoded")
	}
	if _, e := cbackend.Emit(p, "main::main", cbackend.Stage0WindowsMSVCProfile()); e == nil {
		t.Fatal("unchecked array subview emitted")
	}
}

func TestArraySubviewBorrowBounds(t *testing.T) {
	for _, tc := range []struct{ name, bounds string }{
		{"reversed", "2..1"}, {"past-end", "..3"}, {"start-past-end", "3.."},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := `module main;fn i32 main(){let i64[2] a=[1,2];let ref i64[] v=ref a[` + tc.bounds + `];return 0;}`
			r, e := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", source)})
			if e != nil || !r.Verified {
				t.Fatalf("check: %v %+v %+v", e, r.Semantic.Diagnostics, r.Semantic.Unsupported)
			}
			out, e := interpreter.Execute(r.MIR, "main::main")
			if e != nil || out.Kind != interpreter.OutcomeTrap || out.TrapCategory != "bounds" {
				t.Fatalf("execute: %+v %v", out, e)
			}
		})
	}
}

func TestArraySubviewBorrowBoundReturn(t *testing.T) {
	for _, bounds := range []string{`{return 0;0}..{assert(false);1}`, `0..{return 0;1}`} {
		source := `module main;fn void use(ref mut i64[] v,i32 next){assert(false);}fn i32 main(){var i64[2] a=[1,2];use(ref mut a[` + bounds + `],{assert(false);1});assert(false);return 1;}`
		r, e := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", source)})
		if e != nil || !r.Verified {
			t.Fatalf("check: %v %+v %+v", e, r.Semantic.Diagnostics, r.Semantic.Unsupported)
		}
		out, e := interpreter.Execute(r.MIR, "main::main")
		if e != nil || out.Kind != interpreter.OutcomeReturn || out.Value != 0 {
			t.Fatalf("execute: %+v %v", out, e)
		}
		if _, e := cbackend.Emit(r.MIR, "main::main", cbackend.Stage0WindowsMSVCProfile()); e != nil {
			t.Fatal(e)
		}
	}
}
