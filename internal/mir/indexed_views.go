package mir

import "github.com/0disoft/fairyc/internal/hir"

// Match actual local definitions, not operand annotations, before interpreting
// an integer descriptor. In particular it cannot be relabeled as a byte view.
func verifyIndexedViewOperands(op Operation, types map[LocalID]hir.Type) string {
	if len(op.Operands) == 0 {
		return "" // Shape verification reports missing operands.
	}
	actual := fieldStoreOperandType(op.Operands[0], types)
	viewElement, _ := hir.IndexedViewElement(actual)
	switch op.Kind {
	case OperationArrayBorrowView:
		element, _, array := hir.FixedArrayParts(actual)
		if hir.NonByteIndexedView(op.Type) || array && element != hir.TypeU8 {
			if actual != op.OwnerType || !hir.ArrayViewBorrow(actual, op.Type, op.Operator) {
				return "integer array-view borrow disagrees with its owner definition"
			}
		}
	case OperationViewLength, OperationViewEmpty, OperationViewIndex, OperationViewIndexCheck, OperationViewSlice, OperationViewSliceCheck, OperationViewIndexStore:
		if hir.NonByteIndexedView(actual) || hir.NonByteIndexedView(op.OwnerType) || hir.NonByteIndexedView(op.Type) {
			if !hir.NonByteIndexedView(actual) || actual != op.OwnerType {
				return "integer view operation disagrees with its receiver definition"
			}
			bounds := op.Operands[1:]
			if op.Kind == OperationViewIndexStore && len(op.Operands) == 3 {
				if fieldStoreOperandType(op.Operands[2], types) != viewElement {
					return "integer view store value is not an actual matching element value"
				}
				bounds = op.Operands[1:2]
			}
			for _, bound := range bounds {
				if fieldStoreOperandType(bound, types) != hir.TypeUSize {
					return "integer view bound is not an actual usize value"
				}
			}
		}
	}
	return ""
}
