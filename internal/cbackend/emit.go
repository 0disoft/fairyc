// Package cbackend emits the constrained Stage 0 C reference transport.
package cbackend

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/layout"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/target"
)

const (
	// DialectC11 identifies the C11 compiler dialect; target-qualified typedefs may require
	// an explicit feature-test macro and provider header within that dialect.
	DialectC11 = "c11"
	// InstrumentationDropTraceV0 identifies the closed native-conformance drop observer.
	InstrumentationDropTraceV0 = "drop-trace-v0"
)

// Profile names the target, C dialect, and closed instrumentation accepted by one emission run.
type Profile struct {
	Target          target.ID
	Dialect         string
	Instrumentation string
}

// Stage0Profile returns the only currently implemented C emission profile.
func Stage0Profile() Profile {
	return Profile{Target: target.X8664LinuxGNU, Dialect: DialectC11}
}

// Stage0DropTraceProfile returns the native-conformance profile that observes compiler-owned drops.
func Stage0DropTraceProfile() Profile {
	return Profile{
		Target: target.X8664LinuxGNU, Dialect: DialectC11,
		Instrumentation: InstrumentationDropTraceV0,
	}
}

// DropSite identifies one source-linked resource leaf that an instrumented drop may observe.
type DropSite struct {
	ID           uint32
	ResourceType string
	Source       string
	StartByte    int
	EndByte      int
}

// Unit is one deterministic generated-C translation unit plus bounded stage facts.
type Unit struct {
	Target          string
	Dialect         string
	Instrumentation string
	Entry           string
	Functions       []string
	Features        []string
	DropSites       []DropSite
	Source          string
}

// CapabilityError reports verified MIR outside the implemented C transport subset.
type CapabilityError struct {
	Function string
	Feature  string
}

func (e CapabilityError) Error() string {
	if e.Function == "" {
		return fmt.Sprintf("C backend does not support %s", e.Feature)
	}
	return fmt.Sprintf("C backend does not support %s in %s", e.Feature, e.Function)
}

type emitter struct {
	functions          []mir.Function
	bySymbol           map[hir.SymbolID]int
	foreignFunctions   []mir.ForeignFunction
	foreignBySymbol    map[hir.SymbolID]int
	entryIndex         int
	structTypes        []structCType
	structByType       map[hir.Type]int
	enumTypes          []enumCType
	enumByType         map[hir.Type]int
	resourceTypes      []resourceCType
	resourceByType     map[hir.Type]int
	taggedTypes        []taggedCType
	taggedByType       map[hir.Type]int
	resourceLocals     []map[hir.SymbolID]mir.ResourceLocal
	features           map[string]struct{}
	trapCategories     map[string]struct{}
	dynamicTrapTargets map[int]map[mir.BlockID]struct{}
	localTypes         []map[mir.LocalID]hir.Type
	needsBitcast       bool
	needsShift         bool
	cIntLayout         target.Layout
	cSizeTLayout       target.Layout
	cSSizeTLayout      target.Layout
	instrumentation    string
	dropSites          []DropSite
	dropSiteByKey      map[dropSiteKey]uint32
}

type structCType struct {
	definition mir.StructType
	layout     layout.Result
	layoutType layout.Type
	fieldIndex map[string]int
}

type resourceCType struct {
	definition mir.ResourceType
	layout     layout.Result
	layoutType layout.Type
	fieldIndex map[string]int
}

type enumCType struct {
	definition   mir.EnumType
	variantIndex map[string]int
	fieldIndex   []map[string]int
}

func targetCScalar(spec target.Spec, name string) (target.Scalar, bool) {
	for _, scalar := range spec.CScalars {
		if scalar.Name == name {
			return scalar, true
		}
	}
	return target.Scalar{}, false
}

// Emit verifies MIR, rejects unsupported semantic families, and emits one deterministic C unit.
func Emit(program mir.Program, entry string, profile Profile) (Unit, error) {
	if profile != Stage0Profile() && profile != Stage0DropTraceProfile() {
		return Unit{}, fmt.Errorf(
			"unsupported C emission profile target=%q dialect=%q instrumentation=%q",
			profile.Target, profile.Dialect, profile.Instrumentation,
		)
	}
	if failures := mir.Verify(program); len(failures) != 0 {
		return Unit{}, fmt.Errorf("C emission requires verified MIR: %s at %s", failures[0].Code, failures[0].Entity)
	}
	targetSpec, found := target.Lookup(profile.Target)
	if !found {
		return Unit{}, fmt.Errorf("C emission target %q is missing", profile.Target)
	}
	cInt, found := targetCScalar(targetSpec, "c.int")
	if !found {
		return Unit{}, fmt.Errorf("C emission target %q has no c.int layout", profile.Target)
	}
	cSizeT, found := targetCScalar(targetSpec, "c.size_t")
	if !found {
		return Unit{}, fmt.Errorf("C emission target %q has no c.size_t layout", profile.Target)
	}
	if cSizeT.Header != "stddef.h" || cSizeT.Availability != "iso-c11" {
		return Unit{}, fmt.Errorf("C emission target %q has invalid c.size_t provider", profile.Target)
	}
	cSSizeT, found := targetCScalar(targetSpec, "c.ssize_t")
	if !found {
		return Unit{}, fmt.Errorf("C emission target %q has no c.ssize_t layout", profile.Target)
	}
	if cSSizeT.Header != "sys/types.h" || cSSizeT.Availability != "posix.1-2008" {
		return Unit{}, fmt.Errorf("C emission target %q has invalid c.ssize_t provider", profile.Target)
	}
	layoutEngine, err := layout.New(targetSpec)
	if err != nil {
		return Unit{}, fmt.Errorf("create C resource layout engine: %w", err)
	}
	structTypes, structByType, err := prepareStructTypes(program.StructTypes, layoutEngine)
	if err != nil {
		return Unit{}, err
	}
	enumTypes, enumByType, err := prepareEnumTypes(program.EnumTypes)
	if err != nil {
		return Unit{}, err
	}
	resourceTypes, resourceByType, err := prepareResourceTypes(program.ResourceTypes, layoutEngine)
	if err != nil {
		return Unit{}, err
	}
	taggedTypes, taggedByType, err := prepareResourceTaggedTypes(program, resourceTypes, resourceByType, layoutEngine)
	if err != nil {
		return Unit{}, err
	}
	functions := slices.Clone(program.Functions)
	slices.SortFunc(functions, func(left, right mir.Function) int {
		return strings.Compare(left.Name, right.Name)
	})
	foreignFunctions := slices.Clone(program.ForeignFunctions)
	slices.SortFunc(foreignFunctions, func(left, right mir.ForeignFunction) int {
		if value := strings.Compare(left.LinkName, right.LinkName); value != 0 {
			return value
		}
		return int(left.Symbol - right.Symbol)
	})
	foreignFunctions, foreignBySymbol, err := normalizeForeignFunctions(foreignFunctions)
	if err != nil {
		return Unit{}, err
	}
	value := &emitter{
		functions:          functions,
		bySymbol:           make(map[hir.SymbolID]int, len(functions)),
		foreignFunctions:   foreignFunctions,
		foreignBySymbol:    foreignBySymbol,
		entryIndex:         -1,
		structTypes:        structTypes,
		structByType:       structByType,
		enumTypes:          enumTypes,
		enumByType:         enumByType,
		resourceTypes:      resourceTypes,
		resourceByType:     resourceByType,
		taggedTypes:        taggedTypes,
		taggedByType:       taggedByType,
		resourceLocals:     make([]map[hir.SymbolID]mir.ResourceLocal, len(functions)),
		features:           make(map[string]struct{}),
		trapCategories:     make(map[string]struct{}),
		dynamicTrapTargets: make(map[int]map[mir.BlockID]struct{}),
		localTypes:         make([]map[mir.LocalID]hir.Type, len(functions)),
		cIntLayout:         cInt.Layout,
		cSizeTLayout:       cSizeT.Layout,
		cSSizeTLayout:      cSSizeT.Layout,
		instrumentation:    profile.Instrumentation,
	}
	if len(structTypes) != 0 {
		value.features["inert-struct"] = struct{}{}
	}
	if len(enumTypes) != 0 {
		value.features["inert-enum"] = struct{}{}
	}
	for _, structure := range structTypes {
		if structure.definition.ReprC {
			value.features["repr-c-struct"] = struct{}{}
			break
		}
	}
	if len(resourceTypes) != 0 {
		value.features["flat-i32-resource"] = struct{}{}
	}
	for index := range functions {
		function := &functions[index]
		if _, collision := value.foreignBySymbol[function.Symbol]; collision {
			return Unit{}, fmt.Errorf("MIR symbol %d is both internal and foreign", function.Symbol)
		}
		if _, duplicate := value.bySymbol[function.Symbol]; duplicate {
			return Unit{}, fmt.Errorf("duplicate MIR function symbol %d", function.Symbol)
		}
		value.bySymbol[function.Symbol] = index
		if function.Name == entry {
			if value.entryIndex >= 0 {
				return Unit{}, fmt.Errorf("duplicate C entry function %q", entry)
			}
			value.entryIndex = index
		}
		locals := make(map[hir.SymbolID]mir.ResourceLocal, len(function.Resources))
		for _, resource := range function.Resources {
			if _, duplicate := locals[resource.Symbol]; duplicate {
				return Unit{}, fmt.Errorf("duplicate resource local %d in %s", resource.Symbol, function.Name)
			}
			locals[resource.Symbol] = resource
		}
		value.resourceLocals[index] = locals
	}
	if value.entryIndex < 0 {
		return Unit{}, fmt.Errorf("C entry function %q is missing", entry)
	}
	if err := value.inspectForeignFunctions(); err != nil {
		return Unit{}, err
	}
	if err := value.inspectEnumTypes(); err != nil {
		return Unit{}, err
	}
	entryFunction := &functions[value.entryIndex]
	if len(entryFunction.Parameters) != 0 || entryFunction.ResultType != hir.TypeI32 {
		return Unit{}, fmt.Errorf("C entry %q must have type fn() -> i32", entry)
	}
	for index := range functions {
		localTypes, err := value.inferLocalTypes(index)
		if err != nil {
			return Unit{}, err
		}
		value.localTypes[index] = localTypes
	}
	for index := range functions {
		if err := value.inspectFunction(index); err != nil {
			return Unit{}, err
		}
	}
	if err := value.prepareDropTraceSites(); err != nil {
		return Unit{}, err
	}

	var output strings.Builder
	if err := value.emitPreamble(&output); err != nil {
		return Unit{}, err
	}
	for index := range foreignFunctions {
		value.emitForeignMacroGuard(&output, index)
	}
	for index := range foreignFunctions {
		if err := value.emitForeignPrototype(&output, index); err != nil {
			return Unit{}, err
		}
	}
	if len(foreignFunctions) != 0 {
		output.WriteByte('\n')
	}
	for index := range functions {
		if err := value.emitPrototype(&output, index); err != nil {
			return Unit{}, err
		}
	}
	output.WriteByte('\n')
	for index := range functions {
		if err := value.emitFunction(&output, index); err != nil {
			return Unit{}, err
		}
	}
	fmt.Fprintf(&output, "int main(void) {\n    return (int)fy_fn_%d();\n}\n", value.entryIndex)

	names := make([]string, len(functions))
	for index := range functions {
		names[index] = functions[index].Name
	}
	features := make([]string, 0, len(value.features))
	for feature := range value.features {
		features = append(features, feature)
	}
	slices.Sort(features)
	return Unit{
		Target: string(profile.Target), Dialect: profile.Dialect, Instrumentation: profile.Instrumentation,
		Entry: entry, Functions: names, Features: features,
		DropSites: slices.Clone(value.dropSites), Source: output.String(),
	}, nil
}

func prepareStructTypes(definitions []mir.StructType, engine *layout.Engine) ([]structCType, map[hir.Type]int, error) {
	ordered := slices.Clone(definitions)
	slices.SortFunc(ordered, func(left, right mir.StructType) int {
		return strings.Compare(string(left.Type), string(right.Type))
	})
	result := make([]structCType, 0, len(ordered))
	byType := make(map[hir.Type]int, len(ordered))
	publicNames := make(map[string]hir.Type)
	for _, definition := range ordered {
		if len(definition.Fields) == 0 {
			return nil, nil, CapabilityError{Feature: "empty struct type " + strconv.Quote(string(definition.Type))}
		}
		if definition.ReprC {
			if !validCABIIdentifier(definition.Name) {
				return nil, nil, CapabilityError{Feature: "repr(C) struct name " + strconv.Quote(definition.Name)}
			}
			if previous, duplicate := publicNames[definition.Name]; duplicate {
				return nil, nil, fmt.Errorf("repr(C) structs %q and %q share C tag %q", previous, definition.Type, definition.Name)
			}
			publicNames[definition.Name] = definition.Type
		}
		fields := make([]layout.Field, 0, len(definition.Fields))
		fieldIndex := make(map[string]int, len(definition.Fields))
		for index, field := range definition.Fields {
			if definition.ReprC && !validCABIIdentifier(field.Name) {
				return nil, nil, CapabilityError{Feature: "repr(C) struct " + strconv.Quote(definition.Name) + " field name " + strconv.Quote(field.Name)}
			}
			fieldType, err := structFieldLayoutType(field.Type)
			if err != nil {
				return nil, nil, CapabilityError{Feature: "struct type " + strconv.Quote(string(definition.Type)) + " field " + strconv.Quote(field.Name) + " of type " + strconv.Quote(string(field.Type))}
			}
			fieldIndex[field.Name] = index
			layoutName := fmt.Sprintf("f_%d", index)
			if definition.ReprC {
				layoutName = field.Name
			}
			fields = append(fields, layout.Field{Name: layoutName, Type: fieldType})
		}
		aggregateType := layout.Type{Kind: layout.KindStruct, Fields: fields}
		computed, err := engine.Of(aggregateType)
		if err != nil {
			return nil, nil, fmt.Errorf("compute C transport layout for struct %q: %w", definition.Type, err)
		}
		if _, duplicate := byType[definition.Type]; duplicate {
			return nil, nil, fmt.Errorf("duplicate C struct type %q", definition.Type)
		}
		byType[definition.Type] = len(result)
		result = append(result, structCType{definition: definition, layout: computed, layoutType: aggregateType, fieldIndex: fieldIndex})
	}
	return result, byType, nil
}

