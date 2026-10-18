package mir

import "github.com/0disoft/fairyc/internal/hir"

func validRegionExitKind(value string) bool {
	return value == "fallthrough" || value == "return" || value == "propagate" || value == "break" || value == "continue"
}

func validSwitchShape(terminator Terminator) bool {
	switch terminator.SwitchKind {
	case "":
		return len(terminator.SwitchOperands) == 0 && len(terminator.Successors) == 2
	case "bool":
		return len(terminator.SwitchOperands) == 1 && len(terminator.Successors) == 2
	case "assert":
		return terminator.SwitchType == hir.TypeBool && terminator.Operator == "assert" &&
			len(terminator.SwitchOperands) == 1 && validAssertOperand(terminator.SwitchOperands[0]) && len(terminator.Successors) == 2
	case "result":
		return len(terminator.SwitchOperands) == 1 && len(terminator.Successors) == 2
	case "option":
		return len(terminator.SwitchOperands) == 1 && len(terminator.Successors) == 2
	case "enum":
		return terminator.SwitchType != hir.ParseType("") && terminator.SwitchType != hir.TypeInvalid && len(terminator.SwitchOperands) == 1 && len(terminator.Successors) > 0 && len(terminator.Successors) == len(terminator.SwitchVariants) && uniqueNonEmpty(terminator.SwitchVariants)
	case "checked-division":
		return len(terminator.SwitchOperands) == 2 && len(terminator.Successors) == 2 &&
			(terminator.Operator == "/" || terminator.Operator == "%") && hir.CheckedDivisionType(terminator.SwitchType)
	case "checked-view-index":
		return len(terminator.SwitchOperands) == 2 && len(terminator.Successors) == 2 && terminator.Operator == "[]"
	case "checked-array-index":
		return len(terminator.SwitchOperands) == 2 && len(terminator.Successors) == 2 && terminator.Operator == "[]"
	case "checked-view-slice":
		return len(terminator.SwitchOperands) == 3 && len(terminator.Successors) == 2 && terminator.Operator == ".."
	case "checked-unary", "checked-arithmetic", "checked-shift":
		switchKind, _, valid := checkedArithmeticSpec(terminator.Operator, len(terminator.SwitchOperands))
		if !valid || switchKind != terminator.SwitchKind || len(terminator.Successors) != 2 {
			return false
		}
		if terminator.SwitchKind == "checked-arithmetic" {
			return hir.CheckedArithmeticType(terminator.SwitchType)
		}
		if terminator.SwitchKind == "checked-shift" {
			return hir.CheckedShiftType(terminator.SwitchType)
		}
		return signedIntegerType(terminator.SwitchType)
	default:
		return false
	}
}

func validAssertOperand(operand Operand) bool {
	return operand.Type == hir.TypeBool && (operand.Kind == OperandLocal || operand.Kind == OperandI32 && (operand.I32 == 0 || operand.I32 == 1))
}

func checkedArithmeticSpec(operator string, operandCount int) (string, []string, bool) {
	switch {
	case operator == "-" && operandCount == 1:
		return "checked-unary", []string{"integer-overflow"}, true
	case (operator == "+" || operator == "-" || operator == "*") && operandCount == 2:
		return "checked-arithmetic", []string{"integer-overflow"}, true
	case operator == ">>" && operandCount == 2:
		return "checked-shift", []string{"invalid-shift"}, true
	case operator == "<<" && operandCount == 2:
		return "checked-shift", []string{"invalid-shift", "integer-overflow"}, true
	default:
		return "", nil, false
	}
}

func validCheckedArithmeticShape(operation Operation) bool {
	if len(operation.Operands) == 0 {
		return false
	}
	if operation.Operator == "+" || operation.Operator == "-" || operation.Operator == "*" {
		if len(operation.Operands) == 2 {
			return hir.CheckedArithmeticType(operation.OwnerType) && operation.Type == operation.OwnerType &&
				scalarOperandMatches(operation.Operands[0], operation.OwnerType) &&
				scalarOperandMatches(operation.Operands[1], operation.OwnerType)
		}
	}
	if len(operation.Operands) == 1 && operation.Operator == "-" {
		if !signedIntegerType(operation.OwnerType) || operation.Type != operation.OwnerType {
			return false
		}
		return scalarOperandMatches(operation.Operands[0], operation.OwnerType)
	}
	if len(operation.Operands) == 2 && (operation.Operator == ">>" || operation.Operator == "<<") {
		return hir.CheckedShiftType(operation.OwnerType) && operation.Type == operation.OwnerType &&
			scalarOperandMatches(operation.Operands[0], operation.OwnerType) &&
			scalarOperandMatches(operation.Operands[1], operation.OwnerType)
	}
	return false
}

