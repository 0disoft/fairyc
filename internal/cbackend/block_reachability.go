package cbackend

import (
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

// reachableBlocksForC omits MIR blocks that have no executable path from the
// entry. MIR verification and canonical block-ID checks run before emission.
func reachableBlocksForC(function mir.Function, nonReturning map[hir.SymbolID]bool) []bool {
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
func nonReturningFunctionsForC(functions []mir.Function) map[hir.SymbolID]bool {
	known := make(map[hir.SymbolID]bool)
	for {
		changed := false
		for _, function := range functions {
			if known[function.Symbol] {
				continue
			}
			reachable := reachableBlocksForC(function, known)
			returns := false
			for index, block := range function.Blocks {
				if reachable[index] && block.Terminator.Kind == mir.TerminatorReturn {
					returns = true
					break
				}
			}
			if !returns {
				known[function.Symbol] = true
				changed = true
			}
		}
		if !changed {
			return known
		}
	}
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