func prepareEnumTypes(definitions []mir.EnumType) ([]enumCType, map[hir.Type]int, error) {
	ordered := slices.Clone(definitions)
	slices.SortFunc(ordered, func(left, right mir.EnumType) int {
		return strings.Compare(string(left.Type), string(right.Type))
	})
	result := make([]enumCType, 0, len(ordered))
	byType := make(map[hir.Type]int, len(ordered))
	for _, definition := range ordered {
		if len(definition.Variants) == 0 {
			return nil, nil, CapabilityError{Feature: "empty enum type " + strconv.Quote(string(definition.Type))}
		}
		if _, duplicate := byType[definition.Type]; duplicate {
			return nil, nil, fmt.Errorf("duplicate C enum type %q", definition.Type)
		}
		value := enumCType{
			definition: definition, variantIndex: make(map[string]int, len(definition.Variants)),
			fieldIndex: make([]map[string]int, len(definition.Variants)),
		}
		for variantIndex, variant := range definition.Variants {
			value.variantIndex[variant.Name] = variantIndex
			value.fieldIndex[variantIndex] = make(map[string]int, len(variant.Fields))
			for fieldIndex, field := range variant.Fields {
				value.fieldIndex[variantIndex][field.Name] = fieldIndex
			}
		}
		byType[definition.Type] = len(result)
		result = append(result, value)
	}
	return result, byType, nil
}

func structFieldLayoutType(value hir.Type) (layout.Type, error) {
	value = canonicalCValueType(value)
	switch value {
	case hir.TypeU8, hir.TypeI32, hir.TypeU32, hir.TypeUSize:
		return layout.Type{Kind: layout.KindLanguageScalar, Name: string(value)}, nil
	default:
		if _, _, pointer := hir.RawPointerElement(value); pointer {
			return layout.Type{Kind: layout.KindPointer}, nil
		}
		return layout.Type{}, fmt.Errorf("unsupported struct field type %q", value)
	}
}

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

func prepareResourceTypes(definitions []mir.ResourceType, engine *layout.Engine) ([]resourceCType, map[hir.Type]int, error) {
	ordered := slices.Clone(definitions)
	slices.SortFunc(ordered, func(left, right mir.ResourceType) int {
		return strings.Compare(string(left.Type), string(right.Type))
	})
	result := make([]resourceCType, 0, len(ordered))
	byType := make(map[hir.Type]int, len(ordered))
	for _, definition := range ordered {
		if definition.DropKind != hir.ResourceDropEmpty {
			return nil, nil, CapabilityError{Feature: "resource type " + strconv.Quote(string(definition.Type)) + " with non-empty drop behavior"}
		}
		if len(definition.Fields) == 0 {
			return nil, nil, CapabilityError{Feature: "empty resource type " + strconv.Quote(string(definition.Type))}
		}
		fields := make([]layout.Field, 0, len(definition.Fields))
		fieldIndex := make(map[string]int, len(definition.Fields))
		for index, field := range definition.Fields {
			if field.Type != hir.TypeI32 {
				return nil, nil, CapabilityError{Feature: "resource type " + strconv.Quote(string(definition.Type)) + " field " + strconv.Quote(field.Name) + " of type " + strconv.Quote(string(field.Type))}
			}
			fieldIndex[field.Name] = index
			fields = append(fields, layout.Field{
				Name: fmt.Sprintf("f_%d", index),
				Type: layout.Type{Kind: layout.KindLanguageScalar, Name: string(hir.TypeI32)},
			})
		}
		aggregateType := layout.Type{Kind: layout.KindStruct, Fields: fields}
		computed, err := engine.Of(aggregateType)
		if err != nil {
			return nil, nil, fmt.Errorf("compute C transport layout for resource %q: %w", definition.Type, err)
		}
		if len(computed.Fields) != len(definition.Fields) {
			return nil, nil, fmt.Errorf("resource %q layout returned %d fields for %d definitions", definition.Type, len(computed.Fields), len(definition.Fields))
		}
		if _, duplicate := byType[definition.Type]; duplicate {
			return nil, nil, fmt.Errorf("duplicate C resource type %q", definition.Type)
		}
		byType[definition.Type] = len(result)
		result = append(result, resourceCType{definition: definition, layout: computed, layoutType: aggregateType, fieldIndex: fieldIndex})
	}
	return result, byType, nil
}

func (e *emitter) inferLocalTypes(functionIndex int) (map[mir.LocalID]hir.Type, error) {
	function := &e.functions[functionIndex]
	localTypes := make(map[mir.LocalID]hir.Type)
	for index, parameter := range function.Parameters {
		if index >= len(function.ParameterTypes) {
			return nil, fmt.Errorf("MIR function %s has %d parameter types for %d parameters", function.Name, len(function.ParameterTypes), len(function.Parameters))
		}
		if _, err := assignLocalType(localTypes, parameter, function.ParameterTypes[index]); err != nil {
			return nil, fmt.Errorf("parameter type in %s: %w", function.Name, err)
		}
	}
	for _, resource := range function.Resources {
		if _, err := assignLocalType(localTypes, mir.LocalID(resource.Symbol), resource.Type); err != nil {
			return nil, fmt.Errorf("resource local type in %s: %w", function.Name, err)
		}
	}
	changed := true
	for changed {
		changed = false
		for _, block := range function.Blocks {
			for _, operation := range block.Operations {
				var inferred hir.Type
				switch operation.Kind {
				case mir.OperationAssign, mir.OperationInitialize:
					if len(operation.Operands) == 1 {
						inferred, _ = inferredOperandType(localTypes, operation.Operands[0])
					}
				case mir.OperationUnary, mir.OperationBinary:
					inferred = hir.TypeI32
				case mir.OperationResultConstruct, mir.OperationOptionConstruct,
					mir.OperationResultPayload, mir.OperationOptionPayload,
					mir.OperationEnumConstruct, mir.OperationEnumPayload,
					mir.OperationRawAddress, mir.OperationStructConstruct, mir.OperationResourceConstruct, mir.OperationFieldRead:
					inferred = operation.Type
				}
				if operation.DestinationLocal == mir.NoLocal || inferred == hir.TypeInvalid || inferred == "" {
					continue
				}
				added, err := assignLocalType(localTypes, operation.DestinationLocal, inferred)
				if err != nil {
					return nil, fmt.Errorf("operation %s in %s: %w", operation.Kind, function.Name, err)
				}
				changed = changed || added
			}
			terminator := block.Terminator
			if terminator.Kind == mir.TerminatorCall && terminator.DestinationLocal != mir.NoLocal {
				resultType, found := e.callResultType(terminator.Callee)
				if !found {
					return nil, fmt.Errorf("C call in %s names missing function symbol %d", function.Name, terminator.Callee)
				}
				added, err := assignLocalType(localTypes, terminator.DestinationLocal, resultType)
				if err != nil {
					return nil, fmt.Errorf("call result in %s: %w", function.Name, err)
				}
				changed = changed || added
			}
		}
	}
	for _, local := range collectLocals(*function) {
		if _, found := localTypes[local]; !found {
			localTypes[local] = hir.TypeI32
		}
	}
	return localTypes, nil
}

func (e *emitter) callResultType(symbol hir.SymbolID) (hir.Type, bool) {
	if index, found := e.bySymbol[symbol]; found {
		return e.functions[index].ResultType, true
	}
	if index, found := e.foreignBySymbol[symbol]; found {
		return e.foreignFunctions[index].ResultType, true
	}
	return hir.TypeInvalid, false
}

func (e *emitter) inspectForeignFunctions() error {
	for _, function := range e.foreignFunctions {
		if function.ABI != "C" {
			return CapabilityError{Function: function.Name, Feature: "foreign ABI " + strconv.Quote(function.ABI)}
		}
		if !validExternalCIdentifier(function.LinkName) {
			return CapabilityError{Function: function.Name, Feature: "external C symbol " + strconv.Quote(function.LinkName)}
		}
		if _, err := e.foreignCType(function.ResultType, true); err != nil {
			return CapabilityError{Function: function.Name, Feature: err.Error()}
		}
		e.recordCScalarFeature(function.ResultType)
		for _, parameterType := range function.ParameterTypes {
			if _, err := e.foreignCType(parameterType, false); err != nil {
				return CapabilityError{Function: function.Name, Feature: err.Error()}
			}
			e.recordCScalarFeature(parameterType)
		}
	}
	return nil
}

func (e *emitter) recordCScalarFeature(value hir.Type) {
	switch value {
	case hir.TypeCInt:
		e.features["c-int"] = struct{}{}
	case hir.TypeCSizeT:
		e.features["c-size-t"] = struct{}{}
	case hir.TypeCSSizeT:
		e.features["c-ssize-t"] = struct{}{}
	}
}

func (e *emitter) inspectEnumTypes() error {
	for _, enumeration := range e.enumTypes {
		for _, variant := range enumeration.definition.Variants {
			for _, field := range variant.Fields {
				if _, err := e.enumPayloadCType(field.Type); err != nil {
					return CapabilityError{Feature: "enum type " + strconv.Quote(string(enumeration.definition.Type)) + " variant " + strconv.Quote(variant.Name) + " field " + strconv.Quote(field.Name) + " of type " + strconv.Quote(string(field.Type))}
				}
			}
		}
	}
	return nil
}

func (e *emitter) enumPayloadCType(value hir.Type) (string, error) {
	canonical := canonicalCValueType(value)
	if canonical == hir.TypeU8 || canonical == hir.TypeI32 || canonical == hir.TypeU32 || canonical == hir.TypeUSize || isRawPointer(canonical) {
		return e.cValueType(canonical)
	}
	if _, structure := e.structByType[canonical]; structure {
		return e.cValueType(canonical)
	}
	return "", fmt.Errorf("unsupported enum payload type %q", value)
}

func (e *emitter) foreignCType(value hir.Type, result bool) (string, error) {
	if result && value == hir.TypeVoid {
		return "void", nil
	}
	switch value {
	case hir.TypeU8:
		return "uint8_t", nil
	case hir.TypeI32:
		return "int32_t", nil
	case hir.TypeU32:
		return "uint32_t", nil
	case hir.TypeUSize:
		return "uintptr_t", nil
	case hir.TypeCInt:
		return "int", nil
	case hir.TypeCSizeT:
		return "size_t", nil
	case hir.TypeCSSizeT:
		return "ssize_t", nil
	}
	if element, mutable, pointer := hir.RawPointerElement(value); pointer {
		pointee, err := e.foreignPointerElementCType(element)
		if err != nil {
			return "", err
		}
		if mutable {
			return pointee + " *", nil
		}
		return "const " + pointee + " *", nil
	}
	role := "parameter"
	if result {
		role = "result"
	}
	return "", fmt.Errorf("foreign %s type %q", role, value)
}

func (e *emitter) foreignPointerElementCType(value hir.Type) (string, error) {
	switch value {
	case hir.TypeU8:
		return "uint8_t", nil
	case hir.TypeI32:
		return "int32_t", nil
	case hir.TypeU32:
		return "uint32_t", nil
	case hir.TypeUSize:
		return "uintptr_t", nil
	default:
		if index, found := e.structByType[value]; found {
			definition := e.structTypes[index].definition
			if !definition.ReprC {
				return "", fmt.Errorf("foreign raw-pointer element struct %q is not repr(C)", value)
			}
			return "struct " + definition.Name, nil
		}
		return "", fmt.Errorf("foreign raw-pointer element type %q", value)
	}
}

func validCABIIdentifier(value string) bool {
	if value == "" {
		return false
	}
	if _, reserved := generatedCNames[value]; reserved {
		return false
	}
	for index, character := range value {
		if character > 0x7f || !(character == '_' || character >= 'a' && character <= 'z' || character >= 'A' && character <= 'Z' || index > 0 && character >= '0' && character <= '9') {
			return false
		}
	}
	_, keyword := c11Keywords[value]
	return !keyword
}

