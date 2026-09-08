package compiler_test

import (
	"encoding/json"
	"reflect"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/cruntime"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/genericsource"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/semantic"
)

func aggregatePackageSources(t *testing.T) []genericsource.PackageModule {
	t.Helper()
	return []genericsource.PackageModule{
		{Module: parseCompilerFile(t, 1, "main.fy", `module main; import box as b;
fn i32 main() { let b::Box<i32> value = b::make<i32>(42); return b::read<i32>(ref value); }`).Module, Owner: "app", Imports: map[string]genericplan.PackageID{"box": "library"}},
		{Module: parseCompilerFile(t, 2, "box.fy", `module box; import helper as h;
pub resource Box<T> { T value; cleanup { let i32 observed = h::finish<i32>(1); } }
pub fn Box<T> make<T>(T value) { return Box<T> { .value = value }; }
pub fn T read<T>(ref Box<T> value) { return value.value; }
pub struct Unused<T> { T item; }`).Module, Owner: "library", Imports: map[string]genericplan.PackageID{"helper": "helper_a"}},
		{Module: parseCompilerFile(t, 3, "helper.fy", `module helper; pub fn T finish<T>(T value) { return value; }`).Module, Owner: "helper_a"},
	}
}

func TestGenericAggregatePackageSourceExpansion(t *testing.T) {
	input := aggregatePackageSources(t)
	before, _ := json.Marshal(input)
	expanded, err := genericsource.PreparePackages(input, genericplan.DefaultPlanLimits())
	if err != nil || len(expanded.Plan.Specializations) != 4 || len(expanded.Templates) != 5 {
		t.Fatalf("expand: %v %+v", err, expanded)
	}
	after, _ := json.Marshal(input)
	if string(before) != string(after) {
		t.Fatal("source mutated")
	}
	if len(expanded.Modules[0].Resources) != 0 || len(expanded.Modules[1].Resources) != 1 || len(expanded.Modules[1].Structs) != 0 {
		t.Fatal("specialization left its defining module or unused template emitted")
	}
	prepared := semantic.PackageInput{Imports: []semantic.PackageImport{{FromPackage: 1, ImportPath: "box", ToPackage: 2}, {FromPackage: 2, ImportPath: "helper", ToPackage: 3}}}
	for i, module := range expanded.Modules {
		file := parseCompilerFile(t, hir.PackageID(i+1), input[i].Module.Path+".fy", "module "+input[i].Module.Path+";")
		file.Module = module
		prepared.Files = append(prepared.Files, file)
	}
	result, err := compiler.CheckPackages(prepared)
	if err != nil || !result.Verified {
		t.Fatalf("expanded package semantics: %v %+v", err, result.Semantic)
	}
	outcome, err := interpreter.Execute(result.MIR, hir.QualifiedModulePath(1, "main")+"::main")
	if err != nil || outcome.Kind != interpreter.OutcomeReturn || outcome.Value != 42 || len(outcome.DropTrace) != 1 {
		t.Fatalf("execute: %+v %v", outcome, err)
	}
	if runtime.GOOS == "windows" && runtime.GOARCH == "amd64" {
		toolchain, err := cdriver.ResolveWindowsHostMSVC()
		if err != nil {
			t.Fatal(err)
		}
		unit, err := cbackend.Emit(result.MIR, hir.QualifiedModulePath(1, "main")+"::main", cbackend.Stage0WindowsMSVCProfile())
		if err != nil {
			t.Fatal(err)
		}
		scalarRuntime, err := cruntime.ScalarSource(192)
		if err != nil {
			t.Fatal(err)
		}
		for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
			t.Run(profile.Name, func(t *testing.T) {
				artifact, err := cdriver.BuildWindowsHost(toolchain, profile, unit.Source, scalarRuntime)
				if err != nil {
					t.Fatal(err)
				}
				execution, err := cdriver.ExecuteWindowsHost(artifact)
				if err != nil || execution.ExitCode != 42 || execution.Stdout != "" || execution.Stderr != "" {
					t.Fatalf("native: %+v %v", execution, err)
				}
			})
		}
	}
	// Rebinding the same source alias to another package must change the
	// aggregate closure and its factory, not just the helper's own closure.
	input[1].Imports["helper"] = "helper_b"
	input[2].Owner = "helper_b"
	rebound, err := genericsource.PreparePackages(input, genericplan.DefaultPlanLimits())
	if err != nil {
		t.Fatal(err)
	}
	for _, original := range expanded.Templates {
		if original.Owner != "library" || original.Name == "Unused" {
			continue
		}
		found := false
		for _, current := range rebound.Templates {
			if current.Owner == original.Owner && current.Name == original.Name {
				found = true
				if current.Closure == original.Closure {
					t.Fatalf("package rebind did not change %s closure", original.Name)
				}
			}
		}
		if !found {
			t.Fatalf("missing rebound template %s", original.Name)
		}
	}
}

