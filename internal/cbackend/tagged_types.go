package cbackend

import (
	"fmt"
	"slices"
	"strings"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/layout"
	"github.com/0disoft/fairyc/internal/mir"
)

const (
	taggedOption = "option"
	taggedResult = "result"
)

type taggedCType struct {
	typ       hir.Type
	kind      string
	payloads  []hir.Type
	layout    layout.Result
	needsDrop bool
}

func prepareTaggedTypes(
	program mir.Program,
	structures []structCType,
	structByType map[hir.Type]int,
	resources []resourceCType,
	resourceByType map[hir.Type]int,
	arrayByType map[hir.Type]int,
	engine *layout.Engine,
) ([]taggedCType, map[hir.Type]int, error) {
	candidates := make(map[hir.Type]taggedCType)
	consider := func(value hir.Type) {
		if element, option := hir.OptionElement(value); option {
			if isOptionI32(value) || !supportedTaggedPayload(element, structByType, resourceByType, arrayByType) {
				return
			}
			_, needsDrop := resourceByType[element]
			candidates[value] = taggedCType{
				typ: value, kind: taggedOption, payloads: []hir.Type{element}, needsDrop: needsDrop,
			}
			return
		}
		success, failure, result := hir.ResultElements(value)
		if !result || isResultI32I32(value) {
			return
		}
		if !supportedTaggedPayload(success, structByType, resourceByType, arrayByType) ||
			!supportedTaggedPayload(failure, structByType, resourceByType, arrayByType) {
			return
		}
		_, successNeedsDrop := resourceByType[success]
		_, failureNeedsDrop := resourceByType[failure]
		candidates[value] = taggedCType{
			typ: value, kind: taggedResult, payloads: []hir.Type{success, failure},
			needsDrop: successNeedsDrop || failureNeedsDrop,
		}
	}
	for _, function := range program.Functions {
		consider(function.ResultType)
		for _, value := range function.ParameterTypes {
			consider(value)
		}
		for _, resource := range function.Resources {
			consider(resource.Type)
		}
		for _, block := range function.Blocks {
			for _, operation := range block.Operations {
				consider(operation.Type)
				consider(operation.OwnerType)
			}
		}
	}
	orderedTypes := make([]hir.Type, 0, len(candidates))
	for value := range candidates {
		orderedTypes = append(orderedTypes, value)
	}
	slices.SortFunc(orderedTypes, func(left, right hir.Type) int {
		return strings.Compare(string(left), string(right))
	})
	result := make([]taggedCType, 0, len(orderedTypes))
	byType := make(map[hir.Type]int, len(orderedTypes))
	for _, value := range orderedTypes {
		candidate := candidates[value]
		fields := []layout.Field{{
			Name: "tag",
			Type: layout.Type{Kind: layout.KindLanguageScalar, Name: "u32"},
		}}
		for index, payload := range candidate.payloads {
			payloadType, ok := taggedPayloadLayout(payload, structures, structByType, resources, resourceByType)
			if !ok {
				return nil, nil, fmt.Errorf("tagged type %q has unsupported payload %q", value, payload)
			}
			fields = append(fields, layout.Field{Name: fmt.Sprintf("p_%d", index), Type: payloadType})
		}
		computed, err := engine.Of(layout.Type{Kind: layout.KindStruct, Fields: fields})
		if err != nil {
			return nil, nil, fmt.Errorf("compute C transport layout for tagged type %q: %w", value, err)
		}
		candidate.layout = computed
		byType[value] = len(result)
		result = append(result, candidate)
	}
	return result, byType, nil
}

func supportedTaggedPayload(value hir.Type, structByType map[hir.Type]int, resourceByType map[hir.Type]int, arrayByType map[hir.Type]int) bool {
	value = canonicalCValueType(value)
	if value == hir.SharedBorrow(hir.TypeStr) {
		return true
	}
	if element, _, borrow := hir.BorrowElement(value); borrow {
		element = canonicalCValueType(element)
		if element == hir.TypeU8 || element == hir.TypeI32 || element == hir.TypeU32 || element == hir.TypeUSize {
			return true
		}
		if _, structure := structByType[element]; structure {
			return true
		}
		if _, array := arrayByType[element]; array {
			return true
		}
		_, resource := resourceByType[element]
		return resource
	}
	if value == hir.TypeU8 || value == hir.TypeI32 || value == hir.TypeU32 || value == hir.TypeUSize {
		return true
	}
	if _, structure := structByType[value]; structure {
		return true
	}
	_, resource := resourceByType[value]
	return resource
}

