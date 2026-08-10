package cbackend

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/layout"
	"github.com/0disoft/fairyc/internal/mir"
)

type aggregateLayoutResolver struct {
	engine       *layout.Engine
	structures   map[hir.Type]mir.StructType
	enumerations map[hir.Type]mir.EnumType
	types        map[hir.Type]layout.Type
	results      map[hir.Type]layout.Result
	state        map[hir.Type]uint8
}

func prepareAggregateTypes(
	structDefinitions []mir.StructType,
	enumDefinitions []mir.EnumType,
	engine *layout.Engine,
) ([]structCType, map[hir.Type]int, []enumCType, map[hir.Type]int, error) {
	resolver := aggregateLayoutResolver{
		engine: engine, structures: make(map[hir.Type]mir.StructType, len(structDefinitions)),
		enumerations: make(map[hir.Type]mir.EnumType, len(enumDefinitions)),
		types:        make(map[hir.Type]layout.Type), results: make(map[hir.Type]layout.Result),
		state: make(map[hir.Type]uint8),
	}
	for _, definition := range structDefinitions {
		if _, duplicate := resolver.structures[definition.Type]; duplicate {
			return nil, nil, nil, nil, fmt.Errorf("duplicate C struct type %q", definition.Type)
		}
		resolver.structures[definition.Type] = definition
	}
	for _, definition := range enumDefinitions {
		if _, duplicate := resolver.enumerations[definition.Type]; duplicate {
			return nil, nil, nil, nil, fmt.Errorf("duplicate C enum type %q", definition.Type)
		}
		resolver.enumerations[definition.Type] = definition
	}
	allTypes := make([]hir.Type, 0, len(structDefinitions)+len(enumDefinitions))
	for typ := range resolver.structures {
		allTypes = append(allTypes, typ)
	}
	for typ := range resolver.enumerations {
		if _, collision := resolver.structures[typ]; collision {
			return nil, nil, nil, nil, fmt.Errorf("aggregate type %q is both struct and enum", typ)
		}
		allTypes = append(allTypes, typ)
	}
	slices.SortFunc(allTypes, func(left, right hir.Type) int { return strings.Compare(string(left), string(right)) })
	for _, typ := range allTypes {
		if _, err := resolver.resolve(typ); err != nil {
			return nil, nil, nil, nil, err
		}
	}

	orderedStructs, err := orderStructDefinitions(structDefinitions)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	structures := make([]structCType, 0, len(orderedStructs))
	structByType := make(map[hir.Type]int, len(orderedStructs))
	publicNames := make(map[string]hir.Type)
	for _, definition := range orderedStructs {
		if definition.ReprC {
			if !validCABIRecordIdentifier(definition.Name) {
				return nil, nil, nil, nil, CapabilityError{Feature: "repr(C) struct name " + strconv.Quote(definition.Name)}
			}
			if previous, duplicate := publicNames[definition.Name]; duplicate {
				return nil, nil, nil, nil, fmt.Errorf("repr(C) structs %q and %q share C tag %q", previous, definition.Type, definition.Name)
			}
			publicNames[definition.Name] = definition.Type
		}
		fieldIndex := make(map[string]int, len(definition.Fields))
		for index, field := range definition.Fields {
			if definition.ReprC && !validCABIMemberIdentifier(field.Name) {
				return nil, nil, nil, nil, CapabilityError{Feature: "repr(C) struct " + strconv.Quote(definition.Name) + " field name " + strconv.Quote(field.Name)}
			}
			fieldIndex[field.Name] = index
		}
		structByType[definition.Type] = len(structures)
		structures = append(structures, structCType{
			definition: definition, layout: resolver.results[definition.Type],
			layoutType: resolver.types[definition.Type], fieldIndex: fieldIndex,
		})
	}

	orderedEnums := slices.Clone(enumDefinitions)
	slices.SortFunc(orderedEnums, func(left, right mir.EnumType) int {
		return strings.Compare(string(left.Type), string(right.Type))
	})
	enumerations := make([]enumCType, 0, len(orderedEnums))
	enumByType := make(map[hir.Type]int, len(orderedEnums))
	for _, definition := range orderedEnums {
		value := enumCType{
			definition: definition, layout: resolver.results[definition.Type], layoutType: resolver.types[definition.Type],
			variantIndex: make(map[string]int, len(definition.Variants)), fieldIndex: make([]map[string]int, len(definition.Variants)),
		}
		for variantIndex, variant := range definition.Variants {
			value.variantIndex[variant.Name] = variantIndex
			value.fieldIndex[variantIndex] = make(map[string]int, len(variant.Fields))
			for fieldIndex, field := range variant.Fields {
				value.fieldIndex[variantIndex][field.Name] = fieldIndex
			}
		}
		enumByType[definition.Type] = len(enumerations)
		enumerations = append(enumerations, value)
	}
	return structures, structByType, enumerations, enumByType, nil
}

