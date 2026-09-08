package compiler_test

import (
	"encoding/json"
	"runtime"
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
	"github.com/0disoft/fairyc/internal/semantic"
)

func aggregatePackageSources(t *testing.T) []genericsource.PackageModule {
	t.Helper()
	return []genericsource.PackageModule{
		{Module: parseCompilerFile(t, 1, "main.fy", `module main; import box as b;
fn i32 main() { let b::Box<i32> value = b::make<i32>(42); return value.value; }`).Module, Owner: "app", Imports: map[string]genericplan.PackageID{"box": "library"}},
		{Module: parseCompilerFile(t, 2, "box.fy", `module box; import helper as h;
pub resource Box<T> { T value; cleanup { let i32 observed = h::finish<i32>(1); } }
pub fn Box<T> make<T>(T value) { return Box<T> { .value = value }; }
pub struct Unused<T> { T item; }`).Module, Owner: "library", Imports: map[string]genericplan.PackageID{"helper": "helper_a"}},
		{Module: parseCompilerFile(t, 3, "helper.fy", `module helper; pub fn T finish<T>(T value) { return value; }`).Module, Owner: "helper_a"},
	}
}

func TestGenericAggregatePackageSourceExpansion(t *testing.T) {
	input := aggregatePackageSources(t)
	before, _ := json.Marshal(input)
	expanded, err := genericsource.PreparePackages(input, genericplan.DefaultPlanLimits())
	if err != nil || len(expanded.Plan.Specializations) != 3 || len(expanded.Templates) != 4 {
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
	if err != nil || outcome.Kind != interpreter.OutcomeReturn || outcome.Value != 42 {
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

func TestGenericAggregatePackageInterfaceRemainsClosed(t *testing.T) {
	input, coordinates := genericPackageInput(t)
	input.Files[1] = parseCompilerFile(t, 2, "util.fy", `module util; pub resource Box<T> { T value; cleanup {} }`)
	analysis, err := compiler.AnalyzeGenericPackages(input, coordinates, genericplan.DefaultPlanLimits())
	if err != nil || len(analysis.Analysis.Semantic.Diagnostics) == 0 || len(analysis.Interfaces) != 0 || len(analysis.Specializations) != 0 {
		t.Fatalf("aggregate interface admitted: %v %+v", err, analysis)
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
