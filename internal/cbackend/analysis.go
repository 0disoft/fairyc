package cbackend

import (
	"fmt"
	"slices"
	"strings"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/layout"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/target"
)

func analyzeEmission(program mir.Program, entry string, profile Profile, storageTransfers bool) (*emissionAnalysis, error) {
	targetSpec, err := emissionTarget(profile)
	if err != nil {
		return nil, err
	}
	verify := mir.VerifyStorageQueries
	if storageTransfers {
		verify = mir.VerifyStorageTransfers
	}
	if failures := verify(program); len(failures) != 0 {
		return nil, fmt.Errorf("C emission requires verified MIR: %s at %s", failures[0].Code, failures[0].Entity)
	}
	if err := validateCScalarTransports(); err != nil {
		return nil, err
	}
	cScalars := make(map[hir.Type]target.Scalar, len(cScalarTransports))
	for _, transport := range cScalarTransports {
		scalar, scalarFound := targetCScalar(targetSpec, transport.targetName)
		if !scalarFound {
			continue
		}
		if scalar.Header != transport.header || scalar.Availability != transport.availability {
			return nil, fmt.Errorf("C emission target %q has invalid %s provider", profile.Target, transport.targetName)
		}
		cScalars[transport.sourceType] = scalar
	}
	layoutEngine, err := layout.New(targetSpec)
	if err != nil {
		return nil, fmt.Errorf("create C resource layout engine: %w", err)
	}
	structTypes, structByType, enumTypes, enumByType, err := prepareAggregateTypes(program.StructTypes, program.EnumTypes, layoutEngine)
	if err != nil {
		return nil, err
	}
	resourceTypes, resourceByType, err := prepareResourceTypes(program.ResourceTypes, layoutEngine, structTypes, structByType, enumTypes, enumByType)
	if err != nil {
		return nil, err
	}
	functions := slices.Clone(program.Functions)
	slices.SortFunc(functions, func(left, right mir.Function) int {
		return strings.Compare(left.Name, right.Name)
	})
	bySymbol, entryIndex, err := indexFunctions(functions, entry)
	if err != nil {
		return nil, err
	}
	reachable, reachableForeign, err := reachableFunctionIndices(program, functions, bySymbol, entryIndex)
	if err != nil {
		return nil, err
	}
	functions = filterFunctions(functions, reachable)
	structTypes, structByType, enumTypes, enumByType, resourceTypes, resourceByType =
		filterReachableAggregateTypes(functions, program.ForeignFunctions, reachableForeign,
			structTypes, enumTypes, resourceTypes)
	arrayTypes, arrayByType, err := prepareScalarArrayTypes(functions, structTypes)
	if err != nil {
		return nil, err
	}
	taggedTypes, taggedByType, err := prepareTaggedTypes(
		functions, structTypes, structByType, enumTypes, enumByType, resourceTypes, resourceByType, arrayByType, layoutEngine,
	)
	if err != nil {
		return nil, err
	}
	byteStringStorage, err := collectByteStringStorage(functions)
	if err != nil {
		return nil, err
	}
	stringStorage, err := collectStringStorage(functions)
	if err != nil {
		return nil, err
	}
	cStringStorage, err := collectCStringStorage(functions)
	if err != nil {
		return nil, err
	}
	foreignFunctions := slices.Clone(program.ForeignFunctions)
	slices.SortFunc(foreignFunctions, func(left, right mir.ForeignFunction) int {
		if value := strings.Compare(left.LinkName, right.LinkName); value != 0 {
			return value
		}
		return int(left.Symbol - right.Symbol)
	})
	foreignFunctions, foreignBySymbol, err := normalizeForeignFunctions(foreignFunctions)
	if err != nil {
		return nil, err
	}
	for symbol := range bySymbol {
		if _, collision := foreignBySymbol[symbol]; collision {
			return nil, fmt.Errorf("MIR symbol %d is both internal and foreign", symbol)
		}
	}
	foreignFunctions, foreignBySymbol, err = filterForeignFunctions(foreignFunctions, foreignBySymbol, reachableForeign)
	if err != nil {
		return nil, err
	}
	checkedFailures := assertedCheckedFailuresForC(functions)
	value := &analysisBuilder{
		layoutEngine: layoutEngine,
		emissionAnalysis: emissionAnalysis{
			profile:            profile,
			reachableBlocks:    make([][]bool, len(functions)),
			storageLayouts:     make(map[hir.Type]ValueStorageLayout),
			targetID:           profile.Target,
			functions:          functions,
			bySymbol:           make(map[hir.SymbolID]int, len(functions)),
			nonReturning:       nonReturningFunctionsForC(functions, checkedFailures),
			checkedFailures:    checkedFailures,
			foreignFunctions:   foreignFunctions,
			foreignBySymbol:    foreignBySymbol,
			entryIndex:         -1,
			arrayTypes:         arrayTypes,
			arrayByType:        arrayByType,
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
			usedForeignSymbols: make(map[string]struct{}),
			storageTransfers:   storageTransfers,
			trapCategories:     make(map[string]struct{}),
			dynamicTrapTargets: make(map[int]map[mir.BlockID]struct{}),
			localTypes:         make([]map[mir.LocalID]hir.Type, len(functions)),
			cScalars:           cScalars,
			instrumentation:    profile.Instrumentation,
			byteStringStorage:  byteStringStorage,
			stringStorage:      stringStorage,
			cStringStorage:     cStringStorage,
		}}
	if len(byteStringStorage) != 0 {
		value.features["static-byte-string"] = struct{}{}
	}
	if len(stringStorage) != 0 {
		value.features["static-utf8-string"] = struct{}{}
	}
	if len(cStringStorage) != 0 {
		value.features["static-c-string"] = struct{}{}
	}
	if len(structTypes) != 0 {
		value.features["inert-struct"] = struct{}{}
	}
	if len(arrayTypes) != 0 {
		value.features["scalar-fixed-array"] = struct{}{}
	}
	if len(enumTypes) != 0 {
		value.features["inert-enum"] = struct{}{}
	}
	for _, structure := range structTypes {
		for _, field := range structure.definition.Fields {
			value.recordScalarTaggedFeature(field.Type)
		}
		if structure.definition.ReprC {
			value.features["repr-c-struct"] = struct{}{}
			for _, field := range structure.definition.Fields {
				value.recordCScalarFeature(field.Type)
			}
		}
	}
	for _, enumeration := range enumTypes {
		for _, variant := range enumeration.definition.Variants {
			for _, field := range variant.Fields {
				value.recordScalarTaggedFeature(field.Type)
			}
		}
	}
	for _, resource := range resourceTypes {
		value.features[resource.feature] = struct{}{}
		if resource.definition.DropKind == hir.ResourceDropNonEmpty {
			value.features["non-empty-resource-drop"] = struct{}{}
		}
	}
	for index := range functions {
		function := &functions[index]
		if _, collision := value.foreignBySymbol[function.Symbol]; collision {
			return nil, fmt.Errorf("MIR symbol %d is both internal and foreign", function.Symbol)
		}
		if _, duplicate := value.bySymbol[function.Symbol]; duplicate {
			return nil, fmt.Errorf("duplicate MIR function symbol %d", function.Symbol)
		}
		value.bySymbol[function.Symbol] = index
		if function.Name == entry {
			if value.entryIndex >= 0 {
				return nil, fmt.Errorf("duplicate C entry function %q", entry)
			}
			value.entryIndex = index
		}
		locals := make(map[hir.SymbolID]mir.ResourceLocal, len(function.Resources))
		for _, resource := range function.Resources {
			if _, duplicate := locals[resource.Symbol]; duplicate {
				return nil, fmt.Errorf("duplicate resource local %d in %s", resource.Symbol, function.Name)
			}
			locals[resource.Symbol] = resource
		}
		value.resourceLocals[index] = locals
	}
	if value.entryIndex < 0 {
		return nil, fmt.Errorf("C entry function %q is missing", entry)
	}
	if err := value.inspectForeignFunctions(); err != nil {
		return nil, err
	}
	if err := value.inspectEnumTypes(); err != nil {
		return nil, err
	}
	entryFunction := &functions[value.entryIndex]
	if len(entryFunction.Parameters) != 0 || entryFunction.ResultType != hir.TypeI32 {
		return nil, fmt.Errorf("C entry %q must have type fn() -> i32", entry)
	}
	for index := range functions {
		localTypes, err := value.inferLocalTypes(index)
		if err != nil {
			return nil, err
		}
		value.localTypes[index] = localTypes
	}
	for index := range functions {
		if err := value.inspectFunction(index); err != nil {
			return nil, err
		}
	}
	value.prepareExecutableRequirements()
	if err := value.prepareDropTraceSites(); err != nil {
		return nil, err
	}

	// Detach the published plan from the builder allocation, so output does not
	// retain the analysis-only layout engine through an interior pointer.
	plan := value.emissionAnalysis
	return &plan, nil
}
