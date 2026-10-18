package interpreter

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

func evaluate(locals map[mir.LocalID]runtimeValue, operand mir.Operand) (runtimeValue, error) {
	switch operand.Kind {
	case mir.OperandI64:
		return i64Value(operand.I64), nil
	case mir.OperandI32:
		if operand.Type == hir.TypeBool {
			if operand.I32 != 0 && operand.I32 != 1 {
				return runtimeValue{}, capabilityError("boolean immediate has value %d", operand.I32)
			}
			return runtimeValue{kind: valueI32, scalar: i32Bits(operand.I32), typ: hir.TypeBool}, nil
		}
		return i32Value(operand.I32), nil
	case mir.OperandU8:
		return runtimeValue{kind: valueU8, scalar: operand.U64, typ: hir.TypeU8}, nil
	case mir.OperandU32:
		return runtimeValue{kind: valueU32, scalar: operand.U64, typ: hir.TypeU32}, nil
	case mir.OperandUSize:
		return runtimeValue{kind: valueUSize, scalar: operand.U64, typ: hir.TypeUSize}, nil
	case mir.OperandString, mir.OperandByteString:
		return staticByteView(operand.StaticData, operand.Type), nil
	case mir.OperandNull:
		return runtimeValue{kind: valueNull, typ: operand.Type}, nil
	case mir.OperandLocal:
		value, found := locals[operand.Local]
		if !found {
			return runtimeValue{}, capabilityError("local %d has no executable value", operand.Local)
		}
		return value, nil
	default:
		return runtimeValue{}, capabilityError("operand has no executable value")
	}
}

func evaluateOne(locals map[mir.LocalID]runtimeValue, operands []mir.Operand) (runtimeValue, error) {
	if len(operands) != 1 {
		return runtimeValue{}, fmt.Errorf("operation has %d operands, want 1", len(operands))
	}
	return evaluate(locals, operands[0])
}

func evaluateOneI32(locals map[mir.LocalID]runtimeValue, operands []mir.Operand) (int32, error) {
	value, err := evaluateOne(locals, operands)
	if err != nil {
		return 0, err
	}
	return value.asI32()
}

func evaluateTwoI32(locals map[mir.LocalID]runtimeValue, operands []mir.Operand) (int32, int32, error) {
	leftValue, rightValue, err := evaluateTwo(locals, operands)
	if err != nil {
		return 0, 0, err
	}
	left, err := leftValue.asI32()
	if err != nil {
		return 0, 0, err
	}
	right, err := rightValue.asI32()
	return left, right, err
}

func evaluateTwo(locals map[mir.LocalID]runtimeValue, operands []mir.Operand) (runtimeValue, runtimeValue, error) {
	if len(operands) != 2 {
		return runtimeValue{}, runtimeValue{}, fmt.Errorf("operation has %d operands, want 2", len(operands))
	}
	left, err := evaluate(locals, operands[0])
	if err != nil {
		return runtimeValue{}, runtimeValue{}, err
	}
	right, err := evaluate(locals, operands[1])
	return left, right, err
}

func evaluateViewIndexOperands(locals map[mir.LocalID]runtimeValue, operands []mir.Operand) (runtimeValue, runtimeValue, error) {
	if len(operands) != 2 {
		return runtimeValue{}, runtimeValue{}, fmt.Errorf("byte-view index has %d operands, want 2", len(operands))
	}
	receiver, err := evaluate(locals, operands[0])
	if err != nil {
		return runtimeValue{}, runtimeValue{}, err
	}
	index, err := evaluate(locals, operands[1])
	if err != nil {
		return runtimeValue{}, runtimeValue{}, err
	}
	if receiver.kind != valueView || receiver.view() == nil || !isRuntimeIndexedViewType(receiver.typ) || index.kind != valueUSize || index.typ != hir.TypeUSize {
		return runtimeValue{}, runtimeValue{}, capabilityError("byte-view index received incompatible operands")
	}
	return receiver, index, nil
}

