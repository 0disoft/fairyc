package cbackend

import (
	"fmt"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

// Facts are transient C-emission proofs, never changes to verified MIR. Only
// single-predecessor paths carry them; writes and unknown operations/calls reset
// all state. A copied field is a snapshot: no fact crosses a subsequent write.
type assertionValue struct {
	place     string
	reference bool
	literal   mir.Operand
}
type assertionState struct {
	locals     map[mir.LocalID]assertionValue
	equalities map[mir.LocalID]assertionValue
	facts      map[string]mir.Operand
	addressed  map[mir.LocalID]bool
}
type fieldReturn struct {
	parameter int
	path      string
}
type checkedFailureProofs map[hir.SymbolID]map[mir.BlockID]mir.BlockID

func newAssertionState() assertionState {
	return assertionState{map[mir.LocalID]assertionValue{}, map[mir.LocalID]assertionValue{}, map[string]mir.Operand{}, map[mir.LocalID]bool{}}
}
func (s assertionState) clone() assertionState {
	c := newAssertionState()
	for k, v := range s.locals {
		c.locals[k] = v
	}
	for k, v := range s.equalities {
		c.equalities[k] = v
	}
	for k, v := range s.addressed {
		c.addressed[k] = v
	}
	for k, v := range s.facts {
		c.facts[k] = v
	}
	return c
}
func (s assertionState) value(op mir.Operand) assertionValue {
	if op.Kind == mir.OperandLocal {
		if v, ok := s.locals[op.Local]; ok {
			return v
		}
		return assertionValue{}
	}
	switch op.Kind {
	case mir.OperandI32, mir.OperandI64, mir.OperandU8, mir.OperandU32, mir.OperandUSize, mir.OperandByteString:
		return assertionValue{literal: op}
	}
	return assertionValue{}
}
func (s assertionState) resolve(op mir.Operand) mir.Operand {
	v := s.value(op)
	if v.literal.Kind != "" {
		return v.literal
	}
	if !v.reference {
		if n, ok := s.facts[v.place]; ok {
			return n
		}
	}
	return op
}
func (s *assertionState) operation(op mir.Operation) {
	// Definitions are not SSA: descriptor/scalar rebinding invalidates identities.
	if op.Kind != mir.OperationArithmeticCheck && op.Kind != mir.OperationViewIndexCheck && op.Kind != mir.OperationViewSliceCheck {
		_, defined := s.locals[op.DestinationLocal]
		_, compared := s.equalities[op.DestinationLocal]
		if defined || compared || s.addressed[op.DestinationLocal] {
			*s = newAssertionState()
		}
	}

	var v assertionValue
	if len(op.Operands) == 1 {
		v = s.value(op.Operands[0])
	}
	delete(s.equalities, op.DestinationLocal)
	switch op.Kind {
	case mir.OperationAssign, mir.OperationInitialize:
		s.locals[op.DestinationLocal] = v
	case mir.OperationSharedReborrow:
		if !v.reference {
			v = assertionValue{}
		}
		s.locals[op.DestinationLocal] = v
	case mir.OperationFieldBorrow:
		_, _, borrowed := hir.BorrowElement(op.OwnerType)
		if !v.reference || !borrowed {
			v = assertionValue{}
		} else {
			v.place += "/" + op.Field
		}
		s.locals[op.DestinationLocal] = v
	case mir.OperationFieldRead:
		if v.place != "" {
			v.place += "/" + op.Field
			v.reference = false
			v.literal = mir.Operand{}
		}
		s.locals[op.DestinationLocal] = v
	case mir.OperationUnary:
		switch op.Operator {
		case "*":
			if !v.reference {
				v = assertionValue{}
			} else {
				v.reference = false
			}
		case "&", "&mut":
			if len(op.Operands) != 1 || op.Operands[0].Kind != mir.OperandLocal {
				v = assertionValue{}
			} else {
				s.addressed[op.Operands[0].Local] = true
				v = assertionValue{place: fmt.Sprintf("v%d", op.Operands[0].Local), reference: true}
			}
		default:
			*s = newAssertionState()
			return
		}
		s.locals[op.DestinationLocal] = v
	case mir.OperationBinary:
		if op.Operator != "==" || len(op.Operands) != 2 {
			*s = newAssertionState()
			return
		}
		a, b := s.value(op.Operands[0]), s.value(op.Operands[1])
		if a.literal.Kind != "" {
			a, b = b, a
		}
		if a.place != "" && !a.reference && b.literal.Kind != "" && b.literal.Type != hir.TypeBool {
			s.equalities[op.DestinationLocal] = assertionValue{place: a.place, literal: b.literal}
		}
		delete(s.locals, op.DestinationLocal)
	case mir.OperationArithmeticCheck, mir.OperationViewIndexCheck, mir.OperationViewSliceCheck:
		// Verified marker has no write.
	default:
		*s = newAssertionState()
	}
}

// Only an effect-free, single-block reference projection can be summarized.
// No arbitrary call, mutation, recursion or return-origin metadata is trusted.
func pureFieldReturns(functions []mir.Function) map[hir.SymbolID]fieldReturn {
	result := map[hir.SymbolID]fieldReturn{}
	for _, f := range functions {
		_, _, borrowed := hir.BorrowElement(f.ResultType)
		if !borrowed || len(f.Effects) != 0 || len(f.Blocks) != 1 {
			continue
		}
		b := f.Blocks[0]
		if b.Terminator.Kind != mir.TerminatorReturn || b.Terminator.UnwindSuccessor != mir.NoBlock {
			continue
		}
		s := newAssertionState()
		for i, p := range f.Parameters {
			if _, _, ok := hir.BorrowElement(f.ParameterTypes[i]); ok {
				s.locals[p] = assertionValue{place: fmt.Sprintf("p%d", i), reference: true}
			}
		}
		valid := true
		for _, op := range b.Operations {
			switch op.Kind {
			case mir.OperationAssign, mir.OperationInitialize, mir.OperationSharedReborrow, mir.OperationFieldBorrow:
				s.operation(op)
			default:
				valid = false
			}
		}
		if !valid {
			continue
		}
		v := s.value(b.Terminator.ReturnValue)
		if !v.reference {
			continue
		}
		for i := range f.Parameters {
			root := fmt.Sprintf("p%d", i)
			if v.place == root || len(v.place) > len(root) && v.place[:len(root)+1] == root+"/" {
				result[f.Symbol] = fieldReturn{i, v.place[len(root):]}
				break
			}
		}
	}
	return result
}

func assertedCheckedFailuresForC(functions []mir.Function) checkedFailureProofs {
	proofs := checkedFailureProofs{}
	summaries := pureFieldReturns(functions)
	for _, f := range functions {
		selected := map[mir.BlockID]mir.BlockID{}
		proofs[f.Symbol] = selected
		predecessors := make([][]mir.BlockID, len(f.Blocks))
		for _, b := range f.Blocks {
			for _, next := range b.Terminator.Successors {
				predecessors[int(next)] = append(predecessors[int(next)], b.ID)
			}
			if next := b.Terminator.UnwindSuccessor; next != mir.NoBlock {
				predecessors[int(next)] = append(predecessors[int(next)], b.ID)
			}
		}
		states := map[mir.BlockID]assertionState{}
		visiting := map[mir.BlockID]bool{}
		var analyze func(mir.BlockID) assertionState
		analyze = func(id mir.BlockID) assertionState {
			if s, ok := states[id]; ok {
				return s.clone()
			}
			if visiting[id] {
				return newAssertionState()
			}
			visiting[id] = true
			s := newAssertionState()
			if id == f.Entry {
				for i, p := range f.Parameters {
					_, _, borrowed := hir.BorrowElement(f.ParameterTypes[i])
					s.locals[p] = assertionValue{place: fmt.Sprintf("v%d", p), reference: borrowed}
				}
			} else if pred := predecessors[int(id)]; len(pred) == 1 {
				previous := pred[0]
				s = analyze(previous)
				term := f.Blocks[int(previous)].Terminator
				if term.UnwindSuccessor == id {
					s = newAssertionState()
				}
				if term.Kind == mir.TerminatorSwitch && term.SwitchKind == "assert" && len(term.SwitchOperands) == 1 && term.SwitchOperands[0].Kind == mir.OperandLocal && term.Successors[0] == id {
					if eq, ok := s.equalities[term.SwitchOperands[0].Local]; ok {
						s.facts[eq.place] = eq.literal
					}
				}
			}
			for _, op := range f.Blocks[int(id)].Operations {
				s.operation(op)
			}
			term := f.Blocks[int(id)].Terminator
			if term.Kind == mir.TerminatorSwitch && len(term.Successors) == 2 {
				operands := make([]mir.Operand, len(term.SwitchOperands))
				for i, op := range term.SwitchOperands {
					operands[i] = s.resolve(op)
				}
				if literalCheckedFailure(term, operands) {
					selected[id] = term.Successors[1]
				}
			}
			if term.Kind == mir.TerminatorCall {
				if summary, ok := summaries[term.Callee]; ok && summary.parameter < len(term.Arguments) {
					v := s.value(term.Arguments[summary.parameter])
					if v.reference {
						v.place += summary.path
					} else {
						v = assertionValue{}
					}
					s.locals[term.DestinationLocal] = v
					delete(s.equalities, term.DestinationLocal)
				} else {
					s = newAssertionState()
				}
			}
			states[id] = s.clone()
			delete(visiting, id)
			return s
		}
		// Start at entry; remaining roots have no incoming proof. Cycles are cut.
		analyze(f.Entry)
		for _, b := range f.Blocks {
			analyze(b.ID)
		}
	}
	return proofs
}
