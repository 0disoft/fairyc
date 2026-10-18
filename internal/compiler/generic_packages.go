package compiler

import (
	"context"
	"fmt"
	"slices"

	"github.com/0disoft/fairyc/internal/admission"
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

func prepareGenericPackageGraph(input semantic.PackageInput, coordinates map[hir.PackageID]packagegraph.CoordinateID, limits genericplan.PlanLimits, policy admission.Policy[hir.PackageID]) (semantic.PackageInput, genericsource.PackageExpansion, error) {
	return prepareGenericPackageGraphContext(context.Background(), input, coordinates, limits, policy)
}

func prepareGenericPackageGraphContext(ctx context.Context, input semantic.PackageInput, coordinates map[hir.PackageID]packagegraph.CoordinateID, limits genericplan.PlanLimits, policy admission.Policy[hir.PackageID]) (semantic.PackageInput, genericsource.PackageExpansion, error) {
	if err := ctx.Err(); err != nil {
		return semantic.PackageInput{}, genericsource.PackageExpansion{}, err
	}
	if err := policy.Validate(); err != nil {
		return semantic.PackageInput{}, genericsource.PackageExpansion{}, err
	}
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
	sourcePolicy := admission.Policy[genericplan.PackageID]{Capabilities: policy.Capabilities}
	if policy.StorageOwners != nil {
		sourcePolicy.StorageOwners = make([]genericplan.PackageID, len(policy.StorageOwners))
		for i, owner := range policy.StorageOwners {
			var exists bool
			sourcePolicy.StorageOwners[i], exists = owners[owner]
			if !exists {
				return semantic.PackageInput{}, genericsource.PackageExpansion{}, fmt.Errorf("storage operation owner is absent from package input")
			}
		}
	}
	expansion, err := genericsource.PrepareWithPolicyContext(ctx, modules, limits, sourcePolicy)
	if err != nil {
		return semantic.PackageInput{}, genericsource.PackageExpansion{}, err
	}
	input.Files = slices.Clone(input.Files)
	for i := range input.Files {
		input.Files[i].Module = expansion.Modules[i]
	}
	return input, expansion, nil
}
