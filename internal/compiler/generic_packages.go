package compiler

import (
	"fmt"
	"slices"

	"github.com/0disoft/fairyc/internal/diagnostic"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/genericsource"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/semantic"
)

// CheckGenericPackages is a compiler-private admission path for explicit scalar
// generic functions across a resolved package graph. Coordinates must come from
// the same workspace as input. Source-owned expansion failures become ordinary
// diagnostics; invalid package plumbing remains an error. Stable CLI admission
// remains separate from the compiler-private generic interface and cache paths.
func CheckGenericPackages(input semantic.PackageInput, coordinates map[hir.PackageID]packagegraph.CoordinateID, limits genericplan.PlanLimits) (Result, genericplan.Plan, error) {
	analysis, err := AnalyzeGenericPackages(input, coordinates, limits)
	if err != nil {
		return Result{}, genericplan.Plan{}, err
	}
	result, err := LowerAnalyzed(analysis.Analysis)
	return result, analysis.Plan, err
}

func prepareGenericPackages(input semantic.PackageInput, coordinates map[hir.PackageID]packagegraph.CoordinateID, limits genericplan.PlanLimits) (semantic.PackageInput, genericsource.PackageExpansion, error) {
	owners := map[hir.PackageID]genericplan.PackageID{}
	seen := map[packagegraph.CoordinateID]hir.PackageID{}
	for _, file := range input.Files {
		coordinate, exists := coordinates[file.Package]
		if !exists || coordinate == (packagegraph.CoordinateID{}) {
			return semantic.PackageInput{}, genericsource.PackageExpansion{}, fmt.Errorf("missing generic package coordinate for %d", file.Package)
		}
		if previous, duplicate := seen[coordinate]; duplicate && previous != file.Package {
			return semantic.PackageInput{}, genericsource.PackageExpansion{}, fmt.Errorf("ambiguous generic package coordinate")
		}
		seen[coordinate] = file.Package
		owners[file.Package] = genericPackageOwner(coordinate)
	}
	bindings := map[hir.PackageID]map[string]genericplan.PackageID{}
	for _, imported := range input.Imports {
		_, fromExists := owners[imported.FromPackage]
		to, toExists := owners[imported.ToPackage]
		if !fromExists || !toExists {
			return semantic.PackageInput{}, genericsource.PackageExpansion{}, fmt.Errorf("generic import references missing package")
		}
		if bindings[imported.FromPackage] == nil {
			bindings[imported.FromPackage] = map[string]genericplan.PackageID{}
		}
		bound := bindings[imported.FromPackage]
		if previous, exists := bound[imported.ImportPath]; exists && previous != to {
			return semantic.PackageInput{}, genericsource.PackageExpansion{}, fmt.Errorf("conflicting generic import provider")
		}
		bound[imported.ImportPath] = to
	}
	modules := make([]genericsource.PackageModule, len(input.Files))
	for i, file := range input.Files {
		if file.Module == nil {
			return semantic.PackageInput{}, genericsource.PackageExpansion{}, fmt.Errorf("generic source module is nil")
		}
		// Source expansion can now resolve aggregate imports, but package
		// interfaces and cache artifacts still require their own admission.
		for _, declaration := range file.Module.Structs {
			if len(declaration.TypeParameters) != 0 {
				return semantic.PackageInput{}, genericsource.PackageExpansion{}, packageAggregatePending(file.Module.Path, owners[file.Package], declaration.Span)
			}
		}
		for _, declaration := range file.Module.Resources {
			if len(declaration.TypeParameters) != 0 {
				return semantic.PackageInput{}, genericsource.PackageExpansion{}, packageAggregatePending(file.Module.Path, owners[file.Package], declaration.Span)
			}
		}
		modules[i] = genericsource.PackageModule{Module: file.Module, Owner: owners[file.Package], Imports: bindings[file.Package]}
	}
	expansion, err := genericsource.PreparePackages(modules, limits)
	if err != nil {
		return semantic.PackageInput{}, genericsource.PackageExpansion{}, err
	}
	input.Files = slices.Clone(input.Files)
	for i := range input.Files {
		input.Files[i].Module = expansion.Modules[i]
	}
	return input, expansion, nil
}

func packageAggregatePending(module string, owner genericplan.PackageID, span diagnostic.Span) error {
	return &genericsource.SourceError{Owner: owner, Module: module, Span: span, Kind: genericsource.FailureDeclaration, Cause: fmt.Errorf("generic aggregate package interfaces are not admitted")}
}
