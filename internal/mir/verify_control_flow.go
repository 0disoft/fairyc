package mir

import (
	"slices"

	"github.com/0disoft/fairyc/internal/hir"
)

func verifyAssertEdges(function *Function) []VerificationError {
	blocks := make(map[BlockID]*Block, len(function.Blocks))
	for index := range function.Blocks {
		blocks[function.Blocks[index].ID] = &function.Blocks[index]
	}
	var failures []VerificationError
	for index := range function.Blocks {
		block := &function.Blocks[index]
		terminator := block.Terminator
		if terminator.Kind != TerminatorSwitch || terminator.SwitchKind != "assert" {
			continue
		}
		entity := terminatorEntity(block.ID, terminator)
		if len(terminator.Successors) != 2 || terminator.Successors[0] == terminator.Successors[1] {
			failures = append(failures, VerificationError{Code: "V1501", Entity: entity, Detail: "assertion is missing distinct success and failure edges"})
			continue
		}
		trap := blocks[terminator.Successors[1]]
		if trap == nil || trap.Terminator.Kind != TerminatorTrap || !slices.Equal(trap.Terminator.TrapCategories, []string{"assertion-failed"}) {
			failures = append(failures, VerificationError{Code: "V1501", Entity: entity, Detail: "assertion failure edge does not trap as assertion-failed"})
		}
	}
	return failures
}

func verifyTrapCheckEdges(function *Function) []VerificationError {
	blocks := make(map[BlockID]*Block, len(function.Blocks))
	predecessors := make(map[BlockID][]BlockID, len(function.Blocks))
	for index := range function.Blocks {
		block := &function.Blocks[index]
		blocks[block.ID] = block
		for _, successor := range block.Terminator.Successors {
			predecessors[successor] = append(predecessors[successor], block.ID)
		}
	}
	dominators := blockDominators(function, predecessors)
	var failures []VerificationError
	for blockIndex := range function.Blocks {
		block := &function.Blocks[blockIndex]
		for operationIndex, operation := range block.Operations {
			switchKind, markerKind, categories, trapping := trappingOperationSpec(operation)
			if !trapping {
				continue
			}
			entity := operationEntity(block.ID, operationIndex, operation)
			checkedOperator, checkedOperands := trappingCheckIdentity(operation)
			invalidMarker, complete := completeCheckedDominator(function, blocks, dominators, block.ID, switchKind, markerKind, checkedOperator,
				checkedOperands, operation.OwnerType, operation.Type, categories)
			if !complete {
				failureEntity := entity
				detail := "trapping operation is not dominated by its complete checked edge"
				if invalidMarker != "" {
					failureEntity = invalidMarker
					detail = "trapping operation is missing its complete check marker"
				}
				failures = append(failures, VerificationError{Code: "V1501", Entity: failureEntity, Detail: detail})
			}
		}
	}
	return failures
}

func blockDominators(function *Function, predecessors map[BlockID][]BlockID) map[BlockID]map[BlockID]struct{} {
	all := make(map[BlockID]struct{}, len(function.Blocks))
	for _, block := range function.Blocks {
		all[block.ID] = struct{}{}
	}
	dominators := make(map[BlockID]map[BlockID]struct{}, len(function.Blocks))
	for _, block := range function.Blocks {
		if block.ID == function.Entry {
			dominators[block.ID] = map[BlockID]struct{}{block.ID: {}}
		} else {
			dominators[block.ID] = cloneBlockSet(all)
		}
	}
	changed := true
	for changed {
		changed = false
		for _, block := range function.Blocks {
			if block.ID == function.Entry {
				continue
			}
			incoming := predecessors[block.ID]
			next := make(map[BlockID]struct{})
			if len(incoming) > 0 {
				next = cloneBlockSet(dominators[incoming[0]])
				for _, predecessor := range incoming[1:] {
					for candidate := range next {
						if _, found := dominators[predecessor][candidate]; !found {
							delete(next, candidate)
						}
					}
				}
			}
			next[block.ID] = struct{}{}
			if !sameBlockSet(dominators[block.ID], next) {
				dominators[block.ID] = next
				changed = true
			}
		}
	}
	return dominators
}

func completeCheckedDominator(function *Function, blocks map[BlockID]*Block, dominators map[BlockID]map[BlockID]struct{}, target BlockID,
	switchKind string, markerKind OperationKind, operator string, operands []Operand, ownerType, resultType hir.Type, categories []string) (string, bool) {
	invalidMarker := ""
	for index := range function.Blocks {
		candidate := &function.Blocks[index]
		if candidate.ID == target {
			continue
		}
		if _, dominates := dominators[target][candidate.ID]; !dominates {
			continue
		}
		markerEntity, markerValid := checkedMarkerStatus(candidate, markerKind, operator, operands, ownerType, resultType, categories)
		if markerEntity != "" && !markerValid && invalidMarker == "" {
			invalidMarker = markerEntity
		}
		if !markerValid || len(candidate.Terminator.Successors) != 2 {
			continue
		}
		success := candidate.Terminator.Successors[0]
		if _, successDominates := dominators[target][success]; !successDominates {
			continue
		}
		if checkedTerminatorMatches(candidate.Terminator, success, switchKind, operator, operands, ownerType) &&
			checkedTrapSuccessorMatches(blocks, candidate.Terminator, categories) {
			return "", true
		}
	}
	return invalidMarker, false
}

