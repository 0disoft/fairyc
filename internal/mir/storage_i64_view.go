package mir

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/hir"
)

// The SDK-only shared slice form of storage-borrow carries an element count.
// Its unsafe caller proves alignment, initialized extent and owner provenance.
func validI64StorageViewShape(op Operation, properties typeProperties) bool {
	if op.Kind != OperationStorageBorrow || op.Type != hir.SharedBorrow(hir.TypeI64Slice) || len(op.Operands) != 3 ||
		op.Operands[2].Kind != OperandLocal && op.Operands[2].Kind != OperandUSize {
		return false
	}
	if op.Operands[2].Kind == OperandLocal && (op.Operands[2].Local < 0 || op.Operands[2].Local == op.DestinationLocal) {
		return false
	}
	element := op
	element.Type = hir.SharedBorrow(hir.TypeI64)
	element.Operands = op.Operands[:2]
	return validStorageBorrowShape(element, properties)
}

func (l *functionLowerer) lowerI64StorageView(expression *hir.Expression) (Operand, error) {
	if expression.Kind != string(OperationStorageBorrow) || expression.Type != hir.SharedBorrow(hir.TypeI64Slice) ||
		len(expression.Children) != 3 || expression.Children[2].Type != hir.TypeUSize ||
		expression.UnsafeScope == nil || expression.Operator != "" || len(expression.Effects) != 0 {
		return Operand{}, fmt.Errorf("malformed private i64 storage view")
	}
	operands := make([]Operand, 0, 3)
	sequence := make([]hir.Span, 0, 3)
	for index := range expression.Children {
		child := &expression.Children[index]
		operand, err := l.lowerExpression(child)
		if err != nil {
			return Operand{}, err
		}
		if operand.Kind == OperandInvalid {
			return Operand{}, fmt.Errorf("private i64 storage view argument has no value")
		}
		// Materialize each argument before evaluating the next, including calls.
		retained := l.newTemporary()
		l.appendOperation(Operation{Kind: OperationAssign, Span: child.Span,
			DestinationLocal: retained, Type: child.Type, Operands: []Operand{operand}})
		operands = append(operands, Operand{Kind: OperandLocal, Local: retained})
		sequence = append(sequence, child.Span)
	}
	op := Operation{Kind: OperationStorageBorrow, Entity: sourceEntity("storage-borrow", expression.Span),
		Span: expression.Span, Symbol: hir.NoSymbol, DestinationLocal: l.newTemporary(),
		Type: expression.Type, OwnerType: expression.Children[0].Type, IndexType: expression.Children[1].Type,
		Operands: operands, UnsafeScope: *expression.UnsafeScope}
	if !validI64StorageViewShape(op, l.properties) {
		return Operand{}, fmt.Errorf("malformed private i64 storage view operands")
	}
	l.appendOperation(op)
	l.result.UnsafeOperations = append(l.result.UnsafeOperations, UnsafeOperation{Entity: op.Entity,
		SourceSpanEntity: sourceEntity("span", op.Span), ScopeEntity: sourceEntity("unsafe-scope", op.UnsafeScope)})
	l.result.OrderedEvaluations = append(l.result.OrderedEvaluations, OrderedEvaluationFact{
		Expression: expression.Span, Sequence: sequence, MaterializedBeforeParent: true})
	return Operand{Kind: OperandLocal, Local: op.DestinationLocal}, nil
}
