package mir

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/hir"
)

// OperationStorageLayout is a compiler-private, target-storage query. It has
// no value operand or ownership effect; OwnerType is resolved, but its numeric
// representation is selected only by a backend. Ordinary Verify rejects it.
const OperationStorageLayout OperationKind = "storage-layout"

func storageQueryType(value hir.Type, properties typeProperties) bool {
	switch value {
	case hir.TypeBool, hir.TypeU8, hir.TypeI32, hir.TypeU32, hir.TypeUSize:
		return true
	default:
		return properties.IsStruct(value) || properties.IsResource(value) || properties.Enum(value) != nil
	}
}

func validStorageQueryShape(op Operation) bool {
	return op.Type == hir.TypeUSize && op.DestinationLocal >= 0 && op.Symbol == hir.NoSymbol &&
		(op.Field == "size" || op.Field == "alignment") && len(op.Operands) == 0 &&
		len(op.Fields) == 0 && len(op.TrapCategories) == 0 && op.Operator == "" &&
		op.Destination == "" && op.ProjectionEntity == "" && op.DropFlagEntity == "" &&
		op.IndexType == "" && op.Variant == "" && !op.MovePayload && !op.FlagClearedFirst &&
		op.UnsafeScope == (hir.Span{}) && op.SourceForm == "" && op.ExitKind == ""
}

func (l *functionLowerer) lowerStorageLayout(expression *hir.Expression) (Operand, error) {
	if expression.Type != hir.TypeUSize || len(expression.Children) != 0 || len(expression.Effects) != 0 ||
		expression.UnsafeScope != nil || !storageQueryType(expression.TargetType, l.properties) ||
		(expression.Operator != "size" && expression.Operator != "alignment") {
		return Operand{}, fmt.Errorf("malformed private storage layout query")
	}
	destination := l.newTemporary()
	l.appendOperation(Operation{Kind: OperationStorageLayout, Symbol: hir.NoSymbol,
		Span: expression.Span, DestinationLocal: destination, Type: hir.TypeUSize,
		OwnerType: expression.TargetType, Field: expression.Operator})
	return Operand{Kind: OperandLocal, Local: destination}, nil
}