func validExternalCIdentifier(value string) bool {
	if value == "" || value == "main" || strings.HasPrefix(value, "FY_") {
		return false
	}
	if strings.HasPrefix(value, "fy_") {
		if _, admitted := admittedStandardLibrarySymbols[value]; !admitted {
			return false
		}
	}
	if _, reserved := generatedCNames[value]; reserved {
		return false
	}
	for index, character := range value {
		if character > 0x7f || !(character == '_' || character >= 'a' && character <= 'z' || character >= 'A' && character <= 'Z' || index > 0 && character >= '0' && character <= '9') {
			return false
		}
	}
	_, keyword := c11Keywords[value]
	return !keyword
}

var admittedStandardLibrarySymbols = map[string]struct{}{
	"fy_std_text_utf8_valid_v0":    {},
	"fy_std_text_validate_utf8_v0": {},
}

var generatedCNames = map[string]struct{}{
	"uint8_t": {}, "int32_t": {}, "uint32_t": {}, "uint64_t": {}, "uintptr_t": {}, "size_t": {}, "ssize_t": {},
	"UINT8_C": {}, "INT32_C": {}, "UINT32_C": {}, "UINT64_C": {},
	"INT32_MIN": {}, "INT32_MAX": {}, "UINT32_MAX": {}, "UINTPTR_MAX": {},
}

var c11Keywords = map[string]struct{}{
	"auto": {}, "break": {}, "case": {}, "char": {}, "const": {}, "continue": {}, "default": {},
	"do": {}, "double": {}, "else": {}, "enum": {}, "extern": {}, "float": {}, "for": {}, "goto": {},
	"if": {}, "inline": {}, "int": {}, "long": {}, "register": {}, "restrict": {}, "return": {},
	"short": {}, "signed": {}, "sizeof": {}, "static": {}, "struct": {}, "switch": {}, "typedef": {},
	"union": {}, "unsigned": {}, "void": {}, "volatile": {}, "while": {}, "_Alignas": {}, "_Alignof": {},
	"_Atomic": {}, "_Bool": {}, "_Complex": {}, "_Generic": {}, "_Imaginary": {}, "_Noreturn": {},
	"_Static_assert": {}, "_Thread_local": {},
}

func assignLocalType(localTypes map[mir.LocalID]hir.Type, local mir.LocalID, value hir.Type) (bool, error) {
	if local < 0 {
		return false, fmt.Errorf("local has invalid id %d", local)
	}
	value = canonicalCValueType(value)
	if value == hir.TypeVoid || value == hir.TypeInvalid || value == "" {
		return false, fmt.Errorf("local %d has invalid value type %q", local, value)
	}
	if existing, found := localTypes[local]; found {
		if canonicalCValueType(existing) != value {
			return false, fmt.Errorf("local %d has conflicting types %q and %q", local, existing, value)
		}
		return false, nil
	}
	localTypes[local] = value
	return true, nil
}

func inferredOperandType(localTypes map[mir.LocalID]hir.Type, operand mir.Operand) (hir.Type, bool) {
	switch operand.Kind {
	case mir.OperandI32:
		return hir.TypeI32, true
	case mir.OperandU8:
		return hir.TypeU8, true
	case mir.OperandU32:
		return hir.TypeU32, true
	case mir.OperandUSize:
		return hir.TypeUSize, true
	case mir.OperandNull:
		if _, _, pointer := hir.RawPointerElement(operand.Type); pointer {
			return operand.Type, true
		}
		return hir.TypeInvalid, false
	case mir.OperandLocal:
		value, found := localTypes[operand.Local]
		return value, found
	default:
		return hir.TypeInvalid, false
	}
}

func canonicalCValueType(value hir.Type) hir.Type {
	if value == hir.TypeBool {
		return hir.TypeI32
	}
	return value
}

func isOptionI32(value hir.Type) bool {
	element, ok := hir.OptionElement(value)
	return ok && element == hir.TypeI32
}

func isResultI32I32(value hir.Type) bool {
	success, failure, ok := hir.ResultElements(value)
	return ok && success == hir.TypeI32 && failure == hir.TypeI32
}

func (e *emitter) supportedCValueType(value hir.Type) bool {
	value = canonicalCValueType(value)
	if value == hir.TypeU8 || value == hir.TypeI32 || value == hir.TypeU32 || value == hir.TypeUSize || value == hir.TypeCInt || value == hir.TypeCSizeT || value == hir.TypeCSSizeT || isOptionI32(value) || isResultI32I32(value) {
		return true
	}
	if element, _, pointer := hir.RawPointerElement(value); pointer {
		_, err := e.foreignPointerElementCType(element)
		return err == nil
	}
	_, resource := e.resourceByType[value]
	if resource {
		return true
	}
	if _, structure := e.structByType[value]; structure {
		return true
	}
	if _, enumeration := e.enumByType[value]; enumeration {
		return true
	}
	_, tagged := e.taggedByType[value]
	return tagged
}

func (e *emitter) cValueType(value hir.Type) (string, error) {
	value = canonicalCValueType(value)
	switch {
	case value == hir.TypeU8:
		return "uint8_t", nil
	case value == hir.TypeI32:
		return "int32_t", nil
	case value == hir.TypeU32:
		return "uint32_t", nil
	case value == hir.TypeUSize:
		return "uintptr_t", nil
	case value == hir.TypeCInt:
		return "int", nil
	case value == hir.TypeCSizeT:
		return "size_t", nil
	case value == hir.TypeCSSizeT:
		return "ssize_t", nil
	case isRawPointer(value):
		return e.foreignCType(value, false)
	case isOptionI32(value):
		return "fy_option_i32", nil
	case isResultI32I32(value):
		return "fy_result_i32_i32", nil
	default:
		if index, found := e.structByType[value]; found {
			return e.structCTypeName(index), nil
		}
		if index, found := e.enumByType[value]; found {
			return fmt.Sprintf("fy_enum_%d", index), nil
		}
		if index, found := e.resourceByType[value]; found {
			return fmt.Sprintf("fy_resource_%d", index), nil
		}
		if index, found := e.taggedByType[value]; found {
			return fmt.Sprintf("fy_tagged_%d", index), nil
		}
		return "", fmt.Errorf("unsupported C value type %q", value)
	}
}

func (e *emitter) structCTypeName(index int) string {
	definition := e.structTypes[index].definition
	if definition.ReprC {
		return "struct " + definition.Name
	}
	return fmt.Sprintf("fy_struct_%d", index)
}

func (e *emitter) inspectFunction(functionIndex int) error {
	function := &e.functions[functionIndex]
	if function.ResultType != hir.TypeVoid && !e.supportedCValueType(function.ResultType) {
		return CapabilityError{Function: function.Name, Feature: "result type " + strconv.Quote(string(function.ResultType))}
	}
	e.recordCScalarFeature(function.ResultType)
	if len(function.ParameterTypes) != len(function.Parameters) {
		return fmt.Errorf("MIR function %s has %d parameter types for %d parameters", function.Name, len(function.ParameterTypes), len(function.Parameters))
	}
	for _, parameterType := range function.ParameterTypes {
		if !e.supportedCValueType(parameterType) {
			return CapabilityError{Function: function.Name, Feature: "parameter type " + strconv.Quote(string(parameterType))}
		}
		e.recordCScalarFeature(parameterType)
	}
	for _, resource := range function.Resources {
		if !e.supportedNeedsDropType(resource.Type) {
			return CapabilityError{Function: function.Name, Feature: "resource local type " + strconv.Quote(string(resource.Type))}
		}
	}
	if len(function.DropFlags) != 0 {
		return CapabilityError{Function: function.Name, Feature: "partial resource initialization"}
	}
	if len(function.Loans) != 0 || len(function.ReturnOrigins) != 0 {
		return CapabilityError{Function: function.Name, Feature: "safe-borrow values"}
	}
	if len(function.RegionValues) != 0 {
		return CapabilityError{Function: function.Name, Feature: "Region values"}
	}
	allowedUnsafeEntities := make(map[string]struct{})
	for blockIndex := range function.Blocks {
		block := &function.Blocks[blockIndex]
		if block.ID != mir.BlockID(blockIndex) {
			return fmt.Errorf("C emission requires canonical block ids in %s: block index %d has id %d", function.Name, blockIndex, block.ID)
		}
		for operationIndex := range block.Operations {
			operation := &block.Operations[operationIndex]
			if err := validateOperationShape(function.Name, *operation); err != nil {
				return err
			}
			switch operation.Kind {
			case mir.OperationAssign:
				if err := e.inspectAssignment(functionIndex, *operation); err != nil {
					return err
				}
			case mir.OperationInitialize:
				if err := e.inspectResourceInitialize(functionIndex, *operation); err != nil {
					return err
				}
			case mir.OperationMove, mir.OperationDrop:
				if err := e.inspectResourceDischarge(functionIndex, *operation); err != nil {
					return err
				}
			case mir.OperationRawAddress:
				if err := e.inspectRawAddress(functionIndex, *operation); err != nil {
					return err
				}
				allowedUnsafeEntities[operation.Entity] = struct{}{}
				e.features["raw-address"] = struct{}{}
			case mir.OperationStructConstruct:
				if err := e.inspectStructOperation(functionIndex, *operation); err != nil {
					return err
				}
			case mir.OperationEnumConstruct, mir.OperationEnumPayload:
				if err := e.inspectEnumOperation(functionIndex, *operation); err != nil {
					return err
				}
			case mir.OperationResourceConstruct:
				if err := e.inspectResourceOperation(functionIndex, *operation); err != nil {
					return err
				}
			case mir.OperationFieldRead:
				if _, structure := e.structByType[operation.OwnerType]; structure {
					if err := e.inspectStructOperation(functionIndex, *operation); err != nil {
						return err
					}
				} else if err := e.inspectResourceOperation(functionIndex, *operation); err != nil {
					return err
				}
			case mir.OperationUse:
			case mir.OperationArithmeticCheck:
				e.features["checked-arithmetic"] = struct{}{}
			case mir.OperationCheckedDivision:
				e.features["checked-division"] = struct{}{}
			case mir.OperationUnary:
				if !supportedUnary(operation.Operator) {
					return CapabilityError{Function: function.Name, Feature: "unary operator " + strconv.Quote(operation.Operator)}
				}
				if operation.Operator == "~" {
					e.features["scalar-bitwise"] = struct{}{}
					e.needsBitcast = true
				}
			case mir.OperationBinary:
				if !supportedBinary(operation.Operator) {
					return CapabilityError{Function: function.Name, Feature: "binary operator " + strconv.Quote(operation.Operator)}
				}
				switch operation.Operator {
				case "==", "!=", "<", "<=", ">", ">=":
					e.features["scalar-comparison"] = struct{}{}
				case "&", "^", "|":
					e.features["scalar-bitwise"] = struct{}{}
					e.needsBitcast = true
				case ">>":
					e.features["checked-shift"] = struct{}{}
					e.needsBitcast = true
					e.needsShift = true
				}
			case mir.OperationResultConstruct, mir.OperationResultPayload,
				mir.OperationOptionConstruct, mir.OperationOptionPayload, mir.OperationResultSwitch:
				if err := e.inspectTaggedOperation(functionIndex, *operation); err != nil {
					return err
				}
			default:
				return CapabilityError{Function: function.Name, Feature: "MIR operation " + strconv.Quote(string(operation.Kind))}
			}
		}
		terminator := block.Terminator
		for _, successor := range terminator.Successors {
			if successor < 0 || int(successor) >= len(function.Blocks) {
				return fmt.Errorf("C emission successor %d is outside %s", successor, function.Name)
			}
		}
		switch terminator.Kind {
		case mir.TerminatorGoto:
			e.features["scalar-control-flow"] = struct{}{}
		case mir.TerminatorCall:
			e.features["scalar-call"] = struct{}{}
			calleeResult, calleeParameters, foreign, found := e.callSignature(terminator.Callee)
			if !found {
				return fmt.Errorf("C call in %s names missing function symbol %d", function.Name, terminator.Callee)
			}
			if foreign {
				e.features["foreign-c-call"] = struct{}{}
				allowedUnsafeEntities[terminator.Entity] = struct{}{}
				if terminator.ForeignABI != "C" || terminator.UnsafeScope.Source == "" {
					return fmt.Errorf("foreign C call in %s lacks ABI or unsafe provenance", function.Name)
				}
			} else if terminator.ForeignABI != "" {
				return fmt.Errorf("internal call in %s carries foreign ABI %q", function.Name, terminator.ForeignABI)
			}
			if calleeResult == hir.TypeVoid && terminator.DestinationLocal != mir.NoLocal {
				return fmt.Errorf("void C call in %s has destination local %d", function.Name, terminator.DestinationLocal)
			}
			if calleeResult != hir.TypeVoid && terminator.DestinationLocal < 0 {
				return fmt.Errorf("value C call in %s has no destination local", function.Name)
			}
			if len(terminator.Arguments) != len(calleeParameters) {
				return fmt.Errorf("C call in %s passes %d arguments, want %d", function.Name, len(terminator.Arguments), len(calleeParameters))
			}
			for argumentIndex, argument := range terminator.Arguments {
				argumentType, err := e.operandType(functionIndex, argument)
				if err != nil {
					return fmt.Errorf("call argument %d in %s: %w", argumentIndex, function.Name, err)
				}
				if canonicalCValueType(argumentType) != canonicalCValueType(calleeParameters[argumentIndex]) {
					return fmt.Errorf("call argument %d in %s has type %q, want %q", argumentIndex, function.Name, argumentType, calleeParameters[argumentIndex])
				}
			}
		case mir.TerminatorSwitch:
			e.features["scalar-control-flow"] = struct{}{}
			if err := e.inspectSwitch(functionIndex, terminator); err != nil {
				return err
			}
		case mir.TerminatorReturn:
			if len(terminator.Successors) != 0 {
				return fmt.Errorf("return in %s has successors", function.Name)
			}
			if function.ResultType == hir.TypeVoid && terminator.ReturnValue.Kind != mir.OperandInvalid {
				return fmt.Errorf("void return in %s carries a value", function.Name)
			}
			if function.ResultType != hir.TypeVoid {
				if _, err := cOperand(terminator.ReturnValue); err != nil {
					return fmt.Errorf("return in %s: %w", function.Name, err)
				}
				returnType, err := e.operandType(functionIndex, terminator.ReturnValue)
				if err != nil {
					return fmt.Errorf("return in %s: %w", function.Name, err)
				}
				if canonicalCValueType(returnType) != canonicalCValueType(function.ResultType) {
					return fmt.Errorf("return in %s has type %q, want %q", function.Name, returnType, function.ResultType)
				}
			}
		case mir.TerminatorTrap:
			if len(terminator.Successors) != 0 {
				return fmt.Errorf("trap in %s has successors", function.Name)
			}
			e.features["trap"] = struct{}{}
			if err := e.recordTrapCategories(function.Name, terminator.TrapCategories); err != nil {
				return err
			}
		case mir.TerminatorUnreachable:
			if len(terminator.Successors) != 0 {
				return fmt.Errorf("unreachable terminator in %s has successors", function.Name)
			}
			e.features["trap"] = struct{}{}
			e.trapCategories["unreachable"] = struct{}{}
		default:
			return CapabilityError{Function: function.Name, Feature: "MIR terminator " + strconv.Quote(string(terminator.Kind))}
		}
	}
	for _, operation := range function.UnsafeOperations {
		if _, admitted := allowedUnsafeEntities[operation.Entity]; !admitted {
			return CapabilityError{Function: function.Name, Feature: "unsafe operation outside the admitted raw-address and foreign-call subset"}
		}
	}
	return nil
}

