package cbackend

import (
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

// filterReachableAggregateTypes runs after full MIR verification and aggregate
// layout preparation. Private C declarations are limited to types needed by
// entry-reachable code; repr(C) declarations remain part of the C ABI surface.
func filterReachableAggregateTypes(
	functions []mir.Function,
	foreign []mir.ForeignFunction,
	reachableForeign map[hir.SymbolID]struct{},
	structures []structCType,
	enumerations []enumCType,
	resources []resourceCType,
) ([]structCType, map[hir.Type]int, []enumCType, map[hir.Type]int, []resourceCType, map[hir.Type]int) {
	structs := make(map[hir.Type]structCType, len(structures))
	for _, value := range structures {
		structs[value.definition.Type] = value
	}
	enums := make(map[hir.Type]enumCType, len(enumerations))
	for _, value := range enumerations {
		enums[value.definition.Type] = value
	}
	resourceDefinitions := make(map[hir.Type]resourceCType, len(resources))
	for _, value := range resources {
		resourceDefinitions[value.definition.Type] = value
	}
	used := make(map[hir.Type]struct{})
	var consider func(hir.Type)
	consider = func(value hir.Type) {
		if value == hir.TypeInvalid {
			return
		}
		if _, seen := used[value]; seen {
			return
		}
		used[value] = struct{}{}
		if definition, ok := structs[value]; ok {
			for _, field := range definition.definition.Fields {
				consider(field.Type)
			}
		}
		if definition, ok := enums[value]; ok {
			for _, variant := range definition.definition.Variants {
				for _, field := range variant.Fields {
					consider(field.Type)
				}
			}
		}
		if definition, ok := resourceDefinitions[value]; ok {
			for _, field := range definition.definition.Fields {
				consider(field.Type)
			}
		}
		if element, _, ok := hir.BorrowElement(value); ok {
			consider(element)
		}
		if element, _, ok := hir.RawPointerElement(value); ok {
			consider(element)
		}
		if element, ok := hir.OptionElement(value); ok {
			consider(element)
		}
		if success, failure, ok := hir.ResultElements(value); ok {
			consider(success)
			consider(failure)
		}
		if element, _, ok := hir.FixedArrayParts(value); ok {
			consider(element)
		}
	}
	considerOperand := func(value mir.Operand) { consider(value.Type) }
	// repr(C) declarations are an explicit C ABI contract. Keep their layout
	// assertions even when no entry-reachable function names the record.
	for _, definition := range structures {
		if definition.definition.ReprC {
			consider(definition.definition.Type)
		}
	}
	for _, function := range functions {
		consider(function.ResultType)
		consider(function.DropGlueFor)
		for _, value := range function.ParameterTypes {
			consider(value)
		}
		for _, value := range function.Resources {
			consider(value.Type)
		}
		for _, block := range function.Blocks {
			for _, operation := range block.Operations {
				consider(operation.Type)
				consider(operation.OwnerType)
				consider(operation.IndexType)
				for _, operand := range operation.Operands {
					considerOperand(operand)
				}
			}
			terminator := block.Terminator
			consider(terminator.SwitchType)
			considerOperand(terminator.ReturnValue)
			for _, argument := range terminator.Arguments {
				considerOperand(argument)
			}
			for _, operand := range terminator.SwitchOperands {
				considerOperand(operand)
			}
		}
	}
	for _, declaration := range foreign {
		if _, called := reachableForeign[declaration.Symbol]; !called {
			continue
		}
		consider(declaration.ResultType)
		for _, parameter := range declaration.ParameterTypes {
			consider(parameter)
		}
	}
	keptStructs := make([]structCType, 0, len(structures))
	structByType := make(map[hir.Type]int)
	for _, value := range structures {
		if _, keep := used[value.definition.Type]; keep {
			structByType[value.definition.Type] = len(keptStructs)
			keptStructs = append(keptStructs, value)
		}
	}
	keptEnums := make([]enumCType, 0, len(enumerations))
	enumByType := make(map[hir.Type]int)
	for _, value := range enumerations {
		if _, keep := used[value.definition.Type]; keep {
			enumByType[value.definition.Type] = len(keptEnums)
			keptEnums = append(keptEnums, value)
		}
	}
	keptResources := make([]resourceCType, 0, len(resources))
	resourceByType := make(map[hir.Type]int)
	for _, value := range resources {
		if _, keep := used[value.definition.Type]; keep {
			resourceByType[value.definition.Type] = len(keptResources)
			keptResources = append(keptResources, value)
		}
	}
	return keptStructs, structByType, keptEnums, enumByType, keptResources, resourceByType
}
