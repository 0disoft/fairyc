package mir

import "github.com/0disoft/fairyc/internal/hir"

// verifyBoundedCallbacks limits callback addresses to one designated foreign-call
// argument. They cannot become general function values through forged/cached MIR.
func verifyBoundedCallbacks(program Program, functions map[hir.SymbolID]*Function, foreign map[hir.SymbolID]*ForeignFunction) []VerificationError {
	var failures []VerificationError
	reject := func(entity, detail string) {
		failures = append(failures, VerificationError{Code: CodeForeignDeclaration, Entity: entity, Detail: detail})
	}
	for _, declaration := range program.ForeignFunctions {
		if declaration.ResultType == hir.TypeCCallbackI32UserData || !hir.ValidBoundedCallbackParameters(declaration.ParameterTypes) {
			reject(declaration.Entity, "callback requires the final callback/user-data pair and cannot be returned")
		}
	}
	for _, function := range program.Functions {
		var usage map[LocalID]callbackLocalUsage
		if function.ResultType == hir.TypeCCallbackI32UserData {
			reject(function.Name, "callback result is not admitted")
		}
		for _, typ := range function.ParameterTypes {
			if typ == hir.TypeCCallbackI32UserData {
				reject(function.Name, "internal callback parameter is not admitted")
			}
		}
		for _, block := range function.Blocks {
			term := block.Terminator
			if callee := foreign[term.Callee]; term.Kind == TerminatorCall && callee != nil {
				for index, typ := range callee.ParameterTypes {
					if typ != hir.TypeCCallbackI32UserData {
						continue
					}
					origins := 0
					if index < len(term.Arguments) && term.Arguments[index].Kind == OperandLocal {
						for _, op := range block.Operations {
							if op.Kind == OperationCCallback && op.DestinationLocal == term.Arguments[index].Local {
								origins++
							}
						}
					}
					if origins != 1 {
						reject(function.Name, "foreign callback argument has no unique non-null callback origin")
					}
				}
			}
			for _, op := range block.Operations {
				if op.Kind != OperationCCallback {
					continue
				}
				target := functions[op.Symbol]
				if target == nil || target.ResultType != hir.TypeI32 || len(target.ParameterTypes) != 2 ||
					target.ParameterTypes[0] != hir.TypeI32 || target.ParameterTypes[1] != hir.RawPointer(hir.TypeVoid, true) || len(target.Effects) != 0 {
					reject(function.Name, "callback target has no matching internal signature")
				}
				term := block.Terminator
				callee := foreign[term.Callee]
				if term.Kind != TerminatorCall || term.ForeignABI != "C" || callee == nil || len(term.Arguments) < 2 ||
					len(callee.ParameterTypes) != len(term.Arguments) || !hir.ValidBoundedCallbackParameters(callee.ParameterTypes) ||
					callee.ParameterTypes[len(term.Arguments)-2] != hir.TypeCCallbackI32UserData ||
					term.Arguments[len(term.Arguments)-2].Kind != OperandLocal || term.Arguments[len(term.Arguments)-2].Local != op.DestinationLocal {
					reject(function.Name, "callback is not consumed by its call-bounded foreign argument")
				}
				if usage == nil {
					usage = countCallbackLocalUsage(&function)
				}
				counts := usage[op.DestinationLocal]
				if counts.uses != 1 || counts.definitions != 1 {
					reject(function.Name, "callback address escapes or its temporary is overwritten")
				}
			}
		}
	}
	return failures
}

type callbackLocalUsage struct {
	uses, definitions int
}

// Only callback destinations need counts. Build them once per function, using
// sparse keys because cached MIR can contain arbitrary local IDs. Callers defer
// this work until a callback is found, so other functions allocate no index.
func countCallbackLocalUsage(function *Function) map[LocalID]callbackLocalUsage {
	usage := make(map[LocalID]callbackLocalUsage)
	for _, block := range function.Blocks {
		for _, op := range block.Operations {
			if op.Kind == OperationCCallback {
				usage[op.DestinationLocal] = callbackLocalUsage{}
			}
		}
	}
	define := func(local LocalID) {
		if counts, found := usage[local]; found {
			counts.definitions++
			usage[local] = counts
		}
	}
	use := func(operand Operand) {
		if operand.Kind == OperandLocal {
			if counts, found := usage[operand.Local]; found {
				counts.uses++
				usage[operand.Local] = counts
			}
		}
	}
	for _, parameter := range function.Parameters {
		define(parameter)
	}
	for _, block := range function.Blocks {
		for _, op := range block.Operations {
			define(op.DestinationLocal)
			for _, operand := range op.Operands {
				use(operand)
			}
		}
		define(block.Terminator.DestinationLocal)
		use(block.Terminator.ReturnValue)
		for _, operand := range block.Terminator.Arguments {
			use(operand)
		}
		for _, operand := range block.Terminator.SwitchOperands {
			use(operand)
		}
	}
	return usage
}