func validCheckedDivisionShape(operation Operation) bool {
	return len(operation.Operands) == 2 && (operation.Operator == "/" || operation.Operator == "%") &&
		hir.CheckedDivisionType(operation.OwnerType) && operation.Type == operation.OwnerType &&
		scalarOperandMatches(operation.Operands[0], operation.OwnerType) &&
		scalarOperandMatches(operation.Operands[1], operation.OwnerType)
}

func divisionTrapCategories(ownerType hir.Type) ([]string, bool) {
	if !hir.CheckedDivisionType(ownerType) {
		return nil, false
	}
	categories := []string{"division-by-zero"}
	if signedIntegerType(ownerType) {
		categories = append(categories, "signed-division-overflow")
	}
	return categories, true
}

func validViewIndexShape(operation Operation, destination bool) bool {
	element, indexed := hir.IndexedViewElement(operation.OwnerType)
	if len(operation.Operands) != 2 || !indexed ||
		operation.IndexType != hir.TypeUSize || operation.Type != element || operation.Operator != "[]" {
		return false
	}
	if destination != (operation.DestinationLocal != NoLocal) {
		return false
	}
	receiver := operation.Operands[0]
	index := operation.Operands[1]
	if receiver.Kind != OperandLocal && (operation.OwnerType != hir.SharedBorrow(hir.TypeByteSlice) || receiver.Kind != OperandByteString || receiver.Type != operation.OwnerType) {
		return false
	}
	return index.Kind == OperandLocal || index.Kind == OperandUSize
}

func validViewIndexStoreShape(operation Operation) bool {
	element, indexed := hir.IndexedViewElement(operation.OwnerType)
	if len(operation.Operands) != 3 || !indexed || !hir.MutableIndexedView(operation.OwnerType) ||
		operation.IndexType != hir.TypeUSize || operation.Type != element || operation.Operator != "[]=" || operation.DestinationLocal != NoLocal {
		return false
	}
	return operation.Operands[0].Kind == OperandLocal &&
		(operation.Operands[1].Kind == OperandLocal || operation.Operands[1].Kind == OperandUSize) &&
		scalarOperandMatches(operation.Operands[2], element)
}

func isVerifiedIndexedViewType(value hir.Type) bool {
	_, indexed := hir.IndexedViewElement(value)
	return indexed
}

func validArrayConstructShape(operation Operation) bool {
	elementType, length, array := hir.FixedArrayParts(operation.Type)
	if !array || !hir.ExecutableFixedArrayShape(elementType, length) || operation.OwnerType != elementType ||
		operation.Operator != "list" || operation.DestinationLocal == NoLocal || uint64(len(operation.Operands)) != length {
		return false
	}
	for _, operand := range operation.Operands {
		if !scalarOperandMatches(operand, elementType) {
			return false
		}
	}
	return true
}

func validArrayBorrowViewShape(operation Operation) bool {
	return hir.ArrayViewBorrow(operation.OwnerType, operation.Type, operation.Operator) &&
		operation.DestinationLocal != NoLocal && len(operation.Operands) == 1 && operation.Operands[0].Kind == OperandLocal
}

func validArrayLengthShape(operation Operation) bool {
	elementType, length, array := hir.FixedArrayParts(operation.OwnerType)
	return array && hir.ExecutableFixedArrayShape(elementType, length) && operation.DestinationLocal != NoLocal &&
		operation.Type == hir.TypeUSize && operation.Field == "len" && len(operation.Operands) == 1 &&
		operation.Operands[0].Kind == OperandLocal
}

func validArrayIndexShape(operation Operation, destination bool) bool {
	elementType, length, array := hir.FixedArrayParts(operation.OwnerType)
	if !array || !hir.ExecutableFixedArrayShape(elementType, length) || operation.Type != elementType || operation.IndexType != hir.TypeUSize || operation.Operator != "[]" ||
		len(operation.Operands) != 2 || destination != (operation.DestinationLocal != NoLocal) {
		return false
	}
	return operation.Operands[0].Kind == OperandLocal &&
		(operation.Operands[1].Kind == OperandLocal || operation.Operands[1].Kind == OperandUSize)
}

