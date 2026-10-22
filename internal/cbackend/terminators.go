package cbackend

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
	"strconv"
	"strings"
)

func (e *emitter) emitTerminator(output *strings.Builder, functionIndex int, blockID mir.BlockID, terminator mir.Terminator) error {
	function := &e.analysis.functions[functionIndex]
	switch terminator.Kind {
	case mir.TerminatorGoto:
		target, err := oneSuccessor(terminator)
		if err != nil {
			return fmt.Errorf("goto in %s: %w", function.Name, err)
		}
		fmt.Fprintf(output, "    goto fy_b_%d;\n", target)
	case mir.TerminatorCall:
		target, err := oneSuccessor(terminator)
		if err != nil {
			return fmt.Errorf("call in %s: %w", function.Name, err)
		}
		calleeResult, calleeParameters, foreign, found := e.analysis.callSignature(terminator.Callee)
		if !found {
			return fmt.Errorf("call in %s names missing function symbol %d", function.Name, terminator.Callee)
		}
		if len(terminator.Arguments) != len(calleeParameters) {
			return fmt.Errorf("call in %s passes %d arguments, want %d", function.Name, len(terminator.Arguments), len(calleeParameters))
		}
		output.WriteString("    ")
		if calleeResult != hir.TypeVoid && !e.analysis.nonReturning[terminator.Callee] {
			fmt.Fprintf(output, "fy_l_%d = ", terminator.DestinationLocal)
		}
		if foreign {
			fmt.Fprintf(output, "%s(", e.analysis.foreignFunctions[e.analysis.foreignBySymbol[terminator.Callee]].LinkName)
		} else {
			fmt.Fprintf(output, "fy_fn_%d(", e.analysis.bySymbol[terminator.Callee])
		}
		for index, argument := range terminator.Arguments {
			if index != 0 {
				output.WriteString(", ")
			}
			value, err := cOperand(argument)
			if err != nil {
				return fmt.Errorf("call argument in %s: %w", function.Name, err)
			}
			output.WriteString(value)
		}
		output.WriteString(");\n")
		if !e.analysis.nonReturning[terminator.Callee] {
			fmt.Fprintf(output, "    goto fy_b_%d;\n", target)
		}
	case mir.TerminatorSwitch:
		failure, known := e.analysis.checkedFailures[function.Symbol][blockID]
		if !known {
			failure, known = constantCheckedFailure(function.Blocks[int(blockID)])
		}
		if known {
			// A direct literal no longer appears in the guard. Keep its static
			// storage referenced for strict C consumers, with no runtime read.
			for _, operand := range terminator.SwitchOperands {
				if operand.Kind == mir.OperandByteString {
					fmt.Fprintf(output, "    (void)%s;\n", byteStringStorageSymbol(operand.StaticData))
				}
			}
			fmt.Fprintf(output, "    %sgoto fy_b_%d;\n", e.pendingTrapAssignment(functionIndex, failure, checkedFailureTrapCategory(terminator), terminator.Span), failure)
			return nil
		}
		return e.emitSwitch(output, functionIndex, terminator)
	case mir.TerminatorReturn:
		if e.analysis.resourceHasNestedFields(function.DropGlueFor) {
			fmt.Fprintf(output, "    return fy_l_%d;\n", function.Parameters[0])
			return nil
		}
		if function.ResultType == hir.TypeVoid {
			output.WriteString("    return;\n")
			return nil
		}
		value, err := cOperand(terminator.ReturnValue)
		if err != nil {
			return fmt.Errorf("return in %s: %w", function.Name, err)
		}
		fmt.Fprintf(output, "    return %s;\n", value)
	case mir.TerminatorTrap:
		if _, dynamic := e.analysis.dynamicTrapTargets[functionIndex][blockID]; dynamic {
			output.WriteString("    if (fy_pending_trap_location.length != 0) fy_runtime_trap_at_v0(fy_pending_trap, fy_pending_trap_location.source, fy_pending_trap_location.length, fy_pending_trap_location.start, fy_pending_trap_location.end);\n")
			output.WriteString("    fy_runtime_trap(fy_pending_trap);\n")
			return nil
		}
		if len(terminator.TrapCategories) == 0 {
			return fmt.Errorf("trap in %s has no category", function.Name)
		}
		name, _, ok := cTrapSpec(terminator.TrapCategories[0])
		if !ok {
			return CapabilityError{Function: function.Name, Feature: "trap category " + strconv.Quote(terminator.TrapCategories[0])}
		}
		fmt.Fprintf(output, "    %s;\n", e.trapCall(name, terminator.Span))
	case mir.TerminatorUnreachable:
		fmt.Fprintf(output, "    %s;\n", e.trapCall("FY_TRAP_UNREACHABLE", terminator.Span))
	default:
		return CapabilityError{Function: function.Name, Feature: "MIR terminator " + strconv.Quote(string(terminator.Kind))}
	}
	return nil
}

