package semanticsnapshot

import (
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
)

func TestDocumentationOwnersSeparateNamesSharingSourceSpan(t *testing.T) {
	metadata := map[hir.PackageID]packageMetadata{hir.DefaultPackage: {instance: "instance"}}
	span := hir.Span{Package: hir.DefaultPackage, Source: "main.fy", StartByte: 10, EndByte: 30}
	doc := hir.Span{Package: hir.DefaultPackage, Source: "main.fy", StartByte: 0, EndByte: 9}
	key := DeclarationKey{PackageInstance: "instance", Module: "main", Kind: DeclarationFunction, Name: "first"}
	stable := Span{PackageInstance: "instance", Source: "main.fy", StartByte: 10, EndByte: 30}
	declarations := []Declaration{{Key: key, Span: stable}, {Key: key, Span: stable}}
	declarations[1].Key.Name = "second"
	program := hir.Package{Modules: []hir.Module{{Package: hir.DefaultPackage, Path: "main", Functions: []hir.Function{
		{Name: "first", Span: span, DocumentationSpan: doc},
		{Name: "second", Span: span},
	}}}}
	got, err := buildDocumentationSpans(program, metadata, declarations)
	if err != nil || len(got) != 1 || got[0].Declaration != key || got[0].DeclarationSpan != stable {
		t.Fatalf("shared-span identity lost: %+v %v", got, err)
	}
	for _, test := range []struct {
		name, match string
		mutate      func(*hir.Package, *[]Declaration)
	}{
		{"wrong name", "no exact snapshot declaration owner", func(p *hir.Package, _ *[]Declaration) { p.Modules[0].Functions[0].Name = "missing" }},
		{"wrong module", "no exact snapshot declaration owner", func(p *hir.Package, _ *[]Declaration) { p.Modules[0].Path = "other" }},
		{"wrong span", "no exact snapshot declaration owner", func(p *hir.Package, _ *[]Declaration) { p.Modules[0].Functions[0].Span.EndByte++ }},
		{"duplicate declaration", "documentation owner is duplicated", func(_ *hir.Package, d *[]Declaration) { *d = append(*d, (*d)[0]) }},
		{"duplicate attachment", "repeats documentation", func(p *hir.Package, _ *[]Declaration) {
			p.Modules[0].Functions = append(p.Modules[0].Functions, p.Modules[0].Functions[0])
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			copyProgram := hir.Package{Modules: []hir.Module{{Package: hir.DefaultPackage, Path: "main", Functions: append([]hir.Function(nil), program.Modules[0].Functions...)}}}
			copyDeclarations := append([]Declaration(nil), declarations...)
			test.mutate(&copyProgram, &copyDeclarations)
			if _, err := buildDocumentationSpans(copyProgram, metadata, copyDeclarations); err == nil || !strings.Contains(err.Error(), test.match) {
				t.Fatalf("invalid owner accepted or error changed: %v", err)
			}
		})
	}
}
