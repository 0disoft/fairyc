package compiler

import (
	"fmt"
	"slices"

	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/genericsource"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/semantic"
)

// CheckGenericPackages is a compiler-private admission path for explicit scalar
// generic functions and aggregates across a resolved package graph. Coordinates come from
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
	return prepareGenericPackageGraph(input, coordinates, limits, false, false, false)
}

func prepareGenericPackageGraph(input semantic.PackageInput, coordinates map[hir.PackageID]packagegraph.CoordinateID, limits genericplan.PlanLimits, storage, nominal, transfers bool) (semantic.PackageInput, genericsource.PackageExpansion, error) {
	return prepareGenericPackageGraphOwned(input, coordinates, limits, storage, nominal, transfers, nil, false)
}

func prepareGenericPackageGraphOwned(input semantic.PackageInput, coordinates map[hir.PackageID]packagegraph.CoordinateID, limits genericplan.PlanLimits, storage, nominal, transfers bool, storageOwners []hir.PackageID, sdkTemplates bool) (semantic.PackageInput, genericsource.PackageExpansion, error) {
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
		modules[i] = genericsource.PackageModule{Module: file.Module, Owner: owners[file.Package], Imports: bindings[file.Package]}
	}
	prepare := genericsource.PreparePackages
	if transfers {
		prepare = genericsource.PrepareStorageTransferPackages
	} else if nominal {
		prepare = genericsource.PrepareNominalStoragePackages
	} else if storage {
		prepare = genericsource.PrepareStoragePackages
	}
	var expansion genericsource.PackageExpansion
	var err error
	if storageOwners != nil {
		selected := make([]genericplan.PackageID, len(storageOwners))
		for i, owner := range storageOwners {
			var exists bool
			selected[i], exists = owners[owner]
			if !exists {
				return semantic.PackageInput{}, genericsource.PackageExpansion{}, fmt.Errorf("storage operation owner is absent from package input")
			}
		}
		prepareOwned := genericsource.PrepareOwnedStorageTransferPackages
		if sdkTemplates {
			prepareOwned = genericsource.PrepareSDKStoragePackages
		}
		expansion, err = prepareOwned(modules, limits, selected)
	} else {
		expansion, err = prepare(modules, limits)
	}
	if err != nil {
		return semantic.PackageInput{}, genericsource.PackageExpansion{}, err
	}
	input.Files = slices.Clone(input.Files)
	for i := range input.Files {
		input.Files[i].Module = expansion.Modules[i]
	}
	return input, expansion, nil
}
