package compiler_test

import (
	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/mircodec"
	"github.com/0disoft/fairyc/internal/semantic"
	"testing"
)

func TestPartialAggregateConstructionCleanup(t *testing.T) {
	const declarations = `module main;
 fn i32 observe(i32 n){return n;}
 resource Leaf{i32 n;cleanup{discard observe(this.n);}}
 resource Parent{Leaf a;Leaf b;i32 n;cleanup{discard observe(90);}}
 fn Leaf leaf(i32 n){return Leaf{.n=n};}
 fn Result<i32,i32> step(bool success){return if(success){.ok(42)}else{.err(7)};}
 `
	for _, c := range []struct{ name, setup, fields, trace string }{
		{"before any field", "", `.n=step(false)?,.a=Leaf{.n=65},.b=Leaf{.n=66}`, ""},
		{"inline field", "", `.a=Leaf{.n=65},.n=step(false)?,.b=Leaf{.n=66}`, "A"},
		{"call field", "", `.a=leaf(65),.n=step(false)?,.b=leaf(66)`, "A"},
		{"reordered completed fields", "", `.b=Leaf{.n=66},.a=Leaf{.n=65},.n=step(false)?`, "AB"},
		{"moved source field", `let Leaf original=leaf(65);`, `.a=original,.n=step(false)?,.b=leaf(66)`, "A"},
		{"success transfers once", "", `.a=Leaf{.n=65},.b=Leaf{.n=66},.n=step(true)?`, "ZBA"},
		{"branch propagation", "", `.a=Leaf{.n=65},.b=leaf(66),.n=if(true){step(false)?}else{42}`, "BA"},
		{"older local follows partial fields", `let Leaf older=leaf(67);`, `.a=leaf(65),.n=step(false)?,.b=leaf(66)`, "AC"},
	} {
		t.Run(c.name, func(t *testing.T) {
			text := declarations + `fn Result<Parent,i32> build(){` + c.setup + `return .ok(Parent{` + c.fields + `});}fn i32 main(){discard build();return 42;}`
			checked, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", text)})
			if err != nil || !checked.Verified {
				t.Fatalf("check: %v %+v %+v", err, checked.Semantic.Diagnostics, checked.Semantic.Unsupported)
			}
			raw, err := mircodec.Encode(checked.MIR, 1<<20)
			if err != nil {
				t.Fatal(err)
			}
			program, err := mircodec.Decode(raw, 1<<20)
			if err != nil {
				t.Fatal(err)
			}
			out, err := interpreter.ExecuteWithOptions(program, "main::main", interpreter.Options{TestIntrinsics: map[string]interpreter.TestIntrinsicKind{"main::observe": interpreter.TestIntrinsicWriteStdoutByte}, OutputLimit: 8})
			if err != nil || out.Kind != interpreter.OutcomeReturn || out.Value != 42 || string(out.Stdout) != c.trace {
				t.Fatalf("cleanup: %v %s %d trace=%q want=%q", err, out.Kind, out.Value, out.Stdout, c.trace)
			}
			for _, profile := range []cbackend.Profile{cbackend.Stage0Profile(), cbackend.Stage0WindowsMSVCProfile()} {
				if _, err := cbackend.Emit(program, "main::main", profile); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}
