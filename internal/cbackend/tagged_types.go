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
	typ         hir.Type
	kind        string
	payloads    []hir.Type
	layout      layout.Result
	needsDrop   bool
	owningDepth int
}

func sharedBorrowOptionNiche(value hir.Type, structByType, resourceByType, arrayByType map[hir.Type]int) bool {
	element, option := hir.OptionElement(value)
	if !option {
		return false
	}
	borrowed, mutable, borrow := hir.BorrowElement(canonicalCValueType(element))
	if !borrow || mutable {
		return false
	}
	switch borrowed {
	case hir.TypeBool, hir.TypeU8, hir.TypeI32, hir.TypeI64, hir.TypeU32, hir.TypeUSize:
		return true
	default:
		borrowed = canonicalCValueType(borrowed)
		_, structure := structByType[borrowed]
		_, resource := resourceByType[borrowed]
		_, array := arrayByType[borrowed]
		return structure || resource || array
	}
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
	structByType = closureFreeStructIndex(structures, structByType)
	candidates := make(map[hir.Type]taggedCType)
	// Payload support is fixed for this preparation pass. Remember rejected types
	// too: shared Result branches must not repeatedly expand an unsupported graph.
	considered := make(map[hir.Type]struct{})
	var consider func(hir.Type)
	consider = func(value hir.Type) {
		value = canonicalCValueType(value)
		if _, found := considered[value]; found {
			return
		}
		considered[value] = struct{}{}
		// Extended scalars are admitted only at the permitted shallow roots;
		// retained sibling candidates cannot authorize a nested wrapper.
		_, flatTransport := flatTransportOptionPayload(value)
		_, flatResult := flatTransportResultPayload(value)
		_, nestedOption := nestedTransportOptionPayload(value)
		if !flatTransport && !flatResult && !nestedOption && hir.ContainsTaggedType(value, extendedOptionScalar) {
			return
		}
		if element, option := hir.OptionElement(value); option {
			if isOptionI32(value) {
				return
			}
			element = canonicalCValueType(element)
			supported := flatTransport || supportedTaggedPayload(element, true, structByType, enumByType, resourceByType, arrayByType)
			_, needsDrop := resourceByType[element]
			if !supported {
				consider(element)
				nested, found := candidates[element]
				if !found {
					return
				}
				supported = true
				needsDrop = nested.needsDrop
			}
			if !supported {
				return
			}

			candidate := taggedCType{typ: value, kind: taggedOption, payloads: []hir.Type{element}, needsDrop: needsDrop}
			if needsDrop {
				candidate.owningDepth = 1 + candidates[element].owningDepth
				if candidate.owningDepth > maxOwningTagDepth {
					return
				}
			}
			candidates[value] = candidate
			return
		}
		success, failure, result := hir.ResultElements(value)
		if !result || isResultI32I32(value) {
			return
		}
		success = canonicalCValueType(success)
		failure = canonicalCValueType(failure)
		successSupported := flatResult || supportedTaggedPayload(success, true, structByType, enumByType, resourceByType, arrayByType)
		failureSupported := supportedTaggedPayload(failure, true, structByType, enumByType, resourceByType, arrayByType)
		if !successSupported {
			consider(success)
			_, found := candidates[success]
			successSupported = found
		}
		if !failureSupported {
			consider(failure)
			_, found := candidates[failure]
			failureSupported = found
		}
		if !successSupported || !failureSupported {
			return
		}
		_, successNeedsDrop := resourceByType[success]
		_, failureNeedsDrop := resourceByType[failure]
		candidates[value] = taggedCType{
			typ: value, kind: taggedResult, payloads: []hir.Type{success, failure},
			needsDrop: successNeedsDrop || failureNeedsDrop || candidates[success].needsDrop || candidates[failure].needsDrop,
		}
		candidate := candidates[value]
		if candidate.needsDrop {
			candidate.owningDepth = 1 + max(candidates[success].owningDepth, candidates[failure].owningDepth)
			if candidate.owningDepth > maxOwningTagDepth {
				delete(candidates, value)
				return
			}
			candidates[value] = candidate
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
		return strings.Compare(left.String(), right.String())
	})
	result := make([]taggedCType, 0, len(orderedTypes))
	byType := make(map[hir.Type]int, len(orderedTypes))
	layoutTypes := make(map[hir.Type]layout.Type, len(orderedTypes))
	layoutState := make(map[hir.Type]uint8, len(orderedTypes))
	var resolveLayout func(hir.Type) (layout.Type, error)
	resolveLayout = func(value hir.Type) (layout.Type, error) {
		if resolved, found := layoutTypes[value]; found {
			return resolved, nil
		}
		if layoutState[value] == 1 {
			return layout.Type{}, fmt.Errorf("recursive tagged layout dependency involving %q", value)
		}
		candidate, found := candidates[value]
		if !found {
			return layout.Type{}, fmt.Errorf("missing tagged layout candidate %q", value)
		}
		layoutState[value] = 1
		fields := make([]layout.Field, 0, len(candidate.payloads)+1)
		if !sharedBorrowOptionNiche(value, structByType, resourceByType, arrayByType) {
			fields = append(fields, layout.Field{
				Name: "tag",
				Type: layout.Type{Kind: layout.KindLanguageScalar, Name: "u32"},
			})
		}
		for index, payload := range candidate.payloads {
			payload = canonicalCValueType(payload)
			var payloadType layout.Type
			var ok bool
			if _, nested := candidates[payload]; nested {
				var err error
				payloadType, err = resolveLayout(payload)
				if err != nil {
					return layout.Type{}, err
				}
				ok = true
			} else if element, flat := flatTransportOptionPayload(value); flat && element == payload {
				// Keep the shared Result/aggregate payload layout predicates closed.
				payloadType, ok = layout.Type{Kind: layout.KindLanguageScalar, Name: payload.String()}, true
			} else if success, flat := flatTransportResultPayload(value); flat && index == 0 && success == payload {
				// This success leaf does not widen shared aggregate/error payloads.
				payloadType, ok = layout.Type{Kind: layout.KindLanguageScalar, Name: payload.String()}, true
			} else {
				payloadType, ok = taggedPayloadLayout(payload, structures, structByType, enumerations, enumByType, resources, resourceByType, arrayByType)
			}
			if !ok {
				return layout.Type{}, fmt.Errorf("tagged type %q has unsupported payload %q", value, payload)
			}
			fields = append(fields, layout.Field{Name: fmt.Sprintf("p_%d", index), Type: payloadType})
		}
		resolved := layout.Type{Kind: layout.KindStruct, Fields: fields}
		computed, err := engine.Of(resolved)
		if err != nil {
			return layout.Type{}, fmt.Errorf("compute C transport layout for tagged type %q: %w", value, err)
		}
		candidate.layout = computed
		candidates[value] = candidate
		layoutState[value] = 2
		layoutTypes[value] = resolved
		return resolved, nil
	}
	for _, value := range orderedTypes {
		if _, err := resolveLayout(value); err != nil {
			return nil, nil, err
		}
		candidate := candidates[value]
		byType[value] = len(result)
		result = append(result, candidate)
	}
	return result, byType, nil
}

