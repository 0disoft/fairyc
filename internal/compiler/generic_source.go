package compiler

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/genericsource"
	"github.com/0disoft/fairyc/internal/semantic"
)

// CheckGenericFunctions is the compiler-private source-to-MIR admission path for
// module-local scalar generic functions. It deliberately does not participate in
// stable workspace interfaces or package caches. Every generated function passes
// the ordinary type, ownership, effect, lowering, and MIR verification pipeline.
func CheckGenericFunctions(file semantic.File, limits genericplan.PlanLimits) (Result, genericplan.Plan, error) {
	owner := genericplan.PackageID(fmt.Sprintf("package_%d", file.Package))
	module, plan, err := genericsource.Expand(file.Module, owner, limits)
	if err != nil {
		return Result{}, genericplan.Plan{}, err
	}
	file.Module = module
	result, err := Check([]semantic.File{file})
	return result, plan, err
}
