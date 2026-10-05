package mir

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/hir"
)

func (l *functionLowerer) lowerFieldBorrow(expression *hir.Expression) (Operand, error) {
	_, _, borrowed := hir.BorrowElement(expression.Type)
	if !borrowed || len(expression.Children) != 1 || expression.Operator == "" {
		return Operand{}, fmt.Errorf("malformed field borrow")
	}
	owner, err := l.lowerExpression(&expression.Children[0])
	if err != nil {
		return Operand{}, err
	}
	if l.blockTerminated(l.current) {
		return Operand{}, nil
	}
	if _, _, borrowed := hir.BorrowElement(expression.Children[0].Type); !borrowed || owner.Kind != OperandLocal {
		return Operand{}, fmt.Errorf("field borrow requires a materialized owner reference")
	}
	destination := l.newTemporary()
	l.appendOperation(Operation{Kind: OperationFieldBorrow, Symbol: hir.NoSymbol, Span: expression.Span,
		DestinationLocal: destination, Operands: []Operand{owner},
		OwnerType: expression.Children[0].Type, Type: expression.Type, Field: expression.Operator})
	return Operand{Kind: OperandLocal, Local: destination}, nil
}

func validFieldBorrowShape(op Operation) bool {
	_, mutable, borrowed := hir.BorrowElement(op.Type)
	_, ownerMutable, ownerBorrowed := hir.BorrowElement(op.OwnerType)
	return borrowed && ownerBorrowed && mutable == ownerMutable && op.Field != "" &&
		op.Symbol == hir.NoSymbol && op.DestinationLocal >= 0 && len(op.Operands) == 1 &&
		op.Operands[0].Kind == OperandLocal && op.Operands[0].Local >= 0 && op.Operands[0].Local != op.DestinationLocal &&
		len(op.Fields) == 0 && len(op.TrapCategories) == 0 && op.Operator == "" &&
		op.Destination == "" && op.ProjectionEntity == "" && op.DropFlagEntity == "" &&
		op.IndexType == "" && op.Variant == "" && !op.MovePayload && !op.FlagClearedFirst &&
		op.UnsafeScope == (hir.Span{}) && op.SourceForm == "" && op.ExitKind == ""
}

// Mutable projection preserves only plain, non-owning field addresses.
func validMutableFieldBorrow(op Operation, lookup func(hir.Type) *StructType) bool {
	element, mutable, _ := hir.BorrowElement(op.Type)
	if _, _, array := hir.FixedArrayParts(element); array && !hir.ScalarArrayField(element) {
		return false
	}
	return !mutable || plainFieldStoreStruct(fieldAggregateType(op.OwnerType), lookup) &&
		(hir.PlainViewField(element) || plainFieldStoreStruct(element, lookup))
}
