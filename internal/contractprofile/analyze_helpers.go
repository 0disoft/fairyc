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

func markRecursiveFunctions(nodes []functionAnalysis) {
	indexCounter := 0
	indices := make([]int, len(nodes))
	lowLink := make([]int, len(nodes))
	onStack := make([]bool, len(nodes))
	for index := range indices {
		indices[index] = -1
	}
	var stack []int
	var visit func(int)
	visit = func(nodeIndex int) {
		indices[nodeIndex] = indexCounter
		lowLink[nodeIndex] = indexCounter
		indexCounter++
		stack = append(stack, nodeIndex)
		onStack[nodeIndex] = true

		edges := slices.Clone(nodes[nodeIndex].callTargets)
		slices.Sort(edges)
		edges = slices.Compact(edges)
		for _, target := range edges {
			if indices[target] == -1 {
				visit(target)
				lowLink[nodeIndex] = min(lowLink[nodeIndex], lowLink[target])
			} else if onStack[target] {
				lowLink[nodeIndex] = min(lowLink[nodeIndex], indices[target])
			}
		}
		if lowLink[nodeIndex] != indices[nodeIndex] {
			return
		}
		var component []int
		for len(stack) != 0 {
			last := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			onStack[last] = false
			component = append(component, last)
			if last == nodeIndex {
				break
			}
		}
		recursive := len(component) > 1
		if len(component) == 1 {
			for _, target := range nodes[component[0]].callTargets {
				if target == component[0] {
					recursive = true
					break
				}
			}
		}
		if recursive {
			for _, member := range component {
				nodes[member].recursive = true
			}
		}
	}
	for index := range nodes {
		if indices[index] == -1 {
			visit(index)
		}
	}
}

func controlFlowCycleInfo(function mir.Function) (cycle bool, malformed bool, cycleEdges []CycleEdge) {
	blockIndex := make(map[mir.BlockID]int, len(function.Blocks))
	for index, block := range function.Blocks {
		if _, duplicate := blockIndex[block.ID]; duplicate {
			malformed = true
		}
		blockIndex[block.ID] = index
	}
	if _, found := blockIndex[function.Entry]; !found {
		malformed = true
	}
	adjacency := make([][]int, len(function.Blocks))
	for index, block := range function.Blocks {
		for _, successor := range block.Terminator.Successors {
			if successor == mir.NoBlock {
				continue
			}
			target, found := blockIndex[successor]
			if !found {
				malformed = true
				continue
			}
			adjacency[index] = append(adjacency[index], target)
		}
		slices.Sort(adjacency[index])
		adjacency[index] = slices.Compact(adjacency[index])
	}

	nextIndex := 0
	indices := make([]int, len(function.Blocks))
	lowLink := make([]int, len(function.Blocks))
	onStack := make([]bool, len(function.Blocks))
	component := make([]int, len(function.Blocks))
	for index := range indices {
		indices[index] = -1
		component[index] = -1
	}
	var stack []int
	var components [][]int
	var visit func(int)
	visit = func(block int) {
		indices[block] = nextIndex
		lowLink[block] = nextIndex
		nextIndex++
		stack = append(stack, block)
		onStack[block] = true
		for _, target := range adjacency[block] {
			if indices[target] == -1 {
				visit(target)
				lowLink[block] = min(lowLink[block], lowLink[target])
			} else if onStack[target] {
				lowLink[block] = min(lowLink[block], indices[target])
			}
		}
		if lowLink[block] != indices[block] {
			return
		}
		componentID := len(components)
		var members []int
		for len(stack) != 0 {
			last := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			onStack[last] = false
			component[last] = componentID
			members = append(members, last)
			if last == block {
				break
			}
		}
		components = append(components, members)
	}
	for index := range function.Blocks {
		if indices[index] == -1 {
			visit(index)
		}
	}

	cyclicComponent := make([]bool, len(components))
	for componentID, members := range components {
		if len(members) > 1 {
			cyclicComponent[componentID] = true
			continue
		}
		member := members[0]
		for _, target := range adjacency[member] {
			if target == member {
				cyclicComponent[componentID] = true
				break
			}
		}
	}
	for from, targets := range adjacency {
		for _, to := range targets {
			componentID := component[from]
			if componentID == component[to] && cyclicComponent[componentID] {
				cycleEdges = append(cycleEdges, CycleEdge{
					From: function.Blocks[from].ID,
					To:   function.Blocks[to].ID,
				})
			}
		}
	}
	slices.SortFunc(cycleEdges, func(left, right CycleEdge) int {
		if left.From != right.From {
			return int(left.From - right.From)
		}
		return int(left.To - right.To)
	})
	return len(cycleEdges) != 0, malformed, cycleEdges
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
	return strings.Contains(string(value), "*")
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
		hasUSize = hasUSize || containsTypeToken(string(value), string(hir.TypeUSize))
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
	text := string(value)
	for _, scalar := range deterministicProfileCScalars {
		if containsTypeToken(text, string(scalar)) {
			return true
		}
	}
	return false
}

func operationTypes(operation mir.Operation) []hir.Type {
	result := []hir.Type{operation.Type, operation.OwnerType, operation.IndexType}
	for _, operand := range operation.Operands {
		result = append(result, operand.Type)
	}
	return result
}

func terminatorTypes(terminator mir.Terminator) []hir.Type {
	result := []hir.Type{terminator.ReturnValue.Type, terminator.SwitchType}
	for _, operand := range terminator.Arguments {
		result = append(result, operand.Type)
	}
	for _, operand := range terminator.SwitchOperands {
		result = append(result, operand.Type)
	}
	return result
}

func containsFloatType(value hir.Type) bool {
	text := string(value)
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

func operationContainsRawType(operation mir.Operation) bool {
	if containsRawType(operation.Type) || containsRawType(operation.OwnerType) || containsRawType(operation.IndexType) {
		return true
	}
	for _, operand := range operation.Operands {
		if containsRawType(operand.Type) {
			return true
		}
	}
	return false
}

func operationContainsFloatType(operation mir.Operation) bool {
	if containsFloatType(operation.Type) || containsFloatType(operation.OwnerType) || containsFloatType(operation.IndexType) {
		return true
	}
	for _, operand := range operation.Operands {
		if containsFloatType(operand.Type) {
			return true
		}
	}
	return false
}

func terminatorContainsRawType(terminator mir.Terminator) bool {
	for _, operand := range terminator.Arguments {
		if containsRawType(operand.Type) {
			return true
		}
	}
	return containsRawType(terminator.ReturnValue.Type)
}

func terminatorContainsFloatType(terminator mir.Terminator) bool {
	for _, operand := range terminator.Arguments {
		if containsFloatType(operand.Type) {
			return true
		}
	}
	return containsFloatType(terminator.ReturnValue.Type) || containsFloatType(terminator.SwitchType)
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