func evaluateViewIndexStoreOperands(locals map[mir.LocalID]runtimeValue, operands []mir.Operand) (runtimeValue, runtimeValue, runtimeValue, error) {
	receiver, index, value, err := evaluateThreeOperands(locals, operands, "mutable byte-view index store")
	if err != nil {
		return runtimeValue{}, runtimeValue{}, runtimeValue{}, err
	}
	element, indexed := hir.IndexedViewElement(receiver.typ)
	if !indexed || receiver.kind != valueView || receiver.view() == nil || !hir.MutableIndexedView(receiver.typ) ||
		index.kind != valueUSize || index.typ != hir.TypeUSize || !indexedElementRuntimeMatches(value, element) {
		return runtimeValue{}, runtimeValue{}, runtimeValue{}, capabilityError("mutable byte-view index store received incompatible operands")
	}
	return receiver, index, value, nil
}

func evaluateArrayIndexOperands(locals map[mir.LocalID]runtimeValue, operands []mir.Operand) (runtimeValue, runtimeValue, error) {
	if len(operands) != 2 {
		return runtimeValue{}, runtimeValue{}, fmt.Errorf("fixed-array index has %d operands, want 2", len(operands))
	}
	receiver, err := evaluate(locals, operands[0])
	if err != nil {
		return runtimeValue{}, runtimeValue{}, err
	}
	index, err := evaluate(locals, operands[1])
	if err != nil {
		return runtimeValue{}, runtimeValue{}, err
	}
	if receiver.kind != valueArray || receiver.array() == nil || index.kind != valueUSize || index.typ != hir.TypeUSize {
		return runtimeValue{}, runtimeValue{}, capabilityError("fixed-array index received incompatible operands")
	}
	return receiver, index, nil
}

func evaluateArrayIndexStoreOperands(locals map[mir.LocalID]runtimeValue, operands []mir.Operand) (runtimeValue, runtimeValue, runtimeValue, error) {
	receiver, index, value, err := evaluateThreeOperands(locals, operands, "fixed-array index store")
	if err != nil {
		return runtimeValue{}, runtimeValue{}, runtimeValue{}, err
	}
	if receiver.kind != valueArray || receiver.array() == nil || index.kind != valueUSize || index.typ != hir.TypeUSize {
		return runtimeValue{}, runtimeValue{}, runtimeValue{}, capabilityError("fixed-array index store received incompatible operands")
	}
	return receiver, index, value, nil
}

func evaluateThreeOperands(locals map[mir.LocalID]runtimeValue, operands []mir.Operand, operation string) (runtimeValue, runtimeValue, runtimeValue, error) {
	if len(operands) != 3 {
		return runtimeValue{}, runtimeValue{}, runtimeValue{}, fmt.Errorf("%s has %d operands, want 3", operation, len(operands))
	}
	first, err := evaluate(locals, operands[0])
	if err != nil {
		return runtimeValue{}, runtimeValue{}, runtimeValue{}, err
	}
	second, err := evaluate(locals, operands[1])
	if err != nil {
		return runtimeValue{}, runtimeValue{}, runtimeValue{}, err
	}
	third, err := evaluate(locals, operands[2])
	if err != nil {
		return runtimeValue{}, runtimeValue{}, runtimeValue{}, err
	}
	return first, second, third, nil
}

func isRuntimeIndexedViewType(value hir.Type) bool {
	_, indexed := hir.IndexedViewElement(value)
	return indexed
}

func evaluateViewSliceOperands(locals map[mir.LocalID]runtimeValue, operands []mir.Operand) (runtimeValue, runtimeValue, runtimeValue, error) {
	if len(operands) != 3 {
		return runtimeValue{}, runtimeValue{}, runtimeValue{}, fmt.Errorf("byte-view slice has %d operands, want 3", len(operands))
	}
	receiver, err := evaluate(locals, operands[0])
	if err != nil {
		return runtimeValue{}, runtimeValue{}, runtimeValue{}, err
	}
	start, err := evaluate(locals, operands[1])
	if err != nil {
		return runtimeValue{}, runtimeValue{}, runtimeValue{}, err
	}
	end, err := evaluate(locals, operands[2])
	if err != nil {
		return runtimeValue{}, runtimeValue{}, runtimeValue{}, err
	}
	if receiver.kind != valueView || !isRuntimeIndexedViewType(receiver.typ) ||
		start.kind != valueUSize || start.typ != hir.TypeUSize || end.kind != valueUSize || end.typ != hir.TypeUSize {
		return runtimeValue{}, runtimeValue{}, runtimeValue{}, capabilityError("byte-view slice received incompatible operands")
	}
	return receiver, start, end, nil
}
