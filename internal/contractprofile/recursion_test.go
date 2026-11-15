package contractprofile

import (
	"reflect"
	"slices"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

func TestRecursiveFunctionMaskMatchesReachabilityAndPreservesInput(t *testing.T) {
	t.Parallel()
	// Exhaust every directed graph with up to three nodes, including self-calls.
	for count := range 4 {
		for graph := range 1 << (count * count) {
			nodes := recursionGraphFromBits(count, graph)
			before := slices.Clone(nodes)
			for index := range before {
				before[index].callTargets = slices.Clone(nodes[index].callTargets)
			}
			want := recursiveFunctionsByReachability(nodes)
			got := recursiveFunctionMask(nodes)
			if !slices.Equal(got, want) {
				t.Fatalf("nodes=%d graph=%b: recursive=%v, want %v", count, graph, got, want)
			}
			if !reflect.DeepEqual(nodes, before) {
				t.Fatalf("nodes=%d graph=%b: input was modified", count, graph)
			}
		}
	}
}

func TestRecursiveFunctionMaskSeparatesCompletedComponents(t *testing.T) {
	t.Parallel()
	nodes := []functionAnalysis{
		{callTargets: []int{3, 1, 1}}, // Calls cycles without belonging to them.
		{callTargets: []int{3, 2}},    // Cross-edge to an already completed cycle.
		{callTargets: []int{1, 6}},
		{callTargets: []int{4}},
		{callTargets: []int{6, 3}},
		{callTargets: []int{5, 0}}, // Self-call after every other component completes.
		{},
	}
	want := []bool{false, true, true, true, true, true, false}
	if got := recursiveFunctionMask(nodes); !slices.Equal(got, want) {
		t.Fatalf("recursive = %v, want %v", got, want)
	}
}

func recursionGraphFromBits(count, graph int) []functionAnalysis {
	nodes := make([]functionAnalysis, count)
	for from := range nodes {
		for to := count - 1; to >= 0; to-- {
			if graph&(1<<(from*count+to)) != 0 {
				// Reverse order and duplicate edges must not affect cycle membership.
				nodes[from].callTargets = append(nodes[from].callTargets, to, to)
			}
		}
	}
	return nodes
}

func recursiveFunctionsByReachability(nodes []functionAnalysis) []bool {
	reachable := make([][]bool, len(nodes))
	for from := range nodes {
		reachable[from] = make([]bool, len(nodes))
		for _, to := range nodes[from].callTargets {
			reachable[from][to] = true
		}
	}
	for via := range nodes {
		for from := range nodes {
			for to := range nodes {
				reachable[from][to] = reachable[from][to] || reachable[from][via] && reachable[via][to]
			}
		}
	}
	recursive := make([]bool, len(nodes))
	for index := range nodes {
		recursive[index] = reachable[index][index]
	}
	return recursive
}

func TestAnalyzeRepeatedCallsRetainTransitiveCost(t *testing.T) {
	t.Parallel()
	program := mir.Program{Functions: []mir.Function{
		{
			Symbol: 2, Name: "b::leaf", ResultType: hir.TypeI32, Entry: 0,
			Blocks: []mir.Block{{ID: 0, Terminator: mir.Terminator{Kind: mir.TerminatorReturn}}},
		},
		{
			Symbol: 1, Name: "a::caller", ResultType: hir.TypeI32, Entry: 0,
			Blocks: []mir.Block{
				{ID: 0, Terminator: mir.Terminator{Kind: mir.TerminatorCall, Callee: 2, Successors: []mir.BlockID{1}}},
				{ID: 1, Terminator: mir.Terminator{Kind: mir.TerminatorCall, Callee: 2, Successors: []mir.BlockID{2}}},
				{ID: 2, Terminator: mir.Terminator{Kind: mir.TerminatorReturn}},
			},
		},
	}}
	report := Analyze(program, nil)
	if !report.Accepted || len(report.Functions) != 2 {
		t.Fatalf("report = %+v", report)
	}
	caller, leaf := report.Functions[0], report.Functions[1]
	wantCost := caller.LocalAbstractCost + 2*leaf.LocalAbstractCost
	if !caller.Bounded || !leaf.Bounded || caller.TransitiveCostUpperBound != wantCost {
		t.Fatalf("caller = %+v, leaf = %+v, want caller cost %d", caller, leaf, wantCost)
	}
	if !slices.Equal(caller.InternalCalls, []string{"b::leaf"}) {
		t.Fatalf("internal calls = %v", caller.InternalCalls)
	}
}

func BenchmarkRecursiveFunctions(b *testing.B) {
	b.Run("Dense256", func(b *testing.B) {
		nodes := make([]functionAnalysis, 256)
		for from := range nodes {
			for edge := range len(nodes) {
				// An odd stride visits every target in an unsorted order.
				target := (edge*109 + from*17) % len(nodes)
				nodes[from].callTargets = append(nodes[from].callTargets, target)
			}
		}
		b.ReportAllocs()
		var recursive []bool
		for b.Loop() {
			recursive = recursiveFunctionMask(nodes)
		}
		if !recursive[0] {
			b.Fatal("dense call graph was not recursive")
		}
	})
	b.Run("Chain4096", func(b *testing.B) {
		nodes := make([]functionAnalysis, 4096)
		for from := range len(nodes) - 1 {
			nodes[from].callTargets = []int{from + 1}
		}
		b.ReportAllocs()
		var recursive []bool
		for b.Loop() {
			recursive = recursiveFunctionMask(nodes)
		}
		if recursive[0] {
			b.Fatal("acyclic call graph was recursive")
		}
	})
}
