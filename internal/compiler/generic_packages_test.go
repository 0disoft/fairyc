package compiler_test

import (
	"encoding/json"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/semantic"
)

func genericPackageInput(t *testing.T) (semantic.PackageInput, map[hir.PackageID]packagegraph.CoordinateID) {
	t.Helper()
	input := semantic.PackageInput{
		Files: []semantic.File{
			parseCompilerFile(t, 1, "main.fy", "module main; import util as u; fn i32 main() { return u::wrap<i32>(42); }"),
			parseCompilerFile(t, 2, "util.fy", "module util; import core as c; pub fn T wrap<T>(T value) { return c::id<T>(value); }"),
			parseCompilerFile(t, 3, "core.fy", "module core; pub fn T id<T>(T value) { return value; }"),
		},
		Imports: []semantic.PackageImport{{FromPackage: 1, ImportPath: "util", ToPackage: 2}, {FromPackage: 2, ImportPath: "core", ToPackage: 3}},
	}
	coordinates := map[hir.PackageID]packagegraph.CoordinateID{}
	for _, file := range input.Files {
		coordinate, err := packagegraph.ComputeCoordinateID(packagegraph.Coordinate{PackageUID: file.Module.Path, VersionIdentity: "1"})
		if err != nil {
			t.Fatal(err)
		}
		coordinates[file.Package] = coordinate
	}
	return input, coordinates
}

