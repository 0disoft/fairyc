package mir

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/hir"
)

func (l *functionLowerer) lowerIf(expression *hir.Expression) (Operand, error) {
	if len(expression.Children) < 2 || len(expression.Children) > 3 {
		return Operand{}, fmt.Errorf("if expression has %d children", len(expression.Children))
	}
	condition, err := l.lowerExpression(&expression.Children[0])
	if err != nil {
		return Operand{}, err
	}
	if l.blockTerminated(l.current) {
		return Operand{}, nil
	}
	branchLive := cloneLive(l.live)
	thenBlock := l.newBlock()
	elseBlock := l.newBlock()
	joinBlock := l.newBlock()
	l.terminate(Terminator{
		Kind: TerminatorSwitch, Span: expression.Children[0].Span,
		Successors: []BlockID{thenBlock, elseBlock}, UnwindSuccessor: NoBlock,
		SwitchKind: "bool", SwitchOperands: []Operand{condition},
	})

	destination := NoLocal
	if expression.Type != hir.TypeVoid {
		destination = l.newTemporary()
	}

	l.current = thenBlock
	l.live = cloneLive(branchLive)
	thenValue, err := l.lowerExpression(&expression.Children[1])
	if err != nil {
		return Operand{}, err
	}
	thenContinues := !l.blockTerminated(l.current)
	thenLive := cloneLive(l.live)
	if thenContinues {
		if destination != NoLocal {
			if thenValue.Kind == OperandInvalid {
				return Operand{}, fmt.Errorf("value-producing if has an empty then branch")
			}
			l.appendOperation(Operation{Kind: OperationAssign, Span: expression.Children[1].Span, DestinationLocal: destination, Operands: []Operand{thenValue}})
		}
		l.terminate(Terminator{Kind: TerminatorGoto, Span: expression.Children[1].Span, Successors: []BlockID{joinBlock}, UnwindSuccessor: NoBlock})
	}

	l.current = elseBlock
	l.live = cloneLive(branchLive)
	elseValue := Operand{}
	if len(expression.Children) == 3 {
		elseValue, err = l.lowerExpression(&expression.Children[2])
		if err != nil {
			return Operand{}, err
		}
	}
	elseContinues := !l.blockTerminated(l.current)
	elseLive := cloneLive(l.live)
	if elseContinues {
		if destination != NoLocal {
			if elseValue.Kind == OperandInvalid {
				return Operand{}, fmt.Errorf("value-producing if has an empty else branch")
			}
			l.appendOperation(Operation{Kind: OperationAssign, Span: expressionSpan(&expression.Children[len(expression.Children)-1], expression.Span), DestinationLocal: destination, Operands: []Operand{elseValue}})
		}
		l.terminate(Terminator{Kind: TerminatorGoto, Span: expression.Span, Successors: []BlockID{joinBlock}, UnwindSuccessor: NoBlock})
	}

	l.current = joinBlock
	switch {
	case thenContinues && elseContinues:
		if !sameLiveState(thenLive, elseLive) {
			return Operand{}, fmt.Errorf("if branches disagree on resource state after Flow validation")
		}
		l.live = thenLive
	case thenContinues:
		l.live = thenLive
	case elseContinues:
		l.live = elseLive
	default:
		l.live = cloneLive(branchLive)
		l.terminate(Terminator{Kind: TerminatorUnreachable, Span: expression.Span, UnwindSuccessor: NoBlock})
		return Operand{}, nil
	}
	if destination == NoLocal {
		return Operand{}, nil
	}
	return Operand{Kind: OperandLocal, Local: destination}, nil
}

func (l *functionLowerer) lowerShortCircuit(expression *hir.Expression) (Operand, error) {
	left, err := l.lowerExpression(&expression.Children[0])
	if err != nil {
		return Operand{}, err
	}
	if l.blockTerminated(l.current) {
		return Operand{}, nil
	}
	branchLive := cloneLive(l.live)
	rightBlock := l.newBlock()
	shortBlock := l.newBlock()
	joinBlock := l.newBlock()
	successors := []BlockID{rightBlock, shortBlock}
	shortValue := int32(0)
	if expression.Operator == "||" {
		successors = []BlockID{shortBlock, rightBlock}
		shortValue = 1
	}
	l.terminate(Terminator{
		Kind: TerminatorSwitch, Span: expression.Children[0].Span,
		Successors: successors, UnwindSuccessor: NoBlock,
		SwitchKind: "bool", SwitchOperands: []Operand{left},
	})

	destination := l.newTemporary()
	l.current = shortBlock
	l.live = cloneLive(branchLive)
	l.appendOperation(Operation{Kind: OperationAssign, Span: expression.Children[0].Span, DestinationLocal: destination, Operands: []Operand{{Kind: OperandI32, I32: shortValue, Type: hir.TypeBool}}})
	l.terminate(Terminator{Kind: TerminatorGoto, Span: expression.Span, Successors: []BlockID{joinBlock}, UnwindSuccessor: NoBlock})
	shortLive := cloneLive(l.live)

	l.current = rightBlock
	l.live = cloneLive(branchLive)
	right, err := l.lowerExpression(&expression.Children[1])
	if err != nil {
		return Operand{}, err
	}
	rightContinues := !l.blockTerminated(l.current)
	rightLive := cloneLive(l.live)
	if rightContinues {
		if right.Kind == OperandInvalid {
			return Operand{}, fmt.Errorf("short-circuit right operand has no value")
		}
		l.appendOperation(Operation{Kind: OperationAssign, Span: expression.Children[1].Span, DestinationLocal: destination, Operands: []Operand{right}})
		l.terminate(Terminator{Kind: TerminatorGoto, Span: expression.Span, Successors: []BlockID{joinBlock}, UnwindSuccessor: NoBlock})
	}

	l.current = joinBlock
	if rightContinues {
		if !sameLiveState(shortLive, rightLive) {
			return Operand{}, fmt.Errorf("short-circuit branches disagree on resource state after Flow validation")
		}
		l.live = rightLive
	} else {
		l.live = shortLive
	}
	l.result.EliminatedForms = append(l.result.EliminatedForms, EliminatedFormFact{Form: "source-short-circuit", RemainingCount: 0})
	return Operand{Kind: OperandLocal, Local: destination}, nil
}
