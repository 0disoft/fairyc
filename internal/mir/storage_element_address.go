package mir

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/hir"
)

// OperationStorageElementAddress calculates an SDK-private element address.
// The unsafe caller must prove that the index names an initialized element in
// the live allocation. A null base or size overflow yields a null pointer.
func validStorageElementAddressShape(op Operation, properties typeProperties) bool {
	return op.Kind == OperationStorageElementAddress &&
		op.Type == hir.RawPointer(hir.TypeVoid, true) &&
		storageQueryType(op.OwnerType, properties) && op.IndexType == hir.TypeUSize &&
		op.DestinationLocal >= 0 && op.Symbol == hir.NoSymbol &&
		len(op.Operands) == 2 && op.Operands[0].Kind == OperandLocal && op.Operands[1].Kind == OperandLocal &&
		op.Operands[0].Local >= 0 && op.Operands[1].Local >= 0 &&
		op.Operands[0].Local != op.Operands[1].Local &&
		op.DestinationLocal != op.Operands[0].Local && op.DestinationLocal != op.Operands[1].Local &&
		len(op.Fields) == 0 && len(op.TrapCategories) == 0 && op.Operator == "" && op.Field == "" &&
		op.Destination == "" && op.ProjectionEntity == "" && op.DropFlagEntity == "" &&
		op.Variant == "" && !op.MovePayload && !op.FlagClearedFirst &&
		op.SourceForm == "" && op.ExitKind == ""
}

func (l *functionLowerer) lowerStorageElementAddress(expression *hir.Expression) (Operand, error) {
	if expression.Type != hir.RawPointer(hir.TypeVoid, true) || len(expression.Children) != 2 ||
		expression.Children[0].Type != hir.RawPointer(hir.TypeVoid, true) || expression.Children[1].Type != hir.TypeUSize ||
		!storageQueryType(expression.TargetType, l.properties) || expression.UnsafeScope == nil ||
		expression.Operator != "" || len(expression.Effects) != 0 {
		return Operand{}, fmt.Errorf("malformed private storage element address")
	}
	base, err := l.lowerExpression(&expression.Children[0])
	if err != nil {
		return Operand{}, err
	}
	if base.Kind != OperandLocal {
		return Operand{}, fmt.Errorf("private storage element address requires a materialized base")
	}
	retainedBase := l.newTemporary()
	l.appendOperation(Operation{Kind: OperationAssign, Span: expression.Children[0].Span,
		DestinationLocal: retainedBase, Type: expression.Children[0].Type, Operands: []Operand{base}})
	index, err := l.lowerExpression(&expression.Children[1])
	if err != nil {
		return Operand{}, err
	}
	if index.Kind == OperandInvalid {
		return Operand{}, fmt.Errorf("private storage element address requires an index value")
	}
	retainedIndex := l.newTemporary()
	l.appendOperation(Operation{Kind: OperationAssign, Span: expression.Children[1].Span,
		DestinationLocal: retainedIndex, Type: hir.TypeUSize, Operands: []Operand{index}})
	entity := sourceEntity(string(OperationStorageElementAddress), expression.Span)
	op := Operation{Kind: OperationStorageElementAddress, Entity: entity, Span: expression.Span,
		Symbol: hir.NoSymbol, DestinationLocal: l.newTemporary(), Type: expression.Type,
		OwnerType: expression.TargetType, IndexType: hir.TypeUSize,
		Operands: []Operand{{Kind: OperandLocal, Local: retainedBase}, {Kind: OperandLocal, Local: retainedIndex}}, UnsafeScope: *expression.UnsafeScope}
	if !validStorageElementAddressShape(op, l.properties) {
		return Operand{}, fmt.Errorf("malformed private storage element address operands")
	}
	l.appendOperation(op)
	l.result.UnsafeOperations = append(l.result.UnsafeOperations, UnsafeOperation{Entity: entity,
		SourceSpanEntity: sourceEntity("span", expression.Span), ScopeEntity: sourceEntity("unsafe-scope", *expression.UnsafeScope)})
	l.result.OrderedEvaluations = append(l.result.OrderedEvaluations, OrderedEvaluationFact{
		Expression: expression.Span, Sequence: []hir.Span{expression.Children[0].Span, expression.Children[1].Span}, MaterializedBeforeParent: true})
	return Operand{Kind: OperandLocal, Local: op.DestinationLocal}, nil
}
