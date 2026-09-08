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
	if coordinate == (packagegraph.CoordinateID{}) {
		return Result{}, genericplan.Plan{}, fmt.Errorf("storage generic source requires a nonzero package coordinate")
	}
	original, owner := file, genericPackageOwner(coordinate)
	expansion, err := genericsource.PrepareStorageModule(file.Module, owner, limits)
	if err != nil {
		return Result{}, genericplan.Plan{}, err
	}
	file.Module = expansion.Modules[0]
	result, err := lowerWithVerifier(semantic.Analyze([]semantic.File{file}), mir.VerifyStorageQueries)
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
	prepared, expansion, err := prepareGenericPackageGraph(input, coordinates, limits, true)
	if err != nil {
		return Result{}, genericplan.Plan{}, err
	}
	analysis, err := semantic.AnalyzePackages(prepared)
	if err != nil {
		return Result{}, genericplan.Plan{}, &Error{Stage: StagePackageInput, Cause: err}
	}
	result, err := lowerWithVerifier(analysis, mir.VerifyStorageQueries)
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