func taggedPayloadLayout(
	value hir.Type,
	structures []structCType,
	structByType map[hir.Type]int,
	resources []resourceCType,
	resourceByType map[hir.Type]int,
) (layout.Type, bool) {
	value = canonicalCValueType(value)
	if value == hir.SharedBorrow(hir.TypeStr) {
		return layout.Type{
			Kind: layout.KindStruct,
			Fields: []layout.Field{
				{Name: "data", Type: layout.Type{Kind: layout.KindPointer}},
				{Name: "length", Type: layout.Type{Kind: layout.KindLanguageScalar, Name: "usize"}},
			},
		}, true
	}
	if _, _, borrow := hir.BorrowElement(value); borrow {
		return layout.Type{Kind: layout.KindPointer}, true
	}
	if value == hir.TypeU8 || value == hir.TypeI32 || value == hir.TypeU32 || value == hir.TypeUSize {
		return layout.Type{Kind: layout.KindLanguageScalar, Name: string(value)}, true
	}
	if index, found := structByType[value]; found {
		return structures[index].layoutType, true
	}
	index, found := resourceByType[value]
	if !found {
		return layout.Type{}, false
	}
	return resources[index].layoutType, true
}

func (e *emitter) supportedNeedsDropType(value hir.Type) bool {
	if _, direct := e.resourceByType[value]; direct {
		return true
	}
	tagged, found := e.taggedType(value)
	return found && tagged.needsDrop
}

func (e *emitter) taggedType(value hir.Type) (*taggedCType, bool) {
	index, found := e.taggedByType[value]
	if !found {
		return nil, false
	}
	return &e.taggedTypes[index], true
}

func (e *emitter) optionCSpec(value hir.Type) (hir.Type, string, bool) {
	if isOptionI32(value) {
		return hir.TypeI32, "scalar-option", true
	}
	tagged, found := e.taggedType(value)
	if !found || tagged.kind != taggedOption || len(tagged.payloads) != 1 {
		return hir.TypeInvalid, "", false
	}
	feature := "copy-option"
	if tagged.needsDrop {
		feature = "resource-option"
	}
	return tagged.payloads[0], feature, true
}

func (e *emitter) resultCSpec(value hir.Type) (hir.Type, hir.Type, string, bool) {
	if isResultI32I32(value) {
		return hir.TypeI32, hir.TypeI32, "scalar-result", true
	}
	tagged, found := e.taggedType(value)
	if !found || tagged.kind != taggedResult || len(tagged.payloads) != 2 {
		return hir.TypeInvalid, hir.TypeInvalid, "", false
	}
	feature := "copy-result"
	if tagged.needsDrop {
		feature = "resource-result"
	}
	return tagged.payloads[0], tagged.payloads[1], feature, true
}

func (e *emitter) emitTaggedTypes(output *strings.Builder) error {
	for index, tagged := range e.taggedTypes {
		output.WriteString("typedef struct {\n")
		output.WriteString("    uint32_t tag;\n")
		for payloadIndex, payload := range tagged.payloads {
			cType, err := e.cValueType(payload)
			if err != nil {
				return fmt.Errorf("tagged type %q payload %d: %w", tagged.typ, payloadIndex, err)
			}
			fmt.Fprintf(output, "    %s p_%d;\n", cType, payloadIndex)
		}
		fmt.Fprintf(output, "} fy_tagged_%d;\n", index)
		fmt.Fprintf(output, "_Static_assert(sizeof(fy_tagged_%d) == UINT64_C(%d), \"fairyc tagged %d size\");\n", index, tagged.layout.SizeBytes, index)
		fmt.Fprintf(output, "_Static_assert(_Alignof(fy_tagged_%d) == UINT64_C(%d), \"fairyc tagged %d alignment\");\n", index, tagged.layout.AlignmentBytes, index)
		for fieldIndex, field := range tagged.layout.Fields {
			fieldName := "tag"
			if fieldIndex > 0 {
				fieldName = fmt.Sprintf("p_%d", fieldIndex-1)
			}
			fmt.Fprintf(output, "_Static_assert(offsetof(fy_tagged_%d, %s) == UINT64_C(%d), \"fairyc tagged %d field %d offset\");\n", index, fieldName, field.OffsetBytes, index, fieldIndex)
		}
		output.WriteByte('\n')
	}
	return nil
}
