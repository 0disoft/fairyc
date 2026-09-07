package compiler

import (
	"encoding/hex"
	"fmt"

	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/genericsource"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/semantic"
)

// CheckGenericFunctions is the compiler-private source-to-MIR admission path for
// module-local scalar generic functions. It deliberately does not participate in
// stable workspace interfaces or package caches. Every generated function passes
// the ordinary type, ownership, effect, lowering, and MIR verification pipeline.
func CheckGenericFunctions(file semantic.File, limits genericplan.PlanLimits) (Result, genericplan.Plan, error) {
	owner := genericplan.PackageID(fmt.Sprintf("package_%d", file.Package))
	return checkGenericFunctions(file, owner, limits)
}

// CheckGenericFunctionsForPackage binds the private module-local admission path
// to a resolved package coordinate, independent of disposable semantic numbering.
// The caller must supply the coordinate belonging to file.Package. A coordinate
// is a namespace, not evidence for cache reuse; definition digests still govern
// specialization identity. This does not admit public or imported templates.
func CheckGenericFunctionsForPackage(file semantic.File, coordinate packagegraph.CoordinateID, limits genericplan.PlanLimits) (Result, genericplan.Plan, error) {
	if coordinate == (packagegraph.CoordinateID{}) {
		return Result{}, genericplan.Plan{}, fmt.Errorf("generic source requires a nonzero package coordinate")
	}
	// Encode all digest bytes in the planner's canonical identifier alphabet.
	// Keep a distinct versioned namespace rather than weakening path validation.
	owner := genericPackageOwner(coordinate)
	return checkGenericFunctions(file, owner, limits)
}

func genericPackageOwner(coordinate packagegraph.CoordinateID) genericplan.PackageID {
	return genericplan.PackageID("package_coordinate_v1_" + hex.EncodeToString(coordinate[:]))
}

func checkGenericFunctions(file semantic.File, owner genericplan.PackageID, limits genericplan.PlanLimits) (Result, genericplan.Plan, error) {
	hasTemplates := false
	if file.Module != nil {
		for _, function := range file.Module.Functions {
			hasTemplates = hasTemplates || len(function.TypeParameters) != 0
		}
	}
	module, plan, err := genericsource.Expand(file.Module, owner, limits)
	if err != nil {
		return Result{}, genericplan.Plan{}, err
	}
	file.Module = module
	result, err := Check([]semantic.File{file})
	if result.Verified {
		result.Semantic.RequiresGenericProvenance = hasTemplates
	}
	return result, plan, err
}
