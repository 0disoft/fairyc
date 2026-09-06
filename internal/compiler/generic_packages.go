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
// generic functions across a resolved package graph. Coordinates must come from
// the same workspace as input. Stable CLI, interfaces and cache admission remain
// separate: template definitions are not yet published by the typed interface.
func CheckGenericPackages(input semantic.PackageInput, coordinates map[hir.PackageID]packagegraph.CoordinateID, limits genericplan.PlanLimits) (Result, genericplan.Plan, error) {
	owners := map[hir.PackageID]genericplan.PackageID{}
	seen := map[packagegraph.CoordinateID]hir.PackageID{}
	for _, file := range input.Files {
		coordinate, exists := coordinates[file.Package]
		if !exists || coordinate == (packagegraph.CoordinateID{}) {
			return Result{}, genericplan.Plan{}, fmt.Errorf("missing generic package coordinate for %d", file.Package)
		}
		if previous, duplicate := seen[coordinate]; duplicate && previous != file.Package {
			return Result{}, genericplan.Plan{}, fmt.Errorf("ambiguous generic package coordinate")
		}
		seen[coordinate] = file.Package
		owners[file.Package] = genericPackageOwner(coordinate)
	}
	bindings := map[hir.PackageID]map[string]genericplan.PackageID{}
	for _, imported := range input.Imports {
		_, fromExists := owners[imported.FromPackage]
		to, toExists := owners[imported.ToPackage]
		if !fromExists || !toExists {
			return Result{}, genericplan.Plan{}, fmt.Errorf("generic import references missing package")
		}
		if bindings[imported.FromPackage] == nil {
			bindings[imported.FromPackage] = map[string]genericplan.PackageID{}
		}
		bound := bindings[imported.FromPackage]
		if previous, exists := bound[imported.ImportPath]; exists && previous != to {
			return Result{}, genericplan.Plan{}, fmt.Errorf("conflicting generic import provider")
		}
		bound[imported.ImportPath] = to
	}
	modules := make([]genericsource.PackageModule, len(input.Files))
	for i, file := range input.Files {
		modules[i] = genericsource.PackageModule{Module: file.Module, Owner: owners[file.Package], Imports: bindings[file.Package]}
	}
	expanded, plan, err := genericsource.ExpandPackages(modules, limits)
	if err != nil {
		return Result{}, genericplan.Plan{}, err
	}
	input.Files = slices.Clone(input.Files)
	for i := range input.Files {
		input.Files[i].Module = expanded[i]
	}
	result, err := CheckPackages(input)
	return result, plan, err
}