func (e *emitter) callSignature(symbol hir.SymbolID) (hir.Type, []hir.Type, bool, bool) {
	if index, found := e.bySymbol[symbol]; found {
		function := &e.functions[index]
		return function.ResultType, function.ParameterTypes, false, true
	}
	if index, found := e.foreignBySymbol[symbol]; found {
		function := &e.foreignFunctions[index]
		return function.ResultType, function.ParameterTypes, true, true
	}
	return hir.TypeInvalid, nil, false, false
}

func validateOperationShape(function string, operation mir.Operation) error {
	switch operation.Kind {
	case mir.OperationAssign, mir.OperationInitialize, mir.OperationUnary, mir.OperationBinary,
		mir.OperationRawAddress, mir.OperationStructConstruct, mir.OperationEnumConstruct, mir.OperationEnumPayload,
		mir.OperationResourceConstruct, mir.OperationFieldRead:
		if operation.DestinationLocal < 0 {
			return fmt.Errorf("%s operation in %s has no destination local", operation.Kind, function)
		}
	case mir.OperationMove, mir.OperationDrop:
		if operation.Symbol == hir.NoSymbol {
			return fmt.Errorf("%s operation in %s has no resource symbol", operation.Kind, function)
		}
	}
	return nil
}

func (e *emitter) inspectRawAddress(functionIndex int, operation mir.Operation) error {
	function := &e.functions[functionIndex]
	operand, err := oneOperand(operation.Operands)
	if err != nil {
		return fmt.Errorf("raw address in %s: %w", function.Name, err)
	}
	if operand.Kind != mir.OperandLocal || operation.Entity == "" || operation.UnsafeScope.Source == "" {
		return fmt.Errorf("raw address in %s has incomplete place or unsafe provenance", function.Name)
	}
	element, mutable, pointer := hir.RawPointerElement(operation.Type)
	if !pointer || element != operation.OwnerType || operation.Operator != "&" && operation.Operator != "&mut" || operation.Operator == "&" && mutable {
		return fmt.Errorf("raw address in %s has incompatible source and pointer types", function.Name)
	}
	operandType, err := e.operandType(functionIndex, operand)
	if err != nil {
		return fmt.Errorf("raw address in %s: %w", function.Name, err)
	}
	if canonicalCValueType(operandType) != canonicalCValueType(operation.OwnerType) {
		return fmt.Errorf("raw address in %s has place type %q, want %q", function.Name, operandType, operation.OwnerType)
	}
	if _, err := e.cValueType(operation.Type); err != nil {
		return CapabilityError{Function: function.Name, Feature: "raw address type " + strconv.Quote(string(operation.Type))}
	}
	return nil
}

func (e *emitter) inspectStructOperation(functionIndex int, operation mir.Operation) error {
	function := &e.functions[functionIndex]
	switch operation.Kind {
	case mir.OperationStructConstruct:
		index, found := e.structByType[operation.Type]
		if !found {
			return CapabilityError{Function: function.Name, Feature: "struct construction type " + strconv.Quote(string(operation.Type))}
		}
		definition := &e.structTypes[index]
		if len(operation.Fields) != len(definition.definition.Fields) || len(operation.Operands) != len(definition.definition.Fields) {
			return fmt.Errorf("struct construction in %s does not match type %q", function.Name, operation.Type)
		}
		for fieldIndex, field := range definition.definition.Fields {
			if operation.Fields[fieldIndex] != field.Name {
				return fmt.Errorf("struct construction in %s has field %q at index %d, want %q", function.Name, operation.Fields[fieldIndex], fieldIndex, field.Name)
			}
			operandType, err := e.operandType(functionIndex, operation.Operands[fieldIndex])
			if err != nil {
				return fmt.Errorf("struct construction field %d in %s: %w", fieldIndex, function.Name, err)
			}
			if canonicalCValueType(operandType) != canonicalCValueType(field.Type) {
				return fmt.Errorf("struct construction field %d in %s has type %q, want %q", fieldIndex, function.Name, operandType, field.Type)
			}
		}
	case mir.OperationFieldRead:
		index, found := e.structByType[operation.OwnerType]
		if !found {
			return CapabilityError{Function: function.Name, Feature: "field-read owner type " + strconv.Quote(string(operation.OwnerType))}
		}
		definition := &e.structTypes[index]
		fieldIndex, found := definition.fieldIndex[operation.Field]
		if !found || canonicalCValueType(definition.definition.Fields[fieldIndex].Type) != canonicalCValueType(operation.Type) {
			return fmt.Errorf("struct field read in %s does not match field %q", function.Name, operation.Field)
		}
		receiver, err := oneOperand(operation.Operands)
		if err != nil {
			return fmt.Errorf("struct field read in %s: %w", function.Name, err)
		}
		receiverType, err := e.operandType(functionIndex, receiver)
		if err != nil {
			return fmt.Errorf("struct field read in %s: %w", function.Name, err)
		}
		if canonicalCValueType(receiverType) != canonicalCValueType(operation.OwnerType) {
			return fmt.Errorf("struct field read in %s has receiver type %q, want %q", function.Name, receiverType, operation.OwnerType)
		}
	}
	return nil
}

func (e *emitter) inspectEnumOperation(functionIndex int, operation mir.Operation) error {
	function := &e.functions[functionIndex]
	ownerType := operation.Type
	if operation.Kind == mir.OperationEnumPayload {
		ownerType = operation.OwnerType
	}
	index, found := e.enumByType[ownerType]
	if !found {
		return CapabilityError{Function: function.Name, Feature: "enum operation type " + strconv.Quote(string(ownerType))}
	}
	definition := &e.enumTypes[index]
	variantIndex, found := definition.variantIndex[operation.Variant]
	if !found {
		return fmt.Errorf("enum operation in %s names missing variant %q", function.Name, operation.Variant)
	}
	variant := definition.definition.Variants[variantIndex]
	switch operation.Kind {
	case mir.OperationEnumConstruct:
		if len(operation.Fields) != len(variant.Fields) || len(operation.Operands) != len(variant.Fields) {
			return fmt.Errorf("enum construction in %s does not match variant %q", function.Name, operation.Variant)
		}
		seen := make(map[string]struct{}, len(operation.Fields))
		for operandIndex, fieldName := range operation.Fields {
			fieldIndex, declared := definition.fieldIndex[variantIndex][fieldName]
			if !declared {
				return fmt.Errorf("enum construction in %s names missing field %q", function.Name, fieldName)
			}
			if _, duplicate := seen[fieldName]; duplicate {
				return fmt.Errorf("enum construction in %s repeats field %q", function.Name, fieldName)
			}
			seen[fieldName] = struct{}{}
			operandType, err := e.operandType(functionIndex, operation.Operands[operandIndex])
			if err != nil {
				return fmt.Errorf("enum construction field %q in %s: %w", fieldName, function.Name, err)
			}
			if canonicalCValueType(operandType) != canonicalCValueType(variant.Fields[fieldIndex].Type) {
				return fmt.Errorf("enum construction field %q in %s has type %q, want %q", fieldName, function.Name, operandType, variant.Fields[fieldIndex].Type)
			}
		}
	case mir.OperationEnumPayload:
		fieldIndex, declared := definition.fieldIndex[variantIndex][operation.Field]
		if !declared || canonicalCValueType(variant.Fields[fieldIndex].Type) != canonicalCValueType(operation.Type) {
			return fmt.Errorf("enum payload in %s does not match field %q", function.Name, operation.Field)
		}
		receiver, err := oneOperand(operation.Operands)
		if err != nil {
			return fmt.Errorf("enum payload in %s: %w", function.Name, err)
		}
		receiverType, err := e.operandType(functionIndex, receiver)
		if err != nil {
			return fmt.Errorf("enum payload in %s: %w", function.Name, err)
		}
		if canonicalCValueType(receiverType) != canonicalCValueType(operation.OwnerType) {
			return fmt.Errorf("enum payload in %s has receiver type %q, want %q", function.Name, receiverType, operation.OwnerType)
		}
	}
	return nil
}

func (e *emitter) operandType(functionIndex int, operand mir.Operand) (hir.Type, error) {
	switch operand.Kind {
	case mir.OperandI32:
		return hir.TypeI32, nil
	case mir.OperandU8:
		return hir.TypeU8, nil
	case mir.OperandU32:
		return hir.TypeU32, nil
	case mir.OperandUSize:
		return hir.TypeUSize, nil
	case mir.OperandNull:
		if _, _, pointer := hir.RawPointerElement(operand.Type); pointer {
			return operand.Type, nil
		}
		return hir.TypeInvalid, fmt.Errorf("null operand has invalid type %q", operand.Type)
	case mir.OperandLocal:
		value, found := e.localTypes[functionIndex][operand.Local]
		if !found {
			return hir.TypeInvalid, fmt.Errorf("local operand %d has no inferred C type", operand.Local)
		}
		return value, nil
	default:
		return hir.TypeInvalid, fmt.Errorf("unsupported C operand kind %q", operand.Kind)
	}
}

func (e *emitter) inspectAssignment(functionIndex int, operation mir.Operation) error {
	function := &e.functions[functionIndex]
	operand, err := oneOperand(operation.Operands)
	if err != nil {
		return fmt.Errorf("assign in %s: %w", function.Name, err)
	}
	sourceType, err := e.operandType(functionIndex, operand)
	if err != nil {
		return fmt.Errorf("assign in %s: %w", function.Name, err)
	}
	destinationType, found := e.localTypes[functionIndex][operation.DestinationLocal]
	if !found {
		return fmt.Errorf("assign in %s has no type for destination local %d", function.Name, operation.DestinationLocal)
	}
	if canonicalCValueType(sourceType) != canonicalCValueType(destinationType) {
		return fmt.Errorf("assign in %s has source type %q and destination type %q", function.Name, sourceType, destinationType)
	}
	return nil
}

func (e *emitter) inspectResourceInitialize(functionIndex int, operation mir.Operation) error {
	function := &e.functions[functionIndex]
	resource, found := e.resourceLocals[functionIndex][operation.Symbol]
	if !found || operation.DestinationLocal != mir.LocalID(operation.Symbol) {
		return fmt.Errorf("resource initialization in %s does not name its declared local", function.Name)
	}
	if !e.supportedNeedsDropType(resource.Type) {
		return CapabilityError{Function: function.Name, Feature: "resource initialization type " + strconv.Quote(string(resource.Type))}
	}
	operand, err := oneOperand(operation.Operands)
	if err != nil {
		return fmt.Errorf("resource initialization in %s: %w", function.Name, err)
	}
	valueType, err := e.operandType(functionIndex, operand)
	if err != nil {
		return fmt.Errorf("resource initialization in %s: %w", function.Name, err)
	}
	if canonicalCValueType(valueType) != canonicalCValueType(resource.Type) {
		return fmt.Errorf("resource initialization in %s has value type %q, want %q", function.Name, valueType, resource.Type)
	}
	return nil
}

