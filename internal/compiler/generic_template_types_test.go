package compiler

import (
	"fmt"
	"reflect"
	"slices"
	"testing"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/genericsource"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/semantic"
)

func TestGenericTemplateTypesExactImportScopes(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		t.Run(fmt.Sprint(reverse), func(t *testing.T) {
			input := semantic.PackageInput{
				Files: []semantic.File{
					{Package: 1, Module: &ast.Module{Path: "app", Imports: []ast.Import{{Path: "dep", Alias: "d"}, {Path: "dep", Alias: "alt"}}}},
					{Package: 2, Module: &ast.Module{Path: "dep"}},
					{Package: 3, Module: &ast.Module{Path: "dep"}},
					{Package: 4, Module: &ast.Module{Path: "app", Imports: []ast.Import{{Path: "dep", Alias: "d"}}}},
				},
				Imports: []semantic.PackageImport{
					{FromPackage: 1, ImportPath: "dep", ToPackage: 2},
					{FromPackage: 1, ImportPath: "dep", ToPackage: 2},
					{FromPackage: 4, ImportPath: "dep", ToPackage: 3},
					{FromPackage: 1, ImportPath: "unused", ToPackage: 9},
					{FromPackage: 9, ImportPath: "dep", ToPackage: 3},
				},
			}
			declarations := []GenericDeclaration{
				{Kind: genericsource.OriginStruct, Package: 2, Module: "dep", Name: "Cell", ID: "left", Public: true},
				{Kind: genericsource.OriginResource, Package: 3, Module: "dep", Name: "Cell", ID: "right", Public: true},
				{Kind: genericsource.OriginResource, Package: 2, Module: "dep", Name: "Hidden", ID: "hidden"},
				{Kind: genericsource.OriginStruct, Package: 1, Module: "app", Name: "Local", ID: "local"},
				{Kind: genericsource.OriginResource, Package: 4, Module: "app", Name: "Own", ID: "own"},
				{Kind: genericsource.OriginFunction, Package: 2, Module: "dep", Name: "Function", ID: "function", Public: true},
				{Kind: genericsource.OriginStruct, Package: 2, Module: "other", Name: "WrongModule", ID: "wrong-module", Public: true},
				{Kind: genericsource.OriginStruct, Package: 9, Module: "dep", Name: "WrongPackage", ID: "wrong-package", Public: true},
			}
			declarations = append(declarations, declarations[0])
			want := []GenericTemplateType{
				{1, "app", "Local", "local"}, {1, "app", "alt::Cell", "left"}, {1, "app", "d::Cell", "left"},
				{2, "dep", "Cell", "left"}, {2, "dep", "Hidden", "hidden"},
				{3, "dep", "Cell", "right"},
				{4, "app", "Own", "own"}, {4, "app", "d::Cell", "right"},
			}
			if reverse {
				slices.Reverse(input.Files)
				slices.Reverse(input.Imports)
				slices.Reverse(declarations)
			}
			if got := genericTemplateTypes(input, declarations); !reflect.DeepEqual(got, want) {
				t.Fatalf("template scopes:\n got %+v\nwant %+v", got, want)
			}
			// Rebuild from current bindings; no scope may leak from the prior call.
			input.Imports = nil
			want = slices.DeleteFunc(want, func(fact GenericTemplateType) bool {
				return fact.Name == "d::Cell" || fact.Name == "alt::Cell"
			})
			if got := genericTemplateTypes(input, declarations); !reflect.DeepEqual(got, want) {
				t.Fatalf("unbound imports: got %+v, want %+v", got, want)
			}
			if got := genericTemplateTypes(input, nil); got != nil {
				t.Fatalf("empty declarations: %+v", got)
			}
		})
	}
}

func TestGenericTemplateTypesRetainsConflictingProviders(t *testing.T) {
	input := semantic.PackageInput{
		Files: []semantic.File{{Package: 1, Module: &ast.Module{Path: "app", Imports: []ast.Import{{Path: "dep", Alias: "d"}}}}},
		Imports: []semantic.PackageImport{
			{FromPackage: 1, ImportPath: "dep", ToPackage: 3},
			{FromPackage: 1, ImportPath: "dep", ToPackage: 2},
		},
	}
	declarations := []GenericDeclaration{
		{Kind: genericsource.OriginStruct, Package: 2, Module: "dep", Name: "Cell", ID: "left", Public: true},
		{Kind: genericsource.OriginStruct, Package: 3, Module: "dep", Name: "Cell", ID: "right", Public: true},
	}
	// This projection must preserve contradictions for downstream validation.
	want := []GenericTemplateType{{1, "app", "d::Cell", "left"}, {1, "app", "d::Cell", "right"}}
	if got := genericTemplateTypes(input, declarations); !reflect.DeepEqual(got, want) {
		t.Fatalf("conflicting providers: got %+v, want %+v", got, want)
	}
}

func TestGenericTemplateTypesWithoutAggregates(t *testing.T) {
	input := semantic.PackageInput{Files: []semantic.File{{Package: 1}}}
	for _, declarations := range [][]GenericDeclaration{nil, {{Kind: genericsource.OriginFunction, Public: true}}} {
		if got := genericTemplateTypes(input, declarations); got != nil {
			t.Fatalf("non-aggregate declarations produced template types: %+v", got)
		}
	}
}

var genericTemplateTypesBenchmarkSink []GenericTemplateType

func BenchmarkGenericTemplateTypes(b *testing.B) {
	for _, files := range []int{16, 64, 256} {
		b.Run(fmt.Sprint(files), func(b *testing.B) {
			var input semantic.PackageInput
			var declarations []GenericDeclaration
			for file := range files {
				pkg := hir.PackageID(file + 1)
				module := &ast.Module{Path: fmt.Sprintf("module_%d", file)}
				for dependency := range 4 {
					provider := (file + dependency + 1) % files
					path := fmt.Sprintf("module_%d", provider)
					module.Imports = append(module.Imports, ast.Import{Path: path, Alias: fmt.Sprintf("dep%d", dependency)})
					input.Imports = append(input.Imports, semantic.PackageImport{FromPackage: pkg, ImportPath: path, ToPackage: hir.PackageID(provider + 1)})
				}
				input.Files = append(input.Files, semantic.File{Package: pkg, Module: module})
				for declaration := range 8 {
					kind := genericsource.OriginStruct
					if declaration%2 == 1 {
						kind = genericsource.OriginResource
					}
					if declaration == 7 {
						kind = genericsource.OriginFunction
					}
					declarations = append(declarations, GenericDeclaration{Kind: kind, Package: pkg, Module: module.Path,
						Name: fmt.Sprintf("Type_%d", declaration), ID: genericplan.DeclarationID(fmt.Sprintf("%d:%d", file, declaration)), Public: declaration%4 != 0})
				}
			}
			b.ReportAllocs()
			for b.Loop() {
				result := genericTemplateTypes(input, declarations)
				if len(result) != files*27 {
					b.Fatalf("template scopes=%d, want %d", len(result), files*27)
				}
				genericTemplateTypesBenchmarkSink = result
			}
		})
	}
}
