package semanticsnapshot_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/semanticsnapshot"
	"github.com/0disoft/fairyc/internal/symbolidentity"
)

func genericSourceDeclarations(t *testing.T, text string) ([]semanticsnapshot.Declaration, semanticsnapshot.PackageInput, error) {
	t.Helper()
	file := semantic.File{Package: hir.DefaultPackage, Path: "main.fy", Module: analyzeSyntax(t, "main.fy", text)}
	result, err := compiler.CheckExplicitFiles([]semantic.File{file})
	if err != nil || !result.Verified {
		t.Fatalf("compiler result: %+v %v", result.Semantic, err)
	}
	input := packageInput(t, hir.DefaultPackage, "example.generic.snapshot", "generic", "1.0.0", "main.fy", text, "root")
	declarations, err := semanticsnapshot.GenericDeclarations(result.GenericProvenance, []semanticsnapshot.PackageInput{input})
	return declarations, input, err
}

func TestOriginalGenericSourceProducesAlphaStableSymbolID(t *testing.T) {
	text := "module main; fn T id<T>(T value) { return value; } fn i32 main() { assert(id<i32>(42) == 42); return 0; }"
	identity := func(text string) symbolidentity.ID {
		declarations, input, err := genericSourceDeclarations(t, text)
		if err != nil || len(declarations) != 1 {
			t.Fatalf("declarations: %+v %v", declarations, err)
		}
		declaration := declarations[0]
		if declaration.Key.Name != "id" || declaration.Type.Kind != semanticsnapshot.TypeParameter || declaration.Type.Name != "0" ||
			!reflect.DeepEqual(declaration.Type, declaration.Parameters[0].Type) {
			t.Fatalf("original binder structure lost: %+v", declaration)
		}
		symbols, err := symbolidentity.FromDeclarations(declarations, map[string]string{input.Instance.String(): input.UID}, map[string]string{input.Coordinate.String(): input.UID})
		if err != nil || len(symbols) != 1 {
			t.Fatalf("identity: %+v %v", symbols, err)
		}
		return symbols[0].ID
	}
	before := identity(text)
	after := identity(strings.ReplaceAll(text, "T", "Renamed"))
	if before != after {
		t.Fatal("alpha-renaming changed the original generic SymbolId")
	}
	if identity(strings.Replace(text, "id<i32>(42)", "id<u32>(42)", 1)) != before {
		t.Fatal("observed concrete specialization changed the original generic SymbolId")
	}
	unused := strings.Replace(text, "assert(id<i32>(42) == 42); ", "", 1)
	if identity(unused) != before {
		t.Fatal("removing all specializations changed the original generic SymbolId")
	}
	if identity(strings.Replace(unused, "fn T id", "@effects(console) fn T id", 1)) == before {
		t.Fatal("declared effects were omitted from the original generic SymbolId")
	}
}

func TestGenericHeaderPreservesNestedTypesAndDeclaredEffects(t *testing.T) {
	text := "module main; @effects(console, filesystem) fn Result<T, u32> unused<T>(ref T value, T[0x2] pair) { return missing; } fn i32 main() { return 0; }"
	declarations, _, err := genericSourceDeclarations(t, text)
	if err != nil || len(declarations) != 1 {
		t.Fatalf("declarations: %+v %v", declarations, err)
	}
	d := declarations[0]
	if d.Type.Kind != semanticsnapshot.TypeResult || d.Type.Arguments[0].Kind != semanticsnapshot.TypeParameter ||
		d.Parameters[0].Type.Kind != semanticsnapshot.TypeBorrow || d.Parameters[1].Type.Kind != semanticsnapshot.TypeFixedArray ||
		d.Parameters[1].Type.Length != 2 || !reflect.DeepEqual(d.Effects, []string{"console", "filesystem"}) {
		t.Fatalf("nested signature/effects lost: %+v", d)
	}
	// The body is intentionally uninstantiated and unresolved; no executable or
	// body-validity facts are claimed by this source declaration adapter.
}

func TestGenericHeaderRejectsUnresolvedTypesAndEffects(t *testing.T) {
	for _, header := range []string{
		"fn Unknown unused<T>(T value)",
		"fn T unused<T>(T[SIZE] value)",
		"@effects(unknown) fn T unused<T>(T value)",
		"@effects(console, console) fn T unused<T>(T value)",
	} {
		t.Run(header, func(t *testing.T) {
			_, _, err := genericSourceDeclarations(t, "module main; "+header+" { return missing; } fn i32 main() { return 0; }")
			if err == nil {
				t.Fatal("unresolved source header was guessed")
			}
		})
	}
}
