package contractprofile

import (
	"cmp"
	"reflect"
	"slices"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

func TestControlFlowCycleInfoMatchesReachabilityAndPreservesInput(t *testing.T) {
	t.Parallel()
	for count := range 4 {
		for graph := range 1 << (count * count) {
			function, reachable := controlFlowGraphFixture(count, graph)
			before := function
			before.Blocks = slices.Clone(function.Blocks)
			for index := range before.Blocks {
				before.Blocks[index].Terminator.Successors = slices.Clone(function.Blocks[index].Terminator.Successors)
			}
			want := expectedControlFlowCycleEdges(function, reachable)
			cycle, malformed, edges := controlFlowCycleInfo(function)
			if cycle != (len(want) != 0) || malformed != (count == 0) || !slices.Equal(edges, want) {
				t.Fatalf("nodes=%d graph=%b: cycle=%t malformed=%t edges=%v, want %v", count, graph, cycle, malformed, edges, want)
			}
			if !reflect.DeepEqual(function, before) {
				t.Fatalf("nodes=%d graph=%b: caller-owned blocks or successors changed", count, graph)
			}
		}
	}
}

func controlFlowGraphFixture(count, graph int) (mir.Function, [][]bool) {
	function := mir.Function{Entry: mir.BlockID(count * 7), Blocks: make([]mir.Block, count)}
	reachable := make([][]bool, count)
	for from := range count {
		function.Blocks[from].ID = mir.BlockID((count - from) * 7)
		reachable[from] = make([]bool, count)
		function.Blocks[from].Terminator.Successors = []mir.BlockID{mir.NoBlock}
		for to := count - 1; to >= 0; to-- {
			if graph&(1<<(from*count+to)) != 0 {
				target := mir.BlockID((count - to) * 7)
				function.Blocks[from].Terminator.Successors = append(function.Blocks[from].Terminator.Successors, target, target)
				reachable[from][to] = true
			}
		}
	}
	for via := range count {
		for from := range count {
			for to := range count {
				reachable[from][to] = reachable[from][to] || reachable[from][via] && reachable[via][to]
			}
		}
	}
	return function, reachable
}

func expectedControlFlowCycleEdges(function mir.Function, reachable [][]bool) []CycleEdge {
	var edges []CycleEdge
	for from, block := range function.Blocks {
		for to, target := range function.Blocks {
			if slices.Contains(block.Terminator.Successors, target.ID) && reachable[to][from] {
				edges = append(edges, CycleEdge{From: block.ID, To: target.ID})
			}
		}
	}
	slices.SortFunc(edges, func(left, right CycleEdge) int {
		if order := cmp.Compare(left.From, right.From); order != 0 {
			return order
		}
		return cmp.Compare(left.To, right.To)
	})
	return edges
}

func TestControlFlowCycleInfoPreservesMalformedGraphResults(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name     string
		function mir.Function
		edges    []CycleEdge
	}{
		{
			name: "missing entry retains self-cycle",
			function: mir.Function{Entry: 9, Blocks: []mir.Block{
				{ID: 1, Terminator: mir.Terminator{Successors: []mir.BlockID{1}}},
			}},
			edges: []CycleEdge{{From: 1, To: 1}},
		},
		{
			name: "unknown successor retains known self-cycle",
			function: mir.Function{Entry: 1, Blocks: []mir.Block{
				{ID: 1, Terminator: mir.Terminator{Successors: []mir.BlockID{9, 1, mir.NoBlock}}},
			}},
			edges: []CycleEdge{{From: 1, To: 1}},
		},
		{
			name: "duplicate block uses last target",
			function: mir.Function{Entry: 1, Blocks: []mir.Block{
				{ID: 1, Terminator: mir.Terminator{Successors: []mir.BlockID{1}}}, {ID: 1},
			}},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			cycle, malformed, edges := controlFlowCycleInfo(test.function)
			if !malformed || cycle != (len(test.edges) != 0) || !slices.Equal(edges, test.edges) {
				t.Fatalf("cycle=%t malformed=%t edges=%v, want %v", cycle, malformed, edges, test.edges)
			}
		})
	}
}

func TestBuildMeterPlanOmitsBridgeBetweenCyclicComponents(t *testing.T) {
	t.Parallel()
	program := mir.Program{Functions: []mir.Function{{
		Symbol: 1, Name: "main::main", ResultType: hir.TypeI32, Entry: 0,
		Blocks: []mir.Block{
			{ID: 0, Terminator: mir.Terminator{Kind: mir.TerminatorGoto, Successors: []mir.BlockID{1}}},
			{ID: 1, Terminator: mir.Terminator{Kind: mir.TerminatorGoto, Successors: []mir.BlockID{2}}},
			{ID: 2, Terminator: mir.Terminator{Kind: mir.TerminatorSwitch, Successors: []mir.BlockID{1, 3}}},
			{ID: 3, Terminator: mir.Terminator{Kind: mir.TerminatorGoto, Successors: []mir.BlockID{4}}},
			{ID: 4, Terminator: mir.Terminator{Kind: mir.TerminatorSwitch, Successors: []mir.BlockID{3, 5}}},
			{ID: 5, Terminator: mir.Terminator{Kind: mir.TerminatorReturn}},
		},
	}}}
	plan, err := BuildMeterPlan(program, map[string]uint64{"main::main": 7})
	if err != nil || len(plan.Functions) != 1 {
		t.Fatalf("plan=%+v error=%v", plan, err)
	}
	want := []CycleEdge{{From: 1, To: 2}, {From: 2, To: 1}, {From: 3, To: 4}, {From: 4, To: 3}}
	if got := plan.Functions[0].CycleEdges; !slices.Equal(got, want) {
		t.Fatalf("cycle edges=%v, want %v", got, want)
	}
}

func BenchmarkControlFlowCycleInfo(b *testing.B) {
	for _, cycle := range []bool{false, true} {
		name := "Chain4096"
		if cycle {
			name = "Cycle4096"
		}
		b.Run(name, func(b *testing.B) {
			function := mir.Function{Entry: 0, Blocks: make([]mir.Block, 4096)}
			for index := range function.Blocks {
				function.Blocks[index].ID = mir.BlockID(index)
				if index+1 < len(function.Blocks) {
					function.Blocks[index].Terminator.Successors = []mir.BlockID{mir.BlockID(index + 1)}
				}
			}
			if cycle {
				function.Blocks[len(function.Blocks)-1].Terminator.Successors = []mir.BlockID{0}
			}
			b.ReportAllocs()
			for b.Loop() {
				gotCycle, malformed, edges := controlFlowCycleInfo(function)
				if gotCycle != cycle || malformed || cycle && len(edges) != len(function.Blocks) {
					b.Fatalf("cycle=%t malformed=%t edge count=%d", gotCycle, malformed, len(edges))
				}
			}
		})
	}
}
