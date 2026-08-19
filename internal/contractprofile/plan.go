package contractprofile

import (
	"fmt"
	"slices"
	"strings"

	"github.com/0disoft/fairyc/internal/mir"
)

// CycleEdge is one compiler-private MIR edge that a backend must meter.
// Block identities are deliberately absent from the public JSON report.
type CycleEdge struct {
	From mir.BlockID
	To   mir.BlockID
}

// FunctionMeterPlan is the exact handoff consumed by a future contract backend.
type FunctionMeterPlan struct {
	Name                         string
	LoopBudget                   uint64
	CycleEdges                   []CycleEdge
	AbstractCostUpperBound       uint64
	StorageGrowthUpperBoundBytes uint64
}

// MeterPlan binds one accepted profile decision to exact compiler-private MIR edges.
type MeterPlan struct {
	Profile   string
	CostUnit  string
	Functions []FunctionMeterPlan
}

// BuildMeterPlan returns exact cycle edges only when the complete verified program passes the
// deterministic profile. A backend must not emit executable code from a rejected plan.
func BuildMeterPlan(program mir.Program, loopBudgets map[string]uint64) (MeterPlan, error) {
	report := Analyze(program, loopBudgets)
	if !report.Accepted {
		if len(report.Violations) != 0 {
			first := report.Violations[0]
			return MeterPlan{}, fmt.Errorf("contract profile rejected by %s: %s", first.Code, first.Message)
		}
		return MeterPlan{}, fmt.Errorf("contract profile rejected")
	}

	functions := slices.Clone(program.Functions)
	slices.SortFunc(functions, func(left, right mir.Function) int {
		return strings.Compare(left.Name, right.Name)
	})
	reports := make(map[string]FunctionReport, len(report.Functions))
	for _, function := range report.Functions {
		reports[function.Name] = function
	}

	plan := MeterPlan{
		Profile:   ProfileID,
		CostUnit:  CostUnit,
		Functions: make([]FunctionMeterPlan, 0, len(functions)),
	}
	for _, function := range functions {
		_, malformed, edges := controlFlowCycleInfo(function)
		if malformed {
			return MeterPlan{}, fmt.Errorf("contract profile accepted malformed CFG for %s", function.Name)
		}
		functionReport, found := reports[function.Name]
		if !found || !functionReport.Bounded {
			return MeterPlan{}, fmt.Errorf("contract profile accepted function without a bound: %s", function.Name)
		}
		plan.Functions = append(plan.Functions, FunctionMeterPlan{
			Name:                         function.Name,
			LoopBudget:                   loopBudgets[function.Name],
			CycleEdges:                   slices.Clone(edges),
			AbstractCostUpperBound:       functionReport.TransitiveCostUpperBound,
			StorageGrowthUpperBoundBytes: functionReport.StorageGrowthUpperBoundBytes,
		})
	}
	return plan, nil
}
