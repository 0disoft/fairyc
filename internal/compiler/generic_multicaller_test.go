package compiler_test

import (
	"encoding/json"
	"reflect"
	"runtime"
	"slices"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/semantic"
)

func TestGenericMultipleCallersShareOnlyExactProviderSpecializations(t *testing.T) {
	for _, distinct := range []bool{false, true} {
		name, want := "shared-provider", 1
		if distinct {
			name, want = "distinct-identical-provider", 2
		}
		t.Run(name, func(t *testing.T) {
			input, coordinates := genericMulticallerInput(t, distinct)
			before, _ := json.Marshal(input)
			limits := genericplan.PlanLimits{MaxSpecializations: want, MaxSpecializationsPerDeclaration: 1, MaxExpansionDepth: 1}
			result, plan, err := compiler.CheckGenericPackages(input, coordinates, limits)
			if err != nil || !result.Verified || len(plan.Specializations) != want || len(result.MIR.Functions) != 3+want {
				t.Fatalf("compile/dedup: %v; plan=%+v semantic=%+v", err, plan, result.Semantic)
			}
			after, _ := json.Marshal(input)
			if string(before) != string(after) {
				t.Fatal("shared source input mutated")
			}
			declarations := map[genericplan.DeclarationID]compiler.GenericDeclaration{}
			for _, declaration := range result.GenericProvenance.Declarations() {
				declarations[declaration.ID] = declaration
			}
			owners := map[hir.PackageID]int{}
			for _, origin := range result.GenericProvenance.Specializations() {
				declaration, found := declarations[origin.Declaration]
				if !found {
					t.Fatal("missing original declaration")
				}
				owners[declaration.Package]++
				qualified := hir.QualifiedModulePath(declaration.Package, declaration.Module) + "::" + origin.GeneratedName
				matches := 0
				for _, function := range result.MIR.Functions {
					if function.Name == qualified {
						matches++
					}
				}
				if matches != 1 {
					t.Fatal("specialization lacks one exact defining-package MIR owner")
				}
			}
			if owners[4] != 1 || len(owners) != want || (distinct && owners[5] != 1) {
				t.Fatalf("provider ownership: %+v", owners)
			}
			entry := hir.QualifiedModulePath(1, "main") + "::main"
			outcome, err := interpreter.Execute(result.MIR, entry)
			if err != nil || outcome.Kind != interpreter.OutcomeReturn || outcome.Value != 42 {
				t.Fatalf("execute: %+v %v", outcome, err)
			}
			if distinct {
				limits.MaxSpecializations = 1
				failed, _, err := compiler.CheckGenericPackages(input, coordinates, limits)
				if err != nil || failed.Verified || !reflect.DeepEqual(failed.MIR, mir.Program{}) || len(failed.Semantic.Diagnostics) != 1 || failed.Semantic.Diagnostics[0].Code != "E4503" {
					t.Fatalf("distinct providers bypassed budget: %v %+v", err, failed.Semantic)
				}
				limits.MaxSpecializations = want
			}
			for i := range input.Files {
				input.Files[i].Package += 100
			}
			for i := range input.Imports {
				input.Imports[i].FromPackage += 100
				input.Imports[i].ToPackage += 100
			}
			renumbered := map[hir.PackageID]packagegraph.CoordinateID{}
			for id, coordinate := range coordinates {
				renumbered[id+100] = coordinate
			}
			slices.Reverse(input.Files)
			slices.Reverse(input.Imports)
			rebuilt, reordered, err := compiler.CheckGenericPackages(input, renumbered, limits)
			if err != nil || !rebuilt.Verified || !reflect.DeepEqual(plan, reordered) {
				t.Fatal("package renumbering or order changed specialization plan")
			}
			t.Run("native", func(t *testing.T) {
				if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
					t.Skip("native evidence requires Windows amd64")
				}
				toolchain, err := cdriver.ResolveWindowsHostMSVC()
				if err != nil {
					t.Fatal(err)
				}
				runGenericNative(t, toolchain, result, entry, 42)
			})
		})
	}
}

func genericMulticallerInput(t *testing.T, distinct bool) (semantic.PackageInput, map[hir.PackageID]packagegraph.CoordinateID) {
	t.Helper()
	const provider = "module core; pub fn T id<T>(T value) { return value; }"
	input := semantic.PackageInput{Files: []semantic.File{
		parseCompilerFile(t, 1, "main.fy", "module main; import left as l; import right as r; fn i32 main() { return l::read() + r::read(); }"),
		parseCompilerFile(t, 2, "left.fy", "module left; import core as c; pub fn i32 read() { return c::id<i32>(21); }"),
		parseCompilerFile(t, 3, "right.fy", "module right; import core as c; pub fn i32 read() { return c::id<i32>(21); }"),
		parseCompilerFile(t, 4, "core.fy", provider),
		parseCompilerFile(t, 5, "core.fy", provider),
	}, Imports: []semantic.PackageImport{{FromPackage: 1, ImportPath: "left", ToPackage: 2}, {FromPackage: 1, ImportPath: "right", ToPackage: 3}, {FromPackage: 2, ImportPath: "core", ToPackage: 4}, {FromPackage: 3, ImportPath: "core", ToPackage: 4}}}
	if distinct {
		input.Imports[3].ToPackage = 5
	}
	coordinates := map[hir.PackageID]packagegraph.CoordinateID{}
	for id, uid := range []string{"root", "left", "right", "provider-a", "provider-b"} {
		coordinate, err := packagegraph.ComputeCoordinateID(packagegraph.Coordinate{PackageUID: uid, VersionIdentity: "1"})
		if err != nil {
			t.Fatal(err)
		}
		coordinates[hir.PackageID(id+1)] = coordinate
	}
	return input, coordinates
}
