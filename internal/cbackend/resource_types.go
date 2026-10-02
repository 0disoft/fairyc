package cbackend

import (
	"fmt"
	"slices"
	"strings"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

func orderResourceDefinitions(definitions []mir.ResourceType) ([]mir.ResourceType, error) {
	byType := make(map[hir.Type]mir.ResourceType, len(definitions))
	roots := make([]hir.Type, 0, len(definitions))
	for _, definition := range definitions {
		if _, duplicate := byType[definition.Type]; duplicate {
			return nil, fmt.Errorf("duplicate C resource type %q", definition.Type)
		}
		byType[definition.Type] = definition
		roots = append(roots, definition.Type)
	}
	slices.SortFunc(roots, func(a, b hir.Type) int { return strings.Compare(string(a), string(b)) })
	state := make(map[hir.Type]uint8, len(definitions))
	ordered := make([]mir.ResourceType, 0, len(definitions))
	var visit func(hir.Type) error
	visit = func(value hir.Type) error {
		if state[value] == 2 {
			return nil
		}
		if state[value] == 1 {
			return CapabilityError{Feature: fmt.Sprintf("cyclic by-value resource layout at %q", value)}
		}
		state[value] = 1
		definition := byType[value]
		for _, field := range definition.Fields {
			if _, nested := byType[field.Type]; nested {
				if err := visit(field.Type); err != nil {
					return err
				}
			}
		}
		state[value] = 2
		ordered = append(ordered, definition)
		return nil
	}
	for _, root := range roots {
		if err := visit(root); err != nil {
			return nil, err
		}
	}
	return ordered, nil
}

func (e *emitter) resourceHasNestedFields(value hir.Type) bool {
	index, found := e.resourceByType[value]
	if !found {
		return false
	}
	for _, field := range e.resourceTypes[index].definition.Fields {
		if _, resource := e.resourceByType[field.Type]; resource {
			return true
		}
	}
	return false
}

// Nested cleanup bodies return their post-cleanup local storage to the private
// drop caller. Child destruction must observe any changes the body made to this,
// not a stale by-value copy. The source and MIR cleanup signatures remain void.
func (e *emitter) emittedFunctionResultType(function *mir.Function) (string, error) {
	if e.nonReturning[function.Symbol] {
		return "void", nil
	}
	if e.resourceHasNestedFields(function.DropGlueFor) {
		return e.cValueType(function.DropGlueFor)
	}
	return e.cFunctionResultType(function.ResultType)
}
