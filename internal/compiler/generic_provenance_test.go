package compiler_test

import (
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/semantic"
)

func TestGenericProvenanceRetainsUnusedDeclarationsAndDetachedBindings(t *testing.T) {
	text := "module main; fn T id<T>(T value) { return value; } fn T unused<T>(T value) { return value; } fn i32 main() { return id<i32>(42); }"
	file := parseCompilerFile(t, hir.DefaultPackage, "main.fy", text)
	result, err := compiler.CheckExplicitFiles([]semantic.File{file})
	if err != nil || !result.Verified {
		t.Fatalf("compile: %+v %v", result.Semantic, err)
	}
	declarations := result.GenericProvenance.Declarations()
	origins := result.GenericProvenance.Specializations()
	if len(declarations) != 2 || len(origins) != 1 {
		t.Fatalf("provenance: %+v %+v", declarations, origins)
	}
	for _, declaration := range declarations {
		if declaration.Span.Source != "main.fy" || text[declaration.NameSpan.StartByte:declaration.NameSpan.EndByte] != declaration.Name || len(declaration.Parameters) != 1 || declaration.Parameters[0] != "T" {
			t.Fatalf("source declaration: %+v", declaration)
		}
		if len(declaration.Signature.Arguments) != 1 || declaration.Signature.Arguments[0].Name != "value" ||
			declaration.Signature.Arguments[0].Type.Path != "T" || declaration.Signature.Result.Path != "T" || declaration.Signature.Unsafe {
			t.Fatalf("original signature was lost: %+v", declaration.Signature)
		}
	}
	if origins[0].Declaration != declarations[0].ID || declarations[0].Name != "id" || len(origins[0].Bindings) != 1 || origins[0].Bindings[0] != (genericplan.Binding{Parameter: "T", Type: "i32"}) {
		t.Fatalf("binding: %+v", origins)
	}
	found := false
	for _, function := range result.MIR.Functions {
		found = found || function.Name == "main::"+origins[0].GeneratedName
	}
	if !found {
		t.Fatal("origin does not identify an emitted function")
	}
	declarations[0].Parameters[0] = "changed"
	declarations[0].Signature.Arguments[0].Type.Path = "bool"
	file.Module.Functions[0].Parameters[0].Type.Path = "u32"
	file.Module.Functions[0].ResultType.Path = "u32"
	origins[0].Bindings[0].Type = "bool"
	if result.GenericProvenance.Declarations()[0].Parameters[0] != "T" || result.GenericProvenance.Specializations()[0].Bindings[0].Type != "i32" {
		t.Fatal("provenance getter leaked mutable storage")
	}
	if got := result.GenericProvenance.Declarations()[0].Signature; got.Arguments[0].Type.Path != "T" || got.Result.Path != "T" {
		t.Fatalf("signature aliases the source or getter: %+v", got)
	}
	invalid := parseCompilerFile(t, hir.DefaultPackage, "main.fy", strings.Replace(text, "id<i32>", "id<Unknown>", 1))
	rejected, err := compiler.CheckExplicitFiles([]semantic.File{invalid})
	if err != nil || rejected.Verified || len(rejected.GenericProvenance.Declarations()) != 0 || len(rejected.GenericProvenance.Specializations()) != 0 {
		t.Fatalf("rejection published provenance: %+v %v", rejected.Semantic, err)
	}
}
