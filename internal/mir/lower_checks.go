package mir

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/hir"
	"slices"
)

func (l *functionLowerer) lowerAssert(expression *hir.Expression, statementSpan hir.Span) error {
	if expression == nil || expression.Type != hir.TypeBool {
		return fmt.Errorf("malformed assert statement")
	}
	if expression.Kind == hir.ExpressionLiteral && expression.Value == "false" {
		l.terminate(Terminator{Kind: TerminatorTrap, Span: statementSpan, UnwindSuccessor: NoBlock, TrapCategories: []string{"assertion-failed"}})
		l.result.TrapPaths = append(l.result.TrapPaths, TrapPathFact{
			Operation: statementSpan, Categories: []string{"assertion-failed"}, CleanupActions: []CleanupAction{}, HasSuccessor: false,
		})
		return nil
	}
	condition, err := l.lowerExpression(expression)
	if err != nil {
		return err
	}
	if condition.Kind == OperandInvalid {
		return fmt.Errorf("assert statement has no condition value")
	}
	condition.Type = hir.TypeBool
	continuation := l.newBlock()
	trap := l.newBlock()
	l.terminate(Terminator{
		Kind: TerminatorSwitch, Span: statementSpan, Successors: []BlockID{continuation, trap}, UnwindSuccessor: NoBlock,
		SwitchKind: "assert", SwitchType: hir.TypeBool, SwitchOperands: []Operand{condition}, Operator: "assert",
	})
	l.current = trap
	l.terminate(Terminator{Kind: TerminatorTrap, Span: statementSpan, UnwindSuccessor: NoBlock, TrapCategories: []string{"assertion-failed"}})
	l.result.TrapPaths = append(l.result.TrapPaths, TrapPathFact{
		Operation: statementSpan, Categories: []string{"assertion-failed"}, CleanupActions: []CleanupAction{}, HasSuccessor: false,
	})
	l.current = continuation
	return nil
}

func (l *functionLowerer) lowerCheckedDivision(expression *hir.Expression, left, right Operand) error {
	categories := []string{"division-by-zero"}
	if expression.Type == hir.TypeI32 || expression.Type == hir.TypeI64 {
		categories = append(categories, "signed-division-overflow")
	}
	l.appendOperation(Operation{
		Kind: OperationCheckedDivision, Span: expression.Span, Operands: []Operand{left, right},
		OwnerType: expression.Children[0].Type, Type: expression.Type,
		Operator: expression.Operator, TrapCategories: slices.Clone(categories),
	})
	continuation := l.newBlock()
	trap := l.newBlock()
	l.terminate(Terminator{
		Kind: TerminatorSwitch, Span: expression.Span, Successors: []BlockID{continuation, trap}, UnwindSuccessor: NoBlock,
		SwitchKind: "checked-division", SwitchType: expression.Children[0].Type,
		SwitchOperands: []Operand{left, right}, Operator: expression.Operator,
	})
	l.current = trap
	l.terminate(Terminator{Kind: TerminatorTrap, Span: expression.Span, UnwindSuccessor: NoBlock, TrapCategories: slices.Clone(categories)})
	l.result.TrapPaths = append(l.result.TrapPaths, TrapPathFact{
		Operation: expression.Span, Categories: slices.Clone(categories), CleanupActions: []CleanupAction{}, HasSuccessor: false,
	})
	l.current = continuation
	return nil
}

func (l *functionLowerer) lowerCheckedArithmetic(expression *hir.Expression, operands []Operand, switchKind string, categories []string) {
	l.appendOperation(Operation{
		Kind: OperationArithmeticCheck, Span: expression.Span, Operands: slices.Clone(operands),
		OwnerType: expression.Children[0].Type, Type: expression.Type,
		Operator: expression.Operator, TrapCategories: slices.Clone(categories),
	})
	continuation := l.newBlock()
	trap := l.newBlock()
	l.terminate(Terminator{
		Kind: TerminatorSwitch, Span: expression.Span, Successors: []BlockID{continuation, trap}, UnwindSuccessor: NoBlock,
		SwitchKind: switchKind, SwitchType: expression.Children[0].Type,
		SwitchOperands: slices.Clone(operands), Operator: expression.Operator,
	})
	l.current = trap
	l.terminate(Terminator{Kind: TerminatorTrap, Span: expression.Span, UnwindSuccessor: NoBlock, TrapCategories: slices.Clone(categories)})
	l.result.TrapPaths = append(l.result.TrapPaths, TrapPathFact{
		Operation: expression.Span, Categories: slices.Clone(categories), CleanupActions: []CleanupAction{}, HasSuccessor: false,
	})
	l.current = continuation
}
