package compiler

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/genericsource"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/semantic"
)

// CheckNominalGenericFunctions is an import-free, module-private experiment.
// It does not activate nominal arguments in stable, explicit-file, package-cache
// or Atlas entry points. Type names are bound to the supplied exact coordinate.
func CheckNominalGenericFunctions(file semantic.File, coordinate packagegraph.CoordinateID, limits genericplan.PlanLimits) (Result, genericplan.Plan, error) {
	if coordinate == (packagegraph.CoordinateID{}) {
		return Result{}, genericplan.Plan{}, fmt.Errorf("nominal generic source requires a nonzero package coordinate")
	}
	original, owner := file, genericPackageOwner(coordinate)
	expansion, err := genericsource.PrepareNominalModule(file.Module, owner, limits)
	if err != nil {
		return Result{}, genericplan.Plan{}, err
	}
	file.Module = expansion.Modules[0]
	files := []semantic.File{file}
	result, err := LowerAnalyzed(Analysis{Semantic: restrictGenericSignatures(semantic.Analyze(files), files, expansion)})
	if result.Verified {
		result.GenericProvenance, err = genericProvenance(semantic.PackageInput{Files: []semantic.File{original}},
			map[hir.PackageID]genericplan.PackageID{file.Package: owner}, expansion, result.Semantic.NominalTypes)
		if err != nil {
			return Result{}, genericplan.Plan{}, err
		}
		result.Semantic.RequiresGenericProvenance = len(expansion.Declarations) != 0
	}
	return result, expansion.Plan, err
}
