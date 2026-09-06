package compiler

import (
	"reflect"
	"testing"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/diagnostic"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/genericsource"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/semantic"
)

func TestGenericSignatureDetachesEveryTypeEdge(t *testing.T) {
	leaf := ast.TypeReference{Kind: ast.TypeKindNamed, Path: "T"}
	shape := ast.TypeReference{Kind: ast.TypeKindForeignFunction, ABI: "C", Element: &leaf, Result: &leaf,
		Arguments: []ast.TypeReference{leaf}, Parameters: []ast.TypeReference{leaf}, Length: &ast.TypeLength{Text: "2"}}
	original := GenericSignature{Unsafe: true, Arguments: []ast.Parameter{{Name: "value", Type: shape}}, Result: shape}
	original.Attributes = []ast.Attribute{{Name: "effects", Arguments: []ast.AttributeArgument{{Raw: "console"}}}}
	want := cloneGenericSignature(original)
	copy := cloneGenericSignature(original)
	if !reflect.DeepEqual(copy, original) {
		t.Fatal("cloning changed written signature data")
	}
	mutate := func(value *ast.TypeReference) {
		value.Element.Path = "changed"
		value.Result.Path = "changed"
		value.Arguments[0].Path = "changed"
		value.Parameters[0].Path = "changed"
		value.Length.Text = "99"
	}
	mutate(&copy.Arguments[0].Type)
	mutate(&copy.Result)
	copy.Arguments[0].Name = "changed"
	copy.Attributes[0].Arguments[0].Raw = "changed"
	if !reflect.DeepEqual(original, want) {
		t.Fatal("signature cloning leaked a mutable edge")
	}
}

func TestGenericProvenanceRejectsMismatchedOriginalHeaders(t *testing.T) {
	span := diagnostic.Span{StartByte: 0, EndByte: 20}
	nameSpan := diagnostic.Span{StartByte: 5, EndByte: 7}
	fn := ast.Function{Name: "id", Span: span, NameSpan: nameSpan, TypeParameters: []string{"T"}}
	owner := genericplan.PackageID("owner")
	expansion := genericsource.PackageExpansion{Declarations: []genericsource.DeclarationOrigin{{
		ID: "owner::main::id", Owner: owner, Module: "main", Name: "id", Span: span, NameSpan: nameSpan, Parameters: []string{"T"},
	}}}
	owners := map[hir.PackageID]genericplan.PackageID{hir.DefaultPackage: owner}
	for _, change := range []struct {
		name  string
		apply func(*ast.Module)
	}{
		{"missing", func(m *ast.Module) { m.Functions = nil }},
		{"duplicate", func(m *ast.Module) { m.Functions = append(m.Functions, fn) }},
		{"owner", func(m *ast.Module) { m.Functions[0].Owner = "Thing" }},
		{"span", func(m *ast.Module) { m.Functions[0].Span.EndByte-- }},
		{"name span", func(m *ast.Module) { m.Functions[0].NameSpan.StartByte++ }},
		{"visibility", func(m *ast.Module) { m.Functions[0].Public = true }},
		{"binders", func(m *ast.Module) { m.Functions[0].TypeParameters = []string{"U"} }},
	} {
		t.Run(change.name, func(t *testing.T) {
			module := &ast.Module{Path: "main", SourceSize: 20, Functions: []ast.Function{fn}}
			input := semantic.PackageInput{Files: []semantic.File{{Package: hir.DefaultPackage, Path: "main.fy", Module: module}}}
			if _, err := genericProvenance(input, owners, expansion, nil); err != nil {
				t.Fatalf("baseline: %v", err)
			}
			change.apply(module)
			if _, err := genericProvenance(input, owners, expansion, nil); err == nil {
				t.Fatal("mismatched original header was accepted")
			}
		})
	}
}
