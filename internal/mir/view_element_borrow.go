package mir

import "github.com/0disoft/fairyc/internal/hir"

// References preserve the exact integer or nominal type. Shared references can shorten a
// mutable view; shared views can never produce exclusive element references.
func ValidViewElementBorrowShape(op Operation) bool {
	element, indexed := hir.IndexedViewElement(op.OwnerType)
	referent, mutable, borrowed := hir.BorrowElement(op.Type)
	return indexed && borrowed && referent == element && (!mutable || hir.MutableIndexedView(op.OwnerType)) &&
		op.DestinationLocal >= 0 && op.Symbol == hir.NoSymbol && len(op.Operands) == 2 &&
		op.Operands[0].Kind == OperandLocal && op.Operands[0].Local >= 0 && op.Operands[0].Local != op.DestinationLocal &&
		(op.Operands[1].Kind == OperandUSize || op.Operands[1].Kind == OperandLocal) &&
		op.IndexType == hir.TypeUSize && op.Operator == "[]" && op.Field == "" && len(op.Fields) == 0 &&
		len(op.TrapCategories) == 0 && op.Destination == "" && op.ProjectionEntity == "" && op.DropFlagEntity == "" &&
		op.Variant == "" && !op.MovePayload && !op.FlagClearedFirst && op.UnsafeScope == (hir.Span{}) && op.SourceForm == "" && op.ExitKind == ""
}
