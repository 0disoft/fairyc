package compiler_test

import (
	"context"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/inlinebody"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/mircodec"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/semantic"
)

const inlineSource = `module main;@inline pub fn i32 add(i32 x){return x+2;}fn i32 main(){return add(40);}`

func TestCrossPackageInlineIdentityCodecAndEmission(t *testing.T) {
	checked, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", inlineSource)})
	if err != nil || !checked.Verified {
		t.Fatalf("check: %v %+v %+v", err, checked.Semantic.Diagnostics, checked.Semantic.Unsupported)
	}
	coordinates := map[hir.PackageID]packagegraph.CoordinateID{hir.DefaultPackage: {1}}
	baseline, err := inlinebody.HIR(checked.Semantic.HIR, coordinates)
	if err != nil || baseline[hir.DefaultPackage] == (inlinebody.Hash{}) {
		t.Fatal("identity absent", err)
	}
	for name, text := range map[string]string{"format": strings.Replace(inlineSource, "i32 x){return x+2;}", "i32 renamed) { /* spacing */ return (renamed + 0x2); }", 1), "body": strings.Replace(inlineSource, "x+2", "x+3", 1), "remove": strings.Replace(inlineSource, "@inline ", "", 1)} {
		result, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", text)})
		if err != nil || !result.Verified {
			t.Fatalf("%s: %v %+v", name, err, result.Semantic)
		}
		hash, err := inlinebody.HIR(result.Semantic.HIR, coordinates)
		if err != nil || (hash[hir.DefaultPackage] == baseline[hir.DefaultPackage]) != (name == "format") {
			t.Fatal("inline invalidation", name, err)
		}
	}
	encoded, err := mircodec.Encode(checked.MIR, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	program, err := mircodec.Decode(encoded, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	if err := inlinebody.VerifyProgram(checked.Semantic.HIR, program); err != nil {
		t.Fatal(err)
	}
	for _, profile := range []cbackend.Profile{cbackend.Stage0Profile(), cbackend.Stage0WindowsMSVCProfile()} {
		unit, err := cbackend.Emit(program, "main::main", profile)
		if err != nil || !strings.Contains(unit.Source, "static inline int32_t fy_fn_") {
			t.Fatal("inline C definition missing", err)
		}
		parts, err := cbackend.EmitBuildContext(context.Background(), program, "main::main", profile, false, 1)
		if err != nil || !strings.Contains(parts.Header, "static inline int32_t fy_fn_") {
			t.Fatal("inline body missing from shared header", err)
		}
	}
	found := false
	for i := range program.Functions {
		if program.Functions[i].InlineBody != "" {
			program.Functions[i].InlineBody = ""
			found = true
		}
	}
	if !found || inlinebody.VerifyProgram(checked.Semantic.HIR, program) == nil {
		t.Fatal("cached inline identity mismatch admitted")
	}
	forged := mir.Clone(checked.MIR)
	for i := range forged.Functions {
		if forged.Functions[i].InlineBody != "" {
			forged.Functions[i].InlineBody = "bad"
		}
	}
	if len(mir.Verify(forged)) == 0 {
		t.Fatal("malformed inline identity admitted")
	}
}

func TestCrossPackageInlineRejectsClosedForms(t *testing.T) {
	for name, decl := range map[string]string{
		"private": "@inline fn i32 f(){return 1;}", "unsafe": "@inline pub unsafe fn i32 f(){return 1;}", "argument": "@inline(1) pub fn i32 f(){return 1;}", "duplicate": "@inline @inline pub fn i32 f(){return 1;}", "generic": "@inline pub fn T f<T>(T x){return x;}", "call": "fn i32 h(){return 1;}@inline pub fn i32 f(){return h();}", "local": "@inline pub fn i32 f(){var i64 x=1;return 1;}", "effects": "@inline @effects(console) pub fn i32 f(){return 1;}", "type": "@inline pub fn u32 f(){return 1;}", "mutable": "@inline pub fn i32 f(i32 x){x=2;return x;}", "arity": "@inline pub fn i32 f(i32 a,i32 b,i32 c,i32 d,i32 e){return a;}", "effectful-loop": "@inline pub fn i32 f(){while(true){assert(false);}return 1;}",
	} {
		t.Run(name, func(t *testing.T) {
			r, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", "module main;"+decl+"fn i32 main(){return 0;}")})
			if err == nil && r.Verified {
				t.Fatal("unsupported inline function admitted")
			}
		})
	}
}
