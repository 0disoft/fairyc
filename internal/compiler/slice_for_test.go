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

func TestSliceForRoundTrip(t *testing.T) {
	src, e := os.ReadFile("testdata/slice_for.fy")
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

func TestSliceForBoundsDominance(t *testing.T) {
	source, e := os.ReadFile("testdata/slice_for.fy")
	if e != nil {
		t.Fatal(e)
	}
	r, e := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", string(source))})
	if e != nil || !r.Verified {
		t.Fatal(e)
	}
	data, e := mircodec.Encode(r.MIR, 1<<20)
	if e != nil {
		t.Fatal(e)
	}
	p, e := mircodec.Decode(data, 1<<20)
	if e != nil {
		t.Fatal(e)
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
		t.Fatal("unchecked subview accepted")
	}
	if _, e := mircodec.Encode(p, 1<<20); e == nil {
		t.Fatal("unchecked subview crossed codec")
	}
	if _, e := cbackend.Emit(p, "main::main", cbackend.Stage0WindowsMSVCProfile()); e == nil {
		t.Fatal("unchecked subview emitted")
	}
}
func TestSliceForBoundReturn(t *testing.T) {
	for _, root := range []string{"data", "view"} {
		for _, bounds := range []string{`{return 0;0}..{assert(false);1}`, `0..{return 0;1}`} {
			source := `module main;fn i32 main(){var i64[2] data=[1,2];let ref i64[] view=ref data;for(let i64 item in ` + root + `[` + bounds + `]){assert(false);}assert(false);return 1;}`
			r, e := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", source)})
			if e != nil || !r.Verified {
				t.Fatalf("check %v %+v %+v", e, r.Semantic.Diagnostics, r.Semantic.Unsupported)
			}
			out, e := interpreter.Execute(r.MIR, "main::main")
			if e != nil || out.Kind != interpreter.OutcomeReturn || out.Value != 0 {
				t.Fatalf("execute %+v %v", out, e)
			}
			if _, e := cbackend.Emit(r.MIR, "main::main", cbackend.Stage0WindowsMSVCProfile()); e != nil {
				t.Fatal(e)
			}
		}
	}
}
