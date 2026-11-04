package mir

import (
	"slices"

	"github.com/0disoft/fairyc/internal/hir"
)

// Keep the smallest admitted entities while scanning unordered region state.
// This bounds storage without allocating/sorting a copy of every map key.
func verifyRegionExits(regions map[hir.SymbolID]string) []VerificationError {
	var entities []string
	omitted := false
	for _, entity := range regions {
		if entity == "" {
			continue // Caller-owned capabilities need no reset here.
		}
		index, _ := slices.BinarySearch(entities, entity)
		if len(entities) == DefaultVerificationReportLimit {
			omitted = true
			if index == len(entities) {
				continue
			}
			entities = entities[:len(entities)-1]
		}
		entities = slices.Insert(entities, index, entity)
	}
	var failures []VerificationError
	for _, entity := range entities {
		failures = appendVerificationErrors(failures, VerificationError{Code: "V1403", Entity: entity, Detail: "normal return exits an active region without reset"})
	}
	if omitted {
		failures = appendVerificationErrors(failures, VerificationError{Code: CodeVerificationReportLimit})
	}
	return failures
}

// Region identities are passed only to Region parameters of ordinary internal
// calls. A live scalar with the same local number cannot substitute for one.
func verifyRegionCall(state verificationState, call Terminator, callee *Function, entity string) []VerificationError {
	var failures []VerificationError
	for index, argument := range call.Arguments {
		if verificationReportsFull(failures) {
			return failures
		}
		wanted := callee != nil && call.ForeignABI == "" && index < len(callee.ParameterTypes) && callee.ParameterTypes[index] == hir.TypeRegion
		actual := argument.Kind == OperandRegion
		_, active := state.regions[hir.SymbolID(argument.Local)]
		if wanted != actual || wanted && !active {
			failures = appendVerificationErrors(failures, VerificationError{Code: "V1403", Entity: entity, Detail: "Region call argument must name one active capability and a Region parameter"})
		}
	}
	return failures
}
