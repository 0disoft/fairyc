package mir

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/hir"
	"slices"
)

func (l *functionLowerer) lowerIndexedAssignment(statement hir.Statement) error {
	target := statement.Target
	operator := statement.Operator
	if operator == "" {
		operator = "="
	}
	if target == nil || statement.Expression == nil || len(target.Children) != 2 || target.Operator != "[]" ||
		target.Children[1].Type != hir.TypeUSize || statement.Expression.Type != target.Type {
		return fmt.Errorf("malformed indexed assignment")
	}
	if operator != "=" && !hir.CompoundAssignmentType(operator, target.Type) {
		return fmt.Errorf("malformed indexed compound assignment %q", operator)
	}
	receiver, err := l.lowerExpression(&target.Children[0])
	if err != nil {
		return err
	}
	if l.blockTerminated(l.current) {
		return nil
	}
	index, err := l.lowerExpression(&target.Children[1])
	if err != nil {
		return err
	}
	if l.blockTerminated(l.current) {
		return nil
	}
	if receiver.Kind == OperandInvalid || index.Kind == OperandInvalid {
		return fmt.Errorf("indexed assignment has an invalid place operand")
	}
	checkKind := OperationArrayIndexCheck
	storeKind := OperationArrayIndexStore
	switchKind := "checked-array-index"
	if target.Kind == hir.ExpressionViewIndex {
		element, indexed := hir.IndexedViewElement(target.Children[0].Type)
		if !indexed || !hir.MutableIndexedView(target.Children[0].Type) || target.Type != element {
			return fmt.Errorf("mutable byte-view assignment has incompatible type metadata")
		}
		checkKind = OperationViewIndexCheck
		storeKind = OperationViewIndexStore
		switchKind = "checked-view-index"
		receiver = l.freezeScalarOperand(receiver, target.Children[0].Span)
	} else {
		elementType, _, array := hir.FixedArrayParts(target.Children[0].Type)
		if target.Kind != hir.ExpressionArrayIndex || !array || elementType != target.Type {
			return fmt.Errorf("fixed-array assignment has incompatible type metadata")
		}
	}
	index = l.freezeScalarOperand(index, target.Children[1].Span)
	placeOperands := []Operand{receiver, index}
	l.result.OrderedEvaluations = append(l.result.OrderedEvaluations, OrderedEvaluationFact{
		Expression:               statement.Span,
		Sequence:                 []hir.Span{target.Children[0].Span, target.Children[1].Span, statement.Expression.Span},
		MaterializedBeforeParent: true,
	})
	l.appendOperation(Operation{
		Kind: checkKind, Span: target.Span, DestinationLocal: NoLocal, Operands: slices.Clone(placeOperands),
		OwnerType: target.Children[0].Type, IndexType: hir.TypeUSize, Type: target.Type, Operator: "[]",
		TrapCategories: []string{"bounds"},
	})
	continuation := l.newBlock()
	trap := l.newBlock()
	l.terminate(Terminator{
		Kind: TerminatorSwitch, Span: target.Span, Successors: []BlockID{continuation, trap}, UnwindSuccessor: NoBlock,
		SwitchKind: switchKind, SwitchOperands: slices.Clone(placeOperands), Operator: "[]",
	})
	l.current = trap
	l.terminate(Terminator{Kind: TerminatorTrap, Span: target.Span, UnwindSuccessor: NoBlock, TrapCategories: []string{"bounds"}})
	l.result.TrapPaths = append(l.result.TrapPaths, TrapPathFact{
		Operation: target.Span, Categories: []string{"bounds"}, CleanupActions: []CleanupAction{}, HasSuccessor: false,
	})
	l.current = continuation
	var prior Operand
	if operator != "=" {
		loadKind := OperationArrayIndex
		if target.Kind == hir.ExpressionViewIndex {
			loadKind = OperationViewIndex
		}
		destination := l.newTemporary()
		l.appendOperation(Operation{
			Kind: loadKind, Span: target.Span, DestinationLocal: destination, Operands: slices.Clone(placeOperands),
			OwnerType: target.Children[0].Type, IndexType: hir.TypeUSize, Type: target.Type, Operator: "[]",
		})
		prior = Operand{Kind: OperandLocal, Local: destination}
	}
	value, err := l.lowerExpression(statement.Expression)
	if err != nil {
		return err
	}
	// A terminating RHS has no stored value and no normal continuation.
	if l.blockTerminated(l.current) {
		return nil
	}
	if value.Kind == OperandInvalid {
		return fmt.Errorf("indexed assignment has an invalid value operand")
	}
	if operator != "=" {
		binary, _ := hir.CompoundAssignmentBinary(operator)
		combined := &hir.Expression{
			Kind: hir.ExpressionBinary, Type: target.Type, Span: statement.Span, Operator: binary,
			Children: []hir.Expression{{Type: target.Type, Span: target.Span}, *statement.Expression},
		}
		value, err = l.lowerBinaryOperands(combined, prior, value)
		if err != nil {
			return err
		}
	}
	l.appendOperation(Operation{
		Kind: storeKind, Span: statement.Span, DestinationLocal: NoLocal,
		Operands: append(slices.Clone(placeOperands), value), OwnerType: target.Children[0].Type,
		IndexType: hir.TypeUSize, Type: target.Type, Operator: "[]=",
	})
	return nil
}

func (l *functionLowerer) freezeScalarOperand(value Operand, at hir.Span) Operand {
	if value.Kind == OperandInvalid {
		return value
	}
	destination := l.newTemporary()
	l.appendOperation(Operation{Kind: OperationAssign, Span: at, DestinationLocal: destination, Operands: []Operand{value}})
	return Operand{Kind: OperandLocal, Local: destination}
}
