package mir

import (
	"cmp"
	"fmt"
	"slices"

	"github.com/0disoft/fairyc/internal/hir"
)

// VerificationError identifies one internal MIR invariant failure.
type VerificationError struct {
	Code   string
	Entity string
	Detail string
}

func (e VerificationError) Error() string {
	return fmt.Sprintf("%s at %s: %s", e.Code, e.Entity, e.Detail)
}

type flagState struct {
	initialized bool
	active      bool
}

type verificationState struct {
	live  map[hir.SymbolID]bool
	flags map[string]flagState
}

// Verify checks the executable invariants required before MIR handoff.
func Verify(program Program) []VerificationError {
	var failures []VerificationError
	for index := range program.Functions {
		failures = append(failures, verifyFunction(&program.Functions[index])...)
	}
	slices.SortFunc(failures, func(left, right VerificationError) int {
		if value := cmp.Compare(left.Code, right.Code); value != 0 {
			return value
		}
		return cmp.Compare(left.Entity, right.Entity)
	})
	return slices.CompactFunc(failures, func(left, right VerificationError) bool {
		return left.Code == right.Code && left.Entity == right.Entity
	})
}

func verifyFunction(function *Function) []VerificationError {
	if int(function.Entry) < 0 || int(function.Entry) >= len(function.Blocks) {
		return []VerificationError{{Code: "V1001", Entity: function.Name, Detail: "entry block is missing"}}
	}
	resources := make(map[hir.SymbolID]ResourceLocal, len(function.Resources))
	initial := verificationState{
		live:  make(map[hir.SymbolID]bool, len(function.Resources)),
		flags: make(map[string]flagState, len(function.DropFlags)),
	}
	for _, resource := range function.Resources {
		resources[resource.Symbol] = resource
		initial.live[resource.Symbol] = resource.Parameter
	}
	for _, parameter := range function.Parameters {
		initial.live[hir.SymbolID(parameter)] = true
	}
	flags := make(map[string]DropFlag, len(function.DropFlags))
	for _, flag := range function.DropFlags {
		flags[flag.Entity] = flag
		initial.flags[flag.Entity] = flagState{}
	}
	type pendingBlock struct {
		id    BlockID
		state verificationState
	}
	queue := []pendingBlock{{id: function.Entry, state: initial}}
	seen := make(map[BlockID]verificationState)
	var failures []VerificationError
	for len(queue) > 0 {
		item := queue[0]
		queue = queue[1:]
		if previous, found := seen[item.id]; found {
			merged, changed, incompatible := mergeVerificationState(previous, item.state, resources)
			if incompatible {
				failures = append(failures, VerificationError{Code: "V1002", Entity: blockEntityFor(function, item.id), Detail: "incoming initialization states disagree"})
				continue
			}
			if !changed {
				continue
			}
			item.state = merged
		}
		seen[item.id] = cloneVerificationState(item.state)
		if int(item.id) < 0 || int(item.id) >= len(function.Blocks) {
			failures = append(failures, VerificationError{Code: "V1001", Entity: blockEntityFor(function, item.id), Detail: "successor block is missing"})
			continue
		}
		block := function.Blocks[item.id]
		state := cloneVerificationState(item.state)
		for operationIndex, operation := range block.Operations {
			entity := operationEntity(item.id, operationIndex, operation)
			if operation.SourceForm != "" {
				failures = append(failures, VerificationError{Code: "V1601", Entity: entity, Detail: "source-only form remains: " + operation.SourceForm})
			}
			if operation.MovePayload && operation.Kind != OperationResultPayload && operation.Kind != OperationOptionPayload {
				failures = append(failures, VerificationError{Code: "V1001", Entity: entity, Detail: "payload move is attached to a non-payload operation"})
			}
			switch operation.Kind {
			case OperationInitialize:
				failures = append(failures, verifyOperands(state, operation.Operands, entity)...)
				if len(operation.Operands) > 1 || len(operation.Operands) == 1 && operation.DestinationLocal != LocalID(operation.Symbol) {
					failures = append(failures, VerificationError{Code: "V1001", Entity: entity, Detail: "resource initialization shape is invalid"})
				}
				if state.live[operation.Symbol] {
					failures = append(failures, VerificationError{Code: "V1002", Entity: entity, Detail: "resource initialized twice"})
				}
				state.live[operation.Symbol] = true
			case OperationUse:
				if !state.live[operation.Symbol] {
					failures = append(failures, VerificationError{Code: "V1002", Entity: entity, Detail: "value used before initialization"})
				}
			case OperationMove:
				if operation.ProjectionEntity != "" {
					failures = append(failures, VerificationError{Code: "V1103", Entity: operation.ProjectionEntity, Detail: "resource move names a projected place"})
				}
				if resource, found := resources[operation.Symbol]; found && !state.live[operation.Symbol] {
					failures = append(failures, VerificationError{Code: "V1101", Entity: resourceEntity(resource, entity), Detail: "resource moved after discharge"})
				}
				state.live[operation.Symbol] = false
			case OperationDropFlagInit:
				flag := state.flags[operation.DropFlagEntity]
				flag.initialized = true
				flag.active = true
				state.flags[operation.DropFlagEntity] = flag
			case OperationDropFlagClear:
				flag := state.flags[operation.DropFlagEntity]
				if _, declared := flags[operation.DropFlagEntity]; !declared || !flag.initialized {
					failures = append(failures, VerificationError{Code: "V1201", Entity: operation.DropFlagEntity, Detail: "drop flag used before initialization"})
				}
				flag.active = false
				state.flags[operation.DropFlagEntity] = flag
			case OperationDrop:
				resource, found := resources[operation.Symbol]
				if found && !state.live[operation.Symbol] {
					failures = append(failures, VerificationError{Code: "V1101", Entity: resourceEntity(resource, entity), Detail: "resource dropped after discharge"})
				}
				if operation.DropFlagEntity != "" {
					if state.flags[operation.DropFlagEntity].active {
						failures = append(failures, VerificationError{Code: "V1202", Entity: entity, Detail: "drop runs before its active flag is cleared"})
					}
				} else if !operation.FlagClearedFirst {
					failures = append(failures, VerificationError{Code: "V1202", Entity: entity, Detail: "drop does not clear its active flag first"})
				}
				state.live[operation.Symbol] = false
			case OperationCheckedDivision:
				failures = append(failures, verifyOperands(state, operation.Operands, entity)...)
				if !slices.Equal(operation.TrapCategories, []string{"division-by-zero", "signed-division-overflow"}) {
					failures = append(failures, VerificationError{Code: "V1501", Entity: entity, Detail: "division is missing required trap checks"})
				}
			case OperationAssign, OperationUnary, OperationBinary:
				failures = append(failures, verifyOperands(state, operation.Operands, entity)...)
				state.live[hir.SymbolID(operation.DestinationLocal)] = true
			case OperationResultConstruct, OperationResultPayload:
				failures = append(failures, verifyOperands(state, operation.Operands, entity)...)
				if len(operation.Operands) != 1 || operation.DestinationLocal == NoLocal {
					failures = append(failures, VerificationError{Code: "V1001", Entity: entity, Detail: "Result operation must have one operand and one destination"})
				}
				if operation.Variant != "ok" && operation.Variant != "err" {
					failures = append(failures, VerificationError{Code: "V1001", Entity: entity, Detail: "Result operation has an invalid variant"})
				}
				state.live[hir.SymbolID(operation.DestinationLocal)] = true
			case OperationOptionConstruct:
				failures = append(failures, verifyOperands(state, operation.Operands, entity)...)
				validShape := operation.DestinationLocal != NoLocal &&
					(operation.Variant == "some" && len(operation.Operands) == 1 || operation.Variant == "none" && len(operation.Operands) == 0)
				if !validShape {
					failures = append(failures, VerificationError{Code: "V1001", Entity: entity, Detail: "Option construction shape does not match its variant"})
				}
				state.live[hir.SymbolID(operation.DestinationLocal)] = true
			case OperationOptionPayload:
				failures = append(failures, verifyOperands(state, operation.Operands, entity)...)
				if operation.DestinationLocal == NoLocal || operation.Variant != "some" || len(operation.Operands) != 1 {
					failures = append(failures, VerificationError{Code: "V1001", Entity: entity, Detail: "Option payload extraction must target Some"})
				}
				state.live[hir.SymbolID(operation.DestinationLocal)] = true
			case OperationResourceConstruct:
				failures = append(failures, verifyOperands(state, operation.Operands, entity)...)
				if operation.DestinationLocal == NoLocal || operation.Type == hir.TypeInvalid || len(operation.Fields) == 0 || len(operation.Fields) != len(operation.Operands) || !uniqueNonEmpty(operation.Fields) {
					failures = append(failures, VerificationError{Code: "V1001", Entity: entity, Detail: "resource construction shape is invalid"})
				}
				state.live[hir.SymbolID(operation.DestinationLocal)] = true
			case OperationFieldRead:
				failures = append(failures, verifyOperands(state, operation.Operands, entity)...)
				if operation.DestinationLocal == NoLocal || operation.Field == "" || len(operation.Operands) != 1 {
					failures = append(failures, VerificationError{Code: "V1001", Entity: entity, Detail: "field read shape is invalid"})
				}
				state.live[hir.SymbolID(operation.DestinationLocal)] = true
			case OperationResultSwitch:
				failures = append(failures, verifyOperands(state, operation.Operands, entity)...)
				if len(operation.Operands) != 1 {
					failures = append(failures, VerificationError{Code: "V1001", Entity: entity, Detail: "Result switch operation must have one operand"})
				}
			}
		}

		terminator := block.Terminator
		if terminator.Kind == "" {
			failures = append(failures, VerificationError{Code: "V1001", Entity: blockEntityFor(function, item.id), Detail: "reachable block has no terminator"})
			continue
		}
		if terminator.Kind == TerminatorCall && terminator.UnwindSuccessor != NoBlock {
			failures = append(failures, VerificationError{Code: "V1502", Entity: terminatorEntity(item.id, terminator), Detail: "Stage 0 call has an unwind successor"})
		}
		if terminator.Kind == TerminatorCall {
			failures = append(failures, verifyOperands(state, terminator.Arguments, terminatorEntity(item.id, terminator))...)
			state.live[hir.SymbolID(terminator.DestinationLocal)] = true
		}
		if terminator.Kind == TerminatorSwitch {
			failures = append(failures, verifyOperands(state, terminator.SwitchOperands, terminatorEntity(item.id, terminator))...)
			if !validSwitchShape(terminator) {
				failures = append(failures, VerificationError{Code: "V1001", Entity: terminatorEntity(item.id, terminator), Detail: "switch shape does not match its kind"})
			}
		}
		if terminator.Kind == TerminatorReturn {
			failures = append(failures, verifyOperands(state, []Operand{terminator.ReturnValue}, terminatorEntity(item.id, terminator))...)
			for symbol, resource := range resources {
				if state.live[symbol] {
					failures = append(failures, VerificationError{Code: "V1102", Entity: blockEntityFor(function, item.id), Detail: "resource " + resourceEntity(resource, fmt.Sprintf("local:%d", symbol)) + " exits without cleanup"})
				}
			}
			continue
		}
		if terminator.Kind == TerminatorTrap || terminator.Kind == TerminatorUnreachable {
			continue
		}
		for _, successor := range terminator.Successors {
			queue = append(queue, pendingBlock{id: successor, state: cloneVerificationState(state)})
		}
	}

	for _, loan := range function.Loans {
		if loan.OutlivesOrigin {
			failures = append(failures, VerificationError{Code: "V1301", Entity: loan.Entity, Detail: "safe borrow outlives its origin"})
		}
	}
	if len(function.ReturnOrigins) > 0 {
		returned := 0
		for _, origin := range function.ReturnOrigins {
			if origin.Returned {
				returned++
			}
		}
		if returned != 1 {
			failures = append(failures, VerificationError{Code: "V1302", Entity: fallbackEntity(function.ReturnEntity, "terminator:return"), Detail: "safe-borrow return does not have exactly one origin"})
		}
	}
	for _, value := range function.RegionValues {
		if value.Escapes {
			failures = append(failures, VerificationError{Code: "V1401", Entity: value.Entity, Detail: "region-derived value escapes its region"})
		}
		if value.NeedsDrop {
			failures = append(failures, VerificationError{Code: "V1402", Entity: value.OperationEntity, Detail: "plain region receives a NeedsDrop value"})
		}
	}
	for _, operation := range function.UnsafeOperations {
		if operation.SourceSpanEntity == "" || operation.ScopeEntity == "" {
			failures = append(failures, VerificationError{Code: "V1701", Entity: operation.Entity, Detail: "unsafe instruction lacks source provenance"})
		}
	}
	return failures
}

