package cbackend

import (
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

// reachableBlocksForC omits MIR blocks that have no executable path from the
// entry. MIR verification and canonical block-ID checks run before emission.
func reachableBlocksForC(function mir.Function, nonReturning map[hir.SymbolID]bool, proofs map[mir.BlockID]mir.BlockID) []bool {
	reachable := make([]bool, len(function.Blocks))
	queue := []mir.BlockID{function.Entry}
	for len(queue) != 0 {
		id := queue[0]
		queue = queue[1:]
		index := int(id)
		if index < 0 || index >= len(function.Blocks) || reachable[index] {
			continue
		}
		block := &function.Blocks[index]
		reachable[index] = true
		if block.Terminator.Kind == mir.TerminatorCall && nonReturning[block.Terminator.Callee] {
			// A trapping or infinite callee cannot reach the normal continuation.
		} else if hasCheckedFailure(proofs, id) {
			queue = append(queue, proofs[id])
		} else if selected, failure := constantCheckedFailure(*block); failure {
			queue = append(queue, selected)
		} else if selected, constant := constantBooleanSuccessor(block.Terminator); constant {
			queue = append(queue, selected)
		} else {
			queue = append(queue, block.Terminator.Successors...)
		}
		if block.Terminator.UnwindSuccessor != mir.NoBlock {
			queue = append(queue, block.Terminator.UnwindSuccessor)
		}
	}
	return reachable
}

// Infer only from verified local CFGs and already proven local callees. Foreign
// calls remain unknown, and unwind edges are retained conservatively.
func nonReturningFunctionsForC(functions []mir.Function, proofs checkedFailureProofs) map[hir.SymbolID]bool {
	known := make(map[hir.SymbolID]bool)
	callers := make(map[hir.SymbolID][]int)
	pending := make([]int, len(functions))
	queued := make([]bool, len(functions))
	for index, function := range functions {
		pending[index], queued[index] = index, true
		for _, block := range function.Blocks {
			if term := block.Terminator; term.Kind == mir.TerminatorCall {
				callers[term.Callee] = append(callers[term.Callee], index)
			}
		}
	}
	// Only callers can lose a returning path when a callee becomes known.
	// Include dead call sites conservatively; CFG analysis still owns reachability.
	for head := 0; head < len(pending); head++ {
		index := pending[head]
		queued[index] = false
		function := &functions[index]
		if known[function.Symbol] {
			continue
		}
		reachable := reachableBlocksForC(*function, known, proofs[function.Symbol])
		returns := false
		for index, block := range function.Blocks {
			if reachable[index] && block.Terminator.Kind == mir.TerminatorReturn {
				returns = true
				break
			}
		}
		if returns {
			continue
		}
		known[function.Symbol] = true
		for _, caller := range callers[function.Symbol] {
			if !known[functions[caller].Symbol] && !queued[caller] {
				pending = append(pending, caller)
				queued[caller] = true
			}
		}
	}
	return known
}

// Literal bool edges must agree in reachability and C output. Otherwise MSVC
// rejects the impossible fallthrough/return after assert(false) or while(true).
func constantBooleanSuccessor(terminator mir.Terminator) (mir.BlockID, bool) {
	if terminator.Kind != mir.TerminatorSwitch || (terminator.SwitchKind != "assert" && terminator.SwitchKind != "bool") || len(terminator.SwitchOperands) != 1 || len(terminator.Successors) != 2 {
		return mir.NoBlock, false
	}
	operand := terminator.SwitchOperands[0]
	if operand.Kind != mir.OperandI32 || operand.Type != hir.TypeBool || (operand.I32 != 0 && operand.I32 != 1) {
		return mir.NoBlock, false
	}
	if operand.I32 == 0 {
		return terminator.Successors[1], true
	}
	return terminator.Successors[0], true
}

func hasCheckedFailure(proofs map[mir.BlockID]mir.BlockID, id mir.BlockID) bool {
	_, ok := proofs[id]
	return ok
}
