package mir

import "github.com/0disoft/fairyc/internal/hir"

type verificationRetention struct {
	last  map[hir.SymbolID]int
	floor []int
}

// Find the lowest execution-order rank reachable from each block, without
// per-block reachability sets. Storage IDs need not follow execution order:
// a switch continuation can be allocated before either arm.
// Bounds and last mentions take linear work in blocks, edges and instructions.
func verificationLocalRetention(function *Function) verificationRetention {
	r := verificationRetention{last: make(map[hir.SymbolID]int), floor: make([]int, len(function.Blocks))}
	indices := make(map[BlockID]int, len(function.Blocks))
	predecessors := make(map[BlockID][]BlockID)
	for i, block := range function.Blocks {
		if block.ID != BlockID(i) {
			return r // Malformed identity: retain all state for normal validation.
		}
		indices[block.ID] = i
		for _, successor := range block.Terminator.Successors {
			if successor < 0 || int(successor) >= len(function.Blocks) {
				return r
			}
			predecessors[successor] = append(predecessors[successor], block.ID)
		}
	}
	order := dominatorBlockOrder(function, indices, predecessors)
	rank := make([]int, len(function.Blocks))
	for i, block := range order {
		rank[block] = i
	}
	mention := func(symbol hir.SymbolID, block int) {
		if symbol >= 0 {
			r.last[symbol] = max(r.last[symbol], rank[block])
		}
	}
	operands := func(values []Operand, block int) {
		for _, value := range values {
			if value.Kind == OperandLocal || value.Kind == OperandRegion {
				mention(hir.SymbolID(value.Local), block)
			}
		}
	}
	for i, block := range function.Blocks {
		for _, op := range block.Operations {
			mention(op.Symbol, i)
			mention(hir.SymbolID(op.DestinationLocal), i)
			operands(op.Operands, i)
		}
		term := block.Terminator
		mention(hir.SymbolID(term.DestinationLocal), i)
		operands(term.Arguments, i)
		operands(term.SwitchOperands, i)
		operands([]Operand{term.ReturnValue}, i)
	}
	for i := range r.floor {
		r.floor[i] = -1
	}
	// Ascending seeds reach predecessors in order of their minimum reachable rank.
	// Each block is assigned once and each reverse edge is visited once.
	var queue []BlockID
	for _, root := range order {
		if r.floor[root] >= 0 {
			continue
		}
		r.floor[root] = rank[root]
		queue = append(queue[:0], root)
		for len(queue) > 0 {
			block := queue[len(queue)-1]
			queue = queue[:len(queue)-1]
			for _, previous := range predecessors[block] {
				if r.floor[previous] < 0 {
					r.floor[previous] = rank[root]
					queue = append(queue, previous)
				}
			}
		}
	}
	return r
}

func (r verificationRetention) retire(live map[hir.SymbolID]bool, successors []BlockID, resources map[hir.SymbolID]ResourceLocal) {
	floor := len(r.floor)
	for _, successor := range successors {
		if successor < 0 || int(successor) >= len(r.floor) {
			return
		}
		floor = min(floor, r.floor[successor])
	}
	for symbol := range live {
		if _, resource := resources[symbol]; !resource && r.last[symbol] < floor {
			delete(live, symbol)
		}
	}
}
