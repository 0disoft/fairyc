package mir

import "github.com/0disoft/fairyc/internal/hir"

type verificationRetention struct {
	last  map[hir.SymbolID]int
	floor []int
}

// Conservatively bound every block reachable from each entry. Include even
// unreachable later backedges, so this needs no per-block reachability sets.
// Bounds and last mentions take linear work in blocks, edges and instructions.
func verificationLocalRetention(function *Function) verificationRetention {
	r := verificationRetention{last: make(map[hir.SymbolID]int), floor: make([]int, len(function.Blocks))}
	mention := func(symbol hir.SymbolID, block int) {
		if symbol >= 0 {
			r.last[symbol] = block
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
		if block.ID != BlockID(i) {
			return r // Malformed identity: retain all state for normal validation.
		}
		for _, successor := range block.Terminator.Successors {
			if successor < 0 || int(successor) >= len(function.Blocks) {
				return r
			}
		}
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
	bound := len(function.Blocks)
	for i := len(function.Blocks) - 1; i >= 0; i-- {
		bound = min(bound, i)
		for _, successor := range function.Blocks[i].Terminator.Successors {
			bound = min(bound, int(successor))
		}
		r.floor[i] = bound
	}
	// A later backedge may lead to an earlier backedge. Close those chains in
	// ascending order, where every strictly earlier bound is already closed.
	for i, bound := range r.floor {
		if bound < i {
			r.floor[i] = r.floor[bound]
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