func (e *emitter) inspectResourceDischarge(functionIndex int, operation mir.Operation) error {
	function := &e.functions[functionIndex]
	resource, found := e.resourceLocals[functionIndex][operation.Symbol]
	if !found {
		return fmt.Errorf("%s in %s names undeclared resource local %d", operation.Kind, function.Name, operation.Symbol)
	}
	if !e.supportedNeedsDropType(resource.Type) {
		return CapabilityError{Function: function.Name, Feature: "resource discharge type " + strconv.Quote(string(resource.Type))}
	}
	if operation.Kind == mir.OperationMove && operation.Destination == "" {
		return fmt.Errorf("resource move in %s has no destination class", function.Name)
	}
	if operation.Kind == mir.OperationDrop && (operation.DropFlagEntity != "" || !operation.FlagClearedFirst) {
		return CapabilityError{Function: function.Name, Feature: "conditional resource drop"}
	}
	return nil
}

func (e *emitter) inspectResourceOperation(functionIndex int, operation mir.Operation) error {
	function := &e.functions[functionIndex]
	switch operation.Kind {
	case mir.OperationResourceConstruct:
		index, found := e.resourceByType[operation.Type]
		if !found {
			return CapabilityError{Function: function.Name, Feature: "resource construction type " + strconv.Quote(string(operation.Type))}
		}
		definition := &e.resourceTypes[index]
		if len(operation.Fields) != len(definition.definition.Fields) || len(operation.Operands) != len(definition.definition.Fields) {
			return fmt.Errorf("resource construction in %s does not match type %q", function.Name, operation.Type)
		}
		for fieldIndex, field := range definition.definition.Fields {
			if operation.Fields[fieldIndex] != field.Name {
				return fmt.Errorf("resource construction in %s has field %q at index %d, want %q", function.Name, operation.Fields[fieldIndex], fieldIndex, field.Name)
			}
			operandType, err := e.operandType(functionIndex, operation.Operands[fieldIndex])
			if err != nil {
				return fmt.Errorf("resource construction field %d in %s: %w", fieldIndex, function.Name, err)
			}
			if canonicalCValueType(operandType) != hir.TypeI32 {
				return CapabilityError{Function: function.Name, Feature: "non-i32 resource construction field"}
			}
		}
	case mir.OperationFieldRead:
		index, found := e.resourceByType[operation.OwnerType]
		if !found {
			return CapabilityError{Function: function.Name, Feature: "field-read owner type " + strconv.Quote(string(operation.OwnerType))}
		}
		definition := &e.resourceTypes[index]
		if _, found := definition.fieldIndex[operation.Field]; !found || operation.Type != hir.TypeI32 {
			return CapabilityError{Function: function.Name, Feature: "non-i32 resource field read"}
		}
		receiver, err := oneOperand(operation.Operands)
		if err != nil {
			return fmt.Errorf("resource field read in %s: %w", function.Name, err)
		}
		receiverType, err := e.operandType(functionIndex, receiver)
		if err != nil {
			return fmt.Errorf("resource field read in %s: %w", function.Name, err)
		}
		if canonicalCValueType(receiverType) != canonicalCValueType(operation.OwnerType) {
			return fmt.Errorf("resource field read in %s has receiver type %q, want %q", function.Name, receiverType, operation.OwnerType)
		}
	}
	return nil
}

func (e *emitter) inspectTaggedOperation(functionIndex int, operation mir.Operation) error {
	function := &e.functions[functionIndex]
	switch operation.Kind {
	case mir.OperationResultConstruct:
		successType, failureType, feature, supported := e.resultCSpec(operation.Type)
		if !supported {
			return CapabilityError{Function: function.Name, Feature: "Result construction type " + strconv.Quote(string(operation.Type))}
		}
		expectedType := successType
		if operation.Variant == "err" {
			expectedType = failureType
		} else if operation.Variant != "ok" {
			return fmt.Errorf("Result construction in %s has invalid variant %q", function.Name, operation.Variant)
		}
		operand, err := oneOperand(operation.Operands)
		if err != nil {
			return fmt.Errorf("Result construction in %s: %w", function.Name, err)
		}
		payloadType, err := e.operandType(functionIndex, operand)
		if err != nil || canonicalCValueType(payloadType) != canonicalCValueType(expectedType) {
			return CapabilityError{Function: function.Name, Feature: "Result payload type " + strconv.Quote(string(payloadType))}
		}
		e.features[feature] = struct{}{}
	case mir.OperationResultPayload:
		operand, err := oneOperand(operation.Operands)
		if err != nil {
			return fmt.Errorf("Result payload in %s: %w", function.Name, err)
		}
		containerType, err := e.operandType(functionIndex, operand)
		if err != nil {
			return CapabilityError{Function: function.Name, Feature: "Result payload source type " + strconv.Quote(string(containerType))}
		}
		successType, failureType, feature, supported := e.resultCSpec(containerType)
		if !supported {
			return CapabilityError{Function: function.Name, Feature: "Result payload source type " + strconv.Quote(string(containerType))}
		}
		expectedType := successType
		if operation.Variant == "err" {
			expectedType = failureType
		} else if operation.Variant != "ok" {
			return fmt.Errorf("Result payload in %s has invalid variant %q", function.Name, operation.Variant)
		}
		if canonicalCValueType(operation.Type) != canonicalCValueType(expectedType) || operation.MovePayload != e.supportedNeedsDropType(expectedType) {
			return CapabilityError{Function: function.Name, Feature: "Result payload extraction type " + strconv.Quote(string(operation.Type))}
		}
		e.features[feature] = struct{}{}
	case mir.OperationOptionConstruct:
		elementType, feature, supported := e.optionCSpec(operation.Type)
		if !supported {
			return CapabilityError{Function: function.Name, Feature: "Option construction type " + strconv.Quote(string(operation.Type))}
		}
		if operation.Variant == "some" {
			operand, err := oneOperand(operation.Operands)
			if err != nil {
				return fmt.Errorf("Option::Some construction in %s: %w", function.Name, err)
			}
			payloadType, err := e.operandType(functionIndex, operand)
			if err != nil || canonicalCValueType(payloadType) != canonicalCValueType(elementType) {
				return CapabilityError{Function: function.Name, Feature: "Option payload type " + strconv.Quote(string(payloadType))}
			}
		} else if operation.Variant != "none" || len(operation.Operands) != 0 {
			return fmt.Errorf("Option construction in %s has invalid variant %q", function.Name, operation.Variant)
		}
		e.features[feature] = struct{}{}
	case mir.OperationOptionPayload:
		operand, err := oneOperand(operation.Operands)
		if err != nil {
			return fmt.Errorf("Option payload in %s: %w", function.Name, err)
		}
		containerType, err := e.operandType(functionIndex, operand)
		if err != nil {
			return CapabilityError{Function: function.Name, Feature: "Option payload source type " + strconv.Quote(string(containerType))}
		}
		elementType, feature, supported := e.optionCSpec(containerType)
		if !supported {
			return CapabilityError{Function: function.Name, Feature: "Option payload source type " + strconv.Quote(string(containerType))}
		}
		if operation.Variant != "some" || canonicalCValueType(operation.Type) != canonicalCValueType(elementType) || operation.MovePayload != e.supportedNeedsDropType(elementType) {
			return CapabilityError{Function: function.Name, Feature: "Option payload extraction type " + strconv.Quote(string(operation.Type))}
		}
		e.features[feature] = struct{}{}
	case mir.OperationResultSwitch:
		operand, err := oneOperand(operation.Operands)
		if err != nil {
			return fmt.Errorf("Result switch marker in %s: %w", function.Name, err)
		}
		containerType, err := e.operandType(functionIndex, operand)
		if err != nil {
			return CapabilityError{Function: function.Name, Feature: "Result switch source type " + strconv.Quote(string(containerType))}
		}
		_, _, feature, supported := e.resultCSpec(containerType)
		if !supported {
			return CapabilityError{Function: function.Name, Feature: "Result switch source type " + strconv.Quote(string(containerType))}
		}
		e.features[feature] = struct{}{}
	}
	return nil
}

func (e *emitter) inspectSwitch(functionIndex int, terminator mir.Terminator) error {
	function := &e.functions[functionIndex]
	if terminator.SwitchKind == "enum" {
		index, found := e.enumByType[terminator.SwitchType]
		if !found {
			return CapabilityError{Function: function.Name, Feature: "enum switch type " + strconv.Quote(string(terminator.SwitchType))}
		}
		definition := &e.enumTypes[index]
		if len(terminator.Successors) != len(definition.definition.Variants) || len(terminator.SwitchVariants) != len(definition.definition.Variants) {
			return fmt.Errorf("enum switch in %s has an incomplete branch table", function.Name)
		}
		for variantIndex, variant := range definition.definition.Variants {
			if terminator.SwitchVariants[variantIndex] != variant.Name {
				return fmt.Errorf("enum switch in %s has variant %q at index %d, want %q", function.Name, terminator.SwitchVariants[variantIndex], variantIndex, variant.Name)
			}
		}
		operand, err := oneOperand(terminator.SwitchOperands)
		if err != nil {
			return fmt.Errorf("enum switch in %s: %w", function.Name, err)
		}
		valueType, err := e.operandType(functionIndex, operand)
		if err != nil {
			return fmt.Errorf("enum switch in %s: %w", function.Name, err)
		}
		if canonicalCValueType(valueType) != canonicalCValueType(terminator.SwitchType) {
			return fmt.Errorf("enum switch in %s has type %q, want %q", function.Name, valueType, terminator.SwitchType)
		}
		e.features["inert-enum"] = struct{}{}
		e.features["trap"] = struct{}{}
		e.trapCategories["unreachable"] = struct{}{}
		return nil
	}
	if len(terminator.Successors) != 2 {
		return fmt.Errorf("switch in %s has %d successors", function.Name, len(terminator.Successors))
	}
	var categories []string
	switch terminator.SwitchKind {
	case "bool":
		return nil
	case "option":
		operand, err := oneOperand(terminator.SwitchOperands)
		if err != nil {
			return fmt.Errorf("Option switch in %s: %w", function.Name, err)
		}
		valueType, err := e.operandType(functionIndex, operand)
		if err != nil {
			return CapabilityError{Function: function.Name, Feature: "Option switch type " + strconv.Quote(string(valueType))}
		}
		_, feature, supported := e.optionCSpec(valueType)
		if !supported {
			return CapabilityError{Function: function.Name, Feature: "Option switch type " + strconv.Quote(string(valueType))}
		}
		e.features[feature] = struct{}{}
		e.features["trap"] = struct{}{}
		e.trapCategories["unreachable"] = struct{}{}
		return nil
	case "result":
		operand, err := oneOperand(terminator.SwitchOperands)
		if err != nil {
			return fmt.Errorf("Result switch in %s: %w", function.Name, err)
		}
		valueType, err := e.operandType(functionIndex, operand)
		if err != nil {
			return CapabilityError{Function: function.Name, Feature: "Result switch type " + strconv.Quote(string(valueType))}
		}
		_, _, feature, supported := e.resultCSpec(valueType)
		if !supported {
			return CapabilityError{Function: function.Name, Feature: "Result switch type " + strconv.Quote(string(valueType))}
		}
		e.features[feature] = struct{}{}
		e.features["trap"] = struct{}{}
		e.trapCategories["unreachable"] = struct{}{}
		return nil
	case "checked-division":
		e.features["checked-division"] = struct{}{}
		categories = []string{"division-by-zero", "signed-division-overflow"}
	case "checked-unary", "checked-arithmetic":
		e.features["checked-arithmetic"] = struct{}{}
		categories = []string{"integer-overflow"}
	case "checked-shift":
		e.features["checked-shift"] = struct{}{}
		categories = []string{"invalid-shift"}
	default:
		return CapabilityError{Function: function.Name, Feature: "MIR switch " + strconv.Quote(terminator.SwitchKind)}
	}
	if err := e.recordTrapCategories(function.Name, categories); err != nil {
		return err
	}
	targets := e.dynamicTrapTargets[functionIndex]
	if targets == nil {
		targets = make(map[mir.BlockID]struct{})
		e.dynamicTrapTargets[functionIndex] = targets
	}
	targets[terminator.Successors[1]] = struct{}{}
	e.trapCategories["unreachable"] = struct{}{}
	return nil
}

func (e *emitter) recordTrapCategories(function string, categories []string) error {
	if len(categories) == 0 {
		return fmt.Errorf("trap in %s has no category", function)
	}
	for _, category := range categories {
		if _, _, ok := cTrapSpec(category); !ok {
			return CapabilityError{Function: function, Feature: "trap category " + strconv.Quote(category)}
		}
		e.trapCategories[category] = struct{}{}
	}
	return nil
}