func cloneBlockSet(source map[BlockID]struct{}) map[BlockID]struct{} {
	result := make(map[BlockID]struct{}, len(source))
	for value := range source {
		result[value] = struct{}{}
	}
	return result
}

func sameBlockSet(left, right map[BlockID]struct{}) bool {
	if len(left) != len(right) {
		return false
	}
	for value := range left {
		if _, found := right[value]; !found {
			return false
		}
	}
	return true
}

func trappingOperationSpec(operation Operation) (string, OperationKind, []string, bool) {
	if operation.Kind == OperationArrayIndex {
		return "checked-array-index", OperationArrayIndexCheck, []string{"bounds"}, validArrayIndexShape(operation, true)
	}
	if operation.Kind == OperationArrayIndexStore {
		return "checked-array-index", OperationArrayIndexCheck, []string{"bounds"}, validArrayIndexStoreShape(operation)
	}
	if operation.Kind == OperationViewElementBorrow {
		return "checked-view-index", OperationViewIndexCheck, []string{"bounds"}, ValidViewElementBorrowShape(operation)
	}
	if operation.Kind == OperationViewIndex {
		return "checked-view-index", OperationViewIndexCheck, []string{"bounds"}, validViewIndexShape(operation, true)
	}
	if operation.Kind == OperationViewIndexStore {
		return "checked-view-index", OperationViewIndexCheck, []string{"bounds"}, validViewIndexStoreShape(operation)
	}
	if operation.Kind == OperationViewSlice {
		return "checked-view-slice", OperationViewSliceCheck, []string{"bounds"}, validViewSliceShape(operation, true)
	}
	if operation.Kind == OperationUnary {
		switchKind, categories, valid := checkedArithmeticSpec(operation.Operator, len(operation.Operands))
		return switchKind, OperationArithmeticCheck, categories, valid && signedIntegerType(operation.OwnerType) && operation.Type == operation.OwnerType
	}
	if operation.Kind != OperationBinary {
		return "", OperationInvalid, nil, false
	}
	if operation.Operator == "/" || operation.Operator == "%" {
		categories, valid := divisionTrapCategories(operation.OwnerType)
		return "checked-division", OperationCheckedDivision, categories, valid && validBinaryShape(operation)
	}
	switchKind, categories, valid := checkedArithmeticSpec(operation.Operator, len(operation.Operands))
	return switchKind, OperationArithmeticCheck, categories, valid && validBinaryShape(operation)
}

func trappingCheckIdentity(operation Operation) (string, []Operand) {
	if operation.Kind == OperationArrayIndexStore || operation.Kind == OperationViewIndexStore {
		return "[]", operation.Operands[:2]
	}
	return operation.Operator, operation.Operands
}

func checkedTerminatorMatches(terminator Terminator, continuation BlockID, switchKind, operator string, operands []Operand, ownerType hir.Type) bool {
	if terminator.Kind != TerminatorSwitch || terminator.SwitchKind != switchKind || terminator.Operator != operator ||
		len(terminator.Successors) != 2 || terminator.Successors[0] != continuation || !slices.Equal(terminator.SwitchOperands, operands) {
		return false
	}
	if switchKind == "checked-unary" || switchKind == "checked-arithmetic" || switchKind == "checked-division" || switchKind == "checked-shift" {
		return terminator.SwitchType == ownerType
	}
	return true
}

func checkedMarkerStatus(block *Block, markerKind OperationKind, operator string, operands []Operand, ownerType, resultType hir.Type, categories []string) (string, bool) {
	for index := len(block.Operations) - 1; index >= 0; index-- {
		operation := block.Operations[index]
		if operation.Kind == markerKind && operation.Operator == operator && slices.Equal(operation.Operands, operands) {
			typeMatches := (markerKind != OperationArithmeticCheck && markerKind != OperationCheckedDivision) ||
				operation.OwnerType == ownerType && operation.Type == resultType
			return operationEntity(block.ID, index, operation), typeMatches && slices.Equal(operation.TrapCategories, categories)
		}
	}
	return "", false
}

func checkedTrapSuccessorMatches(blocks map[BlockID]*Block, terminator Terminator, categories []string) bool {
	if len(terminator.Successors) != 2 {
		return false
	}
	trap := blocks[terminator.Successors[1]]
	return trap != nil && trap.Terminator.Kind == TerminatorTrap && slices.Equal(trap.Terminator.TrapCategories, categories)
}
