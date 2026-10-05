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

func TestStructArrayStorageRoundTrip(t *testing.T) {
	source, e := os.ReadFile("testdata/struct_array_storage.fy")
	if e != nil {
		t.Fatal(e)
	}
	r, e := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", string(source))})
	if e != nil || !r.Verified {
		t.Fatalf("check: %v %+v %+v", e, r.Semantic.Diagnostics, r.Semantic.Unsupported)
	}
	data, e := mircodec.Encode(r.MIR, 1<<20)
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
	for _, tc := range []struct {
		name string
		typ  hir.Type
		repr bool
	}{
		{"zero", hir.FixedArray(hir.TypeU8, 0), false}, {"oversize", hir.FixedArray(hir.TypeU8, 4097), false}, {"nested", hir.FixedArray(hir.FixedArray(hir.TypeU8, 2), 2), false}, {"foreign", hir.FixedArray(hir.TypeU8, 4), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bad, e := mircodec.Decode(data, 1<<20)
			if e != nil {
				t.Fatal(e)
			}
			found := false
			for i := range bad.StructTypes {
				for j := range bad.StructTypes[i].Fields {
					f := &bad.StructTypes[i].Fields[j]
					if f.Name == "bytes" {
						f.Type = tc.typ
						bad.StructTypes[i].ReprC = tc.repr
						found = true
					}
				}
			}
			if !found || len(mir.Verify(bad)) == 0 {
				t.Fatal("invalid array field accepted")
			}
			if _, e := mircodec.Encode(bad, 1<<20); e == nil {
				t.Fatal("invalid array field encoded")
			}
		})
	}
}
