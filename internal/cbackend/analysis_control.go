package cbackend

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
	"slices"
	"strconv"
)

func (e *analysisBuilder) inspectSwitch(functionIndex int, terminator mir.Terminator, predecessors []uint32) error {
	function := &e.functions[functionIndex]
	if terminator.SwitchKind == "enum" {
		index, found := e.enumByType[terminator.SwitchType]
		if !found {
			return CapabilityError{Function: function.Name, Feature: "enum switch type " + strconv.Quote(terminator.SwitchType.String())}
		}
		definition := &e.enumTypes[index]
		if len(terminator.Successors) != len(definition.definition.Variants) || len(terminator.SwitchVariants) != len(definition.definition.Variants) {
			return fmt.Errorf("enum switch in %s has an incomplete branch table", function.Name)
		}
		for variantIndex, variant := range definition.definition.Variants {
			if terminator.SwitchVariants[variantIndex] != variant.Name {
				return fmt.Errorf("enum switch in %s has variant %q at index %d, want %q", function.Name, terminator.SwitchVariants[variantIndex], variantIndex, variant.Name)
			}
		}
		operand, err := oneOperand(terminator.SwitchOperands)
		if err != nil {
			return fmt.Errorf("enum switch in %s: %w", function.Name, err)
		}
		valueType, err := e.operandType(functionIndex, operand)
		if err != nil {
			return fmt.Errorf("enum switch in %s: %w", function.Name, err)
		}
		if canonicalCValueType(valueType) != canonicalCValueType(terminator.SwitchType) {
			return fmt.Errorf("enum switch in %s has type %q, want %q", function.Name, valueType, terminator.SwitchType)
		}
		e.features["inert-enum"] = struct{}{}
		e.features["trap"] = struct{}{}
		e.trapCategories["unreachable"] = struct{}{}
		return nil
	}
	if len(terminator.Successors) != 2 {
		return fmt.Errorf("switch in %s has %d successors", function.Name, len(terminator.Successors))
	}
	var categories []string
	switch terminator.SwitchKind {
	case "bool":
		return nil
	case "assert":
		operand, err := oneOperand(terminator.SwitchOperands)
		if err != nil {
			return fmt.Errorf("assert in %s: %w", function.Name, err)
		}
		valid := operand.Kind == mir.OperandI32 && operand.Type == hir.TypeBool && (operand.I32 == 0 || operand.I32 == 1)
		if operand.Kind == mir.OperandLocal {
			operandType, typeErr := e.operandType(functionIndex, operand)
			valid = typeErr == nil && operandType == hir.TypeBool
		}
		if !valid || terminator.SwitchType != hir.TypeBool || terminator.Operator != "assert" {
			return fmt.Errorf("assert in %s has incompatible condition metadata", function.Name)
		}
		e.features["assert"] = struct{}{}
		categories = []string{"assertion-failed"}
	case "option":
		operand, err := oneOperand(terminator.SwitchOperands)
		if err != nil {
			return fmt.Errorf("Option switch in %s: %w", function.Name, err)
		}
		valueType, err := e.operandType(functionIndex, operand)
		if err != nil {
			return CapabilityError{Function: function.Name, Feature: "Option switch type " + strconv.Quote(valueType.String())}
		}
		_, feature, supported := e.optionCSpec(valueType)
		if !supported {
			return CapabilityError{Function: function.Name, Feature: "Option switch type " + strconv.Quote(valueType.String())}
		}
		e.features[feature] = struct{}{}
		e.features["trap"] = struct{}{}
		e.trapCategories["unreachable"] = struct{}{}
		return nil
	case "result":
		operand, err := oneOperand(terminator.SwitchOperands)
		if err != nil {
			return fmt.Errorf("Result switch in %s: %w", function.Name, err)
		}
		valueType, err := e.operandType(functionIndex, operand)
		if err != nil {
			return CapabilityError{Function: function.Name, Feature: "Result switch type " + strconv.Quote(valueType.String())}
		}
		_, _, feature, supported := e.resultCSpec(valueType)
		if !supported {
			return CapabilityError{Function: function.Name, Feature: "Result switch type " + strconv.Quote(valueType.String())}
		}
		e.features[feature] = struct{}{}
		e.features["trap"] = struct{}{}
		e.trapCategories["unreachable"] = struct{}{}
		return nil
	case "checked-division":
		left, right, err := twoOperands(terminator.SwitchOperands)
		if err != nil {
			return fmt.Errorf("checked division in %s: %w", function.Name, err)
		}
		leftType, leftErr := e.operandType(functionIndex, left)
		rightType, rightErr := e.operandType(functionIndex, right)
		if leftErr != nil || rightErr != nil || !hir.CheckedDivisionType(terminator.SwitchType) ||
			leftType != terminator.SwitchType || rightType != terminator.SwitchType {
			return fmt.Errorf("checked division in %s has incompatible type metadata", function.Name)
		}
		e.features["checked-division"] = struct{}{}
		categories = []string{"division-by-zero"}
		if terminator.SwitchType == hir.TypeI32 || terminator.SwitchType == hir.TypeI64 {
			categories = append(categories, "signed-division-overflow")
		}
	case "checked-view-index":
		receiver, index, err := twoOperands(terminator.SwitchOperands)
		if err != nil {
			return fmt.Errorf("checked byte-view index in %s: %w", function.Name, err)
		}
		receiverType, err := e.operandType(functionIndex, receiver)
		if err != nil {
			return fmt.Errorf("checked byte-view index receiver in %s: %w", function.Name, err)
		}
		indexType, err := e.operandType(functionIndex, index)
		if err != nil {
			return fmt.Errorf("checked byte-view index offset in %s: %w", function.Name, err)
		}
		if terminator.Operator != "[]" || !isCIndexedViewType(receiverType) || indexType != hir.TypeUSize {
			return fmt.Errorf("checked byte-view index in %s has incompatible operands", function.Name)
		}
		e.features["view-index"] = struct{}{}
		categories = []string{"bounds"}
	case "checked-array-index":
		receiver, index, err := twoOperands(terminator.SwitchOperands)
		if err != nil {
			return fmt.Errorf("checked fixed-array index in %s: %w", function.Name, err)
		}
		receiverType, receiverErr := e.operandType(functionIndex, receiver)
		indexType, indexErr := e.operandType(functionIndex, index)
		if _, _, array := hir.FixedArrayParts(receiverType); receiverErr != nil || indexErr != nil || !array ||
			terminator.Operator != "[]" || indexType != hir.TypeUSize {
			return fmt.Errorf("checked fixed-array index in %s has incompatible operands", function.Name)
		}
		e.features["array-index"] = struct{}{}
		categories = []string{"bounds"}
	case "checked-view-slice":
		receiver, start, end, err := threeOperands(terminator.SwitchOperands)
		if err != nil {
			return fmt.Errorf("checked byte-view slice in %s: %w", function.Name, err)
		}
		receiverType, receiverErr := e.operandType(functionIndex, receiver)
		startType, startErr := e.operandType(functionIndex, start)
		endType, endErr := e.operandType(functionIndex, end)
		if receiverErr != nil || startErr != nil || endErr != nil || terminator.Operator != ".." ||
			!isCIndexedViewType(receiverType) || startType != hir.TypeUSize || endType != hir.TypeUSize {
			return fmt.Errorf("checked byte-view slice in %s has incompatible operands", function.Name)
		}
		e.features["view-slice"] = struct{}{}
		categories = []string{"bounds"}
	case "checked-unary":
		operand, err := oneOperand(terminator.SwitchOperands)
		if err != nil {
			return fmt.Errorf("checked unary in %s: %w", function.Name, err)
		}
		operandType, err := e.operandType(functionIndex, operand)
		if err != nil || (terminator.SwitchType != hir.TypeI32 && terminator.SwitchType != hir.TypeI64) || operandType != terminator.SwitchType {
			return fmt.Errorf("checked unary in %s has incompatible type metadata", function.Name)
		}
		e.features["checked-arithmetic"] = struct{}{}
		categories = []string{"integer-overflow"}
	case "checked-arithmetic":
		left, right, err := twoOperands(terminator.SwitchOperands)
		if err != nil {
			return fmt.Errorf("checked arithmetic in %s: %w", function.Name, err)
		}
		leftType, leftErr := e.operandType(functionIndex, left)
		rightType, rightErr := e.operandType(functionIndex, right)
		if leftErr != nil || rightErr != nil || !hir.CheckedArithmeticType(terminator.SwitchType) ||
			leftType != terminator.SwitchType || rightType != terminator.SwitchType {
			return fmt.Errorf("checked arithmetic in %s has incompatible type metadata", function.Name)
		}
		if terminator.SwitchType == hir.TypeI64 {
			switch terminator.Operator {
			case "+":
				e.i64Helpers.addOverflow = true
			case "-":
				e.i64Helpers.subOverflow = true
			case "*":
				e.i64Helpers.mulOverflow = true
			}
		}
		e.features["checked-arithmetic"] = struct{}{}
		categories = []string{"integer-overflow"}
	case "checked-shift":
		left, right, err := twoOperands(terminator.SwitchOperands)
		if err != nil {
			return fmt.Errorf("checked shift in %s: %w", function.Name, err)
		}
		leftType, leftErr := e.operandType(functionIndex, left)
		rightType, rightErr := e.operandType(functionIndex, right)
		if leftErr != nil || rightErr != nil || !hir.CheckedShiftType(terminator.SwitchType) ||
			leftType != terminator.SwitchType || rightType != terminator.SwitchType {
			return fmt.Errorf("checked shift in %s has incompatible type metadata", function.Name)
		}
		e.features["checked-shift"] = struct{}{}
		switch terminator.Operator {
		case "<<":
			if terminator.SwitchType == hir.TypeI64 {
				e.i64Helpers.shift = true
			}
			categories = []string{"invalid-shift", "integer-overflow"}
		case ">>":
			categories = []string{"invalid-shift"}
		default:
			return fmt.Errorf("checked shift in %s has unsupported operator %q", function.Name, terminator.Operator)
		}
	default:
		return CapabilityError{Function: function.Name, Feature: "MIR switch " + strconv.Quote(terminator.SwitchKind)}
	}
	if err := e.recordTrapCategories(function.Name, categories); err != nil {
		return err
	}
	e.trapCategories["unreachable"] = struct{}{}
	failure := terminator.Successors[1]
	if len(categories) == 1 && failure >= 0 && int(failure) < len(function.Blocks) && predecessors[failure] == 1 {
		trap := function.Blocks[failure]
		if len(trap.Operations) == 0 && trap.Terminator.Kind == mir.TerminatorTrap &&
			slices.Equal(trap.Terminator.TrapCategories, categories) {
			return nil
		}
	}
	targets := e.dynamicTrapTargets[functionIndex]
	if targets == nil {
		targets = make(map[mir.BlockID]struct{})
		e.dynamicTrapTargets[functionIndex] = targets
	}
	targets[failure] = struct{}{}
	return nil
}

func (e *analysisBuilder) recordTrapCategories(function string, categories []string) error {
	if len(categories) == 0 {
		return fmt.Errorf("trap in %s has no category", function)
	}
	for _, category := range categories {
		if _, _, ok := cTrapSpec(category); !ok {
			return CapabilityError{Function: function, Feature: "trap category " + strconv.Quote(category)}
		}
		e.trapCategories[category] = struct{}{}
	}
	return nil
}