func (e *emitter) emitSwitch(output *strings.Builder, functionIndex int, terminator mir.Terminator) error {
	function := &e.analysis.functions[functionIndex]
	if selected, constant := constantBooleanSuccessor(terminator); constant {
		if terminator.SwitchKind == "assert" && terminator.SwitchOperands[0].I32 == 0 {
			output.WriteString("    " + e.pendingTrapAssignment(functionIndex, selected, "FY_TRAP_ASSERTION_FAILED", terminator.Span))
		}
		fmt.Fprintf(output, "    goto fy_b_%d;\n", selected)
		return nil
	}
	if terminator.SwitchKind == "enum" {
		enumIndex, found := e.analysis.enumByType[terminator.SwitchType]
		if !found {
			return CapabilityError{Function: function.Name, Feature: "enum switch type " + strconv.Quote(terminator.SwitchType.String())}
		}
		if len(terminator.Successors) != len(terminator.SwitchVariants) {
			return fmt.Errorf("enum switch in %s has %d successors for %d variants", function.Name, len(terminator.Successors), len(terminator.SwitchVariants))
		}
		operand, err := oneOperand(terminator.SwitchOperands)
		if err != nil {
			return fmt.Errorf("enum switch in %s: %w", function.Name, err)
		}
		value, err := cOperand(operand)
		if err != nil {
			return fmt.Errorf("enum switch in %s: %w", function.Name, err)
		}
		enumeration := &e.analysis.enumTypes[enumIndex]
		for branchIndex, variant := range terminator.SwitchVariants {
			variantIndex, found := enumeration.variantIndex[variant]
			if !found {
				return fmt.Errorf("enum switch in %s names missing variant %q", function.Name, variant)
			}
			if enumeration.definition.CBase != "" {
				fmt.Fprintf(output, "    if ((%s) == (fy_enum_%d)UINT64_C(%d)) goto fy_b_%d;\n", value, enumIndex, variantIndex, terminator.Successors[branchIndex])
			} else {
				fmt.Fprintf(output, "    if ((%s).tag == UINT32_C(%d)) goto fy_b_%d;\n", value, variantIndex, terminator.Successors[branchIndex])
			}
		}
		fmt.Fprintf(output, "    %s;\n", e.trapCall("FY_TRAP_UNREACHABLE", terminator.Span))
		return nil
	}
	if len(terminator.Successors) != 2 {
		return fmt.Errorf("switch in %s has %d successors", function.Name, len(terminator.Successors))
	}
	success, failure := terminator.Successors[0], terminator.Successors[1]
	switch terminator.SwitchKind {
	case "bool":
		operand, err := oneOperand(terminator.SwitchOperands)
		if err != nil {
			return fmt.Errorf("bool switch in %s: %w", function.Name, err)
		}
		value, err := cOperand(operand)
		if err != nil {
			return fmt.Errorf("bool switch in %s: %w", function.Name, err)
		}
		fmt.Fprintf(output, "    if (%s != INT32_C(0)) goto fy_b_%d;\n", value, success)
		fmt.Fprintf(output, "    goto fy_b_%d;\n", failure)
	case "assert":
		operand, err := oneOperand(terminator.SwitchOperands)
		if err != nil {
			return fmt.Errorf("assert in %s: %w", function.Name, err)
		}
		value, err := cOperand(operand)
		if err != nil {
			return fmt.Errorf("assert condition in %s: %w", function.Name, err)
		}
		fmt.Fprintf(output, "    if (%s == INT32_C(0)) { %sgoto fy_b_%d; }\n", value, e.pendingTrapAssignment(functionIndex, failure, "FY_TRAP_ASSERTION_FAILED", terminator.Span), failure)
		fmt.Fprintf(output, "    goto fy_b_%d;\n", success)
	case "option":
		operand, err := oneOperand(terminator.SwitchOperands)
		if err != nil {
			return fmt.Errorf("Option switch in %s: %w", function.Name, err)
		}
		value, err := cOperand(operand)
		if err != nil {
			return fmt.Errorf("Option switch in %s: %w", function.Name, err)
		}
		valueType, err := e.analysis.operandType(functionIndex, operand)
		if err != nil {
			return fmt.Errorf("Option switch in %s: %w", function.Name, err)
		}
		if sharedBorrowOptionNiche(valueType, e.analysis.structByType, e.analysis.resourceByType, e.analysis.arrayByType) {
			fmt.Fprintf(output, "    if ((%s).p_0 != ((void *)0)) goto fy_b_%d;\n", value, success)
			fmt.Fprintf(output, "    goto fy_b_%d;\n", failure)
			break
		}
		fmt.Fprintf(output, "    if ((%s).tag == FY_OPTION_SOME) goto fy_b_%d;\n", value, success)
		fmt.Fprintf(output, "    if ((%s).tag == FY_OPTION_NONE) goto fy_b_%d;\n", value, failure)
		fmt.Fprintf(output, "    %s;\n", e.trapCall("FY_TRAP_UNREACHABLE", terminator.Span))
	case "result":
		operand, err := oneOperand(terminator.SwitchOperands)
		if err != nil {
			return fmt.Errorf("Result switch in %s: %w", function.Name, err)
		}
		value, err := cOperand(operand)
		if err != nil {
			return fmt.Errorf("Result switch in %s: %w", function.Name, err)
		}
		fmt.Fprintf(output, "    if ((%s).tag == FY_RESULT_OK) goto fy_b_%d;\n", value, success)
		fmt.Fprintf(output, "    if ((%s).tag == FY_RESULT_ERR) goto fy_b_%d;\n", value, failure)
		fmt.Fprintf(output, "    %s;\n", e.trapCall("FY_TRAP_UNREACHABLE", terminator.Span))
	case "checked-unary":
		operand, err := oneOperand(terminator.SwitchOperands)
		if err != nil {
			return fmt.Errorf("checked unary in %s: %w", function.Name, err)
		}
		if terminator.Operator != "-" {
			return CapabilityError{Function: function.Name, Feature: "checked unary operator " + strconv.Quote(terminator.Operator)}
		}
		value, err := cOperand(operand)
		if err != nil {
			return fmt.Errorf("checked unary in %s: %w", function.Name, err)
		}
		minimum := "INT32_MIN"
		if terminator.SwitchType == hir.TypeI64 {
			minimum = "INT64_MIN"
		}
		fmt.Fprintf(output, "    if (%s == %s) { %sgoto fy_b_%d; }\n", value, minimum, e.pendingTrapAssignment(functionIndex, failure, "FY_TRAP_INTEGER_OVERFLOW", terminator.Span), failure)
		fmt.Fprintf(output, "    goto fy_b_%d;\n", success)
	case "checked-arithmetic":
		left, right, err := twoOperands(terminator.SwitchOperands)
		if err != nil {
			return fmt.Errorf("checked arithmetic in %s: %w", function.Name, err)
		}
		if terminator.Operator != "+" && terminator.Operator != "-" && terminator.Operator != "*" {
			return CapabilityError{Function: function.Name, Feature: "checked arithmetic operator " + strconv.Quote(terminator.Operator)}
		}
		leftValue, err := cOperand(left)
		if err != nil {
			return fmt.Errorf("checked arithmetic in %s: %w", function.Name, err)
		}
		rightValue, err := cOperand(right)
		if err != nil {
			return fmt.Errorf("checked arithmetic in %s: %w", function.Name, err)
		}
		if terminator.SwitchType == hir.TypeI64 {
			check := map[string]string{"+": "add", "-": "sub", "*": "mul"}[terminator.Operator]
			fmt.Fprintf(output, "    if (fy_i64_%s_overflow(%s, %s)) { %sgoto fy_b_%d; }\n", check, leftValue, rightValue, e.pendingTrapAssignment(functionIndex, failure, "FY_TRAP_INTEGER_OVERFLOW", terminator.Span), failure)
		} else if terminator.SwitchType == hir.TypeI32 {
			expression := fmt.Sprintf("((int64_t)(%s) %s (int64_t)(%s))", leftValue, terminator.Operator, rightValue)
			fmt.Fprintf(output, "    if (%s < (int64_t)INT32_MIN || %s > (int64_t)INT32_MAX) { %sgoto fy_b_%d; }\n", expression, expression, e.pendingTrapAssignment(functionIndex, failure, "FY_TRAP_INTEGER_OVERFLOW", terminator.Span), failure)
		} else {
			maximum, err := unsignedCMaximum(terminator.SwitchType)
			if err != nil {
				return fmt.Errorf("checked arithmetic in %s: %w", function.Name, err)
			}
			switch terminator.Operator {
			case "+":
				fmt.Fprintf(output, "    if ((%s) > %s - (%s)) { %sgoto fy_b_%d; }\n", leftValue, maximum, rightValue, e.pendingTrapAssignment(functionIndex, failure, "FY_TRAP_INTEGER_OVERFLOW", terminator.Span), failure)
			case "-":
				fmt.Fprintf(output, "    if ((%s) < (%s)) { %sgoto fy_b_%d; }\n", leftValue, rightValue, e.pendingTrapAssignment(functionIndex, failure, "FY_TRAP_INTEGER_OVERFLOW", terminator.Span), failure)
			case "*":
				fmt.Fprintf(output, "    if ((%s) != 0 && (%s) > %s / (%s)) { %sgoto fy_b_%d; }\n", rightValue, leftValue, maximum, rightValue, e.pendingTrapAssignment(functionIndex, failure, "FY_TRAP_INTEGER_OVERFLOW", terminator.Span), failure)
			}
		}
		fmt.Fprintf(output, "    goto fy_b_%d;\n", success)
	case "checked-division":
		left, right, err := twoOperands(terminator.SwitchOperands)
		if err != nil {
			return fmt.Errorf("checked division in %s: %w", function.Name, err)
		}
		leftValue, err := cOperand(left)
		if err != nil {
			return fmt.Errorf("checked division in %s: %w", function.Name, err)
		}
		rightValue, err := cOperand(right)
		if err != nil {
			return fmt.Errorf("checked division in %s: %w", function.Name, err)
		}
		fmt.Fprintf(output, "    if (%s == 0) { %sgoto fy_b_%d; }\n", rightValue, e.pendingTrapAssignment(functionIndex, failure, "FY_TRAP_DIVISION_BY_ZERO", terminator.Span), failure)
		if terminator.SwitchType == hir.TypeI64 {
			fmt.Fprintf(output, "    if (%s == INT64_MIN && %s == -INT64_C(1)) { %sgoto fy_b_%d; }\n", leftValue, rightValue, e.pendingTrapAssignment(functionIndex, failure, "FY_TRAP_SIGNED_DIVISION_OVERFLOW", terminator.Span), failure)
		}
		if terminator.SwitchType == hir.TypeI32 {
			fmt.Fprintf(output, "    if (%s == INT32_MIN && %s == -INT32_C(1)) { %sgoto fy_b_%d; }\n", leftValue, rightValue, e.pendingTrapAssignment(functionIndex, failure, "FY_TRAP_SIGNED_DIVISION_OVERFLOW", terminator.Span), failure)
		}
		fmt.Fprintf(output, "    goto fy_b_%d;\n", success)
	case "checked-view-index":
		receiver, index, err := twoOperands(terminator.SwitchOperands)
		if err != nil {
			return fmt.Errorf("checked byte-view index in %s: %w", function.Name, err)
		}
		receiverValue, err := cOperand(receiver)
		if err != nil {
			return fmt.Errorf("checked byte-view index receiver in %s: %w", function.Name, err)
		}
		indexValue, err := cOperand(index)
		if err != nil {
			return fmt.Errorf("checked byte-view index offset in %s: %w", function.Name, err)
		}
		fmt.Fprintf(output, "    if (%s >= (%s).length) { %sgoto fy_b_%d; }\n", indexValue, receiverValue, e.pendingTrapAssignment(functionIndex, failure, "FY_TRAP_BOUNDS", terminator.Span), failure)
		fmt.Fprintf(output, "    goto fy_b_%d;\n", success)
	case "checked-array-index":
		receiver, index, err := twoOperands(terminator.SwitchOperands)
		if err != nil {
			return fmt.Errorf("checked fixed-array index in %s: %w", function.Name, err)
		}
		receiverType, err := e.analysis.operandType(functionIndex, receiver)
		if err != nil {
			return fmt.Errorf("checked fixed-array index receiver in %s: %w", function.Name, err)
		}
		_, length, array := hir.FixedArrayParts(receiverType)
		if !array {
			return fmt.Errorf("checked fixed-array index receiver in %s has type %q", function.Name, receiverType)
		}
		indexValue, err := cOperand(index)
		if err != nil {
			return fmt.Errorf("checked fixed-array index offset in %s: %w", function.Name, err)
		}
		fmt.Fprintf(output, "    if (%s >= ((uintptr_t)UINT64_C(%d))) { %sgoto fy_b_%d; }\n", indexValue, length, e.pendingTrapAssignment(functionIndex, failure, "FY_TRAP_BOUNDS", terminator.Span), failure)
		fmt.Fprintf(output, "    goto fy_b_%d;\n", success)
	case "checked-view-slice":
		receiver, start, end, err := threeOperands(terminator.SwitchOperands)
		if err != nil {
			return fmt.Errorf("checked byte-view slice in %s: %w", function.Name, err)
		}
		receiverValue, err := cOperand(receiver)
		if err != nil {
			return fmt.Errorf("checked byte-view slice receiver in %s: %w", function.Name, err)
		}
		startValue, err := cOperand(start)
		if err != nil {
			return fmt.Errorf("checked byte-view slice start in %s: %w", function.Name, err)
		}
		endValue, err := cOperand(end)
		if err != nil {
			return fmt.Errorf("checked byte-view slice end in %s: %w", function.Name, err)
		}
		if end.Kind == mir.OperandUSize && end.U64 == 0 {
			// The length is unsigned, so only a nonzero start can fail when
			// the end is zero. Avoid generating 0 > length under -Wtype-limits.
			fmt.Fprintf(output, "    if ((%s) != ((uintptr_t)UINT64_C(0))) { %sgoto fy_b_%d; }\n", startValue, e.pendingTrapAssignment(functionIndex, failure, "FY_TRAP_BOUNDS", terminator.Span), failure)
		} else if start.Kind == mir.OperandUSize && start.U64 == 0 {
			fmt.Fprintf(output, "    if ((%s) > (%s).length) { %sgoto fy_b_%d; }\n", endValue, receiverValue, e.pendingTrapAssignment(functionIndex, failure, "FY_TRAP_BOUNDS", terminator.Span), failure)
		} else {
			fmt.Fprintf(output, "    if ((%s) > (%s) || (%s) > (%s).length) { %sgoto fy_b_%d; }\n", startValue, endValue, endValue, receiverValue, e.pendingTrapAssignment(functionIndex, failure, "FY_TRAP_BOUNDS", terminator.Span), failure)
		}
		fmt.Fprintf(output, "    goto fy_b_%d;\n", success)
	case "checked-shift":
		_, amount, err := twoOperands(terminator.SwitchOperands)
		if err != nil {
			return fmt.Errorf("checked shift in %s: %w", function.Name, err)
		}
		amountValue, err := cOperand(amount)
		if err != nil {
			return fmt.Errorf("checked shift in %s: %w", function.Name, err)
		}
		width, err := shiftCWidth(terminator.SwitchType)
		if err != nil {
			return fmt.Errorf("checked shift in %s: %w", function.Name, err)
		}
		if terminator.SwitchType == hir.TypeI32 || terminator.SwitchType == hir.TypeI64 {
			fmt.Fprintf(output, "    if (%s < INT32_C(0) || %s >= %s) { %sgoto fy_b_%d; }\n", amountValue, amountValue, width, e.pendingTrapAssignment(functionIndex, failure, "FY_TRAP_INVALID_SHIFT", terminator.Span), failure)
		} else {
			fmt.Fprintf(output, "    if (%s >= %s) { %sgoto fy_b_%d; }\n", amountValue, width, e.pendingTrapAssignment(functionIndex, failure, "FY_TRAP_INVALID_SHIFT", terminator.Span), failure)
		}
		amountValue = reachableShiftCount(terminator.SwitchType, amount, amountValue)
		if terminator.Operator == "<<" {
			value, _, err := twoOperands(terminator.SwitchOperands)
			if err != nil {
				return fmt.Errorf("checked left shift in %s: %w", function.Name, err)
			}
			valueText, err := cOperand(value)
			if err != nil {
				return fmt.Errorf("checked left shift in %s: %w", function.Name, err)
			}
			if terminator.SwitchType == hir.TypeI64 {
				fmt.Fprintf(output, "    if ((%s) > (INT64_MAX >> (uint64_t)(%s)) || (%s) < fy_sar_i64(INT64_MIN, %s)) { %sgoto fy_b_%d; }\n", valueText, amountValue, valueText, amountValue, e.pendingTrapAssignment(functionIndex, failure, "FY_TRAP_INTEGER_OVERFLOW", terminator.Span), failure)
			} else if terminator.SwitchType == hir.TypeI32 {
				expression := fmt.Sprintf("((int64_t)(%s) * (INT64_C(1) << (uint32_t)(%s)))", valueText, amountValue)
				fmt.Fprintf(output, "    if (%s < (int64_t)INT32_MIN || %s > (int64_t)INT32_MAX) { %sgoto fy_b_%d; }\n", expression, expression, e.pendingTrapAssignment(functionIndex, failure, "FY_TRAP_INTEGER_OVERFLOW", terminator.Span), failure)
			} else {
				maximum, err := unsignedCMaximum(terminator.SwitchType)
				if err != nil {
					return fmt.Errorf("checked left shift in %s: %w", function.Name, err)
				}
				fmt.Fprintf(output, "    if ((%s) > (%s >> (%s))) { %sgoto fy_b_%d; }\n", valueText, maximum, amountValue, e.pendingTrapAssignment(functionIndex, failure, "FY_TRAP_INTEGER_OVERFLOW", terminator.Span), failure)
			}
		}
		fmt.Fprintf(output, "    goto fy_b_%d;\n", success)
	default:
		return CapabilityError{Function: function.Name, Feature: "MIR switch " + strconv.Quote(terminator.SwitchKind)}
	}
	return nil
}

func (e *emitter) pendingTrapAssignment(functionIndex int, failure mir.BlockID, category string, span hir.Span) string {
	if _, dynamic := e.analysis.dynamicTrapTargets[functionIndex][failure]; !dynamic {
		return ""
	}
	arguments, valid := e.trapLocationArguments(span)
	if !valid {
		return "fy_pending_trap = " + category + "; fy_pending_trap_location.length = 0; "
	}
	return "fy_pending_trap = " + category + "; fy_pending_trap_location = (struct fy_runtime_location_v0){" + arguments + "}; "
}
