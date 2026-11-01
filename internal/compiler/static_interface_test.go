package compiler_test

import (
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/semantic"
	"strings"
	"testing"
)

const staticInterfaceSource = `module main;
@interface fn T transform<T>(T value){}
@implements(transform,i32) fn i32 twice(i32 value){return value*2;}
@implements(transform,i64) fn i64 triple(i64 value){return value*3;}
fn T algorithm<T>(T value){return transform<T>(value);}
fn i32 main(){assert(algorithm<i64>(14)==42);return algorithm<i32>(21);}`

func TestStaticInterfaceExecution(t *testing.T) {
	file := parseCompilerFile(t, hir.DefaultPackage, "main.fy", staticInterfaceSource)
	r, err := compiler.CheckExplicitFiles([]semantic.File{file})
	if err != nil || !r.Verified {
		t.Fatalf("compile: %v %+v", err, r.Semantic)
	}
	out, err := interpreter.Execute(r.MIR, "main::main")
	if err != nil || out.Value != int32(42) {
		t.Fatalf("interpret: %+v %v", out, err)
	}
	runExpandedLanguageNative(t, r, "main::main", "")
}
func TestStaticInterfaceRejectsInvalidSources(t *testing.T) {
	for name, text := range map[string]string{
		"missing":         strings.Replace(staticInterfaceSource, "@implements(transform,i32)", "", 1),
		"duplicate":       strings.Replace(staticInterfaceSource, "fn T algorithm", `@implements(transform,i32) fn i32 other(i32 v){return v;}fn T algorithm`, 1),
		"wrong signature": strings.Replace(staticInterfaceSource, "fn i32 twice(i32 value)", "fn i64 twice(i64 value)", 1),
		"body":            strings.Replace(staticInterfaceSource, "(T value){}", "(T value){return value;}", 1),
		"unsafe":          strings.Replace(staticInterfaceSource, "fn i32 twice", "unsafe fn i32 twice", 1),
		"unmarked":        strings.Replace(staticInterfaceSource, "@interface ", "", 1),
		"arguments":       strings.Replace(staticInterfaceSource, "@interface ", "@interface(1) ", 1),
	} {
		t.Run(name, func(t *testing.T) {
			r, err := compiler.CheckExplicitFiles([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", text)})
			if err != nil || r.Verified || len(r.Semantic.Diagnostics) == 0 {
				t.Fatalf("invalid operation did not give source diagnostic: %v %+v", err, r.Semantic)
			}
		})
	}
}