func extendedOptionScalar(value hir.Type) bool {
	return hir.IsIntegerTransportScalar(value) || hir.IsFloatScalar(value)
}

func flatTransportOptionPayload(value hir.Type) (hir.Type, bool) {
	payload, option := hir.OptionElement(value)
	return payload, option && extendedOptionScalar(payload)
}

func nestedTransportOptionPayload(value hir.Type) (hir.Type, bool) {
	inner, outerOption := hir.OptionElement(value)
	leaf, innerOption := hir.OptionElement(inner)
	return leaf, outerOption && innerOption && extendedOptionScalar(leaf)
}

func flatTransportResultPayload(value hir.Type) (hir.Type, bool) {
	success, failure, result := hir.ResultElements(value)
	return success, result && (failure == hir.TypeU32 || failure == hir.TypeI32) && extendedOptionScalar(success)
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
	case hir.TypeU8, hir.TypeI32, hir.TypeI64, hir.TypeU32, hir.TypeUSize:
		return layout.Type{Kind: layout.KindLanguageScalar, Name: value.String()}, true
	default:
		return layout.Type{}, false
	}
}

func supportedTaggedPayload(value hir.Type, allowEnum bool, structByType map[hir.Type]int, enumByType map[hir.Type]int, resourceByType map[hir.Type]int, arrayByType map[hir.Type]int) bool {
	value = canonicalCValueType(value)
	if isOptionI32(value) || isResultI32I32(value) {
		return true
	}
	if _, view := hir.IndexedViewElement(value); value == hir.SharedBorrow(hir.TypeStr) || view {
		return true
	}
	if element, mutable, borrow := hir.BorrowElement(value); borrow {
		element = canonicalCValueType(element)
		if (element == hir.TypeBool && !mutable) || element == hir.TypeU8 || element == hir.TypeI32 || element == hir.TypeI64 || element == hir.TypeU32 || element == hir.TypeUSize {
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
	if value == hir.TypeU8 || value == hir.TypeI32 || value == hir.TypeI64 || value == hir.TypeU32 || value == hir.TypeUSize {
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
	if _, view := hir.IndexedViewElement(value); value == hir.SharedBorrow(hir.TypeStr) || view {
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
	if value == hir.TypeU8 || value == hir.TypeI32 || value == hir.TypeI64 || value == hir.TypeU32 || value == hir.TypeUSize {
		return layout.Type{Kind: layout.KindLanguageScalar, Name: value.String()}, true
	}
	if index, found := structByType[value]; found {
		return structures[index].layoutType, true
	}
	if index, found := enumByType[value]; found {
		return enumerations[index].layoutType, true
	}
	if _, found := arrayByType[value]; found {
		if hir.TransportOnlyArray(value) {
			return layout.Type{}, false
		}
		if result, ok := scalarArrayLayoutType(value); ok {
			return result, true
		}
		element, length, array := hir.FixedArrayParts(value)
		if index, ok := structByType[element]; array && ok {
			elementLayout := structures[index].layoutType
			return layout.Type{Kind: layout.KindArray, Element: &elementLayout, Length: length}, true
		}
		return layout.Type{}, false
	}
	index, found := resourceByType[value]
	if !found {
		return layout.Type{}, false
	}
	return resources[index].layoutType, true
}

func (e *emissionAnalysis) supportedNeedsDropType(value hir.Type) bool {
	if _, direct := e.resourceByType[value]; direct {
		return true
	}
	tagged, found := e.taggedType(value)
	return found && tagged.needsDrop
}

func (e *emissionAnalysis) taggedType(value hir.Type) (*taggedCType, bool) {
	index, found := e.taggedByType[value]
	if !found {
		return nil, false
	}
	return &e.taggedTypes[index], true
}

func (e *emissionAnalysis) optionCSpec(value hir.Type) (hir.Type, string, bool) {
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

func (e *emissionAnalysis) resultCSpec(value hir.Type) (hir.Type, hir.Type, string, bool) {
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

// Definitions follow payload dependencies; stable numeric IDs retain their sorted order.
func (e *emitter) emitTaggedTypes(output *strings.Builder, directCopyScalar bool) error {
	seen := make(map[int]bool)
	var visit func(int) error
	visit = func(index int) error {
		if seen[index] {
			return nil
		}
		seen[index] = true
		tagged := e.analysis.taggedTypes[index]
		_, direct := copyScalarTaggedAggregateLayout(tagged.typ)
		if direct != directCopyScalar || !direct && e.analysis.taggedParticipatesInAggregateGraph(index) {
			return nil
		}
		for _, payload := range tagged.payloads {
			if child, found := e.analysis.taggedByType[payload]; found {
				if err := visit(child); err != nil {
					return err
				}
			}
		}
		return e.emitTaggedType(output, index)
	}
	for index := range e.analysis.taggedTypes {
		if err := visit(index); err != nil {
			return err
		}
	}
	return nil
}

func (e *emissionAnalysis) taggedParticipatesInAggregateGraph(index int) bool {
	tagged := e.taggedTypes[index]
	// This root-only transport family contains a non-aggregate flat Option.
	// Emit both through the tagged dependency visitor so the child is complete.
	if _, nested := nestedTransportOptionPayload(tagged.typ); nested {
		return false
	}
	if _, direct := copyScalarTaggedAggregateLayout(tagged.typ); direct || tagged.needsDrop {
		return false
	}
	foundAggregate := false
	for _, payload := range tagged.payloads {
		payload = canonicalCValueType(payload)
		if _, found := copyScalarTaggedAggregateLayout(payload); found {
			foundAggregate = true
			continue
		}
		if nested, found := e.taggedByType[payload]; found {
			if e.taggedTypes[nested].needsDrop {
				return false
			}
			foundAggregate = true
			continue
		}
		if _, found := e.structByType[payload]; found {
			foundAggregate = true
			continue
		}
		if _, found := e.enumByType[payload]; found {
			foundAggregate = true
			continue
		}
		if payload != hir.TypeU8 && payload != hir.TypeI32 && payload != hir.TypeI64 && payload != hir.TypeU32 && payload != hir.TypeUSize {
			return false
		}
	}
	return foundAggregate
}

func (e *emitter) emitTaggedType(output *strings.Builder, index int) error {
	tagged := e.analysis.taggedTypes[index]
	output.WriteString("typedef struct {\n")
	if !sharedBorrowOptionNiche(tagged.typ, e.analysis.structByType, e.analysis.resourceByType, e.analysis.arrayByType) {
		output.WriteString("    uint32_t tag;\n")
	}
	for payloadIndex, payload := range tagged.payloads {
		cType, err := e.analysis.cValueType(payload)
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
		if sharedBorrowOptionNiche(tagged.typ, e.analysis.structByType, e.analysis.resourceByType, e.analysis.arrayByType) {
			fieldName = fmt.Sprintf("p_%d", fieldIndex)
		} else if fieldIndex > 0 {
			fieldName = fmt.Sprintf("p_%d", fieldIndex-1)
		}
		fmt.Fprintf(output, "_Static_assert(offsetof(fy_tagged_%d, %s) == UINT64_C(%d), \"fairyc tagged %d field %d offset\");\n", index, fieldName, field.OffsetBytes, index, fieldIndex)
	}
	output.WriteByte('\n')
	return nil
}
