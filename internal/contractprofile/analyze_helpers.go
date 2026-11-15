package contractprofile

import (
	"slices"
	"strings"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

func computeTransitiveCost(index int, nodes []functionAnalysis, loopBudgets map[string]uint64, state []uint8) {
	if state[index] == 2 {
		return
	}
	node := &nodes[index]
	if state[index] == 1 || node.recursive {
		node.transitiveBounded = false
		node.transitiveCost = 0
		node.transitiveOverflow = state[index] == 1
		return
	}
	state[index] = 1
	cost := node.localCost
	overflow := node.localCostOverflow
	bounded := !node.cfgMalformed && !node.unknownInternalCall && !node.externalCall
	for _, target := range node.callTargets {
		computeTransitiveCost(target, nodes, loopBudgets, state)
		callee := &nodes[target]
		if !callee.transitiveBounded || callee.transitiveOverflow {
			bounded = false
		}
		var addOverflow bool
		cost, addOverflow = saturatingAdd(cost, callee.transitiveCost, false)
		overflow = overflow || addOverflow
	}
	if node.cfgCycle {
		budget := loopBudgets[node.function.Name]
		if budget == 0 || budget > MaxLoopBudget {
			bounded = false
		} else {
			var multiplyOverflow bool
			cost, multiplyOverflow = saturatingMultiply(cost, budget+1)
			overflow = overflow || multiplyOverflow
		}
	}
	if node.recursive {
		bounded = false
	}
	node.transitiveCost = cost
	node.transitiveBounded = bounded
	node.transitiveOverflow = overflow
	state[index] = 2
}

func operationCost(operation mir.Operation) (uint64, bool) {
	weight := uint64(1)
	switch operation.Kind {
	case mir.OperationCheckedDivision,
		mir.OperationArithmeticCheck,
		mir.OperationArrayIndexCheck,
		mir.OperationViewIndexCheck,
		mir.OperationViewSliceCheck:
		weight = 2
	case mir.OperationDrop:
		weight = 3
	case mir.OperationArrayConstruct,
		mir.OperationStructConstruct,
		mir.OperationResourceConstruct,
		mir.OperationEnumConstruct:
		weight = 2
		var overflow bool
		weight, overflow = saturatingAdd(weight, uint64(len(operation.Operands)), false)
		weight, nextOverflow := saturatingAdd(weight, uint64(len(operation.Fields)), false)
		return weight, overflow || nextOverflow
	case mir.OperationRegionEnter,
		mir.OperationRegionPlace,
		mir.OperationRegionReset:
		weight = 4
	}
	return weight, false
}

func terminatorCost(terminator mir.Terminator) (uint64, bool) {
	weight := uint64(1)
	switch terminator.Kind {
	case mir.TerminatorCall:
		weight = 5
	case mir.TerminatorSwitch:
		weight = 2
		var overflow bool
		weight, overflow = saturatingAdd(weight, uint64(len(terminator.Successors)), false)
		return weight, overflow
	}
	return weight, false
}

func saturatingAdd(left, right uint64, existingOverflow bool) (uint64, bool) {
	maximum := ^uint64(0)
	if maximum-left < right {
		return maximum, true
	}
	return left + right, existingOverflow
}

func saturatingMultiply(left, right uint64) (uint64, bool) {
	maximum := ^uint64(0)
	if left != 0 && right > maximum/left {
		return maximum, true
	}
	return left * right, false
}

func containsRawType(value hir.Type) bool {
	_, _, ok := hir.RawPointerElement(value)
	if ok {
		return true
	}
	return strings.Contains(value.String(), "*")
}

func anyRawType(values []hir.Type) bool {
	for _, value := range values {
		if containsRawType(value) {
			return true
		}
	}
	return false
}

func appendTargetTypeViolations(report *Report, owner string, span hir.Span, context string, values ...hir.Type) {
	if report == nil {
		return
	}
	hasCScalar := false
	hasUSize := false
	for _, value := range values {
		hasCScalar = hasCScalar || containsCScalarType(value)
		hasUSize = hasUSize || containsTypeToken(value.String(), hir.TypeUSize.String())
	}
	if hasCScalar {
		report.Violations = append(report.Violations, violationAt(
			"CP1014",
			owner,
			span,
			context+" contains a target-dependent c::* scalar",
		))
	}
	if hasUSize {
		report.Violations = append(report.Violations, violationAt(
			"CP1015",
			owner,
			span,
			context+" contains target-width usize",
		))
	}
}

func containsCScalarType(value hir.Type) bool {
	text := value.String()
	for _, scalar := range deterministicProfileCScalars {
		if containsTypeToken(text, scalar.String()) {
			return true
		}
	}
	return false
}

func containsFloatType(value hir.Type) bool {
	text := value.String()
	return containsTypeToken(text, "f32") ||
		containsTypeToken(text, "f64") ||
		containsTypeToken(text, "c::float") ||
		containsTypeToken(text, "c::double")
}

func containsTypeToken(text, token string) bool {
	for offset := 0; offset <= len(text)-len(token); {
		index := strings.Index(text[offset:], token)
		if index < 0 {
			return false
		}
		index += offset
		beforeOK := index == 0 || !typeTokenByte(text[index-1])
		after := index + len(token)
		afterOK := after == len(text) || !typeTokenByte(text[after])
		if beforeOK && afterOK {
			return true
		}
		offset = index + 1
	}
	return false
}

func typeTokenByte(value byte) bool {
	return value >= 'a' && value <= 'z' ||
		value >= 'A' && value <= 'Z' ||
		value >= '0' && value <= '9' ||
		value == '_' || value == ':'
}

func anyFloatType(values []hir.Type) bool {
	for _, value := range values {
		if containsFloatType(value) {
			return true
		}
	}
	return false
}

func rawPointerOperation(operation mir.Operation) bool {
	switch operation.Kind {
	case mir.OperationRawAddress,
		mir.OperationRawBorrowPointer,
		mir.OperationStrFromRawParts,
		mir.OperationCStrFromRawParts,
		mir.OperationByteSliceFromRawParts,
		mir.OperationMutByteSliceFromRawParts,
		mir.OperationCStrDataAddress,
		mir.OperationByteSliceDataAddress,
		mir.OperationStrDataAddress:
		return true
	default:
		return false
	}
}

func regionAllocationOperation(kind mir.OperationKind) bool {
	return kind == mir.OperationRegionEnter || kind == mir.OperationRegionPlace || kind == mir.OperationRegionReset
}

func joinEffects(values []hir.Effect) string {
	text := make([]string, len(values))
	for index, value := range values {
		text[index] = string(value)
	}
	slices.Sort(text)
	return strings.Join(text, ", ")
}

func violationAt(code, function string, span hir.Span, message string) Violation {
	return Violation{
		Code:     code,
		Function: function,
		Location: locationFromHIRSpan(span),
		Message:  message,
	}
}