func validSwitchShape(terminator Terminator) bool {
	switch terminator.SwitchKind {
	case "":
		return len(terminator.SwitchOperands) == 0 && len(terminator.Successors) == 2
	case "bool":
		return len(terminator.SwitchOperands) == 1 && len(terminator.Successors) == 2
	case "result":
		return len(terminator.SwitchOperands) == 1 && len(terminator.Successors) == 2
	case "option":
		return len(terminator.SwitchOperands) == 1 && len(terminator.Successors) == 2
	case "checked-division":
		return len(terminator.SwitchOperands) == 2 && len(terminator.Successors) == 2
	default:
		return false
	}
}

func verifyOperands(state verificationState, operands []Operand, entity string) []VerificationError {
	var failures []VerificationError
	for _, operand := range operands {
		if operand.Kind == OperandLocal && !state.live[hir.SymbolID(operand.Local)] {
			failures = append(failures, VerificationError{Code: "V1002", Entity: entity, Detail: fmt.Sprintf("local %d used before initialization", operand.Local)})
		}
	}
	return failures
}

func uniqueNonEmpty(values []string) bool {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if value == "" {
			return false
		}
		if _, duplicate := seen[value]; duplicate {
			return false
		}
		seen[value] = struct{}{}
	}
	return true
}

func cloneVerificationState(input verificationState) verificationState {
	result := verificationState{
		live:  cloneLive(input.live),
		flags: make(map[string]flagState, len(input.flags)),
	}
	for entity, state := range input.flags {
		result.flags[entity] = state
	}
	return result
}

