package semanticsnapshot_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/semanticsnapshot"
	"github.com/0disoft/fairyc/internal/symbolidentity"
)

func TestGenericAggregateHeadersResolveOriginalTemplateScopes(t *testing.T) {
	build := func(rootID, depID hir.PackageID, uid, alias, binder string) map[string]symbolidentity.ID {
		rootText := "module main; import dep as " + alias + "; resource Holder<T> { " + alias + "::Cell<T> payload; cleanup {} } fn T unused<T>(" + alias + "::Cell<T> value, T result) { return result; } fn i32 main() { return 0; }"
		depText := "module dep; pub struct Cell<T> { T value; }"
		rootText, depText = strings.ReplaceAll(rootText, "T", binder), strings.ReplaceAll(depText, "T", binder)
		root := packageInput(t, rootID, "example.aggregate.app", "app", "1", "main.fy", rootText, "root")
		dep := packageInput(t, depID, uid, "dep", "1", "dep.fy", depText, "dependency")
		root.Dependencies = []hir.PackageID{depID}
		input := semantic.PackageInput{Files: []semantic.File{
			{Package: rootID, Path: "main.fy", Module: analyzeSyntax(t, "main.fy", rootText)},
			{Package: depID, Path: "dep.fy", Module: analyzeSyntax(t, "dep.fy", depText)},
		}, Imports: []semantic.PackageImport{{FromPackage: rootID, ImportPath: "dep", ToPackage: depID}}}
		result, _, err := compiler.CheckGenericPackages(input, map[hir.PackageID]packagegraph.CoordinateID{rootID: root.Coordinate, depID: dep.Coordinate}, genericplan.DefaultPlanLimits())
		if err != nil || !result.Verified {
			t.Fatalf("compile: %v %+v", err, result.Semantic)
		}
		facts := result.GenericProvenance.TemplateTypes()
		before := result.GenericProvenance.TemplateTypes()
		if len(facts) != 3 {
			t.Fatalf("template scope count: %+v", facts)
		}
		facts[0].Name = "corrupted"
		if !reflect.DeepEqual(before, result.GenericProvenance.TemplateTypes()) {
			t.Fatal("template facts alias getter")
		}
		declarations, err := semanticsnapshot.GenericDeclarations(result.GenericProvenance, []semanticsnapshot.PackageInput{root, dep})
		if err != nil || len(declarations) != 3 {
			t.Fatalf("headers: %v %+v", err, declarations)
		}
		for _, d := range declarations {
			if d.Key.Name == "Holder" {
				if d.Key.Kind != semanticsnapshot.DeclarationResource || d.DropKind != "source-declared" || d.Type.Kind != semanticsnapshot.TypeNominal || len(d.Fields) != 1 || len(d.Parameters) != 0 {
					t.Fatalf("resource header: %+v", d)
				}
				typ := d.Fields[0].Type
				if typ.Kind != semanticsnapshot.TypeNominal || typ.Name != "dep::Cell" || typ.PackageCoordinate != dep.Coordinate.String() || len(typ.Arguments) != 1 || typ.Arguments[0].Kind != semanticsnapshot.TypeParameter || typ.Arguments[0].Name != "0" {
					t.Fatalf("imported template: %+v", typ)
				}
				d.Fields[0].Type.Arguments[0].Name = "99"
			}
		}
		// Snapshot output is detached from both the compiler and subsequent reads.
		declarations, err = semanticsnapshot.GenericDeclarations(result.GenericProvenance, []semanticsnapshot.PackageInput{root, dep})
		if err != nil {
			t.Fatal(err)
		}
		symbols, err := symbolidentity.FromDeclarations(declarations, map[string]string{root.Instance.String(): root.UID, dep.Instance.String(): dep.UID}, map[string]string{root.Coordinate.String(): root.UID, dep.Coordinate.String(): dep.UID})
		if err != nil {
			t.Fatal(err)
		}
		ids := map[string]symbolidentity.ID{}
		for _, symbol := range symbols {
			ids[symbol.Declaration.Name] = symbol.ID
		}
		return ids
	}
	baseline := build(1, 2, "example.aggregate.dep", "d", "T")
	renamed := build(19, 7, "example.aggregate.dep", "renamed", "Value")
	if !reflect.DeepEqual(baseline, renamed) {
		t.Fatal("binder, alias or package numbering changed aggregate identities")
	}
	other := build(1, 2, "example.other.dep", "d", "T")
	for _, name := range []string{"Holder", "unused", "Cell"} {
		if baseline[name] == other[name] {
			t.Fatalf("provider identity omitted from %s", name)
		}
	}
}
