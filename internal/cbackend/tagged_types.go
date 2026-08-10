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
	functions []mir.Function,
	structures []structCType,
	structByType map[hir.Type]int,
	enumerations []enumCType,
	enumByType map[hir.Type]int,
	resources []resourceCType,
	resourceByType map[hir.Type]int,
	arrayByType map[hir.Type]int,
	engine *layout.Engine,
) ([]taggedCType, map[hir.Type]int, error) {
	candidates := make(map[hir.Type]taggedCType)
	consider := func(value hir.Type) {
		if element, option := hir.OptionElement(value); option {
			if isOptionI32(value) || !supportedTaggedPayload(element, true, structByType, enumByType, resourceByType, arrayByType) {
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
		if !supportedTaggedPayload(success, true, structByType, enumByType, resourceByType, arrayByType) ||
			!supportedTaggedPayload(failure, true, structByType, enumByType, resourceByType, arrayByType) {
			return
		}
		_, successNeedsDrop := resourceByType[success]
		_, failureNeedsDrop := resourceByType[failure]
		candidates[value] = taggedCType{
			typ: value, kind: taggedResult, payloads: []hir.Type{success, failure},
			needsDrop: successNeedsDrop || failureNeedsDrop,
		}
	}
	for _, function := range functions {
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
	for _, structure := range structures {
		for _, field := range structure.definition.Fields {
			consider(field.Type)
		}
	}
	for _, enumeration := range enumerations {
		for _, variant := range enumeration.definition.Variants {
			for _, field := range variant.Fields {
				consider(field.Type)
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
			payloadType, ok := taggedPayloadLayout(payload, structures, structByType, enumerations, enumByType, resources, resourceByType, arrayByType)
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

func copyScalarTaggedAggregateLayout(value hir.Type) (layout.Type, bool) {
	tag := layout.Field{Name: "tag", Type: layout.Type{Kind: layout.KindLanguageScalar, Name: "u32"}}
	if element, option := hir.OptionElement(value); option {
		payload, found := copyScalarTaggedPayloadLayout(element)
		if !found {
			return layout.Type{}, false
		}
		name := "p_0"
		if isOptionI32(value) {
			name = "payload"
		}
		return layout.Type{Kind: layout.KindStruct, Fields: []layout.Field{
			tag,
			{Name: name, Type: payload},
		}}, true
	}
	success, failure, result := hir.ResultElements(value)
	if !result {
		return layout.Type{}, false
	}
	successLayout, successFound := copyScalarTaggedPayloadLayout(success)
	failureLayout, failureFound := copyScalarTaggedPayloadLayout(failure)
	if !successFound || !failureFound {
		return layout.Type{}, false
	}
	if isResultI32I32(value) {
		return layout.Type{Kind: layout.KindStruct, Fields: []layout.Field{
			tag,
			{Name: "payload", Type: successLayout},
		}}, true
	}
	return layout.Type{Kind: layout.KindStruct, Fields: []layout.Field{
		tag,
		{Name: "p_0", Type: successLayout},
		{Name: "p_1", Type: failureLayout},
	}}, true
}

func copyScalarTaggedPayloadLayout(value hir.Type) (layout.Type, bool) {
	value = canonicalCValueType(value)
	switch value {
	case hir.TypeU8, hir.TypeI32, hir.TypeU32, hir.TypeUSize:
		return layout.Type{Kind: layout.KindLanguageScalar, Name: string(value)}, true
	default:
		return layout.Type{}, false
	}
}

func supportedTaggedPayload(value hir.Type, allowEnum bool, structByType map[hir.Type]int, enumByType map[hir.Type]int, resourceByType map[hir.Type]int, arrayByType map[hir.Type]int) bool {
	value = canonicalCValueType(value)
	if isOptionI32(value) || isResultI32I32(value) {
		return true
	}
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
	if _, enumeration := enumByType[value]; allowEnum && enumeration {
		return true
	}
	if _, array := arrayByType[value]; array {
		return true
	}
	_, resource := resourceByType[value]
	return resource
}

func taggedPayloadLayout(
	value hir.Type,
	structures []structCType,
	structByType map[hir.Type]int,
	enumerations []enumCType,
	enumByType map[hir.Type]int,
	resources []resourceCType,
	resourceByType map[hir.Type]int,
	arrayByType map[hir.Type]int,
) (layout.Type, bool) {
	value = canonicalCValueType(value)
	if tagged, found := copyScalarTaggedAggregateLayout(value); found {
		return tagged, true
	}
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
	if index, found := enumByType[value]; found {
		return enumerations[index].layoutType, true
	}
	if _, found := arrayByType[value]; found {
		return scalarArrayLayoutType(value)
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

func (e *emitter) emitTaggedTypes(output *strings.Builder, directCopyScalar bool) error {
	for index, tagged := range e.taggedTypes {
		_, direct := copyScalarTaggedAggregateLayout(tagged.typ)
		if direct != directCopyScalar {
			continue
		}
		if !direct && e.taggedParticipatesInAggregateGraph(index) {
			continue
		}
		if err := e.emitTaggedType(output, index); err != nil {
			return err
		}
	}
	return nil
}

func (e *emitter) taggedParticipatesInAggregateGraph(index int) bool {
	tagged := e.taggedTypes[index]
	if _, direct := copyScalarTaggedAggregateLayout(tagged.typ); direct || tagged.needsDrop {
		return false
	}
	foundAggregate := false
	for _, payload := range tagged.payloads {
		payload = canonicalCValueType(payload)
		if _, found := e.structByType[payload]; found {
			foundAggregate = true
			continue
		}
		if _, found := e.enumByType[payload]; found {
			foundAggregate = true
			continue
		}
		if payload != hir.TypeU8 && payload != hir.TypeI32 && payload != hir.TypeU32 && payload != hir.TypeUSize {
			return false
		}
	}
	return foundAggregate
}

func (e *emitter) emitTaggedType(output *strings.Builder, index int) error {
	tagged := e.taggedTypes[index]
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
	return nil
}
