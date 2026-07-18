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
			if !sameVerificationState(previous, item.state) {
				failures = append(failures, VerificationError{Code: "V1002", Entity: blockEntityFor(function, item.id), Detail: "incoming initialization states disagree"})
			}
			continue
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
			switch operation.Kind {
			case OperationInitialize:
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
				if !slices.Equal(operation.TrapCategories, []string{"division-by-zero", "signed-division-overflow"}) {
					failures = append(failures, VerificationError{Code: "V1501", Entity: entity, Detail: "division is missing required trap checks"})
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
		if terminator.Kind == TerminatorReturn {
			for symbol, resource := range resources {
				if state.live[symbol] {
					failures = append(failures, VerificationError{Code: "V1102", Entity: blockEntityFor(function, item.id), Detail: "resource " + resourceEntity(resource, fmt.Sprintf("local:%d", symbol)) + " exits without cleanup"})
				}
			}
			continue
		}
		if terminator.Kind == TerminatorTrap {
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

func sameVerificationState(left, right verificationState) bool {
	if !sameLiveState(left.live, right.live) || len(left.flags) != len(right.flags) {
		return false
	}
	for entity, state := range left.flags {
		if right.flags[entity] != state {
			return false
		}
	}
	return true
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