func TestGenericPackagesResolveAndExecute(t *testing.T) {
	input, coordinates := genericPackageInput(t)
	before, _ := json.Marshal(input)
	result, plan, err := compiler.CheckGenericPackages(input, coordinates, genericplan.DefaultPlanLimits())
	if err != nil || !result.Verified || len(plan.Specializations) != 2 {
		t.Fatalf("compile: %v %+v %+v", err, result.Semantic, plan)
	}
	outcome, err := interpreter.Execute(result.MIR, hir.QualifiedModulePath(1, "main")+"::main")
	if err != nil || outcome.Kind != interpreter.OutcomeReturn || outcome.Value != 42 {
		t.Fatalf("execute: %+v %v", outcome, err)
	}
	after, _ := json.Marshal(input)
	if string(before) != string(after) {
		t.Fatal("package input mutated")
	}
	ordinary, err := compiler.CheckPackages(input)
	if err != nil || ordinary.Verified {
		t.Fatalf("ordinary compiler admitted generic packages: %v", err)
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
	result, reordered, err := compiler.CheckGenericPackages(input, renumbered, genericplan.DefaultPlanLimits())
	if err != nil || !result.Verified || !reflect.DeepEqual(plan, reordered) {
		t.Fatalf("renumbered/reordered plan changed: %v %+v", err, result.Semantic)
	}
}

func TestGenericPackagesRejectInvalidBoundaries(t *testing.T) {
	for _, name := range []string{"private", "missing-coordinate", "duplicate-coordinate", "missing-provider", "conflicting-provider", "implicit", "alias-collision", "cycle", "global-budget"} {
		t.Run(name, func(t *testing.T) {
			input, coordinates := genericPackageInput(t)
			limits := genericplan.DefaultPlanLimits()
			switch name {
			case "private":
				input.Files[1].Module.Functions[0].Public = false
			case "missing-coordinate":
				delete(coordinates, 2)
			case "duplicate-coordinate":
				coordinates[2] = coordinates[1]
			case "missing-provider":
				input.Imports = input.Imports[1:]
			case "conflicting-provider":
				input.Imports = append(input.Imports, semantic.PackageImport{FromPackage: 1, ImportPath: "util", ToPackage: 3})
			case "implicit":
				input.Files[0] = parseCompilerFile(t, 1, "main.fy", "module main; import util as u; fn i32 main() { return u::wrap(42); }")
			case "alias-collision":
				input.Files[1].Module.Imports[0].Alias = "wrap"
			case "cycle":
				input.Files[2] = parseCompilerFile(t, 3, "core.fy", "module core; import util as u; pub fn T id<T>(T value) { return u::wrap<T>(value); }")
				input.Imports = append(input.Imports, semantic.PackageImport{FromPackage: 3, ImportPath: "util", ToPackage: 2})
			case "global-budget":
				limits.MaxSpecializations = 1
			}
			result, _, err := compiler.CheckGenericPackages(input, coordinates, limits)
			if err == nil || result.Verified {
				t.Fatalf("invalid package boundary admitted: %+v", result.Semantic)
			}
			if name == "global-budget" && genericplan.ErrorCode(err) != genericplan.CodePlanningBudgetExceeded {
				t.Fatalf("wrong budget failure: %v", err)
			}
		})
	}
}

func TestGenericPackagesBindEqualModulePathsToExactProvider(t *testing.T) {
	input, coordinates := genericPackageInput(t)
	input.Files = append(input.Files, parseCompilerFile(t, 4, "core.fy", "module core; pub fn T id<T>(T value) { return value; }"))
	coordinate := coordinates[3]
	coordinate[0] ^= 1
	coordinates[4] = coordinate
	_, first, err := compiler.CheckGenericPackages(input, coordinates, genericplan.DefaultPlanLimits())
	if err != nil {
		t.Fatal(err)
	}
	input.Imports[1].ToPackage = 4
	result, second, err := compiler.CheckGenericPackages(input, coordinates, genericplan.DefaultPlanLimits())
	if err != nil || !result.Verified {
		t.Fatalf("second provider: %v %+v", err, result.Semantic)
	}
	for i := range first.Specializations {
		if first.Specializations[i].ID == second.Specializations[i].ID {
			t.Fatal("provider selection omitted from transitive identity")
		}
	}
	input.Files[3] = parseCompilerFile(t, 4, "core.fy", "module core; pub fn T id<T>(T value) { let T copy = value; return copy; }")
	_, changed, err := compiler.CheckGenericPackages(input, coordinates, genericplan.DefaultPlanLimits())
	if err != nil {
		t.Fatal(err)
	}
	for i := range second.Specializations {
		if second.Specializations[i].ID == changed.Specializations[i].ID {
			t.Fatal("imported template body change omitted from identity")
		}
	}
	for _, specialization := range changed.Specializations {
		if !strings.Contains(string(specialization.Declaration), "package_coordinate_v1_") {
			t.Fatal("disposable owner in plan")
		}
	}
}

func TestGenericPackagesIdentityIncludesOrdinaryImportBinding(t *testing.T) {
	input, coordinates := genericPackageInput(t)
	input.Files[1] = parseCompilerFile(t, 2, "util.fy", "module util; import core as util; fn T value<T>(T x) { return x; } pub fn i32 wrap<T>(T argument) { return util::value(); }")
	input.Files[2] = parseCompilerFile(t, 3, "core.fy", "module core; pub fn i32 value() { return 1; }")
	input.Files = append(input.Files, parseCompilerFile(t, 4, "core.fy", "module core; pub fn i32 value() { return 2; }"))
	coordinate := coordinates[3]
	coordinate[0] ^= 1
	coordinates[4] = coordinate
	var previous genericplan.Plan
	for _, provider := range []hir.PackageID{3, 4} {
		input.Imports[1].ToPackage = provider
		result, plan, err := compiler.CheckGenericPackages(input, coordinates, genericplan.DefaultPlanLimits())
		if err != nil || !result.Verified || len(plan.Specializations) != 1 {
			t.Fatalf("compile: %v %+v", err, result.Semantic)
		}
		outcome, err := interpreter.Execute(result.MIR, hir.QualifiedModulePath(1, "main")+"::main")
		if err != nil || outcome.Kind != interpreter.OutcomeReturn || outcome.Value != int32(provider-2) {
			t.Fatalf("provider: %+v %v", outcome, err)
		}
		if len(previous.Specializations) != 0 && previous.Specializations[0].ID == plan.Specializations[0].ID {
			t.Fatal("ordinary import binding omitted from template identity")
		}
		previous = plan
	}
}