func (r *aggregateLayoutResolver) resolve(typ hir.Type) (layout.Type, error) {
	if value, found := r.types[typ]; found {
		return value, nil
	}
	if r.state[typ] == 1 {
		if _, structure := r.structures[typ]; structure {
			return layout.Type{}, CapabilityError{Feature: "recursive struct dependency involving " + strconv.Quote(string(typ))}
		}
		return layout.Type{}, CapabilityError{Feature: "recursive aggregate dependency involving " + strconv.Quote(string(typ))}
	}
	r.state[typ] = 1
	var value layout.Type
	var err error
	if definition, found := r.structures[typ]; found {
		value, err = r.resolveStruct(definition)
	} else if definition, found := r.enumerations[typ]; found {
		value, err = r.resolveEnum(definition)
	} else {
		err = fmt.Errorf("missing aggregate type %q", typ)
	}
	if err != nil {
		return layout.Type{}, err
	}
	computed, err := r.engine.Of(value)
	if err != nil {
		return layout.Type{}, fmt.Errorf("compute C transport layout for aggregate %q: %w", typ, err)
	}
	r.state[typ] = 2
	r.types[typ] = value
	r.results[typ] = computed
	return value, nil
}

func (r *aggregateLayoutResolver) resolveStruct(definition mir.StructType) (layout.Type, error) {
	if len(definition.Fields) == 0 {
		return layout.Type{}, CapabilityError{Feature: "empty struct type " + strconv.Quote(string(definition.Type))}
	}
	fields := make([]layout.Field, 0, len(definition.Fields))
	for index, field := range definition.Fields {
		fieldType, err := r.resolveStructField(field.Type, definition.ReprC)
		if err != nil {
			return layout.Type{}, CapabilityError{Feature: "struct type " + strconv.Quote(string(definition.Type)) + " field " + strconv.Quote(field.Name) + " of type " + strconv.Quote(string(field.Type)) + ": " + err.Error()}
		}
		name := fmt.Sprintf("f_%d", index)
		if definition.ReprC {
			name = field.Name
		}
		fields = append(fields, layout.Field{Name: name, Type: fieldType})
	}
	return layout.Type{Kind: layout.KindStruct, Fields: fields}, nil
}

func (r *aggregateLayoutResolver) resolveStructField(value hir.Type, reprC bool) (layout.Type, error) {
	value = canonicalCValueType(value)
	if nested, found := r.structures[value]; found {
		if !reprC || !nested.ReprC {
			return layout.Type{}, fmt.Errorf("nested struct type %q is not repr(C)", value)
		}
		return r.resolve(value)
	}
	if _, found := r.enumerations[value]; found {
		if reprC {
			return layout.Type{}, fmt.Errorf("inert enum type %q is not a repr(C) field", value)
		}
		return r.resolve(value)
	}
	if tagged, found := scalarTaggedAggregateLayout(value); found && !reprC {
		return tagged, nil
	}
	switch value {
	case hir.TypeU8, hir.TypeI32, hir.TypeU32, hir.TypeUSize:
		return layout.Type{Kind: layout.KindLanguageScalar, Name: string(value)}, nil
	default:
		if transport, found := cScalarTransportForType(value); found && reprC {
			return layout.Type{Kind: layout.KindCScalar, Name: transport.targetName}, nil
		}
		if _, _, pointer := hir.RawPointerElement(value); pointer {
			return layout.Type{Kind: layout.KindPointer}, nil
		}
		return layout.Type{}, fmt.Errorf("unsupported struct field type %q", value)
	}
}

