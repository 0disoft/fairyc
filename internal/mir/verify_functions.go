package mir

import (
	"fmt"
	"slices"

	"github.com/0disoft/fairyc/internal/hir"
)

func verifyFunctionDefinitions(functions []Function, foreignFunctions map[hir.SymbolID]*ForeignFunction) (map[hir.SymbolID]*Function, []VerificationError) {
	bySymbol := make(map[hir.SymbolID]*Function, len(functions))
	var failures []VerificationError
	for index := range functions {
		function := &functions[index]
		if function.Symbol == hir.NoSymbol {
			failures = append(failures, VerificationError{Code: CodeFunctionIdentity, Entity: function.Name, Detail: "function symbol is missing"})
			continue
		}
		if previous := bySymbol[function.Symbol]; previous != nil {
			failures = append(failures, VerificationError{Code: CodeFunctionIdentity, Entity: function.Name, Detail: "function symbol is duplicated"})
			continue
		}
		if foreignFunctions[function.Symbol] != nil {
			failures = append(failures, VerificationError{Code: CodeFunctionIdentity, Entity: function.Name, Detail: "symbol is both internal and foreign"})
			continue
		}
		bySymbol[function.Symbol] = function
	}
	return bySymbol, failures
}

func verifyDropGlueLinks(resources []ResourceType, functions []Function, bySymbol map[hir.SymbolID]*Function) []VerificationError {
	byResource := make(map[hir.Type]*Function)
	var failures []VerificationError
	for index := range functions {
		function := &functions[index]
		if function.DropGlueFor == hir.ParseType("") {
			continue
		}
		if previous := byResource[function.DropGlueFor]; previous != nil {
			failures = append(failures, VerificationError{Code: CodeDropGlue, Entity: function.Name, Detail: "resource has duplicate drop glue"})
		} else {
			byResource[function.DropGlueFor] = function
		}
		if len(function.Parameters) != 1 || len(function.ParameterTypes) != 1 ||
			function.ParameterTypes[0] != function.DropGlueFor || function.ResultType != hir.TypeVoid {
			failures = append(failures, VerificationError{Code: CodeDropGlue, Entity: function.Name, Detail: "drop glue signature is not fn(Resource) -> void"})
		}
		if len(function.Parameters) == 1 {
			for _, local := range function.Resources {
				if local.Symbol == hir.SymbolID(function.Parameters[0]) {
					failures = append(failures, VerificationError{Code: CodeDropGlue, Entity: function.Name, Detail: "drop receiver owns a recursive cleanup obligation"})
				}
			}
		}
	}
	for index := range resources {
		resource := &resources[index]
		entity := fallbackEntity(resource.Entity, resourceTypeEntity(resource.Type))
		switch resource.DropKind {
		case hir.ResourceDropEmpty:
			if resource.DropSymbol != hir.NoSymbol || len(resource.DropEffects) != 0 {
				failures = append(failures, VerificationError{Code: CodeDropGlue, Entity: entity, Detail: "empty drop has an executable drop symbol"})
			}
		case hir.ResourceDropNonEmpty:
			function := bySymbol[resource.DropSymbol]
			if resource.DropSymbol == hir.NoSymbol || function == nil || function.DropGlueFor != resource.Type || byResource[resource.Type] != function {
				failures = append(failures, VerificationError{Code: CodeDropGlue, Entity: entity, Detail: "non-empty drop has no matching drop glue"})
			} else if !slices.Equal(function.Effects, resource.DropEffects) {
				failures = append(failures, VerificationError{Code: CodeDropGlue, Entity: entity, Detail: "drop glue effects do not match the resource declaration"})
			}
		}
	}
	return failures
}

func verifyForeignFunctions(definitions []ForeignFunction) (map[hir.SymbolID]*ForeignFunction, []VerificationError) {
	result := make(map[hir.SymbolID]*ForeignFunction, len(definitions))
	byLinkName := make(map[string]*ForeignFunction, len(definitions))
	var failures []VerificationError
	for index := range definitions {
		definition := &definitions[index]
		entity := fallbackEntity(definition.Entity, fmt.Sprintf("foreign:%d", definition.Symbol))
		if definition.Symbol == hir.NoSymbol || definition.Name == "" || definition.LinkName == "" || definition.ABI != "C" || definition.ResultType == hir.ParseType("") || definition.ResultType == hir.TypeInvalid {
			failures = append(failures, VerificationError{Code: CodeForeignDeclaration, Entity: entity, Detail: "foreign declaration is incomplete or has an unsupported ABI"})
			continue
		}
		if _, duplicate := result[definition.Symbol]; duplicate {
			failures = append(failures, VerificationError{Code: CodeForeignDeclaration, Entity: entity, Detail: "foreign symbol is duplicated"})
			continue
		}
		result[definition.Symbol] = definition
		if previous, duplicate := byLinkName[definition.LinkName]; duplicate {
			if !CompatibleForeignSignature(*previous, *definition) {
				failures = append(failures, VerificationError{Code: CodeForeignDeclaration, Entity: entity, Detail: "foreign declarations for one link name have conflicting signatures"})
			}
		} else {
			byLinkName[definition.LinkName] = definition
		}
		if _, _, borrow := hir.BorrowElement(definition.ResultType); borrow {
			failures = append(failures, VerificationError{Code: CodeForeignDeclaration, Entity: entity, Detail: "foreign result exposes a safe borrow"})
		}
		for _, parameterType := range definition.ParameterTypes {
			if parameterType == hir.ParseType("") || parameterType == hir.TypeInvalid {
				failures = append(failures, VerificationError{Code: CodeForeignDeclaration, Entity: entity, Detail: "foreign parameter has no concrete type"})
			}
			if _, _, borrow := hir.BorrowElement(parameterType); borrow {
				failures = append(failures, VerificationError{Code: CodeForeignDeclaration, Entity: entity, Detail: "foreign parameter exposes a safe borrow"})
			}
		}
	}
	return result, failures
}
