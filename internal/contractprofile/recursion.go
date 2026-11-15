package contractprofile

import "slices"

const undiscoveredFunction = -1

type callGraphRecursion struct {
	nodes     []functionAnalysis
	indices   []int
	lowLink   []int
	onStack   []bool
	stack     []int
	recursive []bool
	nextIndex int
}

// recursiveFunctionMask computes call-cycle membership without modifying nodes or their edges.
// Tarjan's algorithm needs neither sorted nor unique edges; all working state is local.
func recursiveFunctionMask(nodes []functionAnalysis) []bool {
	analysis := callGraphRecursion{
		nodes:     nodes,
		indices:   make([]int, len(nodes)),
		lowLink:   make([]int, len(nodes)),
		onStack:   make([]bool, len(nodes)),
		stack:     make([]int, 0, len(nodes)),
		recursive: make([]bool, len(nodes)),
	}
	for index := range analysis.indices {
		analysis.indices[index] = undiscoveredFunction
	}
	for index := range nodes {
		if analysis.indices[index] == undiscoveredFunction {
			analysis.visit(index)
		}
	}
	return analysis.recursive
}

func (analysis *callGraphRecursion) visit(node int) {
	analysis.indices[node] = analysis.nextIndex
	analysis.lowLink[node] = analysis.nextIndex
	analysis.nextIndex++
	analysis.stack = append(analysis.stack, node)
	analysis.onStack[node] = true
	for _, target := range analysis.nodes[node].callTargets {
		if analysis.indices[target] == undiscoveredFunction {
			analysis.visit(target)
			analysis.lowLink[node] = min(analysis.lowLink[node], analysis.lowLink[target])
			continue
		}
		if analysis.onStack[target] {
			analysis.lowLink[node] = min(analysis.lowLink[node], analysis.indices[target])
		}
	}
	if analysis.lowLink[node] != analysis.indices[node] {
		return
	}
	analysis.completeComponent(node)
}

func (analysis *callGraphRecursion) completeComponent(root int) {
	start := len(analysis.stack) - 1
	for analysis.stack[start] != root {
		start--
	}
	// Completed members occupy one stack suffix; consume it before reusing that storage.
	members := analysis.stack[start:]
	recursive := len(members) > 1 || slices.Contains(analysis.nodes[root].callTargets, root)
	for _, member := range members {
		analysis.onStack[member] = false
		analysis.recursive[member] = recursive
	}
	analysis.stack = analysis.stack[:start]
}