func (r *aggregateLayoutResolver) resolveEnum(definition mir.EnumType) (layout.Type, error) {
	if len(definition.Variants) == 0 {
		return layout.Type{}, CapabilityError{Feature: "empty enum type " + strconv.Quote(string(definition.Type))}
	}
	unionFields := []layout.Field{{Name: "unused", Type: layout.Type{Kind: layout.KindLanguageScalar, Name: "u8"}}}
	for variantIndex, variant := range definition.Variants {
		if len(variant.Fields) == 0 {
			continue
		}
		fields := make([]layout.Field, 0, len(variant.Fields))
		for fieldIndex, field := range variant.Fields {
			fieldType, err := r.resolveEnumField(field.Type)
			if err != nil {
				return layout.Type{}, CapabilityError{Feature: "enum type " + strconv.Quote(string(definition.Type)) + " variant " + strconv.Quote(variant.Name) + " field " + strconv.Quote(field.Name) + " of type " + strconv.Quote(string(field.Type))}
			}
			fields = append(fields, layout.Field{Name: fmt.Sprintf("f_%d", fieldIndex), Type: fieldType})
		}
		unionFields = append(unionFields, layout.Field{Name: fmt.Sprintf("v_%d", variantIndex), Type: layout.Type{Kind: layout.KindStruct, Fields: fields}})
	}
	return layout.Type{Kind: layout.KindStruct, Fields: []layout.Field{
		{Name: "tag", Type: layout.Type{Kind: layout.KindLanguageScalar, Name: "u32"}},
		{Name: "payload", Type: layout.Type{Kind: layout.KindUnion, Fields: unionFields}},
	}}, nil
}

func (r *aggregateLayoutResolver) resolveEnumField(value hir.Type) (layout.Type, error) {
	value = canonicalCValueType(value)
	if _, found := r.structures[value]; found {
		return r.resolve(value)
	}
	if _, found := r.enumerations[value]; found {
		return r.resolve(value)
	}
	if tagged, found := scalarTaggedAggregateLayout(value); found {
		return tagged, nil
	}
	switch value {
	case hir.TypeU8, hir.TypeI32, hir.TypeU32, hir.TypeUSize:
		return layout.Type{Kind: layout.KindLanguageScalar, Name: string(value)}, nil
	default:
		if _, _, pointer := hir.RawPointerElement(value); pointer {
			return layout.Type{Kind: layout.KindPointer}, nil
		}
		return layout.Type{}, fmt.Errorf("unsupported enum field type %q", value)
	}
}

func scalarTaggedAggregateLayout(value hir.Type) (layout.Type, bool) {
	if !isOptionI32(value) && !isResultI32I32(value) {
		return layout.Type{}, false
	}
	return layout.Type{Kind: layout.KindStruct, Fields: []layout.Field{
		{Name: "tag", Type: layout.Type{Kind: layout.KindLanguageScalar, Name: "u32"}},
		{Name: "payload", Type: layout.Type{Kind: layout.KindLanguageScalar, Name: "i32"}},
	}}, true
}

