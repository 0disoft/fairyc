package interpreter

import (
	"fmt"
	"math"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

func evaluateBinary(locals map[mir.LocalID]runtimeValue, operation mir.Operation) (runtimeValue, string, error) {
	if len(operation.Operands) != 2 {
		return runtimeValue{}, "", fmt.Errorf("binary operation has %d operands, want 2", len(operation.Operands))
	}
	leftValue, err := evaluate(locals, operation.Operands[0])
	if err != nil {
		return runtimeValue{}, "", err
	}
	rightValue, err := evaluate(locals, operation.Operands[1])
	if err != nil {
		return runtimeValue{}, "", err
	}
	if operation.OwnerType == hir.TypeI64 && operation.Type == hir.TypeI64 {
		return checkedI64Operation(operation.Operator, leftValue, rightValue)
	}
	switch operation.Operator {
	case "==", "!=":
		_, _, pointer := hir.RawPointerElement(operation.OwnerType)
		if (!hir.ScalarEqualityType(operation.OwnerType) && !pointer) || operation.Type != hir.TypeBool {
			return runtimeValue{}, "", capabilityError("equality comparison has incompatible owner or result type")
		}
		equal, err := equalScalarValues(leftValue, rightValue, operation.OwnerType)
		if err != nil {
			return runtimeValue{}, "", err
		}
		if operation.Operator == "!=" {
			equal = !equal
		}
		return boolRuntimeValue(equal), "", nil
	case "<", "<=", ">", ">=":
		if !hir.ScalarOrderedType(operation.OwnerType) || operation.Type != hir.TypeBool {
			return runtimeValue{}, "", capabilityError("ordered comparison has incompatible owner or result type")
		}
		result, err := orderedScalarValues(operation.Operator, leftValue, rightValue, operation.OwnerType)
		return boolRuntimeValue(result), "", err
	case "+", "-", "*", "/", "%":
		if !hir.CheckedDivisionType(operation.OwnerType) || operation.Type != operation.OwnerType {
			return runtimeValue{}, "", capabilityError("checked integer operation has incompatible owner or result type")
		}
		if operation.OwnerType != hir.TypeI32 {
			return checkedUnsignedIntegerOperation(operation.Operator, leftValue, rightValue, operation.OwnerType)
		}
	case ">>", "<<":
		if !hir.CheckedShiftType(operation.OwnerType) || operation.Type != operation.OwnerType {
			return runtimeValue{}, "", capabilityError("checked shift has incompatible owner or result type")
		}
		if operation.OwnerType != hir.TypeI32 {
			return checkedUnsignedIntegerOperation(operation.Operator, leftValue, rightValue, operation.OwnerType)
		}
	case "&", "^", "|":
		if !hir.BitwiseIntegerType(operation.OwnerType) || operation.Type != operation.OwnerType {
			return runtimeValue{}, "", capabilityError("bitwise operation has incompatible owner or result type")
		}
		if operation.OwnerType != hir.TypeI32 {
			return checkedUnsignedIntegerOperation(operation.Operator, leftValue, rightValue, operation.OwnerType)
		}
	}
	if operation.OwnerType != hir.TypeI32 || operation.Type != hir.TypeI32 {
		return runtimeValue{}, "", capabilityError("arithmetic operation has incompatible owner or result type")
	}
	left, err := leftValue.asI32()
	if err != nil {
		return runtimeValue{}, "", err
	}
	right, err := rightValue.asI32()
	if err != nil {
		return runtimeValue{}, "", err
	}
	result, trap, err := binary(operation.Operator, left, right)
	return i32Value(result), trap, err
}

func evaluateUnary(locals map[mir.LocalID]runtimeValue, operation mir.Operation) (runtimeValue, string, error) {
	if len(operation.Operands) != 1 {
		return runtimeValue{}, "", fmt.Errorf("unary operation has %d operands, want 1", len(operation.Operands))
	}
	value, err := evaluate(locals, operation.Operands[0])
	if err != nil {
		return runtimeValue{}, "", err
	}
	if operation.Operator == "&" || operation.Operator == "&mut" {
		wantType := hir.SharedBorrow(operation.OwnerType)
		if operation.Operator == "&mut" {
			wantType = hir.MutableBorrow(operation.OwnerType)
		}
		if operation.Type != wantType ||
			value.typ != operation.OwnerType && (!hir.FixedArrayScalarElement(operation.OwnerType) || !scalarStoreValueMatches(value, operation.OwnerType)) {
			return runtimeValue{}, "", capabilityError("safe borrow has incompatible operand or result type")
		}
		value.typ = operation.Type
		if hir.FixedArrayScalarElement(operation.OwnerType) && operation.Operands[0].Kind == mir.OperandLocal {
			value.payload = &scalarReference{locals: locals, local: operation.Operands[0].Local}
		}
		return value, "", nil
	}
	if operation.Operator == "*" {
		element, _, borrow := hir.BorrowElement(operation.OwnerType)
		if !borrow || operation.Type != element || value.typ != operation.OwnerType {
			return runtimeValue{}, "", capabilityError("safe dereference has incompatible operand or result type")
		}
		if value.reference() != nil {
			referent, exists := value.reference().load()
			if !exists || !scalarStoreValueMatches(referent, element) {
				return runtimeValue{}, "", capabilityError("safe dereference has invalid scalar referent")
			}
			return referent.cloneForValueCopy(), "", nil
		}
		value.typ = element
		return value, "", nil
	}
	if operation.OwnerType == hir.TypeI64 && operation.Type == hir.TypeI64 {
		return unaryI64(operation.Operator, value)
	}
	if operation.Operator == "~" && operation.OwnerType != hir.TypeI32 {
		if !hir.BitwiseNotType(operation.OwnerType) || operation.Type != operation.OwnerType ||
			!comparisonRuntimeMatches(value, operation.OwnerType) {
			return runtimeValue{}, "", capabilityError("bitwise not has incompatible operand or result type")
		}
		limit, ok := unsignedArithmeticLimit(operation.OwnerType)
		if !ok {
			return runtimeValue{}, "", capabilityError("bitwise not has unsupported type %q", operation.OwnerType)
		}
		return unsignedRuntimeValue(operation.OwnerType, limit^value.u64()), "", nil
	}
	if operation.Operator == "!" {
		if operation.OwnerType != hir.TypeBool || operation.Type != hir.TypeBool || value.typ != hir.TypeBool {
			return runtimeValue{}, "", capabilityError("logical not has incompatible operand or result type")
		}
	} else if operation.OwnerType != hir.TypeI32 || operation.Type != hir.TypeI32 {
		return runtimeValue{}, "", capabilityError("unary operation has incompatible operand or result type")
	}
	i32, err := value.asI32()
	if err != nil {
		return runtimeValue{}, "", err
	}
	result, trap, err := unary(operation.Operator, i32)
	if operation.Operator == "!" && err == nil && trap == "" {
		return boolRuntimeValue(result != 0), "", nil
	}
	return i32Value(result), trap, err
}

func checkedUnsignedIntegerOperation(operator string, left, right runtimeValue, ownerType hir.Type) (runtimeValue, string, error) {
	if !comparisonRuntimeMatches(left, ownerType) || !comparisonRuntimeMatches(right, ownerType) ||
		ownerType == hir.TypeBool || ownerType == hir.TypeI32 {
		return runtimeValue{}, "", capabilityError("unsigned integer operation received incompatible operands")
	}
	limit, ok := unsignedArithmeticLimit(ownerType)
	if !ok {
		return runtimeValue{}, "", capabilityError("unsigned integer operation has unsupported type %q", ownerType)
	}
	var result uint64
	switch operator {
	case "+":
		if left.u64() > limit-right.u64() {
			return runtimeValue{}, "integer-overflow", nil
		}
		result = left.u64() + right.u64()
	case "-":
		if left.u64() < right.u64() {
			return runtimeValue{}, "integer-overflow", nil
		}
		result = left.u64() - right.u64()
	case "*":
		if right.u64() != 0 && left.u64() > limit/right.u64() {
			return runtimeValue{}, "integer-overflow", nil
		}
		result = left.u64() * right.u64()
	case "/":
		if right.u64() == 0 {
			return runtimeValue{}, "division-by-zero", nil
		}
		result = left.u64() / right.u64()
	case "%":
		if right.u64() == 0 {
			return runtimeValue{}, "division-by-zero", nil
		}
		result = left.u64() % right.u64()
	case "&":
		result = left.u64() & right.u64()
	case "^":
		result = left.u64() ^ right.u64()
	case "|":
		result = left.u64() | right.u64()
	case ">>":
		width := uint64(64)
		if ownerType == hir.TypeU8 {
			width = 8
		} else if ownerType == hir.TypeU32 {
			width = 32
		}
		if right.u64() >= width {
			return runtimeValue{}, "invalid-shift", nil
		}
		result = left.u64() >> uint32(right.u64())
	case "<<":
		width := uint64(64)
		if ownerType == hir.TypeU8 {
			width = 8
		} else if ownerType == hir.TypeU32 {
			width = 32
		}
		if right.u64() >= width {
			return runtimeValue{}, "invalid-shift", nil
		}
		if left.u64() > limit>>uint32(right.u64()) {
			return runtimeValue{}, "integer-overflow", nil
		}
		result = left.u64() << uint32(right.u64())
	default:
		return runtimeValue{}, "", capabilityError("unsupported unsigned integer operator %q", operator)
	}
	return unsignedRuntimeValue(ownerType, result), "", nil
}

func unsignedArithmeticLimit(ownerType hir.Type) (uint64, bool) {
	switch ownerType {
	case hir.TypeU8:
		return math.MaxUint8, true
	case hir.TypeU32:
		return math.MaxUint32, true
	case hir.TypeUSize:
		return math.MaxUint64, true
	default:
		return 0, false
	}
}

func unsignedRuntimeValue(ownerType hir.Type, value uint64) runtimeValue {
	switch ownerType {
	case hir.TypeU8:
		return runtimeValue{kind: valueU8, scalar: value, typ: hir.TypeU8}
	case hir.TypeU32:
		return runtimeValue{kind: valueU32, scalar: value, typ: hir.TypeU32}
	case hir.TypeUSize:
		return runtimeValue{kind: valueUSize, scalar: value, typ: hir.TypeUSize}
	default:
		return runtimeValue{}
	}
}

func equalScalarValues(left, right runtimeValue, ownerType hir.Type) (bool, error) {
	if !comparisonRuntimeMatches(left, ownerType) || !comparisonRuntimeMatches(right, ownerType) {
		return false, capabilityError("comparison operands have different runtime scalar types")
	}
	switch ownerType {
	case hir.TypeI64:
		return left.i64() == right.i64(), nil
	case hir.TypeBool, hir.TypeI32:
		return left.i32() == right.i32(), nil
	case hir.TypeU8, hir.TypeU32, hir.TypeUSize:
		return left.u64() == right.u64(), nil
	default:
		if _, _, pointer := hir.RawPointerElement(ownerType); pointer {
			return true, nil
		}
		return false, capabilityError("value is not an executable equality value")
	}
}

func orderedScalarValues(operator string, left, right runtimeValue, ownerType hir.Type) (bool, error) {
	if !comparisonRuntimeMatches(left, ownerType) || !comparisonRuntimeMatches(right, ownerType) {
		return false, capabilityError("ordered comparison operands have incompatible runtime scalar types")
	}
	if ownerType == hir.TypeI64 {
		switch operator {
		case "<":
			return left.i64() < right.i64(), nil
		case "<=":
			return left.i64() <= right.i64(), nil
		case ">":
			return left.i64() > right.i64(), nil
		case ">=":
			return left.i64() >= right.i64(), nil
		}
	}
	if ownerType == hir.TypeI32 {
		switch operator {
		case "<":
			return left.i32() < right.i32(), nil
		case "<=":
			return left.i32() <= right.i32(), nil
		case ">":
			return left.i32() > right.i32(), nil
		case ">=":
			return left.i32() >= right.i32(), nil
		}
	}
	switch operator {
	case "<":
		return left.u64() < right.u64(), nil
	case "<=":
		return left.u64() <= right.u64(), nil
	case ">":
		return left.u64() > right.u64(), nil
	case ">=":
		return left.u64() >= right.u64(), nil
	default:
		return false, capabilityError("unsupported ordered comparison operator %q", operator)
	}
}

// Struct stores copy an exact inert nominal value, preserving source independence.
func indexedElementRuntimeMatches(value runtimeValue, element hir.Type) bool {
	if value.kind == valueStruct {
		return value.typ == element && value.structure() != nil
	}
	return comparisonRuntimeMatches(value, element)
}

func comparisonRuntimeMatches(value runtimeValue, ownerType hir.Type) bool {
	switch ownerType {
	case hir.TypeI64:
		return value.kind == valueI64 && value.typ == hir.TypeI64
	case hir.TypeBool, hir.TypeI32:
		return value.kind == valueI32
	case hir.TypeU8:
		return value.kind == valueU8 && value.typ == hir.TypeU8 && value.u64() <= math.MaxUint8
	case hir.TypeU32:
		return value.kind == valueU32 && value.typ == hir.TypeU32 && value.u64() <= math.MaxUint32
	case hir.TypeUSize:
		return value.kind == valueUSize && value.typ == hir.TypeUSize
	default:
		_, _, pointer := hir.RawPointerElement(ownerType)
		return pointer && value.kind == valueNull && value.typ == ownerType
	}
}

func valuePreservingIntegerCast(value runtimeValue, sourceType, targetType hir.Type) (runtimeValue, error) {
	sourceMatches := value.typ == sourceType || sourceType == hir.TypeI32 && value.kind == valueI32 && value.typ == hir.ParseType("")
	if !hir.ValuePreservingIntegerCast(sourceType, targetType) || !sourceMatches {
		return runtimeValue{}, capabilityError("integer cast has incompatible source or target type")
	}
	if targetType == hir.TypeI64 {
		switch sourceType {
		case hir.TypeI32:
			if value.kind == valueI32 {
				return i64Value(int64(value.i32())), nil
			}
		case hir.TypeU8, hir.TypeU32:
			if comparisonRuntimeMatches(value, sourceType) {
				return i64Value(int64(value.u64())), nil
			}
		}
		return runtimeValue{}, capabilityError("invalid i64 widening operand")
	}
	switch sourceType {
	case hir.TypeI32:
		if value.kind == valueI32 && targetType == hir.TypeCInt {
			return runtimeValue{kind: valueI32, scalar: i32Bits(value.i32()), typ: hir.TypeCInt}, nil
		}
	case hir.TypeU8:
		if value.kind != valueU8 || value.u64() > math.MaxUint8 {
			return runtimeValue{}, capabilityError("integer cast received an invalid u8 value")
		}
		switch targetType {
		case hir.TypeI32:
			return i32Value(int32(value.u64())), nil
		case hir.TypeU32:
			return runtimeValue{kind: valueU32, scalar: value.u64(), typ: hir.TypeU32}, nil
		case hir.TypeUSize:
			return runtimeValue{kind: valueUSize, scalar: value.u64(), typ: hir.TypeUSize}, nil
		}
	case hir.TypeU32:
		if value.kind == valueU32 && value.u64() <= math.MaxUint32 && targetType == hir.TypeUSize {
			return runtimeValue{kind: valueUSize, scalar: value.u64(), typ: hir.TypeUSize}, nil
		}
	case hir.TypeCInt:
		if value.kind == valueI32 && value.typ == hir.TypeCInt && targetType == hir.TypeI32 {
			return i32Value(value.i32()), nil
		}
	}
	return runtimeValue{}, capabilityError("integer cast is outside the executable widening subset")
}

func unary(operator string, value int32) (int32, string, error) {
	switch operator {
	case "+":
		return value, "", nil
	case "-":
		if value == math.MinInt32 {
			return 0, "integer-overflow", nil
		}
		return -value, "", nil
	case "~":
		return ^value, "", nil
	case "!":
		if value != 0 && value != 1 {
			return 0, "", fmt.Errorf("logical not received non-boolean value %d", value)
		}
		return value ^ 1, "", nil
	default:
		return 0, "", capabilityError("unsupported unary operator %q", operator)
	}
}

func binary(operator string, left, right int32) (int32, string, error) {
	switch operator {
	case "+":
		return checkedInteger(int64(left) + int64(right))
	case "-":
		return checkedInteger(int64(left) - int64(right))
	case "*":
		return checkedInteger(int64(left) * int64(right))
	case "/":
		if right == 0 {
			return 0, "division-by-zero", nil
		}
		if left == math.MinInt32 && right == -1 {
			return 0, "signed-division-overflow", nil
		}
		return left / right, "", nil
	case "%":
		if right == 0 {
			return 0, "division-by-zero", nil
		}
		if left == math.MinInt32 && right == -1 {
			return 0, "signed-division-overflow", nil
		}
		return left % right, "", nil
	case "<<", ">>":
		if right < 0 || right >= 32 {
			return 0, "invalid-shift", nil
		}
		if operator == "<<" {
			return checkedInteger(int64(left) << uint32(right))
		}
		return left >> uint32(right), "", nil
	case "&":
		return left & right, "", nil
	case "^":
		return left ^ right, "", nil
	case "|":
		return left | right, "", nil
	case "==":
		return boolValue(left == right), "", nil
	case "!=":
		return boolValue(left != right), "", nil
	case "<":
		return boolValue(left < right), "", nil
	case "<=":
		return boolValue(left <= right), "", nil
	case ">":
		return boolValue(left > right), "", nil
	case ">=":
		return boolValue(left >= right), "", nil
	default:
		return 0, "", capabilityError("unsupported binary operator %q", operator)
	}
}

func checkedInteger(value int64) (int32, string, error) {
	if value < math.MinInt32 || value > math.MaxInt32 {
		return 0, "integer-overflow", nil
	}
	return int32(value), "", nil
}

func boolValue(value bool) int32 {
	if value {
		return 1
	}
	return 0
}

func boolRuntimeValue(value bool) runtimeValue {
	return runtimeValue{kind: valueI32, scalar: i32Bits(boolValue(value)), typ: hir.TypeBool}
}
