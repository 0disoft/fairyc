package interpreter

import (
	"fmt"
	"math"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

func oneSuccessor(terminator mir.Terminator) (mir.BlockID, error) {
	if len(terminator.Successors) != 1 {
		return mir.NoBlock, fmt.Errorf("terminator %q has %d successors, want 1", terminator.Kind, len(terminator.Successors))
	}
	return terminator.Successors[0], nil
}

func evaluateSwitch(locals map[mir.LocalID]runtimeValue, terminator mir.Terminator) (mir.BlockID, string, error) {
	switch terminator.SwitchKind {
	case "bool":
		value, err := evaluateOneI32(locals, terminator.SwitchOperands)
		if err != nil {
			return mir.NoBlock, "", err
		}
		if len(terminator.Successors) != 2 {
			return mir.NoBlock, "", fmt.Errorf("bool switch has %d successors, want 2", len(terminator.Successors))
		}
		if value != 0 && value != 1 {
			return mir.NoBlock, "", fmt.Errorf("bool switch received non-boolean value %d", value)
		}
		return terminator.Successors[value^1], "", nil
	case "assert":
		value, err := evaluateOne(locals, terminator.SwitchOperands)
		if err != nil {
			return mir.NoBlock, "", err
		}
		if value.kind != valueI32 || value.typ != hir.TypeBool || (value.i32() != 0 && value.i32() != 1) {
			return mir.NoBlock, "", capabilityError("assert received a non-boolean value")
		}
		if len(terminator.Successors) != 2 {
			return mir.NoBlock, "", fmt.Errorf("assert switch has %d successors, want 2", len(terminator.Successors))
		}
		if value.i32() == 0 {
			return mir.NoBlock, "assertion-failed", nil
		}
		return terminator.Successors[0], "", nil
	case "result":
		value, err := evaluateOne(locals, terminator.SwitchOperands)
		if err != nil {
			return mir.NoBlock, "", err
		}
		if value.kind != valueResult || value.tagged() == nil || value.tagged().payload == nil || value.tagged().dropped {
			return mir.NoBlock, "", capabilityError("Result switch received a non-Result value")
		}
		if len(terminator.Successors) != 2 {
			return mir.NoBlock, "", fmt.Errorf("Result switch has %d successors, want 2", len(terminator.Successors))
		}
		if value.tagged().variant == "ok" {
			return terminator.Successors[0], "", nil
		}
		if value.tagged().variant == "err" {
			return terminator.Successors[1], "", nil
		}
		return mir.NoBlock, "", fmt.Errorf("Result switch received invalid variant %q", value.tagged().variant)
	case "option":
		value, err := evaluateOne(locals, terminator.SwitchOperands)
		if err != nil {
			return mir.NoBlock, "", err
		}
		if value.kind != valueOption || value.tagged() == nil || value.tagged().dropped {
			return mir.NoBlock, "", capabilityError("Option switch received a non-Option value")
		}
		if len(terminator.Successors) != 2 {
			return mir.NoBlock, "", fmt.Errorf("Option switch has %d successors, want 2", len(terminator.Successors))
		}
		if value.tagged().variant == "some" && value.tagged().payload != nil {
			return terminator.Successors[0], "", nil
		}
		if value.tagged().variant == "none" && value.tagged().payload == nil {
			return terminator.Successors[1], "", nil
		}
		return mir.NoBlock, "", fmt.Errorf("Option switch received invalid variant %q", value.tagged().variant)
	case "enum":
		value, err := evaluateOne(locals, terminator.SwitchOperands)
		if err != nil {
			return mir.NoBlock, "", err
		}
		if value.kind != valueEnum || value.enum() == nil || value.enum().definition == nil || value.enum().definition.Type != terminator.SwitchType {
			return mir.NoBlock, "", capabilityError("enum switch received a non-matching enum value")
		}
		if len(terminator.Successors) != len(terminator.SwitchVariants) {
			return mir.NoBlock, "", fmt.Errorf("enum switch has %d successors for %d variants", len(terminator.Successors), len(terminator.SwitchVariants))
		}
		for index, variant := range terminator.SwitchVariants {
			if value.enum().variant == variant {
				return terminator.Successors[index], "", nil
			}
		}
		return mir.NoBlock, "", fmt.Errorf("enum switch received invalid variant %q", value.enum().variant)
	case "checked-division":
		left, right, err := evaluateTwo(locals, terminator.SwitchOperands)
		if err != nil {
			return mir.NoBlock, "", err
		}
		if terminator.SwitchType == hir.TypeI64 {
			_, trap, err := checkedI64Operation(terminator.Operator, left, right)
			if err != nil || trap != "" {
				return mir.NoBlock, trap, err
			}
		} else if terminator.SwitchType == hir.TypeI32 {
			leftI32, leftErr := left.asI32()
			rightI32, rightErr := right.asI32()
			if leftErr != nil || rightErr != nil {
				return mir.NoBlock, "", capabilityError("checked i32 division received incompatible operands")
			}
			if rightI32 == 0 {
				return mir.NoBlock, "division-by-zero", nil
			}
			if leftI32 == math.MinInt32 && rightI32 == -1 {
				return mir.NoBlock, "signed-division-overflow", nil
			}
		} else {
			_, trap, err := checkedUnsignedIntegerOperation(terminator.Operator, left, right, terminator.SwitchType)
			if err != nil {
				return mir.NoBlock, "", err
			}
			if trap != "" {
				return mir.NoBlock, trap, nil
			}
		}
		return terminator.Successors[0], "", nil
	case "checked-view-index":
		receiver, index, err := evaluateViewIndexOperands(locals, terminator.SwitchOperands)
		if err != nil {
			return mir.NoBlock, "", err
		}
		if len(terminator.Successors) != 2 {
			return mir.NoBlock, "", fmt.Errorf("checked byte-view index has %d successors, want 2", len(terminator.Successors))
		}
		if receiver.view() == nil || index.u64() >= receiver.view().length {
			return mir.NoBlock, "bounds", nil
		}
		return terminator.Successors[0], "", nil
	case "checked-array-index":
		receiver, index, err := evaluateArrayIndexOperands(locals, terminator.SwitchOperands)
		if err != nil {
			return mir.NoBlock, "", err
		}
		_, length, array := hir.FixedArrayParts(receiver.typ)
		if !array {
			return mir.NoBlock, "", capabilityError("checked fixed-array index received a non-array value")
		}
		if len(terminator.Successors) != 2 {
			return mir.NoBlock, "", fmt.Errorf("checked fixed-array index has %d successors, want 2", len(terminator.Successors))
		}
		if index.u64() >= length {
			return mir.NoBlock, "bounds", nil
		}
		return terminator.Successors[0], "", nil
	case "checked-view-slice":
		receiver, start, end, err := evaluateViewSliceOperands(locals, terminator.SwitchOperands)
		if err != nil {
			return mir.NoBlock, "", err
		}
		if len(terminator.Successors) != 2 {
			return mir.NoBlock, "", fmt.Errorf("checked byte-view slice has %d successors, want 2", len(terminator.Successors))
		}
		if receiver.view() == nil || start.u64() > end.u64() || end.u64() > receiver.view().length {
			return mir.NoBlock, "bounds", nil
		}
		return terminator.Successors[0], "", nil
	case "checked-unary":
		if terminator.SwitchType == hir.TypeI64 {
			value, err := evaluateOne(locals, terminator.SwitchOperands)
			if err != nil {
				return mir.NoBlock, "", err
			}
			_, trap, err := unaryI64(terminator.Operator, value)
			if err != nil || trap != "" {
				return mir.NoBlock, trap, err
			}
			return terminator.Successors[0], "", nil
		}
		value, err := evaluateOneI32(locals, terminator.SwitchOperands)
		if err != nil {
			return mir.NoBlock, "", err
		}
		_, trap, err := unary(terminator.Operator, value)
		if err != nil {
			return mir.NoBlock, "", err
		}
		if trap != "" {
			return mir.NoBlock, trap, nil
		}
		return terminator.Successors[0], "", nil
	case "checked-arithmetic":
		left, right, err := evaluateTwo(locals, terminator.SwitchOperands)
		if err != nil {
			return mir.NoBlock, "", err
		}
		var trap string
		if terminator.SwitchType == hir.TypeI64 {
			_, trap, err = checkedI64Operation(terminator.Operator, left, right)
		} else if terminator.SwitchType == hir.TypeI32 {
			leftI32, leftErr := left.asI32()
			rightI32, rightErr := right.asI32()
			if leftErr != nil || rightErr != nil {
				return mir.NoBlock, "", capabilityError("checked i32 arithmetic received incompatible operands")
			}
			_, trap, err = binary(terminator.Operator, leftI32, rightI32)
		} else {
			_, trap, err = checkedUnsignedIntegerOperation(terminator.Operator, left, right, terminator.SwitchType)
		}
		if err != nil {
			return mir.NoBlock, "", err
		}
		if trap != "" {
			return mir.NoBlock, trap, nil
		}
		return terminator.Successors[0], "", nil
	case "checked-shift":
		left, right, err := evaluateTwo(locals, terminator.SwitchOperands)
		if err != nil {
			return mir.NoBlock, "", err
		}
		var trap string
		if terminator.SwitchType == hir.TypeI64 {
			_, trap, err = checkedI64Operation(terminator.Operator, left, right)
		} else if terminator.SwitchType == hir.TypeI32 {
			leftI32, leftErr := left.asI32()
			rightI32, rightErr := right.asI32()
			if leftErr != nil || rightErr != nil {
				return mir.NoBlock, "", capabilityError("checked i32 right shift received incompatible operands")
			}
			_, trap, err = binary(terminator.Operator, leftI32, rightI32)
		} else {
			_, trap, err = checkedUnsignedIntegerOperation(terminator.Operator, left, right, terminator.SwitchType)
		}
		if err != nil {
			return mir.NoBlock, "", err
		}
		if trap != "" {
			return mir.NoBlock, trap, nil
		}
		return terminator.Successors[0], "", nil
	default:
		return mir.NoBlock, "", capabilityError("unsupported MIR switch %q", terminator.SwitchKind)
	}
}
