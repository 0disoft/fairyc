package contractstatemir

import (
	"fmt"
	"slices"

	"github.com/0disoft/fairyc/internal/hir"
)

const (
	maxTransactionHelpers          = 32
	maxTransactionHelperOperations = 1024
)

func verifyHelpers(program Program, bound boundLayout) (map[string][]hir.Effect, []VerificationError) {
	effects := make(map[string][]hir.Effect, len(program.Helpers))
	failures := make([]VerificationError, 0)
	if len(program.Helpers) > maxTransactionHelpers {
		failures = append(failures, VerificationError{Code: "CSM1009", Entity: "helpers", Detail: "helper count exceeds 32"})
	}
	totalOperations := len(program.Operations)
	for _, block := range program.Blocks {
		totalOperations += len(block.Operations)
	}
	for index, helper := range program.Helpers {
		entity := fmt.Sprintf("helper %d", index)
		if helper.Name == "" || !validStateBodyLogicalName(helper.Name) {
			failures = append(failures, VerificationError{Code: "CSM1009", Entity: entity, Detail: "helper requires one canonical module-qualified name"})
		}
		if index > 0 && program.Helpers[index-1].Name >= helper.Name {
			failures = append(failures, VerificationError{Code: "CSM1009", Entity: entity, Detail: "helpers must be sorted and unique by name"})
		}
		if _, duplicate := effects[helper.Name]; duplicate {
			failures = append(failures, VerificationError{Code: "CSM1009", Entity: entity, Detail: "helper name is duplicated"})
		}
		effects[helper.Name] = slices.Clone(helper.Effects)
		totalOperations += len(helper.Operations)
		for _, block := range helper.Blocks {
			totalOperations += len(block.Operations)
		}
	}
	if totalOperations > maxTransactionHelperOperations {
		failures = append(failures, VerificationError{Code: "CSM1009", Entity: "helpers", Detail: "root and helper operation count exceeds 1024"})
	}

	dependencies := make(map[string][]string, len(program.Helpers)+1)
	dependencies[""] = bodyCalls(program.Operations, program.Blocks)
	for index, helper := range program.Helpers {
		dependencies[helper.Name] = bodyCalls(helper.Operations, helper.Blocks)
		body := Program{
			Layout: program.Layout, Capability: program.Capability, Effects: helper.Effects,
			Operations: helper.Operations, Exit: helper.Exit, TrapCategory: helper.TrapCategory,
			Entry: helper.Entry, Blocks: helper.Blocks,
		}
		for _, failure := range verifyProgramBody(body, bound, effects, true) {
			failure.Entity = fmt.Sprintf("helper %d %s", index, failure.Entity)
			failures = append(failures, failure)
		}
	}

	seen := make(map[string]bool, len(program.Helpers))
	active := make(map[string]bool, len(program.Helpers))
	var visit func(string)
	visit = func(name string) {
		if active[name] {
			failures = append(failures, VerificationError{Code: "CSM1009", Entity: "helper " + name, Detail: "helper call graph contains recursion"})
			return
		}
		if seen[name] {
			return
		}
		seen[name], active[name] = true, true
		for _, callee := range dependencies[name] {
			if _, found := effects[callee]; !found {
				continue
			}
			visit(callee)
		}
		active[name] = false
	}
	for _, callee := range dependencies[""] {
		if _, found := effects[callee]; found {
			visit(callee)
		}
	}
	for _, helper := range program.Helpers {
		if !seen[helper.Name] {
			failures = append(failures, VerificationError{Code: "CSM1009", Entity: "helper " + helper.Name, Detail: "helper is unreachable from the root transaction"})
		}
	}
	return effects, failures
}

func bodyCalls(operations []Operation, blocks []Block) []string {
	result := make([]string, 0)
	for _, operation := range operations {
		if operation.Kind == OperationCall {
			result = append(result, operation.Callee)
		}
	}
	for _, block := range blocks {
		for _, operation := range block.Operations {
			if operation.Kind == OperationCall {
				result = append(result, operation.Callee)
			}
		}
	}
	return result
}

func shareHelperCapability(program *Program) {
	for index := range program.Operations {
		program.Operations[index].Capability = program.Capability
	}
	for blockIndex := range program.Blocks {
		for operationIndex := range program.Blocks[blockIndex].Operations {
			program.Blocks[blockIndex].Operations[operationIndex].Capability = program.Capability
		}
	}
	for helperIndex := range program.Helpers {
		helper := &program.Helpers[helperIndex]
		for operationIndex := range helper.Operations {
			helper.Operations[operationIndex].Capability = program.Capability
		}
		for blockIndex := range helper.Blocks {
			for operationIndex := range helper.Blocks[blockIndex].Operations {
				helper.Blocks[blockIndex].Operations[operationIndex].Capability = program.Capability
			}
		}
	}
}

func cloneStateOperations(values []Operation) []Operation {
	result := make([]Operation, len(values))
	for index, value := range values {
		result[index] = value
		result[index].Key.Encoded = slices.Clone(value.Key.Encoded)
		result[index].Value.Encoded = slices.Clone(value.Value.Encoded)
	}
	return result
}

func cloneStateBlocks(values []Block) []Block {
	if values == nil {
		return nil
	}
	result := make([]Block, len(values))
	for index, value := range values {
		result[index] = value
		result[index].Parameters = slices.Clone(value.Parameters)
		result[index].Operations = cloneStateOperations(value.Operations)
		result[index].Terminator.Arguments = slices.Clone(value.Terminator.Arguments)
		result[index].Terminator.TrueArguments = slices.Clone(value.Terminator.TrueArguments)
		result[index].Terminator.FalseArguments = slices.Clone(value.Terminator.FalseArguments)
	}
	return result
}

func validateCurrentHelperHIR(input hir.Package, packageID hir.PackageID, root hir.Function, program Program) error {
	bySymbol := make(map[hir.SymbolID]string)
	byName := make(map[string]hir.Function)
	functions := make(map[hir.SymbolID]hir.Function)
	for _, module := range input.Modules {
		if module.Package != packageID {
			continue
		}
		for _, function := range module.Functions {
			name := module.Path + "::" + function.Name
			bySymbol[function.Symbol] = name
			byName[name] = function
			functions[function.Symbol] = function
		}
	}
	validate := func(function hir.Function, operations []Operation, blocks []Block) error {
		calls, err := hirStaticStateCalls(function.Body)
		if err != nil {
			return err
		}
		if err := verifyHIRStateCallEffects(function.Body, functions); err != nil {
			return err
		}
		stableCalls := bodyCalls(operations, blocks)
		if len(calls) != len(stableCalls) {
			return fmt.Errorf("current helper call sequence differs from the stable transaction body")
		}
		for index, symbol := range calls {
			if bySymbol[symbol] != stableCalls[index] {
				return fmt.Errorf("current helper call %d differs from the stable transaction body", index)
			}
		}
		return nil
	}
	if err := validate(root, program.Operations, program.Blocks); err != nil {
		return err
	}
	for _, helper := range program.Helpers {
		function, found := byName[helper.Name]
		if !found {
			return fmt.Errorf("current package is missing stable helper %q", helper.Name)
		}
		if len(function.Parameters) != 0 || function.ResultType != hir.TypeVoid || !slices.Equal(function.Effects, helper.Effects) {
			return fmt.Errorf("current helper %q signature or effects differ from the stable body", helper.Name)
		}
		if err := validate(function, helper.Operations, helper.Blocks); err != nil {
			return fmt.Errorf("current helper %q: %w", helper.Name, err)
		}
	}
	return nil
}
