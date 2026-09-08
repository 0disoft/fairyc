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

const nominalPackageClient = `module main;
import genericlib as g;
import types as x;
fn i32 observe(i32 value) { return value; }
fn i32 check_drop() { return 1; }
struct Point { i32 value; }
resource Token { i32 value; cleanup { let i32 marked = observe(65); } }
fn i32 exercise() {
    let Point local = g::relay<Point>(Point { .value = 20 });
    let x::Point external = g::pass<x::Point>(x::Point { .value = 21 });
    let g::Box<Point> box = g::wrap<Point>(local);
    let Token owned = Token { .value = box.value.value + external.value };
    let Token moved = g::pass<Token>(owned);
    return moved.value;
}
fn i32 main() { return exercise() + check_drop(); }
`

const nominalPackageLibrary = `module genericlib;
pub struct Box<T> { T value; }
pub fn T pass<T>(T value) { return value; }
pub fn T relay<T>(T value) { return pass<T>(value); }
pub fn Box<T> wrap<T>(T value) { return Box<T> { .value = value }; }
pub fn usize size<T>() { return storage_size<T>(); }
`

func nominalPackageInput(t *testing.T, client semantic.File) (semantic.PackageInput, map[hir.PackageID]packagegraph.CoordinateID) {
	t.Helper()
	input := semantic.PackageInput{Files: []semantic.File{client,
		parseCompilerFile(t, 2, "genericlib.fy", nominalPackageLibrary),
		parseCompilerFile(t, 3, "types.fy", "module types; pub struct Point { i32 value; } resource Secret { i32 value; cleanup {} }"),
	}, Imports: []semantic.PackageImport{{FromPackage: client.Package, ImportPath: "genericlib", ToPackage: 2}, {FromPackage: client.Package, ImportPath: "types", ToPackage: 3}}}
	first := genericCoordinate(t)
	second, third := first, first
	second[0] ^= 1
	third[0] ^= 2
	return input, map[hir.PackageID]packagegraph.CoordinateID{client.Package: first, 2: second, 3: third}
}

func checkNominalPackages(t *testing.T, file semantic.File) (compiler.Result, error) {
	t.Helper()
	input, coordinates := nominalPackageInput(t, file)
	result, _, err := compiler.CheckNominalStoragePackages(input, coordinates, genericplan.DefaultPlanLimits())
	return result, err
}

