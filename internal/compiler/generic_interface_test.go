package compiler_test

import (
	"reflect"
	"slices"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/moduleinterface"
	"github.com/0disoft/fairyc/internal/packagegraph"
)

func TestGenericInterfaceRetainsUninstantiatedTemplates(t *testing.T) {
	input, coordinates := genericPackageInput(t)
	used, err := compiler.AnalyzeGenericPackages(input, coordinates, genericplan.DefaultPlanLimits())
	if err != nil || len(used.Interfaces) != 3 || len(used.Plan.Specializations) != 2 {
		t.Fatalf("used: %+v %v", used, err)
	}
	input.Files[0] = parseCompilerFile(t, 1, "main.fy", "module main; import util as u; fn i32 main() { return 0; }")
	unused, err := compiler.AnalyzeGenericPackages(input, coordinates, genericplan.DefaultPlanLimits())
	if err != nil || len(unused.Plan.Specializations) != 0 || !reflect.DeepEqual(used.Interfaces, unused.Interfaces) {
		t.Fatalf("instantiation leaked into interfaces: %v", err)
	}
	input.Files[2] = parseCompilerFile(t, 3, "core.fy", "module core; pub fn T id<T>(T value) { let T copy = value; return copy; }")
	changed, err := compiler.AnalyzeGenericPackages(input, coordinates, genericplan.DefaultPlanLimits())
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []hir.PackageID{2, 3} {
		if changed.Interfaces[id] == unused.Interfaces[id] {
			t.Fatalf("unused exported template change missing from package %d", id)
		}
	}
	if changed.Interfaces[1] != unused.Interfaces[1] {
		t.Fatal("private caller changed source interface")
	}
	for i := range input.Files {
		input.Files[i].Package += 10
	}
	for i := range input.Imports {
		input.Imports[i].FromPackage += 10
		input.Imports[i].ToPackage += 10
	}
	renumbered := map[hir.PackageID]packagegraph.CoordinateID{}
	for id, coordinate := range coordinates {
		renumbered[id+10] = coordinate
	}
	slices.Reverse(input.Files)
	slices.Reverse(input.Imports)
	reordered, err := compiler.AnalyzeGenericPackages(input, renumbered, genericplan.DefaultPlanLimits())
	if err != nil {
		t.Fatal(err)
	}
	for id, hash := range changed.Interfaces {
		if reordered.Interfaces[id+10] != hash {
			t.Fatal("package numbering or input order changed interface")
		}
	}
}

func TestGenericInterfaceTracksReachablePrivateTemplates(t *testing.T) {
	input, coordinates := genericPackageInput(t)
	input.Files[0] = parseCompilerFile(t, 1, "main.fy", "module main; import util as u; fn i32 main() { return 0; }")
	compile := func(helperBody, unusedBody string) compiler.GenericAnalysis {
		t.Helper()
		input.Files[2] = parseCompilerFile(t, 3, "core.fy", "module core; pub fn T id<T>(T value) { return copy<T>(value); } fn T copy<T>(T x) { "+helperBody+" } fn T unrelated<T>(T x) { "+unusedBody+" }")
		result, err := compiler.AnalyzeGenericPackages(input, coordinates, genericplan.DefaultPlanLimits())
		if err != nil || len(result.Interfaces) != 3 {
			t.Fatalf("analyze: %v", err)
		}
		return result
	}
	first := compile("return x;", "return x;")
	unrelated := compile("return x;", "let T spare = x; return spare;")
	if !reflect.DeepEqual(first.Interfaces, unrelated.Interfaces) {
		t.Fatal("unreachable private template changed public interface")
	}
	reachable := compile("let T result = x; return result;", "return x;")
	for _, id := range []hir.PackageID{2, 3} {
		if first.Interfaces[id] == reachable.Interfaces[id] {
			t.Fatal("reachable private template missing from closure")
		}
	}
}

func TestGenericInterfaceAbsentOnSemanticFailure(t *testing.T) {
	input, coordinates := genericPackageInput(t)
	input.Files[2] = parseCompilerFile(t, 3, "core.fy", "module core; pub fn T id<T>(T value) { return false; }")
	result, err := compiler.AnalyzeGenericPackages(input, coordinates, genericplan.DefaultPlanLimits())
	if err != nil || len(result.Analysis.Semantic.Diagnostics) == 0 || result.Interfaces != nil {
		t.Fatalf("rejected source published interfaces: %+v %v", result, err)
	}
	if len(result.Analysis.GenericProvenance.PackageInterfaces()) != 0 {
		t.Fatal("rejected analysis retained package interfaces")
	}
}

func TestGenericPackageInterfacesSurviveVerifiedLoweringWithoutAliasing(t *testing.T) {
	input, coordinates := genericPackageInput(t)
	analysis, err := compiler.AnalyzeGenericPackages(input, coordinates, genericplan.DefaultPlanLimits())
	if err != nil {
		t.Fatal(err)
	}
	interfaces := analysis.Analysis.GenericProvenance.PackageInterfaces()
	if len(interfaces) != 3 {
		t.Fatalf("retained interface count: %d", len(interfaces))
	}
	for i, item := range interfaces {
		if i > 0 && interfaces[i-1].Package >= item.Package {
			t.Fatal("interface order is not canonical")
		}
		if item.Coordinate != coordinates[item.Package] || item.Generic != analysis.Interfaces[item.Package] || item.Ordinary != analysis.OrdinaryInterfaces[item.Package] {
			t.Fatal("retained interface contradicts its source analysis")
		}
	}
	first := interfaces[0]
	interfaces[0] = compiler.GenericPackageInterface{}
	analysis.Interfaces[first.Package] = moduleinterface.GenericHash{}
	analysis.OrdinaryInterfaces[first.Package] = moduleinterface.Hash{}
	coordinates[first.Package] = packagegraph.CoordinateID{}
	result, err := compiler.LowerAnalyzed(analysis.Analysis)
	if err != nil || !result.Verified {
		t.Fatalf("lowering: %v", err)
	}
	retained := result.GenericProvenance.PackageInterfaces()
	if len(retained) != 3 || retained[0] != first {
		t.Fatal("public maps or accessor alias compiler provenance")
	}
	retained[0] = compiler.GenericPackageInterface{}
	if result.GenericProvenance.PackageInterfaces()[0] != first {
		t.Fatal("result accessor aliases retained interface")
	}
	input, coordinates = genericPackageInput(t)
	checked, _, err := compiler.CheckGenericPackages(input, coordinates, genericplan.DefaultPlanLimits())
	if err != nil || !checked.Verified || !reflect.DeepEqual(checked.GenericProvenance.PackageInterfaces(), result.GenericProvenance.PackageInterfaces()) {
		t.Fatalf("checked result lost source interfaces: %v", err)
	}
}
