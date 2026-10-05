package compiler_test

import (
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

func TestInlineNominalArrayFieldRoundTrip(t *testing.T) {
	source, err := os.ReadFile("testdata/inline_nominal_array_fields.fy")
	if err != nil {
		t.Fatal(err)
	}
	checked, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", string(source))})
	if err != nil || !checked.Verified {
		t.Fatalf("check: %v %+v %+v", err, checked.Semantic.Diagnostics, checked.Semantic.Unsupported)
	}
	encoded, err := mircodec.Encode(checked.MIR, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	p, err := mircodec.Decode(encoded, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	out, err := interpreter.Execute(p, "main::main")
	if err != nil || out.Kind != interpreter.OutcomeReturn || out.Value != 0 {
		t.Fatalf("execute: %+v %v", out, err)
	}
	for _, profile := range []cbackend.Profile{cbackend.Stage0WindowsMSVCProfile(), cbackend.Stage0Profile()} {
		if _, err := cbackend.Emit(p, "main::main", profile); err != nil {
			t.Fatal(err)
		}
	}
	for _, mutation := range []string{"repr-c", "wrong-element", "recursive-array", "element-array"} {
		t.Run(mutation, func(t *testing.T) {
			bad, err := mircodec.Decode(encoded, 1<<20)
			if err != nil {
				t.Fatal(err)
			}
			for i := range bad.StructTypes {
				s := &bad.StructTypes[i]
				if s.Name == "APage" {
					switch mutation {
					case "repr-c":
						s.ReprC = true
					case "wrong-element":
						s.Fields[0].Type = hir.FixedArray(hir.Type("main::Missing"), 2)
					case "recursive-array":
						s.Fields[0].Type = hir.FixedArray(s.Type, 2)
					}
				}
				if mutation == "element-array" && s.Name == "ZToken" {
					s.Fields[0].Type = hir.FixedArray(hir.TypeU8, 2)
				}
			}
			found := false
			for _, failure := range mir.Verify(bad) {
				if failure.Code == "V1001" && failure.Detail == "invalid inline array struct field" {
					found = true
				}
			}
			if !found {
				t.Fatal("invalid inline nominal array field lacks its structural rejection")
			}
			if _, err := mircodec.Encode(bad, 1<<20); err == nil {
				t.Fatal("invalid inline nominal array field encoded")
			}
		})
	}
}
