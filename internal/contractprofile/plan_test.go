package contractprofile

import (
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

func TestBuildMeterPlanExposesEveryEdgeInsideCyclicComponents(t *testing.T) {
	t.Parallel()
	program := mir.Program{Functions: []mir.Function{{
		Symbol:     1,
		Name:       "main::main",
		ResultType: hir.TypeI32,
		Entry:      0,
		Blocks: []mir.Block{
			{ID: 0, Terminator: mir.Terminator{Kind: mir.TerminatorGoto, Successors: []mir.BlockID{1}}},
			{ID: 1, Terminator: mir.Terminator{Kind: mir.TerminatorSwitch, Successors: []mir.BlockID{2, 3}}},
			{ID: 2, Terminator: mir.Terminator{Kind: mir.TerminatorGoto, Successors: []mir.BlockID{1}}},
			{ID: 3, Terminator: mir.Terminator{Kind: mir.TerminatorReturn}},
		},
	}}}
	plan, err := BuildMeterPlan(program, map[string]uint64{"main::main": 7})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Profile != ProfileID || plan.CostUnit != CostUnit || len(plan.Functions) != 1 {
		t.Fatalf("plan = %+v", plan)
	}
	function := plan.Functions[0]
	if function.LoopBudget != 7 || function.AbstractCostUpperBound == 0 || len(function.CycleEdges) != 2 {
		t.Fatalf("function plan = %+v", function)
	}
	want := []CycleEdge{{From: 1, To: 2}, {From: 2, To: 1}}
	for index := range want {
		if function.CycleEdges[index] != want[index] {
			t.Fatalf("cycle edges = %+v, want %+v", function.CycleEdges, want)
		}
	}
}

func TestBuildMeterPlanRefusesRejectedProgram(t *testing.T) {
	t.Parallel()
	program := mir.Program{Functions: []mir.Function{{
		Symbol:     1,
		Name:       "main::main",
		ResultType: hir.TypeI32,
		Entry:      0,
		Blocks: []mir.Block{{
			ID:         0,
			Terminator: mir.Terminator{Kind: mir.TerminatorGoto, Successors: []mir.BlockID{0}},
		}},
	}}}
	if _, err := BuildMeterPlan(program, nil); err == nil {
		t.Fatal("BuildMeterPlan accepted an unbounded cycle")
	}
}