func TestPrivateNominalBindingPackagesSource(t *testing.T) {
	file := parseCompilerFile(t, hir.DefaultPackage, "main.fy", nominalPackageClient)
	input, coordinates := nominalPackageInput(t, file)
	before, _ := json.Marshal(input)
	result, plan, err := compiler.CheckNominalStoragePackages(input, coordinates, genericplan.DefaultPlanLimits())
	if err != nil || !result.Verified {
		t.Fatalf("nominal package source: %v %+v", err, result.Semantic)
	}
	seen := map[genericplan.TypeID]bool{}
	for _, specialization := range plan.Specializations {
		for _, binding := range specialization.Bindings {
			if strings.HasPrefix(string(binding.Type), "source_nominal_v0::") {
				seen[binding.Type] = true
			}
		}
	}
	if len(seen) != 3 {
		t.Fatalf("nominal identities collapsed: %+v", seen)
	}
	outcome, err := interpreter.ExecuteWithOptions(result.MIR, "main::main", interpreter.Options{
		TestIntrinsics: map[string]interpreter.TestIntrinsicKind{"main::observe": interpreter.TestIntrinsicWriteStdoutByte}, OutputLimit: 10,
	})
	if err != nil || outcome.Kind != interpreter.OutcomeReturn || outcome.Value != 42 || len(outcome.DropTrace) != 1 || string(outcome.Stdout) != "A" {
		t.Fatalf("nominal package ownership: %+v %v", outcome, err)
	}
	after, _ := json.Marshal(input)
	if string(before) != string(after) {
		t.Fatal("nominal graph mutated")
	}
	ordinary, _, err := compiler.CheckStorageGenericPackages(input, coordinates, genericplan.DefaultPlanLimits())
	if err == nil && ordinary.Verified {
		t.Fatal("scalar graph admitted nominal arguments")
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
	repeated, repeatedPlan, err := compiler.CheckNominalStoragePackages(input, renumbered, genericplan.DefaultPlanLimits())
	if err != nil || !repeated.Verified || !reflect.DeepEqual(plan, repeatedPlan) {
		t.Fatalf("nominal plan order/ID drift: %v", err)
	}
	coordinate := renumbered[13]
	coordinate[0] ^= 4
	renumbered[13] = coordinate
	rebound, reboundPlan, err := compiler.CheckNominalStoragePackages(input, renumbered, genericplan.DefaultPlanLimits())
	if err != nil || !rebound.Verified || reboundPlan.CatalogDigest == plan.CatalogDigest {
		t.Fatalf("owner rebind ignored: %v", err)
	}
	changedInput, changedCoordinates := nominalPackageInput(t, file)
	changedInput.Files[2] = parseCompilerFile(t, 3, "types.fy", "module types; pub struct Point { i32 value; } resource Secret { i32 value; cleanup {} } fn i32 dependency() { return 7; }")
	changed, changedPlan, err := compiler.CheckNominalStoragePackages(changedInput, changedCoordinates, genericplan.DefaultPlanLimits())
	if err != nil || !changed.Verified || changedPlan.CatalogDigest == plan.CatalogDigest {
		t.Fatalf("closed definition graph change ignored: %v", err)
	}
	for _, before := range plan.Specializations {
		for _, after := range changedPlan.Specializations {
			if before.Declaration == after.Declaration && reflect.DeepEqual(before.Bindings, after.Bindings) && before.ID == after.ID {
				t.Fatal("nominal specialization retained stale closed-graph identity")
			}
		}
	}
	runGenericAggregateNativeWithCompiler(t, nominalPackageClient, func(file semantic.File) (compiler.Result, error) { return checkNominalPackages(t, file) })
}

func TestPrivateNominalBindingPackagesStorage(t *testing.T) {
	text := strings.Replace(nominalPackageClient, "fn i32 exercise() {", "fn i32 exercise() { assert(g::size<Point>() == 4 && g::size<x::Point>() == 4 && g::size<Token>() == 4);", 1)
	file := parseCompilerFile(t, hir.DefaultPackage, "main.fy", text)
	result, err := checkNominalPackages(t, file)
	if err != nil || !result.Verified {
		t.Fatalf("nominal layout source: %v %+v", err, result.Semantic)
	}
	if _, err := interpreter.Execute(result.MIR, "main::main"); err == nil {
		t.Fatal("targetless interpreter admitted layouts")
	}
	runGenericAggregateNativeWithCompiler(t, text, func(file semantic.File) (compiler.Result, error) { return checkNominalPackages(t, file) })
}

func TestPrivateNominalBindingPackagesVisibility(t *testing.T) {
	for name, text := range map[string]string{
		"private type": strings.Replace(nominalPackageClient, "g::pass<Token>(owned)", "g::pass<x::Secret>(owned)", 1),
		"wrong owner":  strings.Replace(nominalPackageClient, "g::pass<Token>(owned)", "g::pass<x::Point>(owned)", 1),
		"moved donor":  strings.Replace(nominalPackageClient, "return moved.value;", "return owned.value;", 1),
	} {
		t.Run(name, func(t *testing.T) {
			result, err := checkNominalPackages(t, parseCompilerFile(t, hir.DefaultPackage, "main.fy", text))
			if err == nil && result.Verified {
				t.Fatal("invalid nominal use admitted")
			}
		})
	}
	input, coordinates := nominalPackageInput(t, parseCompilerFile(t, hir.DefaultPackage, "main.fy", nominalPackageClient))
	input.Files[1] = parseCompilerFile(t, 2, "genericlib.fy", strings.Replace(nominalPackageLibrary, "pub fn T pass<T>(T value) { return value; }", "pub fn T pass<T>(T value) { let i32 hidden = value.value; return value; }", 1))
	result, _, err := compiler.CheckNominalStoragePackages(input, coordinates, genericplan.DefaultPlanLimits())
	if err != nil {
		t.Fatal(err)
	}
	if result.Verified || !slices.ContainsFunc(result.Semantic.Diagnostics, func(d semantic.Diagnostic) bool { return d.Code == "E0103" }) {
		t.Fatal("generic provider gained resource field access")
	}
}
