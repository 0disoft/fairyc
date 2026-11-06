package compiler_test

import (
	"fmt"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/semantic"
	"strings"
	"testing"
)

func TestInlineLocalsIdentityAndExecution(t *testing.T) {
	const declaration = "@inline pub fn i32 f(i32 x){let i32 a=x+2;let b=a*2;return b/2;}"
	const expected = "fyinlinefn1-sha256-6aac2a75fa26963e634daa3970fe133abdd3cd7e440ae96e9a12cefd94231aaf"
	for name, decl := range map[string]string{
		"baseline": declaration,
		"rename":   "@inline pub fn i32 f(i32 input){let left=(input+0x2);let i32 right=left*0b10;return (right/2);}",
	} {
		t.Run(name, func(t *testing.T) {
			r, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", "module main;"+decl+"fn i32 main(){return f(40);}")})
			if err != nil || !r.Verified {
				t.Fatalf("check: %v %+v %+v", err, r.Semantic.Diagnostics, r.Semantic.Unsupported)
			}
			if got := r.Semantic.HIR.Modules[0].Functions[0].InlineBody; got != expected {
				t.Fatalf("identity %s want %s", got, expected)
			}
			out, err := interpreter.Execute(r.MIR, "main::main")
			if err != nil || out.Value != 42 {
				t.Fatalf("execute: %+v %v", out, err)
			}
		})
	}
}

func TestInlineLocalsClosedForms(t *testing.T) {
	var many strings.Builder
	for i := 0; i < 17; i++ {
		fmt.Fprintf(&many, "let i32 v%d=%d;", i, i)
	}
	bodies := map[string]string{
		"mutable": "var i32 x=1;return x;", "assign": "let i32 x=1;x=2;return x;",
		"call": "let i32 x=h();return x;", "discard": "let i32 x=1;discard x;return x;",
		"control": "let i32 x=1;return if(true){x}else{2};", "type": "let i64 x=1;return 42;",
		"forward": "let i32 a=b;let i32 b=2;return a;", "self": "let i32 x=x;return x;",
		"after_return": "return 42;let i32 x=1;", "many": many.String() + "return 42;",
		"budget": "let i32 unused=1;let i32 second=1;return " + strings.Repeat("1+", 31) + "1;",
	}
	for name, body := range bodies {
		t.Run(name, func(t *testing.T) {
			r, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", "module main;fn i32 h(){return 1;}@inline pub fn i32 f(){"+body+"}fn i32 main(){return 0;}")})
			if err == nil && r.Verified {
				t.Fatal("closed inline body admitted")
			}
		})
	}
}

func TestInlineLocalsUnusedInitializerTraps(t *testing.T) {
	for name, body := range map[string]string{
		"division":       "let i32 unused=10/0;return 42;",
		"overflow_first": "let i32 unused=2147483647+1;let i32 other=10/0;return 42;",
	} {
		t.Run(name, func(t *testing.T) {
			r, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", "module main;@inline pub fn i32 f(){"+body+"}fn i32 main(){return f();}")})
			if err != nil || !r.Verified {
				t.Fatalf("check: %v %+v", err, r.Semantic)
			}
			out, err := interpreter.Execute(r.MIR, "main::main")
			want := "division-by-zero"
			if name == "overflow_first" {
				want = "integer-overflow"
			}
			if err != nil || out.Kind != interpreter.OutcomeTrap || out.TrapCategory != want {
				t.Fatalf("trap: %+v %v", out, err)
			}
		})
	}
}

func TestInlineLocalsBoundsAndIdentitySensitivity(t *testing.T) {
	var bindings strings.Builder
	bindings.WriteString("let i32 v0=27;")
	for i := 1; i < 16; i++ {
		fmt.Fprintf(&bindings, "let i32 v%d=v%d+1;", i, i-1)
	}
	for _, body := range []string{bindings.String() + "return v15;", "let i32 unused=1;return " + strings.Repeat("1+", 31) + "1;"} {
		r, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", "module main;@inline pub fn i32 f(){"+body+"}fn i32 main(){return f();}")})
		if err != nil || !r.Verified {
			t.Fatalf("bounded body: %v %+v", err, r.Semantic)
		}
	}
	var ids []string
	for _, body := range []string{
		"let i32 a=1;let i32 b=2;return 42;",
		"let i32 a=2;let i32 b=1;return 42;",
		"let i32 a=1;let i32 b=3;return 42;",
		"return 42;",
	} {
		r, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", "module main;@inline pub fn i32 f(){"+body+"}fn i32 main(){return f();}")})
		if err != nil || !r.Verified {
			t.Fatalf("check: %v %+v", err, r.Semantic)
		}
		id := r.Semantic.HIR.Modules[0].Functions[0].InlineBody
		for _, prior := range ids {
			if id == prior {
				t.Fatal("initializer order, unused value or sequence erased from identity")
			}
		}
		ids = append(ids, id)
	}
}