func TestGenericAggregatePackageInterfacesIgnoreInstantiationDemand(t *testing.T) {
	sources := aggregatePackageSources(t)
	input := semantic.PackageInput{Imports: []semantic.PackageImport{{FromPackage: 1, ImportPath: "box", ToPackage: 2}, {FromPackage: 2, ImportPath: "helper", ToPackage: 3}}}
	coordinates := map[hir.PackageID]packagegraph.CoordinateID{}
	for i, source := range sources {
		id := hir.PackageID(i + 1)
		file := parseCompilerFile(t, id, source.Module.Path+".fy", "module "+source.Module.Path+";")
		file.Module = source.Module
		input.Files = append(input.Files, file)
		coordinate, err := packagegraph.ComputeCoordinateID(packagegraph.Coordinate{PackageUID: source.Module.Path, VersionIdentity: "1"})
		if err != nil {
			t.Fatal(err)
		}
		coordinates[id] = coordinate
	}
	used, err := compiler.AnalyzeGenericPackages(input, coordinates, genericplan.DefaultPlanLimits())
	if err != nil || len(used.Interfaces) != 3 || len(used.Plan.Specializations) != 4 {
		t.Fatalf("used: %v %+v", err, used)
	}
	result, err := compiler.LowerAnalyzed(used.Analysis)
	if err != nil || !result.Verified || len(result.GenericProvenance.Declarations()) != 5 || len(result.GenericProvenance.Specializations()) != 4 {
		t.Fatalf("lower: %v %+v", err, result)
	}
	input.Files[0] = parseCompilerFile(t, 1, "main.fy", `module main; import box as b; fn i32 main() { return 0; }`)
	unused, err := compiler.AnalyzeGenericPackages(input, coordinates, genericplan.DefaultPlanLimits())
	if err != nil || len(unused.Plan.Specializations) != 0 || !reflect.DeepEqual(used.Interfaces, unused.Interfaces) || !reflect.DeepEqual(used.OrdinaryInterfaces, unused.OrdinaryInterfaces) {
		t.Fatalf("demand leaked into interface: %v", err)
	}
	input.Files[2] = parseCompilerFile(t, 3, "helper.fy", `module helper; pub fn T finish<T>(T value) { let T copy = value; return copy; }`)
	changed, err := compiler.AnalyzeGenericPackages(input, coordinates, genericplan.DefaultPlanLimits())
	if err != nil || changed.Interfaces[2] == unused.Interfaces[2] || changed.Interfaces[3] == unused.Interfaces[3] || changed.Interfaces[1] != unused.Interfaces[1] {
		t.Fatalf("cleanup closure propagation: %v", err)
	}
	renumbered := map[hir.PackageID]packagegraph.CoordinateID{}
	for id, coordinate := range coordinates {
		renumbered[id+10] = coordinate
	}
	for i := range input.Files {
		input.Files[i].Package += 10
	}
	for i := range input.Imports {
		input.Imports[i].FromPackage += 10
		input.Imports[i].ToPackage += 10
	}
	slices.Reverse(input.Files)
	reordered, err := compiler.AnalyzeGenericPackages(input, renumbered, genericplan.DefaultPlanLimits())
	if err != nil {
		t.Fatal(err)
	}
	for id, hash := range changed.Interfaces {
		if reordered.Interfaces[id+10] != hash {
			t.Fatal("package numbering changed aggregate interface")
		}
	}
}

func TestGenericAggregatePackageSourceRejectsInvalidRequests(t *testing.T) {
	for _, test := range []struct {
		name, from, to string
		provider       bool
	}{
		{"private", "pub resource", "resource", true},
		{"arity", "b::Box<i32>", "b::Box<i32, i32>", false},
		{"missing-alias", "b::Box", "missing::Box", false},
		{"generated", "b::Box<i32>", "b::_fy_generic_forged<i32>", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			input := aggregatePackageSources(t)
			i := 0
			if test.provider {
				i = 1
			}
			text := `module main; import box as b; fn i32 main() { let b::Box<i32> v = b::Box<i32> { .value = 42 }; return v.value; }`
			if test.provider {
				text = `module box; pub resource Box<T> { T value; cleanup {} }`
			}
			input[i].Module = parseCompilerFile(t, hir.PackageID(i+1), input[i].Module.Path+".fy", strings.ReplaceAll(text, test.from, test.to)).Module
			if result, err := genericsource.PreparePackages(input, genericplan.DefaultPlanLimits()); err == nil || len(result.Modules) != 0 || len(result.Templates) != 0 {
				t.Fatalf("invalid source published: %v %+v", err, result)
			}
		})
	}
}
