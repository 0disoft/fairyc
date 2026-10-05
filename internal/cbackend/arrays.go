package cbackend

import (
	"fmt"
	"slices"
	"strings"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/layout"
	"github.com/0disoft/fairyc/internal/mir"
)

// arrayCType is a private, assignable C wrapper for one Fairy fixed array. The wrapper keeps
// internal calls and returns value-semantic without exposing C's non-assignable raw array rules as
// a Fairy ABI promise.
type arrayCType struct {
	typ     hir.Type
	element hir.Type
	length  uint64
}

func prepareScalarArrayTypes(functions []mir.Function, structures ...[]structCType) ([]arrayCType, map[hir.Type]int, error) {
	seen := make(map[hir.Type]arrayCType)
	var record func(hir.Type) error
	record = func(value hir.Type) error {
		if element, option := hir.OptionElement(value); option {
			return record(element)
		}
		if success, failure, result := hir.ResultElements(value); result {
			if err := record(success); err != nil {
				return err
			}
			return record(failure)
		}
		element, length, array := hir.FixedArrayParts(value)
		if !array {
			return nil
		}
		if !hir.ExecutableFixedArrayShape(element, length) {
			return fmt.Errorf("fixed array %q is outside the bounded executable C subset", value)
		}
		seen[value] = arrayCType{typ: value, element: element, length: length}
		return nil
	}
	for functionIndex := range functions {
		function := &functions[functionIndex]
		if err := record(function.ResultType); err != nil {
			return nil, nil, err
		}
		for _, parameterType := range function.ParameterTypes {
			if err := record(parameterType); err != nil {
				return nil, nil, err
			}
		}
		for blockIndex := range function.Blocks {
			block := &function.Blocks[blockIndex]
			for operationIndex := range block.Operations {
				operation := &block.Operations[operationIndex]
				if err := record(operation.Type); err != nil {
					return nil, nil, err
				}
				if err := record(operation.OwnerType); err != nil {
					return nil, nil, err
				}
			}
		}
	}
	for _, definitions := range structures {
		for _, structure := range definitions {
			for _, field := range structure.definition.Fields {
				if err := record(field.Type); err != nil {
					return nil, nil, err
				}
			}
		}
	}
	values := make([]arrayCType, 0, len(seen))
	for _, value := range seen {
		values = append(values, value)
	}
	slices.SortFunc(values, func(left, right arrayCType) int {
		return strings.Compare(string(left.typ), string(right.typ))
	})
	byType := make(map[hir.Type]int, len(values))
	for index := range values {
		byType[values[index].typ] = index
	}
	return values, byType, nil
}

func scalarArrayLayoutType(value hir.Type) (layout.Type, bool) {
	element, length, array := hir.FixedArrayParts(value)
	if !array || !hir.ExecutableFixedArrayShape(element, length) {
		return layout.Type{}, false
	}
	element = canonicalCValueType(element)
	if element != hir.TypeU8 && element != hir.TypeI32 && element != hir.TypeI64 && element != hir.TypeU32 && element != hir.TypeUSize {
		return layout.Type{}, false
	}
	elementLayout := layout.Type{Kind: layout.KindLanguageScalar, Name: string(element)}
	return layout.Type{Kind: layout.KindArray, Element: &elementLayout, Length: length}, true
}

func (e *emitter) arrayCTypeName(index int) string {
	return fmt.Sprintf("fy_array_%d", index)
}

func (e *emitter) emitScalarArrayTypes(output *strings.Builder, nominal bool) error {
	for index, array := range e.arrayTypes {
		if nominal == hir.FixedArrayScalarElement(array.element) {
			continue
		}
		elementType, err := e.cValueType(array.element)
		if err != nil {
			return fmt.Errorf("emit fixed array %q: %w", array.typ, err)
		}
		fmt.Fprintf(output, "typedef struct { %s data[UINT64_C(%d)]; } %s;\n", elementType, array.length, e.arrayCTypeName(index))
	}
	if len(e.arrayTypes) != 0 {
		output.WriteString("\n")
	}
	return nil
}