func (e *emitter) emitPreamble(output *strings.Builder) error {
	_, usesCSSizeT := e.features["c-ssize-t"]
	if usesCSSizeT {
		output.WriteString("#define _POSIX_C_SOURCE 200809L\n")
	}
	transportProfile := "ISO C11"
	if usesCSSizeT {
		transportProfile += " with POSIX.1-2008 typedefs"
	}
	fmt.Fprintf(output, "/* fairyc constrained C transport: %s, %s */\n", target.X8664LinuxGNU, transportProfile)
	output.WriteString("#include <stdint.h>\n")
	_, usesCSizeT := e.features["c-size-t"]
	if len(e.structTypes) != 0 || len(e.resourceTypes) != 0 || usesCSizeT {
		output.WriteString("#include <stddef.h>\n")
	}
	if usesCSSizeT {
		output.WriteString("#include <sys/types.h>\n")
	}
	if e.needsBitcast {
		output.WriteString("#include <string.h>\n")
	}
	output.WriteByte('\n')
	if _, used := e.features["c-int"]; used {
		fmt.Fprintf(output, "_Static_assert(sizeof(int) == UINT64_C(%d), \"fairyc target c.int size\");\n", e.cIntLayout.SizeBytes)
		fmt.Fprintf(output, "_Static_assert(_Alignof(int) == UINT64_C(%d), \"fairyc target c.int alignment\");\n\n", e.cIntLayout.AlignmentBytes)
	}
	if usesCSizeT {
		fmt.Fprintf(output, "_Static_assert(sizeof(size_t) == UINT64_C(%d), \"fairyc target c.size_t size\");\n", e.cSizeTLayout.SizeBytes)
		fmt.Fprintf(output, "_Static_assert(_Alignof(size_t) == UINT64_C(%d), \"fairyc target c.size_t alignment\");\n", e.cSizeTLayout.AlignmentBytes)
		output.WriteString("_Static_assert(((size_t)-1) > (size_t)0, \"fairyc target c.size_t unsigned\");\n\n")
	}
	if usesCSSizeT {
		fmt.Fprintf(output, "_Static_assert(sizeof(ssize_t) == UINT64_C(%d), \"fairyc target c.ssize_t size\");\n", e.cSSizeTLayout.SizeBytes)
		fmt.Fprintf(output, "_Static_assert(_Alignof(ssize_t) == UINT64_C(%d), \"fairyc target c.ssize_t alignment\");\n", e.cSSizeTLayout.AlignmentBytes)
		output.WriteString("_Static_assert(((ssize_t)-1) < (ssize_t)0, \"fairyc target c.ssize_t signed\");\n\n")
	}
	for index, structure := range e.structTypes {
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
	}
	for enumIndex, enumeration := range e.enumTypes {
		fmt.Fprintf(output, "typedef union {\n    uint8_t unused;\n")
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
		fmt.Fprintf(output, "typedef struct { uint32_t tag; fy_enum_payload_%d payload; } fy_enum_%d;\n\n", enumIndex, enumIndex)
	}
	for index, resource := range e.resourceTypes {
		output.WriteString("typedef struct {\n")
		for fieldIndex := range resource.definition.Fields {
			fmt.Fprintf(output, "    int32_t f_%d;\n", fieldIndex)
		}
		fmt.Fprintf(output, "} fy_resource_%d;\n", index)
		fmt.Fprintf(output, "_Static_assert(sizeof(fy_resource_%d) == UINT64_C(%d), \"fairyc resource %d size\");\n", index, resource.layout.SizeBytes, index)
		fmt.Fprintf(output, "_Static_assert(_Alignof(fy_resource_%d) == UINT64_C(%d), \"fairyc resource %d alignment\");\n", index, resource.layout.AlignmentBytes, index)
		for fieldIndex, field := range resource.layout.Fields {
			fmt.Fprintf(output, "_Static_assert(offsetof(fy_resource_%d, f_%d) == UINT64_C(%d), \"fairyc resource %d field %d offset\");\n", index, fieldIndex, field.OffsetBytes, index, fieldIndex)
		}
		output.WriteByte('\n')
	}
	_, scalarOption := e.features["scalar-option"]
	_, resourceOption := e.features["resource-option"]
	if scalarOption || resourceOption {
		output.WriteString("#define FY_OPTION_NONE UINT32_C(0)\n")
		output.WriteString("#define FY_OPTION_SOME UINT32_C(1)\n")
	}
	if scalarOption {
		output.WriteString("typedef struct { uint32_t tag; int32_t payload; } fy_option_i32;\n\n")
	}
	_, scalarResult := e.features["scalar-result"]
	_, resourceResult := e.features["resource-result"]
	if scalarResult || resourceResult {
		output.WriteString("#define FY_RESULT_OK UINT32_C(0)\n")
		output.WriteString("#define FY_RESULT_ERR UINT32_C(1)\n")
	}
	if scalarResult {
		output.WriteString("typedef struct { uint32_t tag; int32_t payload; } fy_result_i32_i32;\n\n")
	}
	if err := e.emitResourceTaggedTypes(output); err != nil {
		return err
	}
	if e.instrumentation == InstrumentationDropTraceV0 {
		output.WriteString("void fy_runtime_trace_drop(uint32_t site_id);\n\n")
	}
	if len(e.trapCategories) != 0 {
		categories := make([]string, 0, len(e.trapCategories))
		for category := range e.trapCategories {
			categories = append(categories, category)
		}
		slices.Sort(categories)
		for _, category := range categories {
			name, code, _ := cTrapSpec(category)
			fmt.Fprintf(output, "#define %s UINT32_C(%d)\n", name, code)
		}
		output.WriteString("_Noreturn void fy_runtime_trap(uint32_t category);\n\n")
	}
	if e.needsBitcast {
		output.WriteString("static int32_t fy_bits_to_i32(uint32_t bits) {\n")
		output.WriteString("    int32_t value;\n    memcpy(&value, &bits, sizeof value);\n    return value;\n}\n\n")
	}
	if e.needsShift {
		output.WriteString("static int32_t fy_sar_i32(int32_t value, int32_t amount) {\n")
		output.WriteString("    uint32_t bits = (uint32_t)value;\n")
		output.WriteString("    if (amount == 0 || value >= 0) {\n        return fy_bits_to_i32(bits >> (uint32_t)amount);\n    }\n")
		output.WriteString("    return fy_bits_to_i32((bits >> (uint32_t)amount) | (UINT32_MAX << (32u - (uint32_t)amount)));\n}\n\n")
	}
	return nil
}

func (e *emitter) structCFieldName(structIndex, fieldIndex int) string {
	definition := e.structTypes[structIndex].definition
	if definition.ReprC {
		return definition.Fields[fieldIndex].Name
	}
	return fmt.Sprintf("f_%d", fieldIndex)
}

func (e *emitter) emitPrototype(output *strings.Builder, functionIndex int) error {
	function := &e.functions[functionIndex]
	resultType, err := e.cFunctionResultType(function.ResultType)
	if err != nil {
		return fmt.Errorf("prototype for %s: %w", function.Name, err)
	}
	fmt.Fprintf(output, "static %s fy_fn_%d(", resultType, functionIndex)
	if err := e.emitParameters(output, function.Parameters, function.ParameterTypes); err != nil {
		return fmt.Errorf("prototype for %s: %w", function.Name, err)
	}
	output.WriteString(");\n")
	return nil
}

func (e *emitter) emitForeignPrototype(output *strings.Builder, functionIndex int) error {
	function := &e.foreignFunctions[functionIndex]
	resultType, err := e.foreignCType(function.ResultType, true)
	if err != nil {
		return fmt.Errorf("foreign prototype for %s: %w", function.Name, err)
	}
	fmt.Fprintf(output, "extern %s %s(", resultType, function.LinkName)
	if len(function.ParameterTypes) == 0 {
		output.WriteString("void")
	} else {
		for index, parameterType := range function.ParameterTypes {
			if index != 0 {
				output.WriteString(", ")
			}
			cType, typeErr := e.foreignCType(parameterType, false)
			if typeErr != nil {
				return fmt.Errorf("foreign prototype for %s: %w", function.Name, typeErr)
			}
			fmt.Fprintf(output, "%s fy_a_%d", cType, index)
		}
	}
	output.WriteString(");\n")
	return nil
}

func (e *emitter) emitForeignMacroGuard(output *strings.Builder, functionIndex int) {
	linkName := e.foreignFunctions[functionIndex].LinkName
	fmt.Fprintf(output, "#ifdef %s\n#undef %s\n#endif\n", linkName, linkName)
}

func (e *emitter) emitFunction(output *strings.Builder, functionIndex int) error {
	function := &e.functions[functionIndex]
	resultType, err := e.cFunctionResultType(function.ResultType)
	if err != nil {
		return fmt.Errorf("function %s: %w", function.Name, err)
	}
	fmt.Fprintf(output, "static %s fy_fn_%d(", resultType, functionIndex)
	if err := e.emitParameters(output, function.Parameters, function.ParameterTypes); err != nil {
		return fmt.Errorf("function %s: %w", function.Name, err)
	}
	output.WriteString(") {\n")
	parameters := make(map[mir.LocalID]struct{}, len(function.Parameters))
	for _, parameter := range function.Parameters {
		parameters[parameter] = struct{}{}
		fmt.Fprintf(output, "    (void)fy_l_%d;\n", parameter)
	}
	for _, local := range collectLocals(*function) {
		if _, parameter := parameters[local]; parameter {
			continue
		}
		localType, found := e.localTypes[functionIndex][local]
		if !found {
			return fmt.Errorf("function %s local %d has no C type", function.Name, local)
		}
		cType, err := e.cValueType(localType)
		if err != nil {
			return fmt.Errorf("function %s local %d: %w", function.Name, local, err)
		}
		fmt.Fprintf(output, "    %s fy_l_%d;\n", cType, local)
	}
	dynamicTargets := e.dynamicTrapTargets[functionIndex]
	if len(dynamicTargets) != 0 {
		output.WriteString("    uint32_t fy_pending_trap = FY_TRAP_UNREACHABLE;\n")
	}
	fmt.Fprintf(output, "    goto fy_b_%d;\n", function.Entry)
	blocks := slices.Clone(function.Blocks)
	slices.SortFunc(blocks, func(left, right mir.Block) int { return int(left.ID - right.ID) })
	for blockIndex := range blocks {
		block := &blocks[blockIndex]
		fmt.Fprintf(output, "fy_b_%d:\n", block.ID)
		for operationIndex := range block.Operations {
			if err := e.emitOperation(output, functionIndex, block.Operations[operationIndex]); err != nil {
				return err
			}
		}
		if err := e.emitTerminator(output, functionIndex, block.ID, block.Terminator); err != nil {
			return err
		}
	}
	output.WriteString("}\n\n")
	return nil
}

