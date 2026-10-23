package mir

import "github.com/0disoft/fairyc/internal/hir"

// Region identities are passed only to Region parameters of ordinary internal
// calls. A live scalar with the same local number cannot substitute for one.
func verifyRegionCall(state verificationState, call Terminator, callee *Function, entity string) []VerificationError {
	var failures []VerificationError
	for index, argument := range call.Arguments {
		wanted := callee != nil && call.ForeignABI == "" && index < len(callee.ParameterTypes) && callee.ParameterTypes[index] == hir.TypeRegion
		actual := argument.Kind == OperandRegion
		_, active := state.regions[hir.SymbolID(argument.Local)]
		if wanted != actual || wanted && !active {
			failures = append(failures, VerificationError{Code: "V1403", Entity: entity, Detail: "Region call argument must name one active capability and a Region parameter"})
		}
	}
	return failures
}
