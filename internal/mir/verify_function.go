package mir

import (
	"fmt"
	"slices"

	"github.com/0disoft/fairyc/internal/hir"
)

func verifyFunction(function *Function, structTypes map[hir.Type]*StructType, enumTypes map[hir.Type]*EnumType, resourceTypes map[hir.Type]*ResourceType, foreignFunctions map[hir.SymbolID]*ForeignFunction, functions map[hir.SymbolID]*Function) []VerificationError {
	storeTypes := fieldStoreLocalTypes(function, functions, foreignFunctions)
	if int(function.Entry) < 0 || int(function.Entry) >= len(function.Blocks) {
		return []VerificationError{{Code: CodeControlFlow, Entity: function.Name, Detail: "entry block is missing"}}
	}
	var failures []VerificationError
	resources := make(map[hir.SymbolID]ResourceLocal, len(function.Resources))
	dropEffects := make(map[hir.Type][]hir.Effect)
	initial := verificationState{
		live:    make(map[hir.SymbolID]bool, len(function.Resources)),
		flags:   make(map[string]flagState, len(function.DropFlags)),
		regions: make(map[hir.SymbolID]string),
	}
	for _, resource := range function.Resources {
		resources[resource.Symbol] = resource
		initial.live[resource.Symbol] = resource.Parameter
		if !containsDefinedResource(resource.Type, resourceTypes) {
			failures = append(failures, VerificationError{Code: CodeAggregateUse, Entity: resourceTypeEntity(resource.Type), Detail: "resource local type has no definition"})
		}
	}
	for _, parameter := range function.Parameters {
		initial.live[hir.SymbolID(parameter)] = true
	}
	flags := make(map[string]DropFlag, len(function.DropFlags))
	for _, flag := range function.DropFlags {
		flags[flag.Entity] = flag
		initial.flags[flag.Entity] = flagState{}
	}
	unsafeByEntity := make(map[string]UnsafeOperation, len(function.UnsafeOperations))
	for _, operation := range function.UnsafeOperations {
		if operation.Entity == "" {
			failures = append(failures, VerificationError{Code: "V1701", Entity: function.Name, Detail: "unsafe instruction has no entity identity"})
			continue
		}
		if _, duplicate := unsafeByEntity[operation.Entity]; duplicate {
			failures = append(failures, VerificationError{Code: "V1701", Entity: operation.Entity, Detail: "unsafe instruction entity is duplicated"})
			continue
		}
		unsafeByEntity[operation.Entity] = operation
	}
	verifier := operationVerifier{
		function:       function,
		structTypes:    structTypes,
		enumTypes:      enumTypes,
		resourceTypes:  resourceTypes,
		resources:      resources,
		dropEffects:    dropEffects,
		flags:          flags,
		unsafeByEntity: unsafeByEntity,
		storeTypes:     storeTypes,
	}
	type pendingBlock struct {
		id    BlockID
		state verificationState
	}
	queue := []pendingBlock{{id: function.Entry, state: initial}}
	seen := make(map[BlockID]verificationState)
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
			failures = append(failures, VerificationError{Code: CodeControlFlow, Entity: blockEntityFor(function, item.id), Detail: "successor block is missing"})
			continue
		}
		block := function.Blocks[item.id]
		state := cloneVerificationState(item.state)
		for operationIndex, operation := range block.Operations {
			failures = append(failures, verifier.verifyOperation(state, operation, operationEntity(item.id, operationIndex, operation))...)
		}

		terminator := block.Terminator
		if terminator.Kind == "" {
			failures = append(failures, VerificationError{Code: CodeControlFlow, Entity: blockEntityFor(function, item.id), Detail: "reachable block has no terminator"})
			continue
		}
		if terminator.Kind == TerminatorCall && terminator.UnwindSuccessor != NoBlock {
			failures = append(failures, VerificationError{Code: "V1502", Entity: terminatorEntity(item.id, terminator), Detail: "Stage 0 call has an unwind successor"})
		}
		if terminator.Kind == TerminatorCall {
			entity := terminatorEntity(item.id, terminator)
			failures = append(failures, verifyOperands(state, terminator.Arguments, entity)...)
			foreign := foreignFunctions[terminator.Callee]
			internal := functions[terminator.Callee]
			if terminator.ForeignABI != "" {
				provenance, hasProvenance := unsafeByEntity[entity]
				callEntity := sourceEntity("call", terminator.Span)
				sourceSpanEntity := sourceEntity("span", terminator.Span)
				scopeEntity := sourceEntity("unsafe-scope", terminator.UnsafeScope)
				scopeContainsCall := sameSpanSource(terminator.UnsafeScope, terminator.Span) &&
					terminator.UnsafeScope.StartByte <= terminator.Span.StartByte && terminator.UnsafeScope.EndByte >= terminator.Span.EndByte
				if foreign == nil || terminator.ForeignABI != "C" || terminator.Entity != callEntity || !scopeContainsCall ||
					!hasProvenance || provenance.SourceSpanEntity != sourceSpanEntity || provenance.ScopeEntity != scopeEntity {
					failures = append(failures, VerificationError{Code: "V1701", Entity: entity, Detail: "foreign call lacks a matching declaration or unsafe provenance"})
				}
			} else if foreign != nil {
				failures = append(failures, VerificationError{Code: "V1701", Entity: entity, Detail: "foreign call is missing its ABI and unsafe provenance"})
			} else if internal == nil {
				failures = append(failures, VerificationError{Code: CodeCallSignature, Entity: entity, Detail: "internal call names a missing function"})
			} else {
				if len(terminator.Arguments) != len(internal.ParameterTypes) {
					failures = append(failures, VerificationError{Code: CodeCallSignature, Entity: entity, Detail: "internal call argument count does not match the callee signature"})
				}
				if (internal.ResultType == hir.TypeVoid && terminator.DestinationLocal != NoLocal) ||
					(internal.ResultType != hir.TypeVoid && terminator.DestinationLocal == NoLocal) {
					failures = append(failures, VerificationError{Code: CodeCallSignature, Entity: entity, Detail: "internal call destination does not match the callee result"})
				}
			}
			if terminator.DestinationLocal != NoLocal {
				state.live[hir.SymbolID(terminator.DestinationLocal)] = true
			}
		}
		if terminator.Kind == TerminatorSwitch {
			failures = append(failures, verifyOperands(state, terminator.SwitchOperands, terminatorEntity(item.id, terminator))...)
			if !validSwitchShape(terminator) {
				failures = append(failures, VerificationError{Code: CodeControlFlow, Entity: terminatorEntity(item.id, terminator), Detail: "switch shape does not match its kind"})
			}
			if terminator.SwitchKind == "enum" {
				definition := enumTypes[terminator.SwitchType]
				if definition == nil || !slices.Equal(terminator.SwitchVariants, enumVariantNames(definition.Variants)) {
					failures = append(failures, VerificationError{Code: CodeControlFlow, Entity: terminatorEntity(item.id, terminator), Detail: "enum switch variants do not match the enum definition"})
				}
			}
		}
		if terminator.Kind == TerminatorReturn {
			if function.ResultType == hir.TypeVoid {
				if terminator.ReturnValue.Kind != OperandInvalid {
					failures = append(failures, VerificationError{Code: CodeReturnShape, Entity: terminatorEntity(item.id, terminator), Detail: "void return carries a value"})
				}
			} else {
				failures = append(failures, verifyOperands(state, []Operand{terminator.ReturnValue}, terminatorEntity(item.id, terminator))...)
				if detail := verifyStorageViewReturn(function, terminator.ReturnValue, storeTypes); detail != "" {
					failures = append(failures, VerificationError{Code: CodeStorageReturn, Entity: terminatorEntity(item.id, terminator), Detail: detail})
				}
			}
			for symbol, resource := range resources {
				if state.live[symbol] {
					failures = append(failures, VerificationError{Code: "V1102", Entity: blockEntityFor(function, item.id), Detail: "resource " + resourceEntity(resource, fmt.Sprintf("local:%d", symbol)) + " exits without cleanup"})
				}
			}
			for _, entity := range state.regions {
				failures = append(failures, VerificationError{Code: "V1403", Entity: entity, Detail: "normal return exits an active region without reset"})
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
	failures = append(failures, verifyTrapCheckEdges(function)...)
	failures = append(failures, verifyAssertEdges(function)...)

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
