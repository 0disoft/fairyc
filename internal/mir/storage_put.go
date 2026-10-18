package mir

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/hir"
)

// OperationStoragePut transfers a typed value into uninitialized raw storage.
// The unsafe caller owns sufficient disjoint writable storage and must establish
// exactly one initialized element obligation after the write. No allocation,
// destruction, replacement of a live owner or recoverable failure is performed.
func validStoragePutShape(op Operation) bool {
	return op.DestinationLocal == NoLocal && op.Symbol == hir.NoSymbol && op.Type == hir.TypeVoid &&
		len(op.Operands) == 2 && op.Operands[0].Kind == OperandLocal && op.Operands[1].Kind == OperandLocal &&
		op.Operands[0].Local >= 0 && op.Operands[1].Local >= 0 && op.Operands[0].Local != op.Operands[1].Local &&
		len(op.Fields) == 0 && len(op.TrapCategories) == 0 && op.Operator == "" &&
		op.Destination == "" && op.ProjectionEntity == "" && op.DropFlagEntity == "" &&
		op.IndexType == hir.ParseType("") && op.Field == "" && op.Variant == "" &&
		!op.MovePayload && !op.FlagClearedFirst && op.SourceForm == "" && op.ExitKind == ""
}

func (l *functionLowerer) lowerStoragePut(expression *hir.Expression) (Operand, error) {
	if expression.Type != hir.TypeVoid || len(expression.Children) != 2 ||
		expression.Children[0].Type != hir.RawPointer(hir.TypeVoid, true) || expression.Children[1].Type != expression.TargetType ||
		!storageQueryType(expression.TargetType, l.properties) || expression.UnsafeScope == nil ||
		expression.Operator != "" || len(expression.Effects) != 0 {
		return Operand{}, fmt.Errorf("malformed private storage put")
	}
	pointer, err := l.lowerExpression(&expression.Children[0])
	if err != nil {
		return Operand{}, err
	}
	if pointer.Kind != OperandLocal {
		return Operand{}, fmt.Errorf("private storage put requires a materialized pointer")
	}
	// Freeze the destination before evaluating the value, including any calls.
	address := l.newTemporary()
	l.appendOperation(Operation{Kind: OperationAssign, Span: expression.Children[0].Span,
		DestinationLocal: address, Operands: []Operand{pointer}, Type: expression.Children[0].Type})
	value, err := l.lowerExpression(&expression.Children[1])
	if err != nil {
		return Operand{}, err
	}
	if path := l.liveResourcePath(&expression.Children[1]); path != nil {
		value = l.materializeMove(path, value, "storage-put", true)
	} else {
		materialized := l.newTemporary()
		l.appendOperation(Operation{Kind: OperationAssign, Span: expression.Children[1].Span,
			DestinationLocal: materialized, Operands: []Operand{value}, Type: expression.TargetType})
		value = Operand{Kind: OperandLocal, Local: materialized}
	}
	entity := sourceEntity("storage-put", expression.Span)
	l.appendOperation(Operation{Kind: OperationStoragePut, Entity: entity, Span: expression.Span,
		Symbol: hir.NoSymbol, DestinationLocal: NoLocal, Type: hir.TypeVoid, OwnerType: expression.TargetType,
		Operands: []Operand{{Kind: OperandLocal, Local: address}, value}, UnsafeScope: *expression.UnsafeScope})
	l.result.UnsafeOperations = append(l.result.UnsafeOperations, UnsafeOperation{Entity: entity,
		SourceSpanEntity: sourceEntity("span", expression.Span), ScopeEntity: sourceEntity("unsafe-scope", *expression.UnsafeScope)})
	l.result.OrderedEvaluations = append(l.result.OrderedEvaluations, OrderedEvaluationFact{
		Expression: expression.Span, Sequence: []hir.Span{expression.Children[0].Span, expression.Children[1].Span}, MaterializedBeforeParent: true})
	return Operand{}, nil
}
