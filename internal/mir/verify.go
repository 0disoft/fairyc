package mir

import (
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

// Verify checks the executable invariants required before MIR handoff.
func Verify(program Program) []VerificationError {
	var failures []VerificationError
	for index := range program.Functions {
		failures = append(failures, verifyFunction(&program.Functions[index])...)
	}
	return failures
}

func verifyFunction(function *Function) []VerificationError {
	if int(function.Entry) < 0 || int(function.Entry) >= len(function.Blocks) {
		return []VerificationError{{Code: "V1001", Entity: function.Name, Detail: "entry block is missing"}}
	}
	resources := make(map[hir.SymbolID]ResourceLocal, len(function.Resources))
	initial := make(map[hir.SymbolID]bool, len(function.Resources))
	for _, resource := range function.Resources {
		resources[resource.Symbol] = resource
		initial[resource.Symbol] = resource.Parameter
	}
	type pendingBlock struct {
		id    BlockID
		state map[hir.SymbolID]bool
	}
	queue := []pendingBlock{{id: function.Entry, state: initial}}
	seen := make(map[BlockID]map[hir.SymbolID]bool)
	var failures []VerificationError
	for len(queue) > 0 {
		item := queue[0]
		queue = queue[1:]
		if previous, found := seen[item.id]; found {
			if !sameState(previous, item.state) {
				failures = append(failures, VerificationError{Code: "V1002", Entity: blockEntity(item.id), Detail: "incoming initialization states disagree"})
			}
			continue
		}
		seen[item.id] = cloneLive(item.state)
		if int(item.id) < 0 || int(item.id) >= len(function.Blocks) {
			failures = append(failures, VerificationError{Code: "V1001", Entity: blockEntity(item.id), Detail: "successor block is missing"})
			continue
		}
		block := function.Blocks[item.id]
		state := cloneLive(item.state)
		for operationIndex, operation := range block.Operations {
			entity := fmt.Sprintf("block:%d.operation:%d", item.id, operationIndex)
			if operation.SourceForm != "" {
				failures = append(failures, VerificationError{Code: "V1601", Entity: entity, Detail: "source-only form remains: " + operation.SourceForm})
			}
			switch operation.Kind {
			case OperationInitialize:
				if state[operation.Symbol] {
					failures = append(failures, VerificationError{Code: "V1002", Entity: entity, Detail: "resource initialized twice"})
				}
				state[operation.Symbol] = true
			case OperationMove:
				if _, resource := resources[operation.Symbol]; resource && !state[operation.Symbol] {
					failures = append(failures, VerificationError{Code: "V1101", Entity: entity, Detail: "resource moved after discharge"})
				}
				state[operation.Symbol] = false
			case OperationDrop:
				if !state[operation.Symbol] {
					failures = append(failures, VerificationError{Code: "V1101", Entity: entity, Detail: "resource dropped after discharge"})
				}
				if !operation.FlagClearedFirst {
					failures = append(failures, VerificationError{Code: "V1202", Entity: entity, Detail: "drop does not clear its active flag first"})
				}
				state[operation.Symbol] = false
			case OperationCheckedDivision:
				if !slices.Equal(operation.TrapCategories, []string{"division-by-zero", "signed-division-overflow"}) {
					failures = append(failures, VerificationError{Code: "V1501", Entity: entity, Detail: "division is missing required trap checks"})
				}
			}
		}

		terminator := block.Terminator
		if terminator.Kind == "" {
			failures = append(failures, VerificationError{Code: "V1001", Entity: blockEntity(item.id), Detail: "reachable block has no terminator"})
			continue
		}
		if terminator.Kind == TerminatorCall && terminator.UnwindSuccessor != NoBlock {
			failures = append(failures, VerificationError{Code: "V1502", Entity: blockEntity(item.id), Detail: "Stage 0 call has an unwind successor"})
		}
		if terminator.Kind == TerminatorReturn {
			for symbol, live := range state {
				if live {
					failures = append(failures, VerificationError{Code: "V1102", Entity: blockEntity(item.id), Detail: fmt.Sprintf("resource %d exits without cleanup", symbol)})
				}
			}
			continue
		}
		if terminator.Kind == TerminatorTrap {
			continue
		}
		for _, successor := range terminator.Successors {
			queue = append(queue, pendingBlock{id: successor, state: cloneLive(state)})
		}
	}
	return failures
}

func sameState(left, right map[hir.SymbolID]bool) bool {
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

func blockEntity(id BlockID) string {
	return fmt.Sprintf("block:%d", id)
}