func orderStructDefinitions(definitions []mir.StructType) ([]mir.StructType, error) {
	sorted := slices.Clone(definitions)
	slices.SortFunc(sorted, func(left, right mir.StructType) int { return strings.Compare(string(left.Type), string(right.Type)) })
	byType := make(map[hir.Type]mir.StructType, len(sorted))
	for _, definition := range sorted {
		if _, duplicate := byType[definition.Type]; duplicate {
			return nil, fmt.Errorf("duplicate C struct type %q", definition.Type)
		}
		byType[definition.Type] = definition
	}
	state := make(map[hir.Type]uint8, len(sorted))
	ordered := make([]mir.StructType, 0, len(sorted))
	var visit func(mir.StructType) error
	visit = func(definition mir.StructType) error {
		switch state[definition.Type] {
		case 1:
			return CapabilityError{Feature: "recursive struct dependency involving " + strconv.Quote(string(definition.Type))}
		case 2:
			return nil
		}
		state[definition.Type] = 1
		dependencies := make([]hir.Type, 0, len(definition.Fields))
		for _, field := range definition.Fields {
			if _, nested := byType[field.Type]; nested {
				dependencies = append(dependencies, field.Type)
			}
		}
		slices.SortFunc(dependencies, func(left, right hir.Type) int { return strings.Compare(string(left), string(right)) })
		for _, dependency := range slices.Compact(dependencies) {
			if err := visit(byType[dependency]); err != nil {
				return err
			}
		}
		state[definition.Type] = 2
		ordered = append(ordered, definition)
		return nil
	}
	for _, definition := range sorted {
		if err := visit(definition); err != nil {
			return nil, err
		}
	}
	return ordered, nil
}

type aggregateEmissionNode struct {
	typ         hir.Type
	structure   int
	enumeration int
}

func (e *emitter) aggregateEmissionOrder() ([]aggregateEmissionNode, error) {
	nodes := make(map[hir.Type]aggregateEmissionNode, len(e.structTypes)+len(e.enumTypes))
	for index, value := range e.structTypes {
		nodes[value.definition.Type] = aggregateEmissionNode{typ: value.definition.Type, structure: index, enumeration: -1}
	}
	for index, value := range e.enumTypes {
		nodes[value.definition.Type] = aggregateEmissionNode{typ: value.definition.Type, structure: -1, enumeration: index}
	}
	roots := make([]hir.Type, 0, len(nodes))
	for _, value := range e.structTypes {
		roots = append(roots, value.definition.Type)
	}
	for _, value := range e.enumTypes {
		roots = append(roots, value.definition.Type)
	}
	state := make(map[hir.Type]uint8, len(nodes))
	ordered := make([]aggregateEmissionNode, 0, len(nodes))
	var visit func(hir.Type) error
	visit = func(typ hir.Type) error {
		if state[typ] == 2 {
			return nil
		}
		if state[typ] == 1 {
			return CapabilityError{Feature: "recursive aggregate emission dependency involving " + strconv.Quote(string(typ))}
		}
		state[typ] = 1
		node := nodes[typ]
		dependencies := make([]hir.Type, 0)
		if node.structure >= 0 {
			for _, field := range e.structTypes[node.structure].definition.Fields {
				if _, found := nodes[canonicalCValueType(field.Type)]; found {
					dependencies = append(dependencies, canonicalCValueType(field.Type))
				}
			}
		} else {
			for _, variant := range e.enumTypes[node.enumeration].definition.Variants {
				for _, field := range variant.Fields {
					if _, found := nodes[canonicalCValueType(field.Type)]; found {
						dependencies = append(dependencies, canonicalCValueType(field.Type))
					}
				}
			}
		}
		slices.SortFunc(dependencies, func(left, right hir.Type) int { return strings.Compare(string(left), string(right)) })
		for _, dependency := range slices.Compact(dependencies) {
			if err := visit(dependency); err != nil {
				return err
			}
		}
		state[typ] = 2
		ordered = append(ordered, node)
		return nil
	}
	for _, typ := range roots {
		if err := visit(typ); err != nil {
			return nil, err
		}
	}
	return ordered, nil
}

