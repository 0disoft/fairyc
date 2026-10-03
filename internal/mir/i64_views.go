package mir

import "github.com/0disoft/fairyc/internal/hir"

// Match actual local definitions, not operand annotations, before interpreting
// an i64 descriptor. In particular it cannot be relabeled as a byte view.
func verifyI64ViewOperands(op Operation, types map[LocalID]hir.Type) string {
	if len(op.Operands) == 0 {
		return "" // Shape verification reports missing operands.
	}
	actual := fieldStoreOperandType(op.Operands[0], types)
	switch op.Kind {
	case OperationArrayBorrowView:
		element, _, array := hir.FixedArrayParts(actual)
		if hir.I64View(op.Type) || array && element == hir.TypeI64 {
			if actual != op.OwnerType || !hir.ArrayViewBorrow(actual, op.Type, op.Operator) {
				return "i64 array-view borrow disagrees with its owner definition"
			}
		}
	case OperationViewLength, OperationViewEmpty, OperationViewIndex, OperationViewIndexCheck, OperationViewSlice, OperationViewSliceCheck, OperationViewIndexStore:
		if hir.I64View(actual) || hir.I64View(op.OwnerType) || hir.I64View(op.Type) {
			if !hir.I64View(actual) || actual != op.OwnerType {
				return "i64 view operation disagrees with its receiver definition"
			}
			bounds := op.Operands[1:]
			if op.Kind == OperationViewIndexStore && len(op.Operands) == 3 {
				if fieldStoreOperandType(op.Operands[2], types) != hir.TypeI64 {
					return "i64 view store value is not an actual i64 value"
				}
				bounds = op.Operands[1:2]
			}
			for _, bound := range bounds {
				if fieldStoreOperandType(bound, types) != hir.TypeUSize {
					return "i64 view bound is not an actual usize value"
				}
			}
		}
	}
	return ""
}
