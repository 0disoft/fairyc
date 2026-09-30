package mir

import (
	"fmt"
	"github.com/0disoft/fairyc/internal/hir"
)

func ValidSharedReborrowShape(op Operation) bool {
	element, mutable, borrowed := hir.BorrowElement(op.OwnerType)
	return borrowed && mutable && op.Type == hir.SharedBorrow(element) &&
		op.DestinationLocal != NoLocal && len(op.Operands) == 1 && op.Operands[0].Kind == OperandLocal &&
		op.Operator == "share" && len(op.TrapCategories) == 0 && len(op.Fields) == 0 && op.Field == ""
}

func (l *functionLowerer) lowerSharedReborrow(expression *hir.Expression) (Operand, error) {
	if len(expression.Children) != 1 {
		return Operand{}, fmt.Errorf("malformed shared reborrow")
	}
	operand, err := l.lowerExpression(&expression.Children[0])
	if err != nil {
		return Operand{}, err
	}
	op := Operation{Kind: OperationSharedReborrow, Span: expression.Span, DestinationLocal: l.newTemporary(), Operands: []Operand{operand}, OwnerType: expression.Children[0].Type, Type: expression.Type, Operator: "share"}
	if !ValidSharedReborrowShape(op) {
		return Operand{}, fmt.Errorf("invalid shared reborrow types")
	}
	l.appendOperation(op)
	return Operand{Kind: OperandLocal, Local: op.DestinationLocal}, nil
}
