package mir

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/hir"
)

// SDK-only shared/exclusive slice storage borrows carry an element count.
// Its unsafe caller proves alignment, initialized extent and owner provenance.
func validStorageViewShape(op Operation, properties typeProperties) bool {
	mutable := op.Kind == OperationStorageBorrowMut
	elementType, view := hir.IndexedViewElement(op.Type)
	if _, nominal := hir.StructViewElement(op.Type); nominal && !properties.plainViewStruct(elementType) {
		return false
	}
	_, resultMutable, _ := hir.BorrowElement(op.Type)
	if (op.Kind != OperationStorageBorrow && !mutable) || !view || resultMutable != mutable || len(op.Operands) != 3 ||
		op.Operands[2].Kind != OperandLocal && op.Operands[2].Kind != OperandUSize {
		return false
	}
	if op.Operands[2].Kind == OperandLocal && (op.Operands[2].Local < 0 || op.Operands[2].Local == op.DestinationLocal) {
		return false
	}
	element := op
	element.Type = hir.SharedBorrow(elementType)
	if mutable {
		element.Type = hir.MutableBorrow(elementType)
	}
	element.Operands = op.Operands[:2]
	return validStorageBorrowShape(element, properties)
}

func (l *functionLowerer) lowerStorageView(expression *hir.Expression) (Operand, error) {
	kind := OperationStorageBorrow
	if expression.Kind == hir.ExpressionStorageBorrowMut {
		kind = OperationStorageBorrowMut
	}
	_, view := hir.IndexedViewElement(expression.Type)
	target, mutable, _ := hir.BorrowElement(expression.Type)
	if expression.Kind != hir.ExpressionStorageBorrow && expression.Kind != hir.ExpressionStorageBorrowMut || !view || target != expression.TargetType || mutable != (kind == OperationStorageBorrowMut) ||
		len(expression.Children) != 3 || expression.Children[2].Type != hir.TypeUSize ||
		expression.UnsafeScope == nil || expression.Operator != "" || len(expression.Effects) != 0 {
		return Operand{}, fmt.Errorf("malformed private storage view")
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
			return Operand{}, fmt.Errorf("private storage view argument has no value")
		}
		// Materialize each argument before evaluating the next, including calls.
		retained := l.newTemporary()
		l.appendOperation(Operation{Kind: OperationAssign, Span: child.Span,
			DestinationLocal: retained, Type: child.Type, Operands: []Operand{operand}})
		operands = append(operands, Operand{Kind: OperandLocal, Local: retained})
		sequence = append(sequence, child.Span)
	}
	op := Operation{Kind: kind, Entity: sourceEntity(string(kind), expression.Span),
		Span: expression.Span, Symbol: hir.NoSymbol, DestinationLocal: l.newTemporary(),
		Type: expression.Type, OwnerType: expression.Children[0].Type, IndexType: expression.Children[1].Type,
		Operands: operands, UnsafeScope: *expression.UnsafeScope}
	if !validStorageViewShape(op, l.properties) {
		return Operand{}, fmt.Errorf("malformed private storage view operands")
	}
	l.appendOperation(op)
	l.result.UnsafeOperations = append(l.result.UnsafeOperations, UnsafeOperation{Entity: op.Entity,
		SourceSpanEntity: sourceEntity("span", op.Span), ScopeEntity: sourceEntity("unsafe-scope", op.UnsafeScope)})
	l.result.OrderedEvaluations = append(l.result.OrderedEvaluations, OrderedEvaluationFact{
		Expression: expression.Span, Sequence: sequence, MaterializedBeforeParent: true})
	return Operand{Kind: OperandLocal, Local: op.DestinationLocal}, nil
}

func isIndexedStorageView(typ hir.Type) bool {
	_, view := hir.IndexedViewElement(typ)
	return view
}

// A private descriptor cannot be relabeled between element types at return.
// The opaque address itself has no element type; the definition and callable
// result must agree rather than trusting the return operand's annotation.
func verifyStorageViewReturn(function *Function, value Operand, types map[LocalID]hir.Type) string {
	for _, block := range function.Blocks {
		for _, op := range block.Operations {
			if (op.Kind == OperationStorageBorrow || op.Kind == OperationStorageBorrowMut) && isIndexedStorageView(op.Type) {
				if isIndexedStorageView(function.ResultType) && fieldStoreOperandType(value, types) != function.ResultType {
					return "private storage view return disagrees with the callable result type"
				}
			}
		}
	}
	return ""
}
