package mir

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/hir"
)

// OperationStorageTake moves one initialized representation out of raw storage.
// The unsafe caller owns that element, permits relocation, and retires its old
// lifetime without dropping it. The operation neither clears nor frees storage.
// It is not a safe pointer read or a proof of the caller's ownership obligations.
const OperationStorageTake OperationKind = "storage-take"

func validStorageTakeShape(op Operation) bool {
	return op.DestinationLocal >= 0 && op.Symbol == hir.NoSymbol &&
		op.OwnerType == hir.RawPointer(hir.TypeVoid, true) &&
		len(op.Operands) == 1 && op.Operands[0].Kind == OperandLocal &&
		op.Operands[0].Local >= 0 && op.Operands[0].Local != op.DestinationLocal &&
		len(op.Fields) == 0 && len(op.TrapCategories) == 0 && op.Operator == "" &&
		op.Destination == "" && op.ProjectionEntity == "" && op.DropFlagEntity == "" &&
		op.IndexType == hir.ParseType("") && op.Field == "" && op.Variant == "" &&
		!op.MovePayload && !op.FlagClearedFirst && op.SourceForm == "" && op.ExitKind == ""
}

func validStorageTransferProvenance(function *Function, op Operation) bool {
	if op.Entity != sourceEntity(string(op.Kind), op.Span) || op.UnsafeScope.Source == "" ||
		!sameSpanSource(op.UnsafeScope, op.Span) || op.UnsafeScope.StartByte > op.Span.StartByte ||
		op.UnsafeScope.EndByte < op.Span.EndByte {
		return false
	}
	count := 0
	for _, entry := range function.UnsafeOperations {
		if entry.Entity == op.Entity && entry.SourceSpanEntity == sourceEntity("span", op.Span) &&
			entry.ScopeEntity == sourceEntity("unsafe-scope", op.UnsafeScope) {
			count++
		}
	}
	return count == 1
}

func (l *functionLowerer) lowerStorageTake(expression *hir.Expression) (Operand, error) {
	if len(expression.Children) != 1 || expression.Children[0].Type != hir.RawPointer(hir.TypeVoid, true) ||
		!storageQueryType(expression.Type, l.properties) || expression.TargetType != expression.Type ||
		expression.UnsafeScope == nil || expression.Operator != "" || len(expression.Effects) != 0 {
		return Operand{}, fmt.Errorf("malformed private storage take")
	}
	pointer, err := l.lowerExpression(&expression.Children[0])
	if err != nil {
		return Operand{}, err
	}
	if pointer.Kind != OperandLocal {
		return Operand{}, fmt.Errorf("private storage take requires a materialized pointer")
	}
	destination := l.newTemporary()
	entity := sourceEntity("storage-take", expression.Span)
	l.appendOperation(Operation{Kind: OperationStorageTake, Entity: entity, Span: expression.Span,
		Symbol: hir.NoSymbol, DestinationLocal: destination, Operands: []Operand{pointer},
		Type: expression.Type, OwnerType: expression.Children[0].Type, UnsafeScope: *expression.UnsafeScope})
	l.result.UnsafeOperations = append(l.result.UnsafeOperations, UnsafeOperation{Entity: entity,
		SourceSpanEntity: sourceEntity("span", expression.Span), ScopeEntity: sourceEntity("unsafe-scope", *expression.UnsafeScope)})
	return Operand{Kind: OperandLocal, Local: destination}, nil
}