func (e *emitter) emitOperation(output *strings.Builder, functionIndex int, operation mir.Operation) error {
	function := e.functions[functionIndex].Name
	switch operation.Kind {
	case mir.OperationAssign, mir.OperationInitialize:
		operand, err := oneOperand(operation.Operands)
		if err != nil {
			return fmt.Errorf("assign in %s: %w", function, err)
		}
		value, err := cOperand(operand)
		if err != nil {
			return fmt.Errorf("assign in %s: %w", function, err)
		}
		fmt.Fprintf(output, "    fy_l_%d = %s;\n", operation.DestinationLocal, value)
	case mir.OperationRawAddress:
		operand, err := oneOperand(operation.Operands)
		if err != nil || operand.Kind != mir.OperandLocal {
			return fmt.Errorf("raw address in %s has invalid local operand", function)
		}
		fmt.Fprintf(output, "    fy_l_%d = &fy_l_%d;\n", operation.DestinationLocal, operand.Local)
	case mir.OperationEnumConstruct:
		enumIndex, found := e.enumByType[operation.Type]
		if !found {
			return CapabilityError{Function: function, Feature: "enum construction type " + strconv.Quote(string(operation.Type))}
		}
		enumeration := &e.enumTypes[enumIndex]
		variantIndex, found := enumeration.variantIndex[operation.Variant]
		if !found {
			return fmt.Errorf("enum construction in %s names missing variant %q", function, operation.Variant)
		}
		fmt.Fprintf(output, "    fy_l_%d = (fy_enum_%d){ .tag = UINT32_C(%d)", operation.DestinationLocal, enumIndex, variantIndex)
		if len(operation.Operands) != 0 {
			fmt.Fprintf(output, ", .payload.v_%d = { ", variantIndex)
			for operandIndex, operand := range operation.Operands {
				if operandIndex != 0 {
					output.WriteString(", ")
				}
				fieldIndex, declared := enumeration.fieldIndex[variantIndex][operation.Fields[operandIndex]]
				if !declared {
					return fmt.Errorf("enum construction in %s names missing field %q", function, operation.Fields[operandIndex])
				}
				value, err := cOperand(operand)
				if err != nil {
					return fmt.Errorf("enum construction field %d in %s: %w", operandIndex, function, err)
				}
				fmt.Fprintf(output, ".f_%d = %s", fieldIndex, value)
			}
			output.WriteString(" }")
		}
		output.WriteString(" };\n")
	case mir.OperationEnumPayload:
		enumIndex, found := e.enumByType[operation.OwnerType]
		if !found {
			return CapabilityError{Function: function, Feature: "enum payload type " + strconv.Quote(string(operation.OwnerType))}
		}
		enumeration := &e.enumTypes[enumIndex]
		variantIndex, found := enumeration.variantIndex[operation.Variant]
		if !found {
			return fmt.Errorf("enum payload in %s names missing variant %q", function, operation.Variant)
		}
		fieldIndex, found := enumeration.fieldIndex[variantIndex][operation.Field]
		if !found {
			return fmt.Errorf("enum payload in %s names missing field %q", function, operation.Field)
		}
		container, err := oneOperand(operation.Operands)
		if err != nil {
			return fmt.Errorf("enum payload in %s: %w", function, err)
		}
		value, err := cOperand(container)
		if err != nil {
			return fmt.Errorf("enum payload in %s: %w", function, err)
		}
		fmt.Fprintf(output, "    fy_l_%d = (%s).payload.v_%d.f_%d;\n", operation.DestinationLocal, value, variantIndex, fieldIndex)
	case mir.OperationStructConstruct:
		index, found := e.structByType[operation.Type]
		if !found {
			return CapabilityError{Function: function, Feature: "struct construction type " + strconv.Quote(string(operation.Type))}
		}
		fmt.Fprintf(output, "    fy_l_%d = (%s){ ", operation.DestinationLocal, e.structCTypeName(index))
		for fieldIndex, operand := range operation.Operands {
			if fieldIndex != 0 {
				output.WriteString(", ")
			}
			value, err := cOperand(operand)
			if err != nil {
				return fmt.Errorf("struct construction field %d in %s: %w", fieldIndex, function, err)
			}
			fmt.Fprintf(output, ".%s = %s", e.structCFieldName(index, fieldIndex), value)
		}
		output.WriteString(" };\n")
	case mir.OperationResourceConstruct:
		index, found := e.resourceByType[operation.Type]
		if !found {
			return CapabilityError{Function: function, Feature: "resource construction type " + strconv.Quote(string(operation.Type))}
		}
		fmt.Fprintf(output, "    fy_l_%d = (fy_resource_%d){ ", operation.DestinationLocal, index)
		for fieldIndex, operand := range operation.Operands {
			if fieldIndex != 0 {
				output.WriteString(", ")
			}
			value, err := cOperand(operand)
			if err != nil {
				return fmt.Errorf("resource construction field %d in %s: %w", fieldIndex, function, err)
			}
			fmt.Fprintf(output, ".f_%d = %s", fieldIndex, value)
		}
		output.WriteString(" };\n")
	case mir.OperationFieldRead:
		if index, found := e.structByType[operation.OwnerType]; found {
			fieldIndex, fieldFound := e.structTypes[index].fieldIndex[operation.Field]
			if !fieldFound {
				return fmt.Errorf("struct field read in %s names missing field %q", function, operation.Field)
			}
			receiver, err := oneOperand(operation.Operands)
			if err != nil {
				return fmt.Errorf("struct field read in %s: %w", function, err)
			}
			value, err := cOperand(receiver)
			if err != nil {
				return fmt.Errorf("struct field read in %s: %w", function, err)
			}
			fmt.Fprintf(output, "    fy_l_%d = (%s).%s;\n", operation.DestinationLocal, value, e.structCFieldName(index, fieldIndex))
			break
		}
		index, found := e.resourceByType[operation.OwnerType]
		if !found {
			return CapabilityError{Function: function, Feature: "field-read owner type " + strconv.Quote(string(operation.OwnerType))}
		}
		fieldIndex, found := e.resourceTypes[index].fieldIndex[operation.Field]
		if !found {
			return fmt.Errorf("resource field read in %s names missing field %q", function, operation.Field)
		}
		receiver, err := oneOperand(operation.Operands)
		if err != nil {
			return fmt.Errorf("resource field read in %s: %w", function, err)
		}
		value, err := cOperand(receiver)
		if err != nil {
			return fmt.Errorf("resource field read in %s: %w", function, err)
		}
		fmt.Fprintf(output, "    fy_l_%d = (%s).f_%d;\n", operation.DestinationLocal, value, fieldIndex)
	case mir.OperationMove:
		fmt.Fprintf(output, "    (void)fy_l_%d;\n", operation.Symbol)
	case mir.OperationDrop:
		fmt.Fprintf(output, "    (void)fy_l_%d;\n", operation.Symbol)
		if e.instrumentation == InstrumentationDropTraceV0 {
			if err := e.emitDropTrace(output, functionIndex, operation); err != nil {
				return fmt.Errorf("drop trace in %s: %w", function, err)
			}
		}
	case mir.OperationUnary:
		operand, err := oneOperand(operation.Operands)
		if err != nil {
			return fmt.Errorf("unary in %s: %w", function, err)
		}
		value, err := cOperand(operand)
		if err != nil {
			return fmt.Errorf("unary in %s: %w", function, err)
		}
		switch operation.Operator {
		case "+":
			fmt.Fprintf(output, "    fy_l_%d = %s;\n", operation.DestinationLocal, value)
		case "-":
			fmt.Fprintf(output, "    fy_l_%d = -(%s);\n", operation.DestinationLocal, value)
		case "!":
			fmt.Fprintf(output, "    fy_l_%d = (%s == INT32_C(0)) ? INT32_C(1) : INT32_C(0);\n", operation.DestinationLocal, value)
		case "~":
			fmt.Fprintf(output, "    fy_l_%d = fy_bits_to_i32(~(uint32_t)(%s));\n", operation.DestinationLocal, value)
		}
	case mir.OperationBinary:
		left, right, err := twoOperands(operation.Operands)
		if err != nil {
			return fmt.Errorf("binary in %s: %w", function, err)
		}
		leftValue, err := cOperand(left)
		if err != nil {
			return fmt.Errorf("binary in %s: %w", function, err)
		}
		rightValue, err := cOperand(right)
		if err != nil {
			return fmt.Errorf("binary in %s: %w", function, err)
		}
		switch operation.Operator {
		case "+", "-", "*", "/", "%":
			fmt.Fprintf(output, "    fy_l_%d = (%s) %s (%s);\n", operation.DestinationLocal, leftValue, operation.Operator, rightValue)
		case ">>":
			fmt.Fprintf(output, "    fy_l_%d = fy_sar_i32(%s, %s);\n", operation.DestinationLocal, leftValue, rightValue)
		case "&", "^", "|":
			fmt.Fprintf(output, "    fy_l_%d = fy_bits_to_i32((uint32_t)(%s) %s (uint32_t)(%s));\n", operation.DestinationLocal, leftValue, operation.Operator, rightValue)
		case "==", "!=", "<", "<=", ">", ">=":
			fmt.Fprintf(output, "    fy_l_%d = ((%s) %s (%s)) ? INT32_C(1) : INT32_C(0);\n", operation.DestinationLocal, leftValue, operation.Operator, rightValue)
		}
	case mir.OperationResultConstruct:
		operand, err := oneOperand(operation.Operands)
		if err != nil {
			return fmt.Errorf("Result construction in %s: %w", function, err)
		}
		payload, err := cOperand(operand)
		if err != nil {
			return fmt.Errorf("Result construction in %s: %w", function, err)
		}
		tag := "FY_RESULT_OK"
		field := "payload"
		cType := "fy_result_i32_i32"
		if operation.Variant == "err" {
			tag = "FY_RESULT_ERR"
		}
		if index, resourceTagged := e.taggedByType[operation.Type]; resourceTagged {
			cType = fmt.Sprintf("fy_tagged_%d", index)
			field = "p_0"
			if operation.Variant == "err" {
				field = "p_1"
			}
		}
		fmt.Fprintf(output, "    fy_l_%d = (%s){ .tag = %s, .%s = %s };\n", operation.DestinationLocal, cType, tag, field, payload)
	case mir.OperationResultPayload:
		operand, err := oneOperand(operation.Operands)
		if err != nil {
			return fmt.Errorf("Result payload in %s: %w", function, err)
		}
		container, err := cOperand(operand)
		if err != nil {
			return fmt.Errorf("Result payload in %s: %w", function, err)
		}
		containerType, err := e.operandType(functionIndex, operand)
		if err != nil {
			return fmt.Errorf("Result payload in %s: %w", function, err)
		}
		field := "payload"
		if _, resourceTagged := e.taggedByType[containerType]; resourceTagged {
			field = "p_0"
			if operation.Variant == "err" {
				field = "p_1"
			}
		}
		fmt.Fprintf(output, "    fy_l_%d = (%s).%s;\n", operation.DestinationLocal, container, field)
	case mir.OperationOptionConstruct:
		cType := "fy_option_i32"
		field := "payload"
		if index, resourceTagged := e.taggedByType[operation.Type]; resourceTagged {
			cType = fmt.Sprintf("fy_tagged_%d", index)
			field = "p_0"
		}
		if operation.Variant == "none" {
			fmt.Fprintf(output, "    fy_l_%d = (%s){ .tag = FY_OPTION_NONE };\n", operation.DestinationLocal, cType)
			break
		}
		operand, err := oneOperand(operation.Operands)
		if err != nil {
			return fmt.Errorf("Option::Some construction in %s: %w", function, err)
		}
		payload, err := cOperand(operand)
		if err != nil {
			return fmt.Errorf("Option::Some construction in %s: %w", function, err)
		}
		fmt.Fprintf(output, "    fy_l_%d = (%s){ .tag = FY_OPTION_SOME, .%s = %s };\n", operation.DestinationLocal, cType, field, payload)
	case mir.OperationOptionPayload:
		operand, err := oneOperand(operation.Operands)
		if err != nil {
			return fmt.Errorf("Option payload in %s: %w", function, err)
		}
		container, err := cOperand(operand)
		if err != nil {
			return fmt.Errorf("Option payload in %s: %w", function, err)
		}
		containerType, err := e.operandType(functionIndex, operand)
		if err != nil {
			return fmt.Errorf("Option payload in %s: %w", function, err)
		}
		field := "payload"
		if _, resourceTagged := e.taggedByType[containerType]; resourceTagged {
			field = "p_0"
		}
		fmt.Fprintf(output, "    fy_l_%d = (%s).%s;\n", operation.DestinationLocal, container, field)
	case mir.OperationUse, mir.OperationArithmeticCheck, mir.OperationCheckedDivision, mir.OperationResultSwitch:
	default:
		return CapabilityError{Function: function, Feature: "MIR operation " + strconv.Quote(string(operation.Kind))}
	}
	return nil
}

func (e *emitter) emitTerminator(output *strings.Builder, functionIndex int, blockID mir.BlockID, terminator mir.Terminator) error {
	function := &e.functions[functionIndex]
	switch terminator.Kind {
	case mir.TerminatorGoto:
		target, err := oneSuccessor(terminator)
		if err != nil {
			return fmt.Errorf("goto in %s: %w", function.Name, err)
		}
		fmt.Fprintf(output, "    goto fy_b_%d;\n", target)
	case mir.TerminatorCall:
		target, err := oneSuccessor(terminator)
		if err != nil {
			return fmt.Errorf("call in %s: %w", function.Name, err)
		}
		calleeResult, calleeParameters, foreign, found := e.callSignature(terminator.Callee)
		if !found {
			return fmt.Errorf("call in %s names missing function symbol %d", function.Name, terminator.Callee)
		}
		if len(terminator.Arguments) != len(calleeParameters) {
			return fmt.Errorf("call in %s passes %d arguments, want %d", function.Name, len(terminator.Arguments), len(calleeParameters))
		}
		output.WriteString("    ")
		if calleeResult != hir.TypeVoid {
			fmt.Fprintf(output, "fy_l_%d = ", terminator.DestinationLocal)
		}
		if foreign {
			fmt.Fprintf(output, "%s(", e.foreignFunctions[e.foreignBySymbol[terminator.Callee]].LinkName)
		} else {
			fmt.Fprintf(output, "fy_fn_%d(", e.bySymbol[terminator.Callee])
		}
		for index, argument := range terminator.Arguments {
			if index != 0 {
				output.WriteString(", ")
			}
			value, err := cOperand(argument)
			if err != nil {
				return fmt.Errorf("call argument in %s: %w", function.Name, err)
			}
			output.WriteString(value)
		}
		output.WriteString(");\n")
		fmt.Fprintf(output, "    goto fy_b_%d;\n", target)
	case mir.TerminatorSwitch:
		return e.emitSwitch(output, functionIndex, terminator)
	case mir.TerminatorReturn:
		if function.ResultType == hir.TypeVoid {
			output.WriteString("    return;\n")
			return nil
		}
		value, err := cOperand(terminator.ReturnValue)
		if err != nil {
			return fmt.Errorf("return in %s: %w", function.Name, err)
		}
		fmt.Fprintf(output, "    return %s;\n", value)
	case mir.TerminatorTrap:
		if _, dynamic := e.dynamicTrapTargets[functionIndex][blockID]; dynamic {
			output.WriteString("    fy_runtime_trap(fy_pending_trap);\n")
			return nil
		}
		if len(terminator.TrapCategories) == 0 {
			return fmt.Errorf("trap in %s has no category", function.Name)
		}
		name, _, ok := cTrapSpec(terminator.TrapCategories[0])
		if !ok {
			return CapabilityError{Function: function.Name, Feature: "trap category " + strconv.Quote(terminator.TrapCategories[0])}
		}
		fmt.Fprintf(output, "    fy_runtime_trap(%s);\n", name)
	case mir.TerminatorUnreachable:
		output.WriteString("    fy_runtime_trap(FY_TRAP_UNREACHABLE);\n")
	default:
		return CapabilityError{Function: function.Name, Feature: "MIR terminator " + strconv.Quote(string(terminator.Kind))}
	}
	return nil
}