func validArrayIndexStoreShape(operation Operation) bool {
	elementType, length, array := hir.FixedArrayParts(operation.OwnerType)
	if !array || !hir.ExecutableFixedArrayShape(elementType, length) || operation.Type != elementType ||
		operation.IndexType != hir.TypeUSize || operation.Operator != "[]=" || len(operation.Operands) != 3 || operation.DestinationLocal != NoLocal {
		return false
	}
	return operation.Operands[0].Kind == OperandLocal &&
		(operation.Operands[1].Kind == OperandLocal || operation.Operands[1].Kind == OperandUSize) &&
		scalarOperandMatches(operation.Operands[2], elementType)
}

func validViewSliceShape(operation Operation, destination bool) bool {
	wantView := operation.OwnerType
	if len(operation.Operands) != 3 || !isIndexedViewType(wantView) ||
		operation.IndexType != hir.TypeUSize || operation.Type != wantView || operation.Operator != ".." {
		return false
	}
	if destination != (operation.DestinationLocal != NoLocal) {
		return false
	}
	receiver := operation.Operands[0]
	if receiver.Kind != OperandLocal && (receiver.Kind != OperandByteString || receiver.Type != wantView) {
		return false
	}
	for _, bound := range operation.Operands[1:] {
		if bound.Kind != OperandLocal && bound.Kind != OperandUSize {
			return false
		}
	}
	return true
}

func scalarCastOperandMatches(operand Operand, sourceType hir.Type) bool {
	if operand.Kind == OperandLocal {
		return true
	}
	return sourceType == hir.TypeI32 && operand.Kind == OperandI32 ||
		sourceType == hir.TypeI64 && operand.Kind == OperandI64 ||
		sourceType == hir.TypeU8 && operand.Kind == OperandU8 ||
		sourceType == hir.TypeU32 && operand.Kind == OperandU32
}

func validBinaryShape(operation Operation) bool {
	if operation.DestinationLocal == NoLocal || len(operation.Operands) != 2 ||
		!scalarOperandMatches(operation.Operands[0], operation.OwnerType) ||
		!scalarOperandMatches(operation.Operands[1], operation.OwnerType) {
		return false
	}
	switch operation.Operator {
	case "==", "!=":
		_, _, pointer := hir.RawPointerElement(operation.OwnerType)
		return (hir.ScalarEqualityType(operation.OwnerType) || pointer) && operation.Type == hir.TypeBool
	case "<", "<=", ">", ">=":
		return hir.ScalarOrderedType(operation.OwnerType) && operation.Type == hir.TypeBool
	case "+", "-", "*":
		return hir.CheckedArithmeticType(operation.OwnerType) && operation.Type == operation.OwnerType
	case "/", "%":
		return hir.CheckedDivisionType(operation.OwnerType) && operation.Type == operation.OwnerType
	case ">>", "<<":
		return hir.CheckedShiftType(operation.OwnerType) && operation.Type == operation.OwnerType
	case "&", "^", "|":
		return hir.BitwiseIntegerType(operation.OwnerType) && operation.Type == operation.OwnerType
	default:
		return false
	}
}

func validUnaryShape(operation Operation) bool {
	if operation.DestinationLocal == NoLocal || len(operation.Operands) != 1 ||
		!scalarOperandMatches(operation.Operands[0], operation.OwnerType) {
		return false
	}
	switch operation.Operator {
	case "&":
		return operation.Type == hir.SharedBorrow(operation.OwnerType)
	case "&mut":
		return operation.Type == hir.MutableBorrow(operation.OwnerType)
	case "*":
		element, _, borrow := hir.BorrowElement(operation.OwnerType)
		return borrow && operation.Type == element
	case "!":
		return operation.OwnerType == hir.TypeBool && operation.Type == hir.TypeBool
	case "+", "-":
		return signedIntegerType(operation.OwnerType) && operation.Type == operation.OwnerType
	case "~":
		return hir.BitwiseNotType(operation.OwnerType) && operation.Type == operation.OwnerType
	default:
		return false
	}
}

func scalarOperandMatches(operand Operand, ownerType hir.Type) bool {
	if operand.Kind == OperandLocal {
		return true
	}
	if _, _, pointer := hir.RawPointerElement(ownerType); pointer {
		return operand.Kind == OperandNull && operand.Type == ownerType
	}
	switch ownerType {
	case hir.TypeI64:
		return operand.Kind == OperandI64
	case hir.TypeBool, hir.TypeI32:
		return operand.Kind == OperandI32
	case hir.TypeU8:
		return operand.Kind == OperandU8
	case hir.TypeU32:
		return operand.Kind == OperandU32
	case hir.TypeUSize:
		return operand.Kind == OperandUSize
	default:
		return false
	}
}
