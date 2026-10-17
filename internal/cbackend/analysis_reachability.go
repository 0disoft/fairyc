package cbackend

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

func normalizeForeignFunctions(input []mir.ForeignFunction) ([]mir.ForeignFunction, map[hir.SymbolID]int, error) {
	result := make([]mir.ForeignFunction, 0, len(input))
	bySymbol := make(map[hir.SymbolID]int, len(input))
	byLinkName := make(map[string]int, len(input))
	for _, function := range input {
		if _, duplicate := bySymbol[function.Symbol]; duplicate {
			return nil, nil, fmt.Errorf("duplicate MIR foreign symbol %d", function.Symbol)
		}
		if index, duplicateName := byLinkName[function.LinkName]; duplicateName {
			canonical := &result[index]
			if !mir.CompatibleForeignSignature(*canonical, function) {
				return nil, nil, fmt.Errorf("conflicting external C declarations %s and %s name %q", canonical.Name, function.Name, function.LinkName)
			}
			bySymbol[function.Symbol] = index
			continue
		}
		index := len(result)
		result = append(result, function)
		byLinkName[function.LinkName] = index
		bySymbol[function.Symbol] = index
	}
	return result, bySymbol, nil
}

// indexFunctions maps every function symbol to its position in a name-sorted
// function slice and resolves the single C entry by name. Entry reachability is
// seeded from this index, so a missing or duplicated entry fails closed here
// before any emission work begins.
func indexFunctions(functions []mir.Function, entry string) (map[hir.SymbolID]int, int, error) {
	bySymbol := make(map[hir.SymbolID]int, len(functions))
	entryIndex := -1
	for index := range functions {
		function := &functions[index]
		if _, duplicate := bySymbol[function.Symbol]; duplicate {
			return nil, -1, fmt.Errorf("duplicate MIR function symbol %d", function.Symbol)
		}
		bySymbol[function.Symbol] = index
		if function.Name == entry {
			if entryIndex >= 0 {
				return nil, -1, fmt.Errorf("duplicate C entry function %q", entry)
			}
			entryIndex = index
		}
	}
	if entryIndex < 0 {
		return nil, -1, fmt.Errorf("C entry function %q is missing", entry)
	}
	return bySymbol, entryIndex, nil
}

// reachableFunctionIndices walks internal call edges, drop-glue edges, and
// tagged drop payloads from the single entry root and returns the visited
// indices into the name-sorted function slice plus the foreign symbols
// referenced by reachable calls. Foreign declarations are filtered separately
// so dead declarations cannot leak prototypes, guards, or scalar features.
func reachableFunctionIndices(program mir.Program, functions []mir.Function, bySymbol map[hir.SymbolID]int, entryIndex int) (map[int]struct{}, map[hir.SymbolID]struct{}, error) {
	resourceTypes := make(map[hir.Type]mir.ResourceType, len(program.ResourceTypes))
	for index := range program.ResourceTypes {
		resourceTypes[program.ResourceTypes[index].Type] = program.ResourceTypes[index]
	}
	visited := make(map[int]struct{})
	reachableForeign := make(map[hir.SymbolID]struct{})
	pending := []int{entryIndex}
	for len(pending) != 0 {
		index := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		if _, seen := visited[index]; seen {
			continue
		}
		visited[index] = struct{}{}
		function := &functions[index]
		locals := make(map[hir.SymbolID]hir.Type, len(function.Resources))
		for _, resource := range function.Resources {
			locals[resource.Symbol] = resource.Type
		}
		for _, block := range function.Blocks {
			for _, operation := range block.Operations {
				if operation.Kind != mir.OperationDrop {
					continue
				}
				resourceType, found := locals[operation.Symbol]
				if !found {
					return nil, nil, fmt.Errorf("drop in %s names undeclared resource local %d", function.Name, operation.Symbol)
				}
				for _, dropSymbol := range dischargeDropSymbols(resourceType, resourceTypes) {
					if dropSymbol == hir.NoSymbol {
						continue
					}
					target, found := bySymbol[dropSymbol]
					if !found {
						return nil, nil, fmt.Errorf("resource %q names missing drop symbol %d", resourceType, dropSymbol)
					}
					pending = append(pending, target)
				}
			}
			terminator := block.Terminator
			if terminator.Kind != mir.TerminatorCall {
				continue
			}
			if terminator.ForeignABI != "" {
				reachableForeign[terminator.Callee] = struct{}{}
				continue
			}
			target, found := bySymbol[terminator.Callee]
			if !found {
				return nil, nil, fmt.Errorf("C call in %s names missing function symbol %d", function.Name, terminator.Callee)
			}
			pending = append(pending, target)
		}
	}
	return visited, reachableForeign, nil
}

// dischargeDropSymbols returns the executable drop glue symbols a dropped local
// can reach: the direct resource type, or the direct resource payloads of an
// Option/Result discharge. Empty drops and non-resource discharges return no
// symbols.
func dischargeDropSymbols(resourceType hir.Type, resourceTypes map[hir.Type]mir.ResourceType) []hir.SymbolID {
	var symbols []hir.SymbolID
	seen := map[hir.Type]bool{}
	var visit func(hir.Type)
	visit = func(value hir.Type) {
		if seen[value] {
			return
		}
		seen[value] = true
		if definition, found := resourceTypes[value]; found {
			symbols = append(symbols, definition.DropSymbol)
			for _, field := range definition.Fields {
				visit(field.Type)
			}
		} else if element, option := hir.OptionElement(value); option {
			visit(element)
		} else if success, failure, result := hir.ResultElements(value); result {
			visit(success)
			visit(failure)
		}
	}
	visit(resourceType)
	return symbols
}

// filterFunctions keeps only visited indices in the original name-sorted order,
// preserving deterministic emission order without mutating the verified MIR.
func filterFunctions(functions []mir.Function, reachable map[int]struct{}) []mir.Function {
	filtered := make([]mir.Function, 0, len(reachable))
	for index := range functions {
		if _, keep := reachable[index]; keep {
			filtered = append(filtered, functions[index])
		}
	}
	return filtered
}

// filterForeignFunctions keeps only foreign declarations referenced by
// reachable calls, preserving the normalized LinkName/symbol ordering and the
// symbol mapping for every declaration aliased onto one canonical LinkName.
// A reachable call whose declaration is missing from the verified set fails
// closed here instead of leaking a dangling prototype.
func filterForeignFunctions(foreignFunctions []mir.ForeignFunction, bySymbol map[hir.SymbolID]int, reachable map[hir.SymbolID]struct{}) ([]mir.ForeignFunction, map[hir.SymbolID]int, error) {
	keep := make(map[int]struct{})
	for symbol := range reachable {
		index, found := bySymbol[symbol]
		if !found {
			return nil, nil, fmt.Errorf("reachable foreign call names missing declaration symbol %d", symbol)
		}
		keep[index] = struct{}{}
	}
	filtered := make([]mir.ForeignFunction, 0, len(keep))
	filteredBySymbol := make(map[hir.SymbolID]int, len(bySymbol))
	position := make(map[int]int, len(keep))
	for index := range foreignFunctions {
		if _, retained := keep[index]; !retained {
			continue
		}
		position[index] = len(filtered)
		filtered = append(filtered, foreignFunctions[index])
	}
	for symbol, index := range bySymbol {
		if filteredIndex, retained := position[index]; retained {
			filteredBySymbol[symbol] = filteredIndex
		}
	}
	return filtered, filteredBySymbol, nil
}
