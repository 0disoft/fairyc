package contractprofile

import (
	"slices"

	"github.com/0disoft/fairyc/internal/mir"
)

const unassignedControlFlowIndex = -1

func controlFlowCycleInfo(function mir.Function) (cycle bool, malformed bool, cycleEdges []CycleEdge) {
	adjacency, malformed := controlFlowAdjacency(function)
	component, cyclic := controlFlowComponentInfo(adjacency)
	cycleEdges = controlFlowCycleEdges(function.Blocks, adjacency, component, cyclic)
	return len(cycleEdges) != 0, malformed, cycleEdges
}

// controlFlowAdjacency validates identities before resolving successors. All arrays are local.
func controlFlowAdjacency(function mir.Function) ([][]int, bool) {
	malformed := false
	blockIndex := make(map[mir.BlockID]int, len(function.Blocks))
	for index, block := range function.Blocks {
		if _, duplicate := blockIndex[block.ID]; duplicate {
			malformed = true
		}
		blockIndex[block.ID] = index
	}
	if _, found := blockIndex[function.Entry]; !found {
		malformed = true
	}
	adjacency := make([][]int, len(function.Blocks))
	for index, block := range function.Blocks {
		targets, invalid := canonicalControlFlowSuccessors(block.Terminator.Successors, blockIndex)
		adjacency[index] = targets
		malformed = malformed || invalid
	}
	return adjacency, malformed
}

func canonicalControlFlowSuccessors(successors []mir.BlockID, blockIndex map[mir.BlockID]int) ([]int, bool) {
	var targets []int
	malformed := false
	for _, successor := range successors {
		if successor == mir.NoBlock {
			continue
		}
		target, found := blockIndex[successor]
		if !found {
			malformed = true
			continue
		}
		targets = append(targets, target)
	}
	slices.Sort(targets)
	return slices.Compact(targets), malformed
}

type controlFlowComponents struct {
	adjacency [][]int
	indices   []int
	lowLink   []int
	onStack   []bool
	stack     []int
	component []int
	cyclic    []bool
	nextIndex int
}

// controlFlowComponentInfo assigns each component its root's block position. It retains no
// member arrays: the active stack already contains every member until that component completes.
func controlFlowComponentInfo(adjacency [][]int) ([]int, []bool) {
	analysis := controlFlowComponents{
		adjacency: adjacency,
		indices:   make([]int, len(adjacency)),
		lowLink:   make([]int, len(adjacency)),
		onStack:   make([]bool, len(adjacency)),
		stack:     make([]int, 0, len(adjacency)),
		component: make([]int, len(adjacency)),
		cyclic:    make([]bool, len(adjacency)),
	}
	for index := range analysis.indices {
		analysis.indices[index] = unassignedControlFlowIndex
		analysis.component[index] = unassignedControlFlowIndex
	}
	for index := range adjacency {
		if analysis.indices[index] == unassignedControlFlowIndex {
			analysis.visit(index)
		}
	}
	return analysis.component, analysis.cyclic
}

func (analysis *controlFlowComponents) visit(block int) {
	analysis.indices[block] = analysis.nextIndex
	analysis.lowLink[block] = analysis.nextIndex
	analysis.nextIndex++
	analysis.stack = append(analysis.stack, block)
	analysis.onStack[block] = true
	for _, target := range analysis.adjacency[block] {
		if analysis.indices[target] == unassignedControlFlowIndex {
			analysis.visit(target)
			analysis.lowLink[block] = min(analysis.lowLink[block], analysis.lowLink[target])
			continue
		}
		if analysis.onStack[target] {
			analysis.lowLink[block] = min(analysis.lowLink[block], analysis.indices[target])
		}
	}
	if analysis.lowLink[block] != analysis.indices[block] {
		return
	}
	analysis.completeComponent(block)
}

func (analysis *controlFlowComponents) completeComponent(root int) {
	start := len(analysis.stack) - 1
	for analysis.stack[start] != root {
		start--
	}
	members := analysis.stack[start:]
	analysis.cyclic[root] = len(members) > 1 || slices.Contains(analysis.adjacency[root], root)
	for _, member := range members {
		analysis.component[member] = root
		analysis.onStack[member] = false
	}
	analysis.stack = analysis.stack[:start]
}

func controlFlowCycleEdges(blocks []mir.Block, adjacency [][]int, component []int, cyclic []bool) []CycleEdge {
	var edges []CycleEdge
	for from, targets := range adjacency {
		componentID := component[from]
		if !cyclic[componentID] {
			continue
		}
		for _, to := range targets {
			if component[to] != componentID {
				continue
			}
			edges = append(edges, CycleEdge{From: blocks[from].ID, To: blocks[to].ID})
		}
	}
	slices.SortFunc(edges, func(left, right CycleEdge) int {
		if left.From != right.From {
			return int(left.From - right.From)
		}
		return int(left.To - right.To)
	})
	return edges
}
