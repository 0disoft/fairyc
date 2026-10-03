package mir

import "github.com/0disoft/fairyc/internal/hir"

// Match actual local definitions, not operand annotations, before interpreting
// an i64 descriptor. In particular it cannot be relabeled as a byte view.
func verifyI64ViewOperands(op Operation, types map[LocalID]hir.Type) string {
	if len(op.Operands) == 0 {
		return "" // Shape verification reports missing operands.
	}
	actual := fieldStoreOperandType(op.Operands[0], types)
	i64View := hir.SharedBorrow(hir.TypeI64Slice)
	switch op.Kind {
	case OperationArrayBorrowView:
		element, _, array := hir.FixedArrayParts(actual)
		if op.Type == i64View || array && element == hir.TypeI64 {
			if actual != op.OwnerType || !hir.ArrayViewBorrow(actual, op.Type, op.Operator) {
				return "i64 array-view borrow disagrees with its owner definition"
			}
		}
	case OperationViewLength, OperationViewEmpty, OperationViewIndex, OperationViewIndexCheck, OperationViewSlice, OperationViewSliceCheck:
		if actual == i64View || op.OwnerType == i64View || op.Type == i64View {
			if actual != i64View || op.OwnerType != i64View {
				return "i64 view operation disagrees with its receiver definition"
			}
			for _, bound := range op.Operands[1:] {
				if fieldStoreOperandType(bound, types) != hir.TypeUSize {
					return "i64 view bound is not an actual usize value"
				}
			}
		}
	}
	return ""
}
