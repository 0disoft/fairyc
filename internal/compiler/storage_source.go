package compiler

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/genericsource"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/semantic"
)

// CheckStorageGenericFunctions is the private source-to-native preparation lane.
// Its MIR may contain symbolic queries rejected by ordinary artifact consumers.
func CheckStorageGenericFunctions(file semantic.File, coordinate packagegraph.CoordinateID, limits genericplan.PlanLimits) (Result, genericplan.Plan, error) {
	return checkStorageGenericFunctions(file, coordinate, limits, false)
}

// CheckStorageTransferFunctions explicitly admits unsafe source storage transfers.
// Its output requires the matching private verifier and C emission entry point.
func CheckStorageTransferFunctions(file semantic.File, coordinate packagegraph.CoordinateID, limits genericplan.PlanLimits) (Result, genericplan.Plan, error) {
	return checkStorageGenericFunctions(file, coordinate, limits, true)
}

func checkStorageGenericFunctions(file semantic.File, coordinate packagegraph.CoordinateID, limits genericplan.PlanLimits, transfers bool) (Result, genericplan.Plan, error) {
	if coordinate == (packagegraph.CoordinateID{}) {
		return Result{}, genericplan.Plan{}, fmt.Errorf("storage generic source requires a nonzero package coordinate")
	}
	original, owner := file, genericPackageOwner(coordinate)
	prepare, verify := genericsource.PrepareStorageModule, mir.VerifyStorageQueries
	if transfers {
		prepare, verify = genericsource.PrepareStorageTransferModule, mir.VerifyStorageTransfers
	}
	expansion, err := prepare(file.Module, owner, limits)
	if err != nil {
		return Result{}, genericplan.Plan{}, err
	}
	file.Module = expansion.Modules[0]
	result, err := lowerWithVerifier(semantic.Analyze([]semantic.File{file}), verify)
	if err == nil && result.Verified {
		result.GenericProvenance, err = genericProvenance(semantic.PackageInput{Files: []semantic.File{original}},
			map[hir.PackageID]genericplan.PackageID{file.Package: owner}, expansion, result.Semantic.NominalTypes)
		if err != nil {
			return Result{}, genericplan.Plan{}, err
		}
		result.Semantic.RequiresGenericProvenance = len(expansion.Declarations) != 0
	}
	return result, expansion.Plan, err
}

// CheckStorageGenericPackages enables private scalar storage queries across a
// resolved graph. It does not publish a stable interface or artifact-cache key.
func CheckStorageGenericPackages(input semantic.PackageInput, coordinates map[hir.PackageID]packagegraph.CoordinateID, limits genericplan.PlanLimits) (Result, genericplan.Plan, error) {
	return checkStorageGenericPackages(input, coordinates, limits, false, false)
}

// CheckNominalStoragePackages admits source-visible nominal arguments through
// exact private bindings, without adding imports or publishing a stable cache key.
func CheckNominalStoragePackages(input semantic.PackageInput, coordinates map[hir.PackageID]packagegraph.CoordinateID, limits genericplan.PlanLimits) (Result, genericplan.Plan, error) {
	return checkStorageGenericPackages(input, coordinates, limits, true, false)
}

// CheckStorageTransferPackages is the private closed-graph ownership-transfer
// lane. Its verified MIR requires the matching private C emitter; it does not
// admit storage transfers to stable workspace or artifact consumers.
func CheckStorageTransferPackages(input semantic.PackageInput, coordinates map[hir.PackageID]packagegraph.CoordinateID, limits genericplan.PlanLimits) (Result, genericplan.Plan, error) {
	return checkStorageGenericPackages(input, coordinates, limits, true, true)
}

func checkStorageGenericPackages(input semantic.PackageInput, coordinates map[hir.PackageID]packagegraph.CoordinateID, limits genericplan.PlanLimits, nominal, transfers bool) (Result, genericplan.Plan, error) {
	prepared, expansion, err := prepareGenericPackageGraph(input, coordinates, limits, true, nominal, transfers)
	if err != nil {
		return Result{}, genericplan.Plan{}, err
	}
	var analysis semantic.Result
	if nominal {
		byOwner := map[genericplan.PackageID]hir.PackageID{}
		for _, file := range input.Files {
			byOwner[genericPackageOwner(coordinates[file.Package])] = file.Package
		}
		bindings := make([]semantic.NominalTypeBinding, 0, len(expansion.NominalBindings))
		for _, binding := range expansion.NominalBindings {
			id, found := byOwner[binding.Owner]
			if !found {
				return Result{}, genericplan.Plan{}, fmt.Errorf("nominal binding owner is absent")
			}
			bindings = append(bindings, semantic.NominalTypeBinding{Key: string(binding.ID), Package: id, Module: binding.Module, Name: binding.Name})
		}
		analysis, err = semantic.AnalyzeBoundNominalPackages(prepared, bindings)
	} else {
		analysis, err = semantic.AnalyzePackages(prepared)
	}
	if err != nil {
		return Result{}, genericplan.Plan{}, &Error{Stage: StagePackageInput, Cause: err}
	}
	verify := mir.VerifyStorageQueries
	if transfers {
		verify = mir.VerifyStorageTransfers
	}
	result, err := lowerWithVerifier(analysis, verify)
	if err == nil && result.Verified {
		owners := make(map[hir.PackageID]genericplan.PackageID, len(coordinates))
		for id, coordinate := range coordinates {
			owners[id] = genericPackageOwner(coordinate)
		}
		result.GenericProvenance, err = genericProvenance(input, owners, expansion, result.Semantic.NominalTypes)
		if err != nil {
			return Result{}, genericplan.Plan{}, err
		}
		result.Semantic.RequiresGenericProvenance = len(expansion.Declarations) != 0
	}
	return result, expansion.Plan, err
}