func (e *emitter) emitAggregateTypes(output *strings.Builder) error {
	ordered, err := e.aggregateEmissionOrder()
	if err != nil {
		return err
	}
	for _, node := range ordered {
		if node.structure >= 0 {
			if err := e.emitStructType(output, node.structure); err != nil {
				return err
			}
		} else if err := e.emitEnumType(output, node.enumeration); err != nil {
			return err
		}
	}
	return nil
}

func (e *emitter) emitStructType(output *strings.Builder, index int) error {
	structure := e.structTypes[index]
	if structure.definition.ReprC {
		fmt.Fprintf(output, "struct %s {\n", structure.definition.Name)
	} else {
		output.WriteString("typedef struct {\n")
	}
	for fieldIndex, field := range structure.definition.Fields {
		cType, err := e.cValueType(field.Type)
		if err != nil {
			return fmt.Errorf("emit struct %q field %q: %w", structure.definition.Type, field.Name, err)
		}
		fmt.Fprintf(output, "    %s %s;\n", cType, e.structCFieldName(index, fieldIndex))
	}
	cType := e.structCTypeName(index)
	if structure.definition.ReprC {
		output.WriteString("};\n")
	} else {
		fmt.Fprintf(output, "} %s;\n", cType)
	}
	fmt.Fprintf(output, "_Static_assert(sizeof(%s) == UINT64_C(%d), \"fairyc struct %d size\");\n", cType, structure.layout.SizeBytes, index)
	fmt.Fprintf(output, "_Static_assert(_Alignof(%s) == UINT64_C(%d), \"fairyc struct %d alignment\");\n", cType, structure.layout.AlignmentBytes, index)
	for fieldIndex, field := range structure.layout.Fields {
		fmt.Fprintf(output, "_Static_assert(offsetof(%s, %s) == UINT64_C(%d), \"fairyc struct %d field %d offset\");\n", cType, e.structCFieldName(index, fieldIndex), field.OffsetBytes, index, fieldIndex)
	}
	output.WriteByte('\n')
	return nil
}

func (e *emitter) emitEnumType(output *strings.Builder, enumIndex int) error {
	enumeration := e.enumTypes[enumIndex]
	output.WriteString("typedef union {\n    uint8_t unused;\n")
	for variantIndex, variant := range enumeration.definition.Variants {
		if len(variant.Fields) == 0 {
			continue
		}
		output.WriteString("    struct {\n")
		for fieldIndex, field := range variant.Fields {
			cType, err := e.enumPayloadCType(field.Type)
			if err != nil {
				return fmt.Errorf("emit enum %q variant %q field %q: %w", enumeration.definition.Type, variant.Name, field.Name, err)
			}
			fmt.Fprintf(output, "        %s f_%d;\n", cType, fieldIndex)
		}
		fmt.Fprintf(output, "    } v_%d;\n", variantIndex)
	}
	fmt.Fprintf(output, "} fy_enum_payload_%d;\n", enumIndex)
	fmt.Fprintf(output, "typedef struct { uint32_t tag; fy_enum_payload_%d payload; } fy_enum_%d;\n", enumIndex, enumIndex)
	fmt.Fprintf(output, "_Static_assert(sizeof(fy_enum_%d) == UINT64_C(%d), \"fairyc enum %d size\");\n", enumIndex, enumeration.layout.SizeBytes, enumIndex)
	fmt.Fprintf(output, "_Static_assert(_Alignof(fy_enum_%d) == UINT64_C(%d), \"fairyc enum %d alignment\");\n", enumIndex, enumeration.layout.AlignmentBytes, enumIndex)
	for fieldIndex, field := range enumeration.layout.Fields {
		fmt.Fprintf(output, "_Static_assert(offsetof(fy_enum_%d, %s) == UINT64_C(%d), \"fairyc enum %d field %d offset\");\n", enumIndex, field.Name, field.OffsetBytes, enumIndex, fieldIndex)
	}
	output.WriteByte('\n')
	return nil
}