func (e *emitter) emitSwitch(output *strings.Builder, functionIndex int, terminator mir.Terminator) error {
	function := &e.functions[functionIndex]
	if terminator.SwitchKind == "enum" {
		enumIndex, found := e.enumByType[terminator.SwitchType]
		if !found {
			return CapabilityError{Function: function.Name, Feature: "enum switch type " + strconv.Quote(string(terminator.SwitchType))}
		}
		if len(terminator.Successors) != len(terminator.SwitchVariants) {
			return fmt.Errorf("enum switch in %s has %d successors for %d variants", function.Name, len(terminator.Successors), len(terminator.SwitchVariants))
		}
		operand, err := oneOperand(terminator.SwitchOperands)
		if err != nil {
			return fmt.Errorf("enum switch in %s: %w", function.Name, err)
		}
		value, err := cOperand(operand)
		if err != nil {
			return fmt.Errorf("enum switch in %s: %w", function.Name, err)
		}
		enumeration := &e.enumTypes[enumIndex]
		for branchIndex, variant := range terminator.SwitchVariants {
			variantIndex, found := enumeration.variantIndex[variant]
			if !found {
				return fmt.Errorf("enum switch in %s names missing variant %q", function.Name, variant)
			}
			fmt.Fprintf(output, "    if ((%s).tag == UINT32_C(%d)) goto fy_b_%d;\n", value, variantIndex, terminator.Successors[branchIndex])
		}
		output.WriteString("    fy_runtime_trap(FY_TRAP_UNREACHABLE);\n")
		return nil
	}
	if len(terminator.Successors) != 2 {
		return fmt.Errorf("switch in %s has %d successors", function.Name, len(terminator.Successors))
	}
	success, failure := terminator.Successors[0], terminator.Successors[1]
	switch terminator.SwitchKind {
	case "bool":
		operand, err := oneOperand(terminator.SwitchOperands)
		if err != nil {
			return fmt.Errorf("bool switch in %s: %w", function.Name, err)
		}
		value, err := cOperand(operand)
		if err != nil {
			return fmt.Errorf("bool switch in %s: %w", function.Name, err)
		}
		fmt.Fprintf(output, "    if (%s != INT32_C(0)) goto fy_b_%d;\n", value, success)
		fmt.Fprintf(output, "    goto fy_b_%d;\n", failure)
	case "option":
		operand, err := oneOperand(terminator.SwitchOperands)
		if err != nil {
			return fmt.Errorf("Option switch in %s: %w", function.Name, err)
		}
		value, err := cOperand(operand)
		if err != nil {
			return fmt.Errorf("Option switch in %s: %w", function.Name, err)
		}
		fmt.Fprintf(output, "    if ((%s).tag == FY_OPTION_SOME) goto fy_b_%d;\n", value, success)
		fmt.Fprintf(output, "    if ((%s).tag == FY_OPTION_NONE) goto fy_b_%d;\n", value, failure)
		output.WriteString("    fy_runtime_trap(FY_TRAP_UNREACHABLE);\n")
	case "result":
		operand, err := oneOperand(terminator.SwitchOperands)
		if err != nil {
			return fmt.Errorf("Result switch in %s: %w", function.Name, err)
		}
		value, err := cOperand(operand)
		if err != nil {
			return fmt.Errorf("Result switch in %s: %w", function.Name, err)
		}
		fmt.Fprintf(output, "    if ((%s).tag == FY_RESULT_OK) goto fy_b_%d;\n", value, success)
		fmt.Fprintf(output, "    if ((%s).tag == FY_RESULT_ERR) goto fy_b_%d;\n", value, failure)
		output.WriteString("    fy_runtime_trap(FY_TRAP_UNREACHABLE);\n")
	case "checked-unary":
		operand, err := oneOperand(terminator.SwitchOperands)
		if err != nil {
			return fmt.Errorf("checked unary in %s: %w", function.Name, err)
		}
		if terminator.Operator != "-" {
			return CapabilityError{Function: function.Name, Feature: "checked unary operator " + strconv.Quote(terminator.Operator)}
		}
		value, err := cOperand(operand)
		if err != nil {
			return fmt.Errorf("checked unary in %s: %w", function.Name, err)
		}
		fmt.Fprintf(output, "    if (%s == INT32_MIN) { fy_pending_trap = FY_TRAP_INTEGER_OVERFLOW; goto fy_b_%d; }\n", value, failure)
		fmt.Fprintf(output, "    goto fy_b_%d;\n", success)
	case "checked-arithmetic":
		left, right, err := twoOperands(terminator.SwitchOperands)
		if err != nil {
			return fmt.Errorf("checked arithmetic in %s: %w", function.Name, err)
		}
		if terminator.Operator != "+" && terminator.Operator != "-" && terminator.Operator != "*" {
			return CapabilityError{Function: function.Name, Feature: "checked arithmetic operator " + strconv.Quote(terminator.Operator)}
		}
		leftValue, err := cOperand(left)
		if err != nil {
			return fmt.Errorf("checked arithmetic in %s: %w", function.Name, err)
		}
		rightValue, err := cOperand(right)
		if err != nil {
			return fmt.Errorf("checked arithmetic in %s: %w", function.Name, err)
		}
		expression := fmt.Sprintf("((int64_t)(%s) %s (int64_t)(%s))", leftValue, terminator.Operator, rightValue)
		fmt.Fprintf(output, "    if (%s < (int64_t)INT32_MIN || %s > (int64_t)INT32_MAX) { fy_pending_trap = FY_TRAP_INTEGER_OVERFLOW; goto fy_b_%d; }\n", expression, expression, failure)
		fmt.Fprintf(output, "    goto fy_b_%d;\n", success)
	case "checked-division":
		left, right, err := twoOperands(terminator.SwitchOperands)
		if err != nil {
			return fmt.Errorf("checked division in %s: %w", function.Name, err)
		}
		leftValue, err := cOperand(left)
		if err != nil {
			return fmt.Errorf("checked division in %s: %w", function.Name, err)
		}
		rightValue, err := cOperand(right)
		if err != nil {
			return fmt.Errorf("checked division in %s: %w", function.Name, err)
		}
		fmt.Fprintf(output, "    if (%s == INT32_C(0)) { fy_pending_trap = FY_TRAP_DIVISION_BY_ZERO; goto fy_b_%d; }\n", rightValue, failure)
		fmt.Fprintf(output, "    if (%s == INT32_MIN && %s == -INT32_C(1)) { fy_pending_trap = FY_TRAP_SIGNED_DIVISION_OVERFLOW; goto fy_b_%d; }\n", leftValue, rightValue, failure)
		fmt.Fprintf(output, "    goto fy_b_%d;\n", success)
	case "checked-shift":
		_, amount, err := twoOperands(terminator.SwitchOperands)
		if err != nil {
			return fmt.Errorf("checked shift in %s: %w", function.Name, err)
		}
		amountValue, err := cOperand(amount)
		if err != nil {
			return fmt.Errorf("checked shift in %s: %w", function.Name, err)
		}
		fmt.Fprintf(output, "    if (%s < INT32_C(0) || %s >= INT32_C(32)) { fy_pending_trap = FY_TRAP_INVALID_SHIFT; goto fy_b_%d; }\n", amountValue, amountValue, failure)
		fmt.Fprintf(output, "    goto fy_b_%d;\n", success)
	default:
		return CapabilityError{Function: function.Name, Feature: "MIR switch " + strconv.Quote(terminator.SwitchKind)}
	}
	return nil
}

func (e *emitter) emitParameters(output *strings.Builder, parameters []mir.LocalID, parameterTypes []hir.Type) error {
	if len(parameters) != len(parameterTypes) {
		return fmt.Errorf("parameter type count %d does not match parameter count %d", len(parameterTypes), len(parameters))
	}
	if len(parameters) == 0 {
		output.WriteString("void")
		return nil
	}
	for index, parameter := range parameters {
		if index != 0 {
			output.WriteString(", ")
		}
		cType, err := e.cValueType(parameterTypes[index])
		if err != nil {
			return err
		}
		fmt.Fprintf(output, "%s fy_l_%d", cType, parameter)
	}
	return nil
}

func collectLocals(function mir.Function) []mir.LocalID {
	values := make(map[mir.LocalID]struct{})
	for _, parameter := range function.Parameters {
		values[parameter] = struct{}{}
	}
	addOperand := func(operand mir.Operand) {
		if operand.Kind == mir.OperandLocal {
			values[operand.Local] = struct{}{}
		}
	}
	for _, block := range function.Blocks {
		for _, operation := range block.Operations {
			if operationHasDestination(operation.Kind) && operation.DestinationLocal != mir.NoLocal {
				values[operation.DestinationLocal] = struct{}{}
			}
			for _, operand := range operation.Operands {
				addOperand(operand)
			}
		}
		terminator := block.Terminator
		if terminator.Kind == mir.TerminatorCall && terminator.DestinationLocal != mir.NoLocal {
			values[terminator.DestinationLocal] = struct{}{}
		}
		for _, operand := range terminator.Arguments {
			addOperand(operand)
		}
		for _, operand := range terminator.SwitchOperands {
			addOperand(operand)
		}
		addOperand(terminator.ReturnValue)
	}
	result := make([]mir.LocalID, 0, len(values))
	for value := range values {
		if value >= 0 {
			result = append(result, value)
		}
	}
	slices.Sort(result)
	return result
}

func operationHasDestination(kind mir.OperationKind) bool {
	switch kind {
	case mir.OperationInitialize, mir.OperationAssign, mir.OperationUnary, mir.OperationBinary,
		mir.OperationResultConstruct, mir.OperationResultPayload,
		mir.OperationOptionConstruct, mir.OperationOptionPayload,
		mir.OperationEnumConstruct, mir.OperationEnumPayload,
		mir.OperationRawAddress, mir.OperationStructConstruct, mir.OperationResourceConstruct, mir.OperationFieldRead, mir.OperationRegionPlace:
		return true
	default:
		return false
	}
}

func (e *emitter) cFunctionResultType(value hir.Type) (string, error) {
	if value == hir.TypeVoid {
		return "void", nil
	}
	return e.cValueType(value)
}

func cOperand(operand mir.Operand) (string, error) {
	switch operand.Kind {
	case mir.OperandLocal:
		if operand.Local < 0 {
			return "", fmt.Errorf("local operand has invalid id %d", operand.Local)
		}
		return fmt.Sprintf("fy_l_%d", operand.Local), nil
	case mir.OperandI32:
		return cI32(operand.I32), nil
	case mir.OperandU8:
		return fmt.Sprintf("UINT8_C(%d)", operand.U64), nil
	case mir.OperandU32:
		return fmt.Sprintf("UINT32_C(%d)", operand.U64), nil
	case mir.OperandUSize:
		return fmt.Sprintf("((uintptr_t)UINT64_C(%d))", operand.U64), nil
	case mir.OperandNull:
		return "((void *)0)", nil
	default:
		return "", fmt.Errorf("unsupported scalar operand kind %q", operand.Kind)
	}
}

func isRawPointer(value hir.Type) bool {
	_, _, pointer := hir.RawPointerElement(value)
	return pointer
}

func cI32(value int32) string {
	if value == -2147483648 {
		return "INT32_MIN"
	}
	if value < 0 {
		return fmt.Sprintf("(-INT32_C(%d))", -int64(value))
	}
	return fmt.Sprintf("INT32_C(%d)", value)
}

// TrapCode returns the fixed Stage 0 runtime code for one canonical trap category.
func TrapCode(category string) (uint32, bool) {
	_, code, ok := cTrapSpec(category)
	return code, ok
}

func cTrapSpec(category string) (string, uint32, bool) {
	switch category {
	case "bounds":
		return "FY_TRAP_BOUNDS", 1, true
	case "integer-overflow":
		return "FY_TRAP_INTEGER_OVERFLOW", 2, true
	case "division-by-zero":
		return "FY_TRAP_DIVISION_BY_ZERO", 3, true
	case "signed-division-overflow":
		return "FY_TRAP_SIGNED_DIVISION_OVERFLOW", 4, true
	case "invalid-shift":
		return "FY_TRAP_INVALID_SHIFT", 5, true
	case "assertion-failed":
		return "FY_TRAP_ASSERTION_FAILED", 6, true
	case "unreachable":
		return "FY_TRAP_UNREACHABLE", 7, true
	default:
		return "", 0, false
	}
}

func supportedUnary(operator string) bool {
	return operator == "+" || operator == "-" || operator == "!" || operator == "~"
}

func supportedBinary(operator string) bool {
	switch operator {
	case "+", "-", "*", "/", "%", ">>", "&", "^", "|", "==", "!=", "<", "<=", ">", ">=":
		return true
	default:
		return false
	}
}

func oneOperand(operands []mir.Operand) (mir.Operand, error) {
	if len(operands) != 1 || operands[0].Kind == mir.OperandInvalid {
		return mir.Operand{}, fmt.Errorf("expected one scalar operand")
	}
	return operands[0], nil
}

func twoOperands(operands []mir.Operand) (mir.Operand, mir.Operand, error) {
	if len(operands) != 2 || operands[0].Kind == mir.OperandInvalid || operands[1].Kind == mir.OperandInvalid {
		return mir.Operand{}, mir.Operand{}, fmt.Errorf("expected two scalar operands")
	}
	return operands[0], operands[1], nil
}

func oneSuccessor(terminator mir.Terminator) (mir.BlockID, error) {
	if len(terminator.Successors) != 1 {
		return mir.NoBlock, fmt.Errorf("expected one successor, got %d", len(terminator.Successors))
	}
	return terminator.Successors[0], nil
}
