package cbackend

import (
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

// Inspect all verified MIR for capability errors, but emit static integer helpers
// only for executable C uses. Pruned guards and continuations must not leave
// unused static functions for strict C consumers.
func (e *emitter) refreshIntegerHelperDemand() {
	e.needsBitcast, e.needsShift = false, false
	e.i64Helpers = i64Helpers{}
	for _, fn := range e.functions {
		reachable := reachableBlocksForC(fn, e.nonReturning, e.checkedFailures[fn.Symbol])
		for i, block := range fn.Blocks {
			if !reachable[i] {
				continue
			}
			for _, op := range block.Operations {
				if op.Kind != mir.OperationUnary && op.Kind != mir.OperationBinary {
					continue
				}
				bits, shift := false, false
				if op.Kind == mir.OperationUnary {
					bits = op.Operator == "-" || op.Operator == "~"
				} else {
					switch op.Operator {
					case "+", "-", "*", "&", "^", "|", "<<":
						bits = true
					case ">>":
						shift = true
					}
				}
				switch op.OwnerType {
				case hir.TypeI32:
					e.needsBitcast = e.needsBitcast || bits || shift
					e.needsShift = e.needsShift || shift
				case hir.TypeI64:
					e.i64Helpers.bitcast = e.i64Helpers.bitcast || bits
					e.i64Helpers.shift = e.i64Helpers.shift || shift
				}
			}
			term := block.Terminator
			if term.Kind != mir.TerminatorSwitch || term.SwitchType != hir.TypeI64 {
				continue
			}
			if hasCheckedFailure(e.checkedFailures[fn.Symbol], block.ID) {
				continue
			}
			if _, pruned := constantCheckedFailure(block); pruned {
				continue
			}
			switch term.SwitchKind {
			case "checked-arithmetic":
				switch term.Operator {
				case "+":
					e.i64Helpers.addOverflow = true
				case "-":
					e.i64Helpers.subOverflow = true
				case "*":
					e.i64Helpers.mulOverflow = true
				}
			case "checked-shift":
				if term.Operator == "<<" {
					e.i64Helpers.shift = true
				}
			}
		}
	}
}
