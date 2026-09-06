package semanticsnapshot_test

import (
	"reflect"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/semanticsnapshot"
	"github.com/0disoft/fairyc/internal/symbolidentity"
)

func TestGenericNominalHeaderUsesCompilerScopeAndStableCoordinates(t *testing.T) {
	build := func(rootID, depID hir.PackageID, depUID, alias, name string) ([]semanticsnapshot.Declaration, []symbolidentity.Symbol, error) {
		rootText := "module main; import dep as " + alias + "; fn T keep<T>(" + alias + "::" + name + " value, T result) { return result; } fn i32 main() { return 0; }"
		depText := "module dep; pub struct Thing { i32 value; } struct Hidden { i32 value; }"
		root := packageInput(t, rootID, "example.generic.app", "app", "1", "main.fy", rootText, "root")
		dep := packageInput(t, depID, depUID, "dep", "1", "dep.fy", depText, "dependency")
		root.Dependencies = []hir.PackageID{depID}
		input := semantic.PackageInput{Files: []semantic.File{
			{Package: rootID, Path: "main.fy", Module: analyzeSyntax(t, "main.fy", rootText)},
			{Package: depID, Path: "dep.fy", Module: analyzeSyntax(t, "dep.fy", depText)},
		}, Imports: []semantic.PackageImport{{FromPackage: rootID, ImportPath: "dep", ToPackage: depID}}}
		result, _, err := compiler.CheckGenericPackages(input, map[hir.PackageID]packagegraph.CoordinateID{rootID: root.Coordinate, depID: dep.Coordinate}, genericplan.DefaultPlanLimits())
		if err != nil || !result.Verified {
			t.Fatalf("compiler: %+v %v", result.Semantic, err)
		}
		facts := result.GenericProvenance.NominalTypes()
		before := result.GenericProvenance.NominalTypes()
		if len(facts) == 0 {
			t.Fatal("compiler dropped nominal scope facts")
		}
		facts[0].Name = "corrupted"
		result.Semantic.NominalTypes[0].Name = "also-corrupted"
		if !reflect.DeepEqual(before, result.GenericProvenance.NominalTypes()) {
			t.Fatal("provenance aliases nominal scope facts")
		}
		declarations, err := semanticsnapshot.GenericDeclarations(result.GenericProvenance, []semanticsnapshot.PackageInput{root, dep})
		if err != nil {
			return nil, nil, err
		}
		if len(declarations) != 1 || len(declarations[0].Parameters) != 2 || declarations[0].Parameters[0].Type.Kind != semanticsnapshot.TypeNominal {
			t.Fatalf("missing nominal declaration: %+v", declarations)
		}
		typ := declarations[0].Parameters[0].Type
		if typ.Name != "dep::Thing" || typ.PackageCoordinate != dep.Coordinate.String() {
			t.Fatalf("nominal origin mismatch: %+v", typ)
		}
		symbols, err := symbolidentity.FromDeclarations(declarations,
			map[string]string{root.Instance.String(): root.UID, dep.Instance.String(): dep.UID},
			map[string]string{root.Coordinate.String(): root.UID, dep.Coordinate.String(): dep.UID})
		return declarations, symbols, err
	}
	_, left, err := build(1, 2, "example.dep", "d", "Thing")
	if err != nil || len(left) != 1 {
		t.Fatalf("baseline: %+v %v", left, err)
	}
	_, right, err := build(91, 7, "example.dep", "renamed", "Thing")
	if err != nil || len(right) != 1 || left[0].ID != right[0].ID {
		t.Fatalf("alias/package-number instability: %+v %v", right, err)
	}
	_, other, err := build(1, 2, "example.other-dep", "d", "Thing")
	if err != nil || len(other) != 1 || left[0].ID == other[0].ID {
		t.Fatalf("same spelling collapsed distinct provider types: %+v %v", other, err)
	}
	if _, _, err := build(1, 2, "example.dep", "d", "Hidden"); err == nil {
		t.Fatal("private imported nominal type was guessed")
	}
}
