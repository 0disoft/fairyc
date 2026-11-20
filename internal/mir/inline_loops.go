package mir

// The loop marker permits natural back edges. Removing them must leave a DAG,
// and disconnected blocks may not use the exemption. The ordinary MIR verifier
// still checks all operand, initialization and arithmetic-check obligations.
func inlineControlFlowStructured(fn *Function) bool {
	blocks := fn.Blocks
	if fn.Entry < 0 || int(fn.Entry) >= len(blocks) {
		return false
	}
	for index, block := range blocks {
		if block.ID != BlockID(index) || len(block.Terminator.Successors) > 2 {
			return false
		}
		for _, successor := range block.Terminator.Successors {
			if successor < 0 || int(successor) >= len(blocks) {
				return false
			}
		}
	}

	state := make([]uint8, len(blocks))
	var dominators *blockDominatorSets
	var walk func(BlockID, bool) bool
	walk = func(id BlockID, reachable bool) bool {
		if state[id] == inlineVisitComplete {
			return true
		}
		state[id] = inlineVisitActive
		term := blocks[id].Terminator
		for _, successor := range term.Successors {
			if state[successor] != inlineVisitActive {
				if !walk(successor, reachable) {
					return false
				}
				continue
			}
			if !reachable || successor == fn.Entry || successor == id || term.Kind != TerminatorGoto || len(term.Successors) != 1 {
				return false
			}
			if dominators == nil {
				dominators = inlineReachableDominators(fn)
			}
			if dominators == nil || !dominators.contains(id, successor) {
				return false
			}
		}
		state[id] = inlineVisitComplete
		return true
	}
	if !walk(fn.Entry, true) {
		return false
	}
	for _, block := range blocks {
		if !walk(block.ID, false) {
			return false
		}
	}
	return true
}

func inlineReachableDominators(fn *Function) *blockDominatorSets {
	seen := make([]bool, len(fn.Blocks))
	incoming := make(map[BlockID][]BlockID, len(fn.Blocks))
	var visit func(BlockID)
	visit = func(id BlockID) {
		if seen[id] {
			return
		}
		seen[id] = true
		for _, successor := range fn.Blocks[id].Terminator.Successors {
			incoming[successor] = append(incoming[successor], id)
			visit(successor)
		}
	}
	visit(fn.Entry)
	return blockDominators(fn, incoming)
}