func mergeVerificationState(previous, incoming verificationState, resources map[hir.SymbolID]ResourceLocal) (verificationState, bool, bool) {
	if len(previous.flags) != len(incoming.flags) {
		return verificationState{}, false, true
	}
	for entity, state := range previous.flags {
		if incoming.flags[entity] != state {
			return verificationState{}, false, true
		}
	}
	merged := cloneVerificationState(previous)
	changed := false
	keys := make(map[hir.SymbolID]struct{}, len(previous.live)+len(incoming.live))
	for symbol := range previous.live {
		keys[symbol] = struct{}{}
	}
	for symbol := range incoming.live {
		keys[symbol] = struct{}{}
	}
	for symbol := range keys {
		left := previous.live[symbol]
		right := incoming.live[symbol]
		if _, resource := resources[symbol]; resource {
			if left != right {
				return verificationState{}, false, true
			}
			continue
		}
		value := left && right
		if merged.live[symbol] != value {
			merged.live[symbol] = value
			changed = true
		}
	}
	return merged, changed, false
}

func sameLiveState(left, right map[hir.SymbolID]bool) bool {
	if len(left) != len(right) {
		return false
	}
	for symbol, live := range left {
		if right[symbol] != live {
			return false
		}
	}
	return true
}

func blockEntityFor(function *Function, id BlockID) string {
	if int(id) >= 0 && int(id) < len(function.Blocks) && function.Blocks[id].Entity != "" {
		return function.Blocks[id].Entity
	}
	return fmt.Sprintf("block:%d", id)
}

func operationEntity(block BlockID, index int, operation Operation) string {
	return fallbackEntity(operation.Entity, fmt.Sprintf("block:%d.operation:%d", block, index))
}

func terminatorEntity(block BlockID, terminator Terminator) string {
	return fallbackEntity(terminator.Entity, fmt.Sprintf("block:%d.terminator", block))
}

func resourceEntity(resource ResourceLocal, fallback string) string {
	return fallbackEntity(resource.Entity, fallback)
}

func fallbackEntity(value, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}
