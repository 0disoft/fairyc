// Package cbackend emits the constrained Stage 0 C reference transport.
package cbackend

import (
	"crypto/sha256"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/0disoft/fairyc/internal/cstdlibcontract"
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

// Stage0Profile returns the Linux GNU Stage 0 C emission profile.
func Stage0Profile() Profile {
	return Profile{Target: target.X8664LinuxGNU, Dialect: DialectC11}
}

// Stage0WindowsMSVCProfile returns the provisional Windows MSVC Stage 0 C
// emission profile. It has no validation-only instrumentation.
func Stage0WindowsMSVCProfile() Profile {
	return Profile{Target: target.X8664WindowsMSVC, Dialect: DialectC11}
}

// Stage0DropTraceProfile returns the native-conformance profile that observes compiler-owned drops.
func Stage0DropTraceProfile() Profile {
	return Profile{
		Target: target.X8664LinuxGNU, Dialect: DialectC11,
		Instrumentation: InstrumentationDropTraceV0,
	}
}

// Stage0WindowsMSVCDropTraceProfile observes compiler-owned drops in the closed
// Windows conformance lane. Production profiles remain uninstrumented.
func Stage0WindowsMSVCDropTraceProfile() Profile {
	return Profile{Target: target.X8664WindowsMSVC, Dialect: DialectC11,
		Instrumentation: InstrumentationDropTraceV0}
}

// DropSite identifies one source-linked resource leaf that an instrumented drop may observe.
type DropSite struct {
	ID           uint32
	Package      hir.PackageID
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
	ForeignSymbols  []string
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
	targetID           target.ID
	layoutEngine       *layout.Engine
	functions          []mir.Function
	bySymbol           map[hir.SymbolID]int
	foreignFunctions   []mir.ForeignFunction
	foreignBySymbol    map[hir.SymbolID]int
	entryIndex         int
	arrayTypes         []arrayCType
	arrayByType        map[hir.Type]int
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
	usedForeignSymbols map[string]struct{}
	storageTransfers   bool
	trapCategories     map[string]struct{}
	dynamicTrapTargets map[int]map[mir.BlockID]struct{}
	localTypes         []map[mir.LocalID]hir.Type
	needsByteSlice     bool
	needsMutByteSlice  bool
	needsStringView    bool
	needsBitcast       bool
	needsShift         bool
	needsI64           bool
	cScalars           map[hir.Type]target.Scalar
	instrumentation    string
	dropSites          []DropSite
	dropSiteByKey      map[dropSiteKey]uint32
	byteStringStorage  map[string]string
	stringStorage      map[string]string
	cStringStorage     map[string]string
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
	feature    string
}

type enumCType struct {
	definition   mir.EnumType
	layout       layout.Result
	layoutType   layout.Type
	variantIndex map[string]int
	fieldIndex   []map[string]int
}

type cScalarTransport struct {
	sourceType   hir.Type
	targetName   string
	cType        string
	feature      string
	header       string
	availability string
}

var cScalarTransports = []cScalarTransport{
	{sourceType: hir.TypeCBool, targetName: "c.bool", cType: "_Bool", feature: "c-bool"},
	{sourceType: hir.TypeCChar, targetName: "c.char", cType: "char", feature: "c-char"},
	{sourceType: hir.TypeCDouble, targetName: "c.double", cType: "double", feature: "c-double"},
	{sourceType: hir.TypeCFloat, targetName: "c.float", cType: "float", feature: "c-float"},
	{sourceType: hir.TypeCInt, targetName: "c.int", cType: "int", feature: "c-int"},
	{sourceType: hir.TypeCLong, targetName: "c.long", cType: "long", feature: "c-long"},
	{sourceType: hir.TypeCLongLong, targetName: "c.longlong", cType: "long long", feature: "c-longlong"},
	{sourceType: hir.TypeCSChar, targetName: "c.schar", cType: "signed char", feature: "c-schar"},
	{sourceType: hir.TypeCShort, targetName: "c.short", cType: "short", feature: "c-short"},
	{sourceType: hir.TypeCSizeT, targetName: "c.size_t", cType: "size_t", feature: "c-size-t", header: "stddef.h", availability: "iso-c11"},
	{sourceType: hir.TypeCSSizeT, targetName: "c.ssize_t", cType: "ssize_t", feature: "c-ssize-t", header: "sys/types.h", availability: "posix.1-2008"},
	{sourceType: hir.TypeCUChar, targetName: "c.uchar", cType: "unsigned char", feature: "c-uchar"},
	{sourceType: hir.TypeCUInt, targetName: "c.uint", cType: "unsigned int", feature: "c-uint"},
	{sourceType: hir.TypeCULong, targetName: "c.ulong", cType: "unsigned long", feature: "c-ulong"},
	{sourceType: hir.TypeCULongLong, targetName: "c.ulonglong", cType: "unsigned long long", feature: "c-ulonglong"},
	{sourceType: hir.TypeCUShort, targetName: "c.ushort", cType: "unsigned short", feature: "c-ushort"},
}

func cScalarTransportForType(value hir.Type) (cScalarTransport, bool) {
	for _, transport := range cScalarTransports {
		if transport.sourceType == value {
			return transport, true
		}
	}
	return cScalarTransport{}, false
}

func validateCScalarTransports() error {
	expected := hir.CScalarTypes()
	if len(cScalarTransports) != len(expected) {
		return fmt.Errorf("C scalar transport table has %d entries, want %d", len(cScalarTransports), len(expected))
	}
	seenTarget := make(map[string]struct{}, len(cScalarTransports))
	seenFeature := make(map[string]struct{}, len(cScalarTransports))
	for index, transport := range cScalarTransports {
		if transport.sourceType != expected[index] || transport.targetName == "" || transport.cType == "" || transport.feature == "" {
			return fmt.Errorf("C scalar transport entry %d is incomplete or out of canonical order", index)
		}
		if _, duplicate := seenTarget[transport.targetName]; duplicate {
			return fmt.Errorf("C scalar transport target %q is duplicated", transport.targetName)
		}
		if _, duplicate := seenFeature[transport.feature]; duplicate {
			return fmt.Errorf("C scalar transport feature %q is duplicated", transport.feature)
		}
		seenTarget[transport.targetName] = struct{}{}
		seenFeature[transport.feature] = struct{}{}
	}
	return nil
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
	return emit(program, entry, profile, false)
}

func emit(program mir.Program, entry string, profile Profile, storageTransfers bool) (Unit, error) {
	targetSpec, err := emissionTarget(profile)
	if err != nil {
		return Unit{}, err
	}
	verify := mir.VerifyStorageQueries
	if storageTransfers {
		verify = mir.VerifyStorageTransfers
	}
	if failures := verify(program); len(failures) != 0 {
		return Unit{}, fmt.Errorf("C emission requires verified MIR: %s at %s", failures[0].Code, failures[0].Entity)
	}
	if err := validateCScalarTransports(); err != nil {
		return Unit{}, err
	}
	cScalars := make(map[hir.Type]target.Scalar, len(cScalarTransports))
	for _, transport := range cScalarTransports {
		scalar, scalarFound := targetCScalar(targetSpec, transport.targetName)
		if !scalarFound {
			continue
		}
		if scalar.Header != transport.header || scalar.Availability != transport.availability {
			return Unit{}, fmt.Errorf("C emission target %q has invalid %s provider", profile.Target, transport.targetName)
		}
		cScalars[transport.sourceType] = scalar
	}
	layoutEngine, err := layout.New(targetSpec)
	if err != nil {
		return Unit{}, fmt.Errorf("create C resource layout engine: %w", err)
	}
	structTypes, structByType, enumTypes, enumByType, err := prepareAggregateTypes(program.StructTypes, program.EnumTypes, layoutEngine)
	if err != nil {
		return Unit{}, err
	}
	resourceTypes, resourceByType, err := prepareResourceTypes(program.ResourceTypes, layoutEngine, structTypes, structByType)
	if err != nil {
		return Unit{}, err
	}
	functions := slices.Clone(program.Functions)
	slices.SortFunc(functions, func(left, right mir.Function) int {
		return strings.Compare(left.Name, right.Name)
	})
	bySymbol, entryIndex, err := indexFunctions(functions, entry)
	if err != nil {
		return Unit{}, err
	}
	reachable, reachableForeign, err := reachableFunctionIndices(program, functions, bySymbol, entryIndex)
	if err != nil {
		return Unit{}, err
	}
	functions = filterFunctions(functions, reachable)
	structTypes, structByType, enumTypes, enumByType, resourceTypes, resourceByType =
		filterReachableAggregateTypes(functions, program.ForeignFunctions, reachableForeign,
			structTypes, enumTypes, resourceTypes)
	arrayTypes, arrayByType, err := prepareScalarArrayTypes(functions)
	if err != nil {
		return Unit{}, err
	}
	taggedTypes, taggedByType, err := prepareTaggedTypes(
		functions, structTypes, structByType, enumTypes, enumByType, resourceTypes, resourceByType, arrayByType, layoutEngine,
	)
	if err != nil {
		return Unit{}, err
	}
	byteStringStorage, err := collectByteStringStorage(functions)
	if err != nil {
		return Unit{}, err
	}
	stringStorage, err := collectStringStorage(functions)
	if err != nil {
		return Unit{}, err
	}
	cStringStorage, err := collectCStringStorage(functions)
	if err != nil {
		return Unit{}, err
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
		return Unit{}, err
	}
	for symbol := range bySymbol {
		if _, collision := foreignBySymbol[symbol]; collision {
			return Unit{}, fmt.Errorf("MIR symbol %d is both internal and foreign", symbol)
		}
	}
	foreignFunctions, foreignBySymbol, err = filterForeignFunctions(foreignFunctions, foreignBySymbol, reachableForeign)
	if err != nil {
		return Unit{}, err
	}
	value := &emitter{
		targetID:           profile.Target,
		layoutEngine:       layoutEngine,
		functions:          functions,
		bySymbol:           make(map[hir.SymbolID]int, len(functions)),
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
	}
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
	value.emitByteStringStorage(&output)
	value.emitStringStorage(&output)
	value.emitCStringStorage(&output)
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
	output.WriteString("int main(void) {\n")
	for index := range functions {
		if index != value.entryIndex {
			fmt.Fprintf(&output, "    (void)&fy_fn_%d;\n", index)
		}
	}
	fmt.Fprintf(&output, "    return (int)fy_fn_%d();\n}\n", value.entryIndex)

	names := make([]string, len(functions))
	for index := range functions {
		names[index] = functions[index].Name
	}
	features := make([]string, 0, len(value.features))
	for feature := range value.features {
		features = append(features, feature)
	}
	slices.Sort(features)
	foreignSymbols := make([]string, 0, len(value.usedForeignSymbols))
	for symbol := range value.usedForeignSymbols {
		foreignSymbols = append(foreignSymbols, symbol)
	}
	slices.Sort(foreignSymbols)
	return Unit{
		Target: string(profile.Target), Dialect: profile.Dialect, Instrumentation: profile.Instrumentation,
		Entry: entry, Functions: names, ForeignSymbols: foreignSymbols, Features: features,
		DropSites: slices.Clone(value.dropSites), Source: output.String(),
	}, nil
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

func prepareResourceTypes(definitions []mir.ResourceType, engine *layout.Engine, structures []structCType, structByType map[hir.Type]int) ([]resourceCType, map[hir.Type]int, error) {
	ordered, err := orderResourceDefinitions(definitions)
	if err != nil {
		return nil, nil, err
	}
	result := make([]resourceCType, 0, len(ordered))
	byType := make(map[hir.Type]int, len(ordered))
	for _, definition := range ordered {
		if definition.DropKind != hir.ResourceDropEmpty && definition.DropKind != hir.ResourceDropNonEmpty {
			return nil, nil, CapabilityError{Feature: "resource type " + strconv.Quote(string(definition.Type)) + " with unsupported drop behavior"}
		}
		if len(definition.Fields) == 0 {
			return nil, nil, CapabilityError{Feature: "empty resource type " + strconv.Quote(string(definition.Type))}
		}
		fields := make([]layout.Field, 0, len(definition.Fields))
		fieldIndex := make(map[string]int, len(definition.Fields))
		allI32 := true
		nestedCopy := false
		nestedResource := false
		for index, field := range definition.Fields {
			fieldType, ok := resourceFieldLayoutType(field.Type)
			if structure, found := structByType[field.Type]; !ok && found {
				fieldType, ok = structures[structure].layoutType, true
				nestedCopy = true
			}
			if resource, found := byType[field.Type]; !ok && found {
				fieldType, ok = result[resource].layoutType, true
				nestedResource = true
			}
			if !ok {
				return nil, nil, CapabilityError{Feature: "resource type " + strconv.Quote(string(definition.Type)) + " field " + strconv.Quote(field.Name) + " of type " + strconv.Quote(string(field.Type))}
			}
			allI32 = allI32 && field.Type == hir.TypeI32
			fieldIndex[field.Name] = index
			fields = append(fields, layout.Field{
				Name: fmt.Sprintf("f_%d", index),
				Type: fieldType,
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
		feature := "flat-copy-resource"
		if allI32 {
			feature = "flat-i32-resource"
		}
		if nestedCopy {
			feature = "nested-copy-resource"
		}
		if nestedResource {
			feature = "nested-resource"
		}
		result = append(result, resourceCType{
			definition: definition, layout: computed, layoutType: aggregateType,
			fieldIndex: fieldIndex, feature: feature,
		})
	}
	return result, byType, nil
}

func resourceFieldLayoutType(value hir.Type) (layout.Type, bool) {
	canonical := canonicalCValueType(value)
	if canonical == hir.TypeU8 || canonical == hir.TypeI32 || canonical == hir.TypeI64 || canonical == hir.TypeU32 || canonical == hir.TypeUSize {
		return layout.Type{Kind: layout.KindLanguageScalar, Name: string(canonical)}, true
	}
	if _, _, pointer := hir.RawPointerElement(value); pointer {
		return layout.Type{Kind: layout.KindPointer}, true
	}
	return layout.Type{}, false
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
				case mir.OperationUnary:
					inferred = operation.Type
				case mir.OperationBinary:
					inferred = operation.Type
				case mir.OperationResultConstruct, mir.OperationOptionConstruct,
					mir.OperationResultPayload, mir.OperationOptionPayload,
					mir.OperationEnumConstruct, mir.OperationEnumPayload,
					mir.OperationRawAddress, mir.OperationRawBorrowPointer, mir.OperationStrFromRawParts, mir.OperationCStrFromRawParts,
					mir.OperationByteSliceFromRawParts, mir.OperationMutByteSliceFromRawParts,
					mir.OperationStrDataAddress, mir.OperationByteSliceDataAddress, mir.OperationCStrDataAddress,
					mir.OperationStructConstruct, mir.OperationResourceConstruct, mir.OperationFieldRead, mir.OperationFieldBorrow, mir.OperationViewLength, mir.OperationViewEmpty, mir.OperationViewIndex, mir.OperationViewSlice,
					mir.OperationStorageLayout, mir.OperationStorageTake, mir.OperationStorageBorrow, mir.OperationStorageBorrowMut, mir.OperationStorageElementAddress, mir.OperationScalarCast, mir.OperationArrayConstruct, mir.OperationArrayBorrowView, mir.OperationArrayLength, mir.OperationArrayIndex,
					mir.OperationRegionPlace:
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
		if !validExternalCIdentifier(function.LinkName, e.targetID) &&
			!(e.storageTransfers && validVectorRuntimeFunction(function, e.targetID)) {
			return CapabilityError{Function: function.Name, Feature: "external C symbol " + strconv.Quote(function.LinkName)}
		}
		if _, err := e.foreignCType(function.ResultType, true); err != nil {
			return CapabilityError{Function: function.Name, Feature: err.Error()}
		}
		if _, structure := e.structByType[function.ResultType]; structure {
			e.features["foreign-c-struct-result"] = struct{}{}
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
	if transport, found := cScalarTransportForType(value); found {
		e.features[transport.feature] = struct{}{}
	}
}

func (e *emitter) recordScalarTaggedFeature(value hir.Type) {
	if isOptionI32(value) {
		e.features["scalar-option"] = struct{}{}
		return
	}
	if isResultI32I32(value) {
		e.features["scalar-result"] = struct{}{}
		return
	}
	if _, direct := copyScalarTaggedAggregateLayout(value); !direct {
		return
	}
	if _, option := hir.OptionElement(value); option {
		e.features["copy-option"] = struct{}{}
	} else {
		e.features["copy-result"] = struct{}{}
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
	if canonical == hir.TypeU8 || canonical == hir.TypeI32 || canonical == hir.TypeI64 || canonical == hir.TypeU32 || canonical == hir.TypeUSize ||
		isRawPointer(canonical) || isOptionI32(canonical) || isResultI32I32(canonical) {
		return e.cValueType(canonical)
	}
	if _, direct := copyScalarTaggedAggregateLayout(canonical); direct {
		return e.cValueType(canonical)
	}
	if index, found := e.taggedByType[canonical]; found && e.taggedParticipatesInAggregateGraph(index) {
		return e.cValueType(canonical)
	}
	if _, structure := e.structByType[canonical]; structure {
		return e.cValueType(canonical)
	}
	if _, enumeration := e.enumByType[canonical]; enumeration {
		return e.cValueType(canonical)
	}
	return "", fmt.Errorf("unsupported enum payload type %q", value)
}

func (e *emitter) foreignCType(value hir.Type, result bool) (string, error) {
	if result && value == hir.TypeVoid {
		return "void", nil
	}
	if transport, found := cScalarTransportForType(value); found {
		if _, available := e.cScalars[value]; !available {
			return "", fmt.Errorf("C scalar %q is unavailable for target %q", transport.targetName, e.targetID)
		}
		return transport.cType, nil
	}
	switch value {
	case hir.TypeU8:
		return "uint8_t", nil
	case hir.TypeI64:
		return "int64_t", nil
	case hir.TypeI32:
		return "int32_t", nil
	case hir.TypeU32:
		return "uint32_t", nil
	case hir.TypeUSize:
		return "uintptr_t", nil
	}
	if element, mutable, pointer := hir.RawPointerElement(value); pointer {
		if nestedElement, nestedMutable, nested := hir.RawPointerElement(element); nested {
			if mutable && nestedMutable && nestedElement == hir.TypeVoid {
				return "void * *", nil
			}
			return "", fmt.Errorf("foreign raw-pointer nesting type %q", value)
		}
		pointee, err := e.foreignPointerElementCType(element)
		if err != nil {
			return "", err
		}
		if mutable {
			return pointee + " *", nil
		}
		return "const " + pointee + " *", nil
	}
	if index, structure := e.structByType[value]; structure {
		if !result {
			return "", fmt.Errorf("foreign parameter type %q", value)
		}
		if !e.structTypes[index].definition.ReprC {
			return "", fmt.Errorf("foreign result struct %q is not repr(C)", value)
		}
		return e.structCTypeName(index), nil
	}
	role := "parameter"
	if result {
		role = "result"
	}
	return "", fmt.Errorf("foreign %s type %q", role, value)
}

func (e *emitter) foreignPointerElementCType(value hir.Type) (string, error) {
	switch value {
	case hir.TypeVoid:
		return "void", nil
	case hir.TypeU8:
		return "uint8_t", nil
	case hir.TypeI64:
		return "int64_t", nil
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

func assignLocalType(localTypes map[mir.LocalID]hir.Type, local mir.LocalID, value hir.Type) (bool, error) {
	if local < 0 {
		return false, fmt.Errorf("local has invalid id %d", local)
	}
	if value == hir.TypeVoid || value == hir.TypeInvalid || value == "" {
		return false, fmt.Errorf("local %d has invalid value type %q", local, value)
	}
	if existing, found := localTypes[local]; found {
		if canonicalCValueType(existing) != canonicalCValueType(value) {
			return false, fmt.Errorf("local %d has conflicting types %q and %q", local, existing, value)
		}
		return false, nil
	}
	localTypes[local] = value
	return true, nil
}

func inferredOperandType(localTypes map[mir.LocalID]hir.Type, operand mir.Operand) (hir.Type, bool) {
	switch operand.Kind {
	case mir.OperandI64:
		return hir.TypeI64, true
	case mir.OperandI32:
		if operand.Type == hir.TypeBool {
			return hir.TypeBool, operand.I32 == 0 || operand.I32 == 1
		}
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
	case mir.OperandByteString:
		if operand.Type == hir.SharedBorrow(hir.TypeByteSlice) {
			return operand.Type, true
		}
		return hir.TypeInvalid, false
	case mir.OperandString:
		if operand.Type == hir.SharedBorrow(hir.TypeStr) && utf8.ValidString(operand.StaticData) {
			return operand.Type, true
		}
		return hir.TypeInvalid, false
	case mir.OperandCString:
		if operand.Type == hir.SharedBorrow(hir.TypeCStr) {
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

func fieldAggregateType(value hir.Type) hir.Type {
	if element, _, borrow := hir.BorrowElement(value); borrow {
		return element
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
	if value == hir.SharedBorrow(hir.TypeByteSlice) || value == hir.MutableBorrow(hir.TypeByteSlice) ||
		value == hir.SharedBorrow(hir.TypeStr) || value == hir.SharedBorrow(hir.TypeCStr) {
		return true
	}
	if _, borrow := e.safeBorrowCType(value); borrow {
		return true
	}
	if value == hir.TypeU8 || value == hir.TypeI32 || value == hir.TypeI64 || value == hir.TypeU32 || value == hir.TypeUSize || hir.IsCScalar(value) || isOptionI32(value) || isResultI32I32(value) {
		return true
	}
	if _, array := e.arrayByType[value]; array {
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
	if transport, found := cScalarTransportForType(value); found {
		if _, available := e.cScalars[value]; !available {
			return "", fmt.Errorf("C scalar %q is unavailable for target %q", transport.targetName, e.targetID)
		}
		return transport.cType, nil
	}
	if cType, borrow := e.safeBorrowCType(value); borrow {
		return cType, nil
	}
	switch {
	case value == hir.SharedBorrow(hir.TypeByteSlice):
		return "fy_byte_slice", nil
	case value == hir.MutableBorrow(hir.TypeByteSlice):
		return "fy_mut_byte_slice", nil
	case value == hir.SharedBorrow(hir.TypeStr):
		return "fy_str", nil
	case value == hir.SharedBorrow(hir.TypeCStr):
		return "const uint8_t *", nil
	case value == hir.TypeU8:
		return "uint8_t", nil
	case value == hir.TypeI64:
		return "int64_t", nil
	case value == hir.TypeI32:
		return "int32_t", nil
	case value == hir.TypeU32:
		return "uint32_t", nil
	case value == hir.TypeUSize:
		return "uintptr_t", nil
	case isRawPointer(value):
		return e.foreignCType(value, false)
	case isOptionI32(value):
		return "fy_option_i32", nil
	case isResultI32I32(value):
		return "fy_result_i32_i32", nil
	default:
		if index, found := e.arrayByType[value]; found {
			return e.arrayCTypeName(index), nil
		}
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

func (e *emitter) safeBorrowCType(value hir.Type) (string, bool) {
	element, mutable, borrow := hir.BorrowElement(value)
	if !borrow {
		return "", false
	}
	pointee := ""
	canonical := canonicalCValueType(element)
	if index, found := e.resourceByType[canonical]; found {
		pointee = fmt.Sprintf("fy_resource_%d", index)
	} else if index, found := e.arrayByType[canonical]; found {
		pointee = e.arrayCTypeName(index)
	} else if index, found := e.structByType[canonical]; found {
		pointee = e.structCTypeName(index)
	} else {
		switch canonical {
		case hir.TypeU8:
			pointee = "uint8_t"
		case hir.TypeI64:
			pointee = "int64_t"
		case hir.TypeI32:
			pointee = "int32_t"
		case hir.TypeU32:
			pointee = "uint32_t"
		case hir.TypeUSize:
			pointee = "uintptr_t"
		}
	}
	if pointee == "" {
		return "", false
	}
	if mutable {
		return pointee + " *", true
	}
	return "const " + pointee + " *", true
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
	predecessors := make([]uint32, len(function.Blocks))
	for _, block := range function.Blocks {
		for _, successor := range block.Terminator.Successors {
			if successor >= 0 && int(successor) < len(predecessors) {
				predecessors[successor]++
			}
		}
	}
	if function.ResultType != hir.TypeVoid && !e.supportedCValueType(function.ResultType) {
		return CapabilityError{Function: function.Name, Feature: "result type " + strconv.Quote(string(function.ResultType))}
	}
	e.recordCValueTypeRequirements(function.ResultType)
	e.recordCScalarFeature(function.ResultType)
	if len(function.ParameterTypes) != len(function.Parameters) {
		return fmt.Errorf("MIR function %s has %d parameter types for %d parameters", function.Name, len(function.ParameterTypes), len(function.Parameters))
	}
	for _, parameterType := range function.ParameterTypes {
		if !e.supportedCValueType(parameterType) {
			return CapabilityError{Function: function.Name, Feature: "parameter type " + strconv.Quote(string(parameterType))}
		}
		e.recordCValueTypeRequirements(parameterType)
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
	if len(function.ReturnOrigins) != 0 {
		return CapabilityError{Function: function.Name, Feature: "safe-borrow returns"}
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
			case mir.OperationMove, mir.OperationDrop, mir.OperationBurn:
				if err := e.inspectResourceDischarge(functionIndex, *operation); err != nil {
					return err
				}
			case mir.OperationRawAddress:
				if err := e.inspectRawAddress(functionIndex, *operation); err != nil {
					return err
				}
				allowedUnsafeEntities[operation.Entity] = struct{}{}
				e.features["raw-address"] = struct{}{}
			case mir.OperationRawBorrowPointer:
				if err := e.inspectRawBorrowPointer(functionIndex, *operation); err != nil {
					return err
				}
				allowedUnsafeEntities[operation.Entity] = struct{}{}
				e.features["raw-borrow-pointer"] = struct{}{}
			case mir.OperationStrFromRawParts, mir.OperationCStrFromRawParts, mir.OperationByteSliceFromRawParts, mir.OperationMutByteSliceFromRawParts:
				if err := e.inspectBorrowedViewFromRawParts(functionIndex, *operation); err != nil {
					return err
				}
				allowedUnsafeEntities[operation.Entity] = struct{}{}
				if operation.Kind == mir.OperationStrFromRawParts {
					e.features["utf8-string-view"] = struct{}{}
				} else if operation.Kind == mir.OperationCStrFromRawParts {
					e.features["c-string-view"] = struct{}{}
				} else if operation.Kind == mir.OperationByteSliceFromRawParts {
					e.features["byte-slice-view"] = struct{}{}
				} else {
					e.features["mutable-byte-slice-view"] = struct{}{}
				}
			case mir.OperationStrDataAddress:
				if err := e.inspectStrDataAddress(functionIndex, *operation); err != nil {
					return err
				}
				allowedUnsafeEntities[operation.Entity] = struct{}{}
				e.features["str-data-address"] = struct{}{}
			case mir.OperationByteSliceDataAddress:
				if err := e.inspectByteSliceDataAddress(functionIndex, *operation); err != nil {
					return err
				}
				allowedUnsafeEntities[operation.Entity] = struct{}{}
				e.features["byte-slice-data-address"] = struct{}{}
			case mir.OperationCStrDataAddress:
				if err := e.inspectCStrDataAddress(functionIndex, *operation); err != nil {
					return err
				}
				allowedUnsafeEntities[operation.Entity] = struct{}{}
				e.features["cstr-data-address"] = struct{}{}
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
			case mir.OperationFieldStore:
				if err := e.inspectFieldStore(functionIndex, *operation); err != nil {
					return err
				}
			case mir.OperationDerefStore:
				if err := e.inspectDerefStore(functionIndex, *operation); err != nil {
					return err
				}
			case mir.OperationFieldBorrow:
				if err := e.inspectFieldBorrow(functionIndex, *operation); err != nil {
					return err
				}
			case mir.OperationFieldRead:
				if _, structure := e.structByType[fieldAggregateType(operation.OwnerType)]; structure {
					if err := e.inspectStructOperation(functionIndex, *operation); err != nil {
						return err
					}
				} else if err := e.inspectResourceOperation(functionIndex, *operation); err != nil {
					return err
				}
			case mir.OperationViewLength:
				if err := e.inspectViewLength(functionIndex, *operation); err != nil {
					return err
				}
				e.features["view-length"] = struct{}{}
			case mir.OperationViewEmpty:
				if err := e.inspectViewEmpty(functionIndex, *operation); err != nil {
					return err
				}
				e.features["view-empty"] = struct{}{}
			case mir.OperationViewIndexCheck:
				if err := e.inspectViewIndex(functionIndex, *operation, false); err != nil {
					return err
				}
				e.features["view-index"] = struct{}{}
			case mir.OperationViewIndex:
				if err := e.inspectViewIndex(functionIndex, *operation, true); err != nil {
					return err
				}
				e.features["view-index"] = struct{}{}
			case mir.OperationViewIndexStore:
				if err := e.inspectViewIndexStore(functionIndex, *operation); err != nil {
					return err
				}
				e.features["mutable-view-index-store"] = struct{}{}
			case mir.OperationViewSliceCheck:
				if err := e.inspectViewSlice(functionIndex, *operation, false); err != nil {
					return err
				}
				e.features["view-slice"] = struct{}{}
			case mir.OperationViewSlice:
				if err := e.inspectViewSlice(functionIndex, *operation, true); err != nil {
					return err
				}
				e.features["view-slice"] = struct{}{}
			case mir.OperationScalarCast:
				if err := e.inspectScalarCast(functionIndex, *operation); err != nil {
					return err
				}
				e.features["scalar-cast"] = struct{}{}
			case mir.OperationArrayConstruct:
				if err := e.inspectArrayConstruction(functionIndex, *operation); err != nil {
					return err
				}
				e.features["scalar-fixed-array"] = struct{}{}
			case mir.OperationArrayBorrowView:
				if err := e.inspectArrayBorrowView(functionIndex, *operation); err != nil {
					return err
				}
				if operation.Type == hir.MutableBorrow(hir.TypeByteSlice) {
					e.features["mutable-array-byte-view-borrow"] = struct{}{}
				} else {
					e.features["array-byte-view-borrow"] = struct{}{}
				}
			case mir.OperationStorageLayout:
				if _, err := e.storageQueryValue(*operation); err != nil {
					return err
				}
				e.features["private-storage-layout"] = struct{}{}
			case mir.OperationStorageTake:
				if err := e.inspectStorageTake(functionIndex, *operation); err != nil {
					return err
				}
				allowedUnsafeEntities[operation.Entity] = struct{}{}
				e.features["private-storage-take"] = struct{}{}
			case mir.OperationStoragePut:
				if err := e.inspectStoragePut(functionIndex, *operation); err != nil {
					return err
				}
				allowedUnsafeEntities[operation.Entity] = struct{}{}
				e.features["private-storage-put"] = struct{}{}
			case mir.OperationStorageBorrow, mir.OperationStorageBorrowMut:
				if err := e.inspectStorageBorrow(functionIndex, *operation); err != nil {
					return err
				}
				allowedUnsafeEntities[operation.Entity] = struct{}{}
				e.features["private-"+string(operation.Kind)] = struct{}{}
			case mir.OperationStorageElementAddress:
				if err := e.inspectStorageElementAddress(functionIndex, *operation); err != nil {
					return err
				}
				allowedUnsafeEntities[operation.Entity] = struct{}{}
				e.features["private-storage-element-address"] = struct{}{}
			case mir.OperationArrayLength:
				if err := e.inspectArrayLength(functionIndex, *operation); err != nil {
					return err
				}
				e.features["array-length"] = struct{}{}
			case mir.OperationArrayIndexCheck:
				if err := e.inspectArrayIndex(functionIndex, *operation, false); err != nil {
					return err
				}
				e.features["array-index"] = struct{}{}
			case mir.OperationArrayIndex:
				if err := e.inspectArrayIndex(functionIndex, *operation, true); err != nil {
					return err
				}
				e.features["array-index"] = struct{}{}
			case mir.OperationArrayIndexStore:
				if err := e.inspectArrayIndexStore(functionIndex, *operation); err != nil {
					return err
				}
				e.features["array-index-store"] = struct{}{}
			case mir.OperationRegionEnter, mir.OperationRegionPlace, mir.OperationRegionReset:
				if err := e.inspectRegionOperation(functionIndex, *operation); err != nil {
					return err
				}
			case mir.OperationUse:
			case mir.OperationArithmeticCheck:
				e.features["checked-arithmetic"] = struct{}{}
				if operation.OwnerType == hir.TypeI32 {
					e.needsBitcast = true
					if operation.Operator == ">>" {
						e.needsShift = true
					}
				}
			case mir.OperationCheckedDivision:
				e.features["checked-division"] = struct{}{}
			case mir.OperationUnary:
				if err := e.inspectUnary(functionIndex, *operation); err != nil {
					return err
				}
				if operation.OwnerType == hir.TypeI64 {
					e.needsI64 = true
				}
				if operation.Operator == "-" || operation.Operator == "~" && operation.OwnerType == hir.TypeI32 {
					e.needsBitcast = true
				}
				if operation.Operator == "~" {
					e.features["scalar-bitwise"] = struct{}{}
				}
			case mir.OperationBinary:
				if err := e.inspectBinary(functionIndex, *operation); err != nil {
					return err
				}
				if operation.OwnerType == hir.TypeI64 {
					e.needsI64 = true
				}
				switch operation.Operator {
				case "==", "!=", "<", "<=", ">", ">=":
					e.features["scalar-comparison"] = struct{}{}
				case "+", "-", "*":
					if operation.OwnerType == hir.TypeI32 {
						e.needsBitcast = true
					}
				case "&", "^", "|":
					e.features["scalar-bitwise"] = struct{}{}
					if operation.OwnerType == hir.TypeI32 {
						e.needsBitcast = true
					}
				case ">>", "<<":
					e.features["checked-shift"] = struct{}{}
					if operation.OwnerType == hir.TypeI32 {
						e.needsBitcast = true
						if operation.Operator == ">>" {
							e.needsShift = true
						}
					}
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
				e.usedForeignSymbols[e.foreignFunctions[e.foreignBySymbol[terminator.Callee]].LinkName] = struct{}{}
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
			if err := e.inspectSwitch(functionIndex, terminator, predecessors); err != nil {
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
			return CapabilityError{Function: function.Name, Feature: "unsafe operation outside the admitted constrained-C subset"}
		}
	}
	return nil
}

func (e *emitter) recordCValueTypeRequirements(value hir.Type) {
	switch canonicalCValueType(value) {
	case hir.SharedBorrow(hir.TypeByteSlice):
		e.needsByteSlice = true
	case hir.MutableBorrow(hir.TypeByteSlice):
		e.needsMutByteSlice = true
	case hir.SharedBorrow(hir.TypeStr):
		e.needsStringView = true
	}
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
		mir.OperationRawAddress, mir.OperationRawBorrowPointer, mir.OperationStrFromRawParts, mir.OperationCStrFromRawParts,
		mir.OperationByteSliceFromRawParts, mir.OperationMutByteSliceFromRawParts,
		mir.OperationStrDataAddress, mir.OperationByteSliceDataAddress, mir.OperationCStrDataAddress,
		mir.OperationStructConstruct, mir.OperationEnumConstruct, mir.OperationEnumPayload,
		mir.OperationResourceConstruct, mir.OperationFieldRead, mir.OperationFieldBorrow, mir.OperationViewLength, mir.OperationViewEmpty, mir.OperationViewIndex, mir.OperationViewSlice, mir.OperationScalarCast,
		mir.OperationStorageLayout, mir.OperationStorageTake, mir.OperationStorageBorrow, mir.OperationStorageBorrowMut, mir.OperationStorageElementAddress, mir.OperationArrayConstruct, mir.OperationArrayBorrowView, mir.OperationArrayLength, mir.OperationArrayIndex,
		mir.OperationRegionPlace:
		if operation.DestinationLocal < 0 {
			return fmt.Errorf("%s operation in %s has no destination local", operation.Kind, function)
		}
	case mir.OperationMove, mir.OperationDrop, mir.OperationBurn:
		if operation.Symbol == hir.NoSymbol {
			return fmt.Errorf("%s operation in %s has no resource symbol", operation.Kind, function)
		}
	case mir.OperationRegionEnter, mir.OperationRegionReset:
		if operation.Symbol == hir.NoSymbol || operation.DestinationLocal != mir.NoLocal {
			return fmt.Errorf("%s operation in %s has invalid Region identity", operation.Kind, function)
		}
	}
	return nil
}

func (e *emitter) inspectViewLength(functionIndex int, operation mir.Operation) error {
	function := &e.functions[functionIndex]
	receiver, err := oneOperand(operation.Operands)
	if err != nil {
		return fmt.Errorf("borrowed-view length in %s: %w", function.Name, err)
	}
	element, _, borrow := hir.BorrowElement(operation.OwnerType)
	if !borrow || element != hir.TypeStr && element != hir.TypeByteSlice || operation.Type != hir.TypeUSize || operation.Field != "len" {
		return fmt.Errorf("borrowed-view length in %s has incompatible owner, result, or field", function.Name)
	}
	receiverType, err := e.operandType(functionIndex, receiver)
	if err != nil {
		return fmt.Errorf("borrowed-view length in %s: %w", function.Name, err)
	}
	if receiverType != operation.OwnerType {
		return fmt.Errorf("borrowed-view length in %s has receiver type %q, want %q", function.Name, receiverType, operation.OwnerType)
	}
	return nil
}

func (e *emitter) inspectArrayConstruction(functionIndex int, operation mir.Operation) error {
	function := &e.functions[functionIndex]
	elementType, length, array := hir.FixedArrayParts(operation.Type)
	if !array || operation.OwnerType != elementType || operation.Operator != "list" ||
		operation.DestinationLocal == mir.NoLocal || uint64(len(operation.Operands)) != length {
		return fmt.Errorf("fixed-array construction in %s has incompatible type or length", function.Name)
	}
	if _, found := e.arrayByType[operation.Type]; !found {
		return CapabilityError{Function: function.Name, Feature: "fixed-array type " + strconv.Quote(string(operation.Type))}
	}
	for index, operand := range operation.Operands {
		operandType, err := e.operandType(functionIndex, operand)
		if err != nil || operandType != elementType {
			return fmt.Errorf("fixed-array construction element %d in %s has incompatible type", index, function.Name)
		}
	}
	return nil
}

func (e *emitter) inspectArrayBorrowView(functionIndex int, operation mir.Operation) error {
	function := &e.functions[functionIndex]
	receiver, err := oneOperand(operation.Operands)
	if err != nil {
		return fmt.Errorf("fixed-array byte-view borrow in %s: %w", function.Name, err)
	}
	elementType, length, array := hir.FixedArrayParts(operation.OwnerType)
	viewValid := (operation.Type == hir.SharedBorrow(hir.TypeByteSlice) && operation.Operator == "&") ||
		(operation.Type == hir.MutableBorrow(hir.TypeByteSlice) && operation.Operator == "&mut")
	if !array || elementType != hir.TypeU8 || !hir.ExecutableFixedArrayShape(elementType, length) ||
		!viewValid {
		return fmt.Errorf("fixed-array byte-view borrow in %s has incompatible owner, result, or operator", function.Name)
	}
	receiverType, err := e.operandType(functionIndex, receiver)
	if err != nil {
		return fmt.Errorf("fixed-array byte-view borrow in %s: %w", function.Name, err)
	}
	if receiver.Kind != mir.OperandLocal || receiverType != operation.OwnerType {
		return fmt.Errorf("fixed-array byte-view borrow in %s has receiver type %q, want local %q", function.Name, receiverType, operation.OwnerType)
	}
	return nil
}

func (e *emitter) inspectArrayLength(functionIndex int, operation mir.Operation) error {
	function := &e.functions[functionIndex]
	receiver, err := oneOperand(operation.Operands)
	if err != nil {
		return fmt.Errorf("fixed-array length in %s: %w", function.Name, err)
	}
	elementType, length, array := hir.FixedArrayParts(operation.OwnerType)
	if !array || !hir.ExecutableFixedArrayShape(elementType, length) || operation.Type != hir.TypeUSize || operation.Field != "len" {
		return fmt.Errorf("fixed-array length in %s has incompatible owner, result, or field", function.Name)
	}
	receiverType, err := e.operandType(functionIndex, receiver)
	if err != nil {
		return fmt.Errorf("fixed-array length in %s: %w", function.Name, err)
	}
	if receiverType != operation.OwnerType {
		return fmt.Errorf("fixed-array length in %s has receiver type %q, want %q", function.Name, receiverType, operation.OwnerType)
	}
	return nil
}

func (e *emitter) inspectArrayIndex(functionIndex int, operation mir.Operation, destination bool) error {
	function := &e.functions[functionIndex]
	receiver, index, err := twoOperands(operation.Operands)
	if err != nil {
		return fmt.Errorf("fixed-array index in %s: %w", function.Name, err)
	}
	elementType, _, array := hir.FixedArrayParts(operation.OwnerType)
	if !array || operation.Type != elementType || operation.IndexType != hir.TypeUSize || operation.Operator != "[]" ||
		destination != (operation.DestinationLocal != mir.NoLocal) {
		return fmt.Errorf("fixed-array index in %s has incompatible owner, result, operator, or destination", function.Name)
	}
	receiverType, receiverErr := e.operandType(functionIndex, receiver)
	indexType, indexErr := e.operandType(functionIndex, index)
	if receiverErr != nil || indexErr != nil || receiverType != operation.OwnerType || indexType != hir.TypeUSize {
		return fmt.Errorf("fixed-array index in %s has incompatible operand types", function.Name)
	}
	if destination && len(operation.TrapCategories) != 0 || !destination && !slices.Equal(operation.TrapCategories, []string{"bounds"}) {
		return fmt.Errorf("fixed-array index in %s has an invalid trap marker", function.Name)
	}
	return nil
}

func (e *emitter) inspectArrayIndexStore(functionIndex int, operation mir.Operation) error {
	function := &e.functions[functionIndex]
	receiver, index, value, err := threeOperands(operation.Operands)
	if err != nil {
		return fmt.Errorf("fixed-array index store in %s: %w", function.Name, err)
	}
	elementType, _, array := hir.FixedArrayParts(operation.OwnerType)
	if !array || operation.Type != elementType || operation.IndexType != hir.TypeUSize || operation.Operator != "[]=" || operation.DestinationLocal != mir.NoLocal {
		return fmt.Errorf("fixed-array index store in %s has incompatible owner, value, operator, or destination", function.Name)
	}
	receiverType, receiverErr := e.operandType(functionIndex, receiver)
	indexType, indexErr := e.operandType(functionIndex, index)
	valueType, valueErr := e.operandType(functionIndex, value)
	if receiverErr != nil || indexErr != nil || valueErr != nil || receiverType != operation.OwnerType || indexType != hir.TypeUSize || valueType != elementType {
		return fmt.Errorf("fixed-array index store in %s has incompatible operand types", function.Name)
	}
	if len(operation.TrapCategories) != 0 {
		return fmt.Errorf("fixed-array index store in %s has an invalid trap marker", function.Name)
	}
	return nil
}

func (e *emitter) inspectViewEmpty(functionIndex int, operation mir.Operation) error {
	function := &e.functions[functionIndex]
	receiver, err := oneOperand(operation.Operands)
	if err != nil {
		return fmt.Errorf("borrowed-view emptiness in %s: %w", function.Name, err)
	}
	element, _, borrow := hir.BorrowElement(operation.OwnerType)
	if !borrow || element != hir.TypeStr && element != hir.TypeByteSlice || operation.Type != hir.TypeBool || operation.Field != "is_empty" {
		return fmt.Errorf("borrowed-view emptiness in %s has incompatible owner, result, or field", function.Name)
	}
	receiverType, err := e.operandType(functionIndex, receiver)
	if err != nil {
		return fmt.Errorf("borrowed-view emptiness in %s: %w", function.Name, err)
	}
	if receiverType != operation.OwnerType {
		return fmt.Errorf("borrowed-view emptiness in %s has receiver type %q, want %q", function.Name, receiverType, operation.OwnerType)
	}
	return nil
}

func (e *emitter) inspectViewIndex(functionIndex int, operation mir.Operation, destination bool) error {
	function := &e.functions[functionIndex]
	receiver, index, err := twoOperands(operation.Operands)
	if err != nil {
		return fmt.Errorf("byte-view index in %s: %w", function.Name, err)
	}
	if !isCByteViewType(operation.OwnerType) || operation.IndexType != hir.TypeUSize || operation.Type != hir.TypeU8 || operation.Operator != "[]" ||
		destination != (operation.DestinationLocal != mir.NoLocal) {
		return fmt.Errorf("byte-view index in %s has incompatible owner, result, operator, or destination", function.Name)
	}
	receiverType, err := e.operandType(functionIndex, receiver)
	if err != nil {
		return fmt.Errorf("byte-view index in %s: %w", function.Name, err)
	}
	indexType, err := e.operandType(functionIndex, index)
	if err != nil {
		return fmt.Errorf("byte-view index in %s: %w", function.Name, err)
	}
	if receiverType != operation.OwnerType || indexType != operation.IndexType {
		return fmt.Errorf("byte-view index in %s has receiver type %q and index type %q", function.Name, receiverType, indexType)
	}
	if destination && len(operation.TrapCategories) != 0 || !destination && !slices.Equal(operation.TrapCategories, []string{"bounds"}) {
		return fmt.Errorf("byte-view index in %s has an invalid trap marker", function.Name)
	}
	return nil
}

func (e *emitter) inspectViewIndexStore(functionIndex int, operation mir.Operation) error {
	function := &e.functions[functionIndex]
	receiver, index, value, err := threeOperands(operation.Operands)
	if err != nil {
		return fmt.Errorf("mutable byte-view index store in %s: %w", function.Name, err)
	}
	if operation.OwnerType != hir.MutableBorrow(hir.TypeByteSlice) || operation.IndexType != hir.TypeUSize || operation.Type != hir.TypeU8 ||
		operation.Operator != "[]=" || operation.DestinationLocal != mir.NoLocal {
		return fmt.Errorf("mutable byte-view index store in %s has incompatible owner, value, operator, or destination", function.Name)
	}
	receiverType, receiverErr := e.operandType(functionIndex, receiver)
	indexType, indexErr := e.operandType(functionIndex, index)
	valueType, valueErr := e.operandType(functionIndex, value)
	if receiverErr != nil || indexErr != nil || valueErr != nil || receiverType != operation.OwnerType || indexType != hir.TypeUSize || valueType != hir.TypeU8 {
		return fmt.Errorf("mutable byte-view index store in %s has incompatible operand types", function.Name)
	}
	if len(operation.TrapCategories) != 0 {
		return fmt.Errorf("mutable byte-view index store in %s has an invalid trap marker", function.Name)
	}
	return nil
}

func isCByteViewType(value hir.Type) bool {
	return value == hir.SharedBorrow(hir.TypeByteSlice) || value == hir.MutableBorrow(hir.TypeByteSlice)
}

func (e *emitter) inspectViewSlice(functionIndex int, operation mir.Operation, destination bool) error {
	function := &e.functions[functionIndex]
	receiver, start, end, err := threeOperands(operation.Operands)
	if err != nil {
		return fmt.Errorf("immutable byte-view slice in %s: %w", function.Name, err)
	}
	wantView := hir.SharedBorrow(hir.TypeByteSlice)
	if operation.OwnerType != wantView || operation.IndexType != hir.TypeUSize || operation.Type != wantView || operation.Operator != ".." ||
		destination != (operation.DestinationLocal != mir.NoLocal) {
		return fmt.Errorf("immutable byte-view slice in %s has incompatible owner, result, operator, or destination", function.Name)
	}
	receiverType, receiverErr := e.operandType(functionIndex, receiver)
	startType, startErr := e.operandType(functionIndex, start)
	endType, endErr := e.operandType(functionIndex, end)
	if receiverErr != nil || startErr != nil || endErr != nil || receiverType != wantView || startType != hir.TypeUSize || endType != hir.TypeUSize {
		return fmt.Errorf("immutable byte-view slice in %s has incompatible operand types", function.Name)
	}
	if destination && len(operation.TrapCategories) != 0 || !destination && !slices.Equal(operation.TrapCategories, []string{"bounds"}) {
		return fmt.Errorf("immutable byte-view slice in %s has an invalid trap marker", function.Name)
	}
	return nil
}

func (e *emitter) inspectScalarCast(functionIndex int, operation mir.Operation) error {
	function := &e.functions[functionIndex]
	operand, err := oneOperand(operation.Operands)
	if err != nil {
		return fmt.Errorf("value-preserving integer cast in %s: %w", function.Name, err)
	}
	if operation.DestinationLocal == mir.NoLocal || operation.Operator != "as" ||
		!hir.ValuePreservingIntegerCast(operation.OwnerType, operation.Type) {
		return fmt.Errorf("value-preserving integer cast in %s has an invalid shape", function.Name)
	}
	operandType, err := e.operandType(functionIndex, operand)
	if err != nil {
		return fmt.Errorf("value-preserving integer cast in %s: %w", function.Name, err)
	}
	if operandType != operation.OwnerType {
		return fmt.Errorf("value-preserving integer cast in %s has operand type %q, want %q", function.Name, operandType, operation.OwnerType)
	}
	if _, err := e.cValueType(operation.Type); err != nil {
		return fmt.Errorf("value-preserving integer cast result in %s: %w", function.Name, err)
	}
	e.recordCScalarFeature(operation.OwnerType)
	e.recordCScalarFeature(operation.Type)
	return nil
}

func (e *emitter) inspectBinary(functionIndex int, operation mir.Operation) error {
	function := &e.functions[functionIndex]
	left, right, err := twoOperands(operation.Operands)
	if err != nil {
		return fmt.Errorf("typed scalar binary operation in %s: %w", function.Name, err)
	}
	if operation.DestinationLocal == mir.NoLocal || !supportedBinary(operation.Operator) {
		return fmt.Errorf("typed scalar binary operation in %s has an invalid destination or operator", function.Name)
	}
	validShape := false
	switch operation.Operator {
	case "==", "!=":
		_, _, pointer := hir.RawPointerElement(operation.OwnerType)
		validShape = (hir.ScalarEqualityType(operation.OwnerType) || pointer) && operation.Type == hir.TypeBool
	case "<", "<=", ">", ">=":
		validShape = hir.ScalarOrderedType(operation.OwnerType) && operation.Type == hir.TypeBool
	case "+", "-", "*":
		validShape = hir.CheckedArithmeticType(operation.OwnerType) && operation.Type == operation.OwnerType
	case "/", "%":
		validShape = hir.CheckedDivisionType(operation.OwnerType) && operation.Type == operation.OwnerType
	case ">>", "<<":
		validShape = hir.CheckedShiftType(operation.OwnerType) && operation.Type == operation.OwnerType
	case "&", "^", "|":
		validShape = hir.BitwiseIntegerType(operation.OwnerType) && operation.Type == operation.OwnerType
	}
	if !validShape {
		return fmt.Errorf("typed scalar binary operation in %s has incompatible owner or result type", function.Name)
	}
	leftType, err := e.operandType(functionIndex, left)
	if err != nil {
		return fmt.Errorf("typed scalar binary left operand in %s: %w", function.Name, err)
	}
	rightType, err := e.operandType(functionIndex, right)
	if err != nil {
		return fmt.Errorf("typed scalar binary right operand in %s: %w", function.Name, err)
	}
	if leftType != operation.OwnerType || rightType != operation.OwnerType {
		return fmt.Errorf("typed scalar binary operation in %s has operand types %q and %q, want %q", function.Name, leftType, rightType, operation.OwnerType)
	}
	return nil
}

func (e *emitter) inspectUnary(functionIndex int, operation mir.Operation) error {
	function := &e.functions[functionIndex]
	operand, err := oneOperand(operation.Operands)
	if err != nil {
		return fmt.Errorf("typed scalar unary operation in %s: %w", function.Name, err)
	}
	if operation.DestinationLocal == mir.NoLocal || !supportedUnary(operation.Operator) {
		return fmt.Errorf("typed scalar unary operation in %s has an invalid destination or operator", function.Name)
	}
	if operation.Operator == "&" || operation.Operator == "&mut" {
		if operand.Kind != mir.OperandLocal {
			return fmt.Errorf("safe borrow in %s does not name a local place", function.Name)
		}
		wantType := hir.SharedBorrow(operation.OwnerType)
		if operation.Operator == "&mut" {
			wantType = hir.MutableBorrow(operation.OwnerType)
		}
		if operation.Type != wantType {
			return fmt.Errorf("safe borrow in %s has type %q, want %q", function.Name, operation.Type, wantType)
		}
		if _, ok := e.safeBorrowCType(operation.Type); !ok {
			return CapabilityError{Function: function.Name, Feature: "safe borrow type " + strconv.Quote(string(operation.Type))}
		}
		operandType, err := e.operandType(functionIndex, operand)
		if err != nil {
			return fmt.Errorf("safe borrow in %s: %w", function.Name, err)
		}
		if operandType != operation.OwnerType {
			return fmt.Errorf("safe borrow in %s has operand type %q, want %q", function.Name, operandType, operation.OwnerType)
		}
		return nil
	}
	if operation.Operator == "*" {
		element, _, borrow := hir.BorrowElement(operation.OwnerType)
		if !borrow || operation.Type != element {
			return fmt.Errorf("safe dereference in %s has incompatible owner or result type", function.Name)
		}
		operandType, err := e.operandType(functionIndex, operand)
		if err != nil {
			return fmt.Errorf("safe dereference in %s: %w", function.Name, err)
		}
		if operandType != operation.OwnerType {
			return fmt.Errorf("safe dereference in %s has operand type %q, want %q", function.Name, operandType, operation.OwnerType)
		}
		return nil
	}
	validShape := false
	switch operation.Operator {
	case "!":
		validShape = operation.OwnerType == hir.TypeBool && operation.Type == hir.TypeBool
	case "+", "-":
		validShape = (operation.OwnerType == hir.TypeI32 || operation.OwnerType == hir.TypeI64) && operation.Type == operation.OwnerType
	case "~":
		validShape = hir.BitwiseNotType(operation.OwnerType) && operation.Type == operation.OwnerType
	}
	if !validShape {
		return fmt.Errorf("typed scalar unary operation in %s has incompatible owner or result type", function.Name)
	}
	operandType, err := e.operandType(functionIndex, operand)
	if err != nil {
		return fmt.Errorf("typed scalar unary operand in %s: %w", function.Name, err)
	}
	if operandType != operation.OwnerType {
		return fmt.Errorf("typed scalar unary operation in %s has operand type %q, want %q", function.Name, operandType, operation.OwnerType)
	}
	return nil
}

func (e *emitter) inspectRegionOperation(functionIndex int, operation mir.Operation) error {
	function := &e.functions[functionIndex]
	switch operation.Kind {
	case mir.OperationRegionEnter:
		operand, err := oneOperand(operation.Operands)
		if err != nil || operand.Kind != mir.OperandLocal {
			return fmt.Errorf("Region entry in %s has no Arena owner local", function.Name)
		}
		ownerType, err := e.operandType(functionIndex, operand)
		if err != nil || ownerType != operation.OwnerType || operation.Type != hir.MutableBorrow(operation.OwnerType) {
			return fmt.Errorf("Region entry in %s has incompatible Arena ownership", function.Name)
		}
		if _, found := e.resourceByType[operation.OwnerType]; !found {
			return CapabilityError{Function: function.Name, Feature: "Region Arena type " + strconv.Quote(string(operation.OwnerType))}
		}
	case mir.OperationRegionPlace:
		operand, err := oneOperand(operation.Operands)
		if err != nil {
			return fmt.Errorf("Region placement in %s: %w", function.Name, err)
		}
		ownerType, err := e.operandType(functionIndex, operand)
		if err != nil || ownerType != operation.OwnerType || e.supportedNeedsDropType(operation.OwnerType) {
			return CapabilityError{Function: function.Name, Feature: "Region placement type " + strconv.Quote(string(operation.OwnerType))}
		}
		successType, failureType, feature, supported := e.resultCSpec(operation.Type)
		placedType, mutable, borrow := hir.BorrowElement(successType)
		errorIndex, errorStruct := e.structByType[failureType]
		if !supported || !borrow || mutable || placedType != operation.OwnerType || !errorStruct ||
			len(e.structTypes[errorIndex].definition.Fields) != 1 || e.structTypes[errorIndex].definition.Fields[0].Type != hir.TypeU32 {
			return CapabilityError{Function: function.Name, Feature: "fallible Region placement result " + strconv.Quote(string(operation.Type))}
		}
		e.features[feature] = struct{}{}
	case mir.OperationRegionReset:
		if len(operation.Operands) != 0 {
			return fmt.Errorf("Region reset in %s has operands", function.Name)
		}
	default:
		return fmt.Errorf("inspect Region operation in %s received %q", function.Name, operation.Kind)
	}
	e.features["region-lifecycle"] = struct{}{}
	e.trapCategories["unreachable"] = struct{}{}
	for _, symbol := range []string{"fy_std_region_begin_v0", "fy_std_region_allocate_zeroed_v0", "fy_std_region_reset_v0"} {
		if !cstdlibcontract.AdmittedForTarget(symbol, e.targetID) {
			return CapabilityError{Function: function.Name, Feature: "Region runtime symbol " + strconv.Quote(symbol)}
		}
		e.usedForeignSymbols[symbol] = struct{}{}
	}
	return nil
}

func unsignedCMaximum(value hir.Type) (string, error) {
	switch value {
	case hir.TypeU8:
		return "UINT8_MAX", nil
	case hir.TypeU32:
		return "UINT32_MAX", nil
	case hir.TypeUSize:
		return "UINTPTR_MAX", nil
	default:
		return "", fmt.Errorf("type %q has no unsigned C maximum", value)
	}
}

func shiftCWidth(value hir.Type) (string, error) {
	switch value {
	case hir.TypeI64:
		return "INT64_C(64)", nil
	case hir.TypeI32:
		return "INT32_C(32)", nil
	case hir.TypeU8:
		return "UINT8_C(8)", nil
	case hir.TypeU32:
		return "UINT32_C(32)", nil
	case hir.TypeUSize:
		return "((uintptr_t)UINT64_C(64))", nil
	default:
		return "", fmt.Errorf("type %q has no checked right-shift width", value)
	}
}

func checkedDivisorName(value hir.Type) (string, error) {
	switch value {
	case hir.TypeI64:
		return "fy_checked_divisor_i64", nil
	case hir.TypeI32:
		return "fy_checked_divisor_i32", nil
	case hir.TypeU8:
		return "fy_checked_divisor_u8", nil
	case hir.TypeU32:
		return "fy_checked_divisor_u32", nil
	case hir.TypeUSize:
		return "fy_checked_divisor_usize", nil
	default:
		return "", fmt.Errorf("type %q has no checked divisor slot", value)
	}
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
	if !pointer || element != operation.OwnerType && element != hir.TypeVoid || operation.Operator != "&" && operation.Operator != "&mut" || operation.Operator == "&" && mutable {
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

func (e *emitter) inspectRawBorrowPointer(functionIndex int, operation mir.Operation) error {
	function := &e.functions[functionIndex]
	operand, err := oneOperand(operation.Operands)
	if err != nil {
		return fmt.Errorf("raw borrow pointer in %s: %w", function.Name, err)
	}
	if operand.Kind != mir.OperandLocal || operation.DestinationLocal == mir.NoLocal ||
		operation.Entity == "" || operation.UnsafeScope.Source == "" || operation.Operator != "as" {
		return fmt.Errorf("raw borrow pointer in %s has incomplete operand or unsafe provenance", function.Name)
	}
	sourceElement, sourceMutable, sourceBorrow := hir.BorrowElement(operation.OwnerType)
	targetElement, targetMutable, targetPointer := hir.RawPointerElement(operation.Type)
	if !sourceBorrow || !targetPointer || (targetElement != sourceElement && targetElement != hir.TypeVoid) ||
		(!sourceMutable && targetMutable) {
		return fmt.Errorf("raw borrow pointer in %s has incompatible source and pointer types", function.Name)
	}
	operandType, err := e.operandType(functionIndex, operand)
	if err != nil {
		return fmt.Errorf("raw borrow pointer in %s: %w", function.Name, err)
	}
	if operandType != operation.OwnerType {
		return fmt.Errorf("raw borrow pointer in %s has operand type %q, want %q", function.Name, operandType, operation.OwnerType)
	}
	if _, ok := e.safeBorrowCType(operation.OwnerType); !ok {
		return CapabilityError{Function: function.Name, Feature: "raw borrow owner type " + strconv.Quote(string(operation.OwnerType))}
	}
	if _, err := e.cValueType(operation.Type); err != nil {
		return CapabilityError{Function: function.Name, Feature: "raw borrow pointer type " + strconv.Quote(string(operation.Type))}
	}
	return nil
}

func (e *emitter) inspectBorrowedViewFromRawParts(functionIndex int, operation mir.Operation) error {
	function := &e.functions[functionIndex]
	if len(operation.Operands) != 3 || operation.Entity == "" || operation.UnsafeScope.Source == "" {
		return fmt.Errorf("borrowed-view raw-parts construction in %s has incomplete operands or unsafe provenance", function.Name)
	}
	_, ownerMutable, ownerBorrow := hir.BorrowElement(operation.OwnerType)
	wantType := hir.SharedBorrow(hir.TypeStr)
	wantOperator := "str::from_raw_parts"
	wantOwnerMutable := false
	wantDataMutable := false
	if operation.Kind == mir.OperationCStrFromRawParts {
		wantType = hir.SharedBorrow(hir.TypeCStr)
		wantOperator = "CStr::from_raw_parts"
	} else if operation.Kind == mir.OperationByteSliceFromRawParts {
		wantType = hir.SharedBorrow(hir.TypeByteSlice)
		wantOperator = "byte_slice::from_raw_parts"
	} else if operation.Kind == mir.OperationMutByteSliceFromRawParts {
		wantType = hir.MutableBorrow(hir.TypeByteSlice)
		wantOperator = "byte_slice::from_raw_parts_mut"
		wantOwnerMutable = true
		wantDataMutable = true
	}
	if !ownerBorrow || ownerMutable != wantOwnerMutable || operation.Type != wantType || operation.Operator != wantOperator {
		return fmt.Errorf("borrowed-view raw-parts construction in %s has incompatible owner or result type", function.Name)
	}
	ownerType, err := e.operandType(functionIndex, operation.Operands[0])
	if err != nil || ownerType != operation.OwnerType {
		return fmt.Errorf("borrowed-view raw-parts owner in %s has type %q, want %q", function.Name, ownerType, operation.OwnerType)
	}
	dataType, err := e.operandType(functionIndex, operation.Operands[1])
	if err != nil {
		return fmt.Errorf("borrowed-view raw-parts data in %s: %w", function.Name, err)
	}
	dataElement, dataMutable, dataPointer := hir.RawPointerElement(dataType)
	if !dataPointer || dataElement != hir.TypeU8 && dataElement != hir.TypeVoid || wantDataMutable && !dataMutable {
		return fmt.Errorf("borrowed-view raw-parts data in %s has incompatible type %q", function.Name, dataType)
	}
	lengthType, err := e.operandType(functionIndex, operation.Operands[2])
	if err != nil || lengthType != hir.TypeUSize {
		return fmt.Errorf("borrowed-view raw-parts length in %s has type %q, want %q", function.Name, lengthType, hir.TypeUSize)
	}
	if _, err := e.cValueType(operation.Type); err != nil {
		return CapabilityError{Function: function.Name, Feature: "borrowed-view raw-parts result type " + strconv.Quote(string(operation.Type))}
	}
	return nil
}

func (e *emitter) inspectByteSliceDataAddress(functionIndex int, operation mir.Operation) error {
	function := &e.functions[functionIndex]
	operand, err := oneOperand(operation.Operands)
	if err != nil {
		return fmt.Errorf("byte-slice data address in %s: %w", function.Name, err)
	}
	if operand.Kind != mir.OperandByteString && operand.Kind != mir.OperandLocal || operation.Entity == "" || operation.UnsafeScope.Source == "" {
		return fmt.Errorf("byte-slice data address in %s has incomplete storage or unsafe provenance", function.Name)
	}
	element, mutable, pointer := hir.RawPointerElement(operation.Type)
	ownerElement, ownerMutable, ownerBorrow := hir.BorrowElement(operation.OwnerType)
	staticStorageValid := operand.Kind != mir.OperandByteString || !ownerMutable
	if !pointer || element != hir.TypeU8 || !ownerBorrow || ownerElement != hir.TypeByteSlice ||
		!staticStorageValid || (!ownerMutable && mutable) || operation.Operator != "as" {
		return fmt.Errorf("byte-slice data address in %s has incompatible source and pointer types", function.Name)
	}
	operandType, err := e.operandType(functionIndex, operand)
	if err != nil {
		return fmt.Errorf("byte-slice data address in %s: %w", function.Name, err)
	}
	if operandType != operation.OwnerType {
		return fmt.Errorf("byte-slice data address in %s has storage type %q, want %q", function.Name, operandType, operation.OwnerType)
	}
	if _, err := e.cValueType(operation.Type); err != nil {
		return CapabilityError{Function: function.Name, Feature: "byte-slice data address type " + strconv.Quote(string(operation.Type))}
	}
	return nil
}

func (e *emitter) inspectStrDataAddress(functionIndex int, operation mir.Operation) error {
	function := &e.functions[functionIndex]
	operand, err := oneOperand(operation.Operands)
	if err != nil {
		return fmt.Errorf("string data address in %s: %w", function.Name, err)
	}
	if operand.Kind != mir.OperandString && operand.Kind != mir.OperandLocal || operation.Entity == "" || operation.UnsafeScope.Source == "" {
		return fmt.Errorf("string data address in %s has incomplete storage or unsafe provenance", function.Name)
	}
	element, mutable, pointer := hir.RawPointerElement(operation.Type)
	if !pointer || mutable || element != hir.TypeU8 || operation.OwnerType != hir.SharedBorrow(hir.TypeStr) || operation.Operator != "as" {
		return fmt.Errorf("string data address in %s has incompatible source and pointer types", function.Name)
	}
	operandType, err := e.operandType(functionIndex, operand)
	if err != nil {
		return fmt.Errorf("string data address in %s: %w", function.Name, err)
	}
	if operandType != operation.OwnerType {
		return fmt.Errorf("string data address in %s has storage type %q, want %q", function.Name, operandType, operation.OwnerType)
	}
	if _, err := e.cValueType(operation.Type); err != nil {
		return CapabilityError{Function: function.Name, Feature: "string data address type " + strconv.Quote(string(operation.Type))}
	}
	return nil
}

func (e *emitter) inspectCStrDataAddress(functionIndex int, operation mir.Operation) error {
	function := &e.functions[functionIndex]
	operand, err := oneOperand(operation.Operands)
	if err != nil {
		return fmt.Errorf("C string data address in %s: %w", function.Name, err)
	}
	if operand.Kind != mir.OperandCString && operand.Kind != mir.OperandLocal || operation.Entity == "" || operation.UnsafeScope.Source == "" {
		return fmt.Errorf("C string data address in %s has incomplete storage or unsafe provenance", function.Name)
	}
	element, mutable, pointer := hir.RawPointerElement(operation.Type)
	if !pointer || mutable || element != hir.TypeU8 || operation.OwnerType != hir.SharedBorrow(hir.TypeCStr) || operation.Operator != "as" {
		return fmt.Errorf("C string data address in %s has incompatible source and pointer types", function.Name)
	}
	operandType, err := e.operandType(functionIndex, operand)
	if err != nil {
		return fmt.Errorf("C string data address in %s: %w", function.Name, err)
	}
	if operandType != operation.OwnerType {
		return fmt.Errorf("C string data address in %s has storage type %q, want %q", function.Name, operandType, operation.OwnerType)
	}
	if _, err := e.cValueType(operation.Type); err != nil {
		return CapabilityError{Function: function.Name, Feature: "C string data address type " + strconv.Quote(string(operation.Type))}
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
		index, found := e.structByType[fieldAggregateType(operation.OwnerType)]
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
	case mir.OperandI64:
		return hir.TypeI64, nil
	case mir.OperandI32:
		if operand.Type == hir.TypeBool {
			if operand.I32 == 0 || operand.I32 == 1 {
				return hir.TypeBool, nil
			}
			return hir.TypeInvalid, fmt.Errorf("boolean immediate has value %d", operand.I32)
		}
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
	case mir.OperandByteString:
		if operand.Type == hir.SharedBorrow(hir.TypeByteSlice) {
			return operand.Type, nil
		}
		return hir.TypeInvalid, fmt.Errorf("byte-string operand has invalid type %q", operand.Type)
	case mir.OperandString:
		if operand.Type == hir.SharedBorrow(hir.TypeStr) && utf8.ValidString(operand.StaticData) {
			return operand.Type, nil
		}
		return hir.TypeInvalid, fmt.Errorf("string operand has invalid type %q", operand.Type)
	case mir.OperandCString:
		if operand.Type == hir.SharedBorrow(hir.TypeCStr) {
			return operand.Type, nil
		}
		return hir.TypeInvalid, fmt.Errorf("C string operand has invalid type %q", operand.Type)
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
	if operation.Kind == mir.OperationBurn && (operation.Type != resource.Type || operation.Destination != "" || operation.DropFlagEntity != "" || operation.FlagClearedFirst) {
		return fmt.Errorf("resource burn in %s has invalid direct-discharge metadata", function.Name)
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
			if operandType != field.Type {
				return fmt.Errorf(
					"resource construction field %d in %s has type %q, want %q",
					fieldIndex, function.Name, operandType, field.Type,
				)
			}
		}
	case mir.OperationFieldRead:
		index, found := e.resourceByType[fieldAggregateType(operation.OwnerType)]
		if !found {
			return CapabilityError{Function: function.Name, Feature: "field-read owner type " + strconv.Quote(string(operation.OwnerType))}
		}
		definition := &e.resourceTypes[index]
		fieldIndex, found := definition.fieldIndex[operation.Field]
		if !found {
			return fmt.Errorf("resource field read in %s names missing field %q", function.Name, operation.Field)
		}
		if operation.Type != definition.definition.Fields[fieldIndex].Type {
			return fmt.Errorf(
				"resource field read in %s has type %q, want %q",
				function.Name, operation.Type, definition.definition.Fields[fieldIndex].Type,
			)
		}
		receiver, err := oneOperand(operation.Operands)
		if err != nil {
			return fmt.Errorf("resource field read in %s: %w", function.Name, err)
		}
		receiverType, err := e.operandType(functionIndex, receiver)
		if err != nil {
			return fmt.Errorf("resource field read in %s: %w", function.Name, err)
		}
		if receiverType != operation.OwnerType {
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

func (e *emitter) inspectSwitch(functionIndex int, terminator mir.Terminator, predecessors []uint32) error {
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
	case "assert":
		operand, err := oneOperand(terminator.SwitchOperands)
		if err != nil {
			return fmt.Errorf("assert in %s: %w", function.Name, err)
		}
		valid := operand.Kind == mir.OperandI32 && operand.Type == hir.TypeBool && (operand.I32 == 0 || operand.I32 == 1)
		if operand.Kind == mir.OperandLocal {
			operandType, typeErr := e.operandType(functionIndex, operand)
			valid = typeErr == nil && operandType == hir.TypeBool
		}
		if !valid || terminator.SwitchType != hir.TypeBool || terminator.Operator != "assert" {
			return fmt.Errorf("assert in %s has incompatible condition metadata", function.Name)
		}
		e.features["assert"] = struct{}{}
		categories = []string{"assertion-failed"}
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
		left, right, err := twoOperands(terminator.SwitchOperands)
		if err != nil {
			return fmt.Errorf("checked division in %s: %w", function.Name, err)
		}
		leftType, leftErr := e.operandType(functionIndex, left)
		rightType, rightErr := e.operandType(functionIndex, right)
		if leftErr != nil || rightErr != nil || !hir.CheckedDivisionType(terminator.SwitchType) ||
			leftType != terminator.SwitchType || rightType != terminator.SwitchType {
			return fmt.Errorf("checked division in %s has incompatible type metadata", function.Name)
		}
		e.features["checked-division"] = struct{}{}
		categories = []string{"division-by-zero"}
		if terminator.SwitchType == hir.TypeI32 || terminator.SwitchType == hir.TypeI64 {
			categories = append(categories, "signed-division-overflow")
		}
	case "checked-view-index":
		receiver, index, err := twoOperands(terminator.SwitchOperands)
		if err != nil {
			return fmt.Errorf("checked byte-view index in %s: %w", function.Name, err)
		}
		receiverType, err := e.operandType(functionIndex, receiver)
		if err != nil {
			return fmt.Errorf("checked byte-view index receiver in %s: %w", function.Name, err)
		}
		indexType, err := e.operandType(functionIndex, index)
		if err != nil {
			return fmt.Errorf("checked byte-view index offset in %s: %w", function.Name, err)
		}
		if terminator.Operator != "[]" || !isCByteViewType(receiverType) || indexType != hir.TypeUSize {
			return fmt.Errorf("checked byte-view index in %s has incompatible operands", function.Name)
		}
		e.features["view-index"] = struct{}{}
		categories = []string{"bounds"}
	case "checked-array-index":
		receiver, index, err := twoOperands(terminator.SwitchOperands)
		if err != nil {
			return fmt.Errorf("checked fixed-array index in %s: %w", function.Name, err)
		}
		receiverType, receiverErr := e.operandType(functionIndex, receiver)
		indexType, indexErr := e.operandType(functionIndex, index)
		if _, _, array := hir.FixedArrayParts(receiverType); receiverErr != nil || indexErr != nil || !array ||
			terminator.Operator != "[]" || indexType != hir.TypeUSize {
			return fmt.Errorf("checked fixed-array index in %s has incompatible operands", function.Name)
		}
		e.features["array-index"] = struct{}{}
		categories = []string{"bounds"}
	case "checked-view-slice":
		receiver, start, end, err := threeOperands(terminator.SwitchOperands)
		if err != nil {
			return fmt.Errorf("checked byte-view slice in %s: %w", function.Name, err)
		}
		receiverType, receiverErr := e.operandType(functionIndex, receiver)
		startType, startErr := e.operandType(functionIndex, start)
		endType, endErr := e.operandType(functionIndex, end)
		if receiverErr != nil || startErr != nil || endErr != nil || terminator.Operator != ".." ||
			receiverType != hir.SharedBorrow(hir.TypeByteSlice) || startType != hir.TypeUSize || endType != hir.TypeUSize {
			return fmt.Errorf("checked byte-view slice in %s has incompatible operands", function.Name)
		}
		e.features["view-slice"] = struct{}{}
		categories = []string{"bounds"}
	case "checked-unary":
		operand, err := oneOperand(terminator.SwitchOperands)
		if err != nil {
			return fmt.Errorf("checked unary in %s: %w", function.Name, err)
		}
		operandType, err := e.operandType(functionIndex, operand)
		if err != nil || (terminator.SwitchType != hir.TypeI32 && terminator.SwitchType != hir.TypeI64) || operandType != terminator.SwitchType {
			return fmt.Errorf("checked unary in %s has incompatible type metadata", function.Name)
		}
		e.features["checked-arithmetic"] = struct{}{}
		categories = []string{"integer-overflow"}
	case "checked-arithmetic":
		left, right, err := twoOperands(terminator.SwitchOperands)
		if err != nil {
			return fmt.Errorf("checked arithmetic in %s: %w", function.Name, err)
		}
		leftType, leftErr := e.operandType(functionIndex, left)
		rightType, rightErr := e.operandType(functionIndex, right)
		if leftErr != nil || rightErr != nil || !hir.CheckedArithmeticType(terminator.SwitchType) ||
			leftType != terminator.SwitchType || rightType != terminator.SwitchType {
			return fmt.Errorf("checked arithmetic in %s has incompatible type metadata", function.Name)
		}
		e.features["checked-arithmetic"] = struct{}{}
		categories = []string{"integer-overflow"}
	case "checked-shift":
		left, right, err := twoOperands(terminator.SwitchOperands)
		if err != nil {
			return fmt.Errorf("checked shift in %s: %w", function.Name, err)
		}
		leftType, leftErr := e.operandType(functionIndex, left)
		rightType, rightErr := e.operandType(functionIndex, right)
		if leftErr != nil || rightErr != nil || !hir.CheckedShiftType(terminator.SwitchType) ||
			leftType != terminator.SwitchType || rightType != terminator.SwitchType {
			return fmt.Errorf("checked shift in %s has incompatible type metadata", function.Name)
		}
		e.features["checked-shift"] = struct{}{}
		switch terminator.Operator {
		case "<<":
			categories = []string{"invalid-shift", "integer-overflow"}
		case ">>":
			categories = []string{"invalid-shift"}
		default:
			return fmt.Errorf("checked shift in %s has unsupported operator %q", function.Name, terminator.Operator)
		}
	default:
		return CapabilityError{Function: function.Name, Feature: "MIR switch " + strconv.Quote(terminator.SwitchKind)}
	}
	if err := e.recordTrapCategories(function.Name, categories); err != nil {
		return err
	}
	e.trapCategories["unreachable"] = struct{}{}
	failure := terminator.Successors[1]
	if len(categories) == 1 && failure >= 0 && int(failure) < len(function.Blocks) && predecessors[failure] == 1 {
		trap := function.Blocks[failure]
		if len(trap.Operations) == 0 && trap.Terminator.Kind == mir.TerminatorTrap &&
			slices.Equal(trap.Terminator.TrapCategories, categories) {
			return nil
		}
	}
	targets := e.dynamicTrapTargets[functionIndex]
	if targets == nil {
		targets = make(map[mir.BlockID]struct{})
		e.dynamicTrapTargets[functionIndex] = targets
	}
	targets[failure] = struct{}{}
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
	fmt.Fprintf(output, "/* fairyc constrained C transport: %s, %s */\n", e.targetID, transportProfile)
	output.WriteString("#include <stdint.h>\n")
	_, usesCSizeT := e.features["c-size-t"]
	_, usesRegion := e.features["region-lifecycle"]
	if len(e.structTypes) != 0 || len(e.enumTypes) != 0 || len(e.resourceTypes) != 0 || len(e.taggedTypes) != 0 || usesCSizeT || usesRegion {
		output.WriteString("#include <stddef.h>\n")
	}
	if usesCSSizeT {
		output.WriteString("#include <sys/types.h>\n")
	}
	_, usesStorageTake := e.features["private-storage-take"]
	_, usesStoragePut := e.features["private-storage-put"]
	if e.needsBitcast || e.needsI64 || usesStorageTake || usesStoragePut {
		output.WriteString("#include <string.h>\n")
	}
	output.WriteByte('\n')
	if usesRegion {
		output.WriteString("typedef struct fy_region {\n")
		output.WriteString("    void *arena;\n    void *mark;\n    uint64_t generation;\n    uint8_t active;\n")
		output.WriteString("} fy_region;\n")
		output.WriteString("extern uint32_t fy_std_region_begin_v0(void *arena, void *output);\n")
		output.WriteString("extern uint32_t fy_std_region_allocate_zeroed_v0(void *region, size_t size, size_t alignment, void **output);\n")
		output.WriteString("extern uint32_t fy_std_region_reset_v0(void *region);\n\n")
	}
	_, usesStaticByteString := e.features["static-byte-string"]
	_, usesArrayByteView := e.features["array-byte-view-borrow"]
	if usesStaticByteString || usesArrayByteView || e.needsByteSlice {
		output.WriteString("typedef struct fy_byte_slice {\n")
		output.WriteString("    const uint8_t *data;\n")
		output.WriteString("    uintptr_t length;\n")
		output.WriteString("} fy_byte_slice;\n\n")
	}
	if _, used := e.features["mutable-array-byte-view-borrow"]; used || e.needsMutByteSlice {
		output.WriteString("typedef struct fy_mut_byte_slice {\n")
		output.WriteString("    uint8_t *data;\n")
		output.WriteString("    uintptr_t length;\n")
		output.WriteString("} fy_mut_byte_slice;\n\n")
	}
	_, usesStaticUTF8String := e.features["static-utf8-string"]
	_, usesUTF8StringView := e.features["utf8-string-view"]
	if usesStaticUTF8String || usesUTF8StringView || e.needsStringView {
		output.WriteString("typedef struct fy_str {\n")
		output.WriteString("    const uint8_t *data;\n")
		output.WriteString("    uintptr_t length;\n")
		output.WriteString("} fy_str;\n\n")
	}
	for _, transport := range cScalarTransports {
		if _, used := e.features[transport.feature]; !used {
			continue
		}
		scalar, available := e.cScalars[transport.sourceType]
		if !available {
			return CapabilityError{Feature: fmt.Sprintf("C scalar %q on target %q", transport.targetName, e.targetID)}
		}
		fmt.Fprintf(output, "_Static_assert(sizeof(%s) == UINT64_C(%d), \"fairyc target %s size\");\n", transport.cType, scalar.Layout.SizeBytes, transport.targetName)
		fmt.Fprintf(output, "_Static_assert(_Alignof(%s) == UINT64_C(%d), \"fairyc target %s alignment\");\n", transport.cType, scalar.Layout.AlignmentBytes, transport.targetName)
		switch scalar.Signedness {
		case "signed":
			fmt.Fprintf(output, "_Static_assert(((%s)-1) < (%s)0, \"fairyc target %s signed\");\n", transport.cType, transport.cType, transport.targetName)
		case "unsigned":
			fmt.Fprintf(output, "_Static_assert(((%s)-1) > (%s)0, \"fairyc target %s unsigned\");\n", transport.cType, transport.cType, transport.targetName)
		case "not-applicable":
		default:
			return fmt.Errorf("C scalar %q has invalid signedness %q", transport.targetName, scalar.Signedness)
		}
		output.WriteByte('\n')
	}
	_, scalarOption := e.features["scalar-option"]
	_, copyOption := e.features["copy-option"]
	_, resourceOption := e.features["resource-option"]
	if scalarOption || copyOption || resourceOption {
		output.WriteString("#define FY_OPTION_NONE UINT32_C(0)\n")
		output.WriteString("#define FY_OPTION_SOME UINT32_C(1)\n")
	}
	if scalarOption {
		output.WriteString("typedef struct { uint32_t tag; int32_t payload; } fy_option_i32;\n\n")
	}
	_, scalarResult := e.features["scalar-result"]
	_, copyResult := e.features["copy-result"]
	_, resourceResult := e.features["resource-result"]
	if scalarResult || copyResult || resourceResult {
		output.WriteString("#define FY_RESULT_OK UINT32_C(0)\n")
		output.WriteString("#define FY_RESULT_ERR UINT32_C(1)\n")
	}
	if scalarResult {
		output.WriteString("typedef struct { uint32_t tag; int32_t payload; } fy_result_i32_i32;\n\n")
	}
	if err := e.emitTaggedTypes(output, true); err != nil {
		return err
	}
	if err := e.emitScalarArrayTypes(output); err != nil {
		return err
	}
	if err := e.emitAggregateTypes(output); err != nil {
		return err
	}
	for index, resource := range e.resourceTypes {
		output.WriteString("typedef struct {\n")
		for fieldIndex, field := range resource.definition.Fields {
			cType, err := e.cValueType(field.Type)
			if err != nil {
				return fmt.Errorf("resource type %q field %q: %w", resource.definition.Type, field.Name, err)
			}
			fmt.Fprintf(output, "    %s f_%d;\n", cType, fieldIndex)
		}
		fmt.Fprintf(output, "} fy_resource_%d;\n", index)
		fmt.Fprintf(output, "_Static_assert(sizeof(fy_resource_%d) == UINT64_C(%d), \"fairyc resource %d size\");\n", index, resource.layout.SizeBytes, index)
		fmt.Fprintf(output, "_Static_assert(_Alignof(fy_resource_%d) == UINT64_C(%d), \"fairyc resource %d alignment\");\n", index, resource.layout.AlignmentBytes, index)
		for fieldIndex, field := range resource.layout.Fields {
			fmt.Fprintf(output, "_Static_assert(offsetof(fy_resource_%d, f_%d) == UINT64_C(%d), \"fairyc resource %d field %d offset\");\n", index, fieldIndex, field.OffsetBytes, index, fieldIndex)
		}
		output.WriteByte('\n')
	}
	if err := e.emitTaggedTypes(output, false); err != nil {
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
	if e.needsI64 {
		output.WriteString(i64CSource)
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
	resultType, err := e.emittedFunctionResultType(function)
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
	resultType, err := e.emittedFunctionResultType(function)
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
		fmt.Fprintf(output, "    (void)&fy_l_%d;\n", local)
	}
	for _, symbol := range functionRegionSymbols(*function) {
		fmt.Fprintf(output, "    fy_region fy_region_%d = {0};\n", symbol)
		fmt.Fprintf(output, "    (void)&fy_region_%d;\n", symbol)
	}
	for _, divisorType := range functionCheckedDivisorTypes(*function) {
		cType, err := e.cValueType(divisorType)
		if err != nil {
			return fmt.Errorf("function %s checked divisor type: %w", function.Name, err)
		}
		name, err := checkedDivisorName(divisorType)
		if err != nil {
			return fmt.Errorf("function %s checked divisor type: %w", function.Name, err)
		}
		fmt.Fprintf(output, "    volatile %s %s;\n", cType, name)
	}
	dynamicTargets := e.dynamicTrapTargets[functionIndex]
	if len(dynamicTargets) != 0 {
		output.WriteString("    uint32_t fy_pending_trap = FY_TRAP_UNREACHABLE;\n")
	}
	fmt.Fprintf(output, "    goto fy_b_%d;\n", function.Entry)
	blocks := slices.Clone(function.Blocks)
	slices.SortFunc(blocks, func(left, right mir.Block) int { return int(left.ID - right.ID) })
	reachable := reachableBlocksForC(*function)
	fixedTrapGroups := make(map[string][]mir.BlockID)
	for _, block := range blocks {
		// Empty, non-returning traps with the same category have identical C behavior.
		// Keep dynamic-category targets separate because their incoming edges set state.
		if !reachable[int(block.ID)] || len(block.Operations) != 0 ||
			block.Terminator.Kind != mir.TerminatorTrap || len(block.Terminator.TrapCategories) != 1 ||
			len(block.Terminator.Successors) != 0 || block.Terminator.UnwindSuccessor != mir.NoBlock {
			continue
		}
		if _, dynamic := dynamicTargets[block.ID]; dynamic {
			continue
		}
		category := block.Terminator.TrapCategories[0]
		fixedTrapGroups[category] = append(fixedTrapGroups[category], block.ID)
	}
	groupedTrapBlocks := make(map[mir.BlockID]struct{})
	var groupedCategories []string
	for category, ids := range fixedTrapGroups {
		if len(ids) < 2 {
			continue
		}
		groupedCategories = append(groupedCategories, category)
		for _, id := range ids {
			groupedTrapBlocks[id] = struct{}{}
		}
	}
	slices.Sort(groupedCategories)
	for blockIndex := range blocks {
		block := &blocks[blockIndex]
		if !reachable[int(block.ID)] {
			continue
		}
		if _, grouped := groupedTrapBlocks[block.ID]; grouped {
			continue
		}
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
	for _, category := range groupedCategories {
		name, _, ok := cTrapSpec(category)
		if !ok {
			return CapabilityError{Function: function.Name, Feature: "trap category " + strconv.Quote(category)}
		}
		for _, id := range fixedTrapGroups[category] {
			fmt.Fprintf(output, "fy_b_%d:\n", id)
		}
		fmt.Fprintf(output, "    fy_runtime_trap(%s);\n", name)
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
	case mir.OperationRawBorrowPointer:
		operand, err := oneOperand(operation.Operands)
		if err != nil || operand.Kind != mir.OperandLocal {
			return fmt.Errorf("raw borrow pointer in %s has invalid local operand", function)
		}
		fmt.Fprintf(output, "    fy_l_%d = fy_l_%d;\n", operation.DestinationLocal, operand.Local)
	case mir.OperationStrFromRawParts, mir.OperationCStrFromRawParts, mir.OperationByteSliceFromRawParts, mir.OperationMutByteSliceFromRawParts:
		if len(operation.Operands) != 3 {
			return fmt.Errorf("borrowed-view raw-parts construction in %s has invalid operands", function)
		}
		owner, err := cOperand(operation.Operands[0])
		if err != nil {
			return fmt.Errorf("borrowed-view raw-parts owner in %s: %w", function, err)
		}
		data, err := cOperand(operation.Operands[1])
		if err != nil {
			return fmt.Errorf("borrowed-view raw-parts data in %s: %w", function, err)
		}
		length, err := cOperand(operation.Operands[2])
		if err != nil {
			return fmt.Errorf("borrowed-view raw-parts length in %s: %w", function, err)
		}
		fmt.Fprintf(output, "    (void)%s;\n", owner)
		if operation.Kind == mir.OperationStrFromRawParts {
			fmt.Fprintf(output, "    fy_l_%d = (fy_str){ .data = ((const uint8_t *)(%s) != (const uint8_t *)0 ? (const uint8_t *)(%s) : (const uint8_t *)\"\"), .length = %s };\n", operation.DestinationLocal, data, data, length)
		} else if operation.Kind == mir.OperationCStrFromRawParts {
			fmt.Fprintf(output, "    (void)%s;\n", length)
			fmt.Fprintf(output, "    fy_l_%d = (const uint8_t *)(%s);\n", operation.DestinationLocal, data)
		} else if operation.Kind == mir.OperationByteSliceFromRawParts {
			fmt.Fprintf(output, "    fy_l_%d = (fy_byte_slice){ .data = (const uint8_t *)(%s), .length = %s };\n", operation.DestinationLocal, data, length)
		} else {
			fmt.Fprintf(output, "    fy_l_%d = (fy_mut_byte_slice){ .data = (uint8_t *)(%s), .length = %s };\n", operation.DestinationLocal, data, length)
		}
	case mir.OperationStrDataAddress:
		operand, err := oneOperand(operation.Operands)
		if err != nil || operand.Kind != mir.OperandString && operand.Kind != mir.OperandLocal {
			return fmt.Errorf("string data address in %s has invalid storage operand", function)
		}
		if operand.Kind == mir.OperandLocal {
			fmt.Fprintf(output, "    fy_l_%d = fy_l_%d.data;\n", operation.DestinationLocal, operand.Local)
		} else {
			fmt.Fprintf(output, "    fy_l_%d = %s;\n", operation.DestinationLocal, stringStorageSymbol(operand.StaticData))
		}
	case mir.OperationByteSliceDataAddress:
		operand, err := oneOperand(operation.Operands)
		if err != nil || operand.Kind != mir.OperandByteString && operand.Kind != mir.OperandLocal {
			return fmt.Errorf("byte-slice data address in %s has invalid storage operand", function)
		}
		if operand.Kind == mir.OperandLocal {
			fmt.Fprintf(output, "    fy_l_%d = fy_l_%d.data;\n", operation.DestinationLocal, operand.Local)
		} else {
			fmt.Fprintf(output, "    fy_l_%d = %s;\n", operation.DestinationLocal, byteStringStorageSymbol(operand.StaticData))
		}
	case mir.OperationCStrDataAddress:
		operand, err := oneOperand(operation.Operands)
		if err != nil || operand.Kind != mir.OperandCString && operand.Kind != mir.OperandLocal {
			return fmt.Errorf("C string data address in %s has invalid storage operand", function)
		}
		if operand.Kind == mir.OperandLocal {
			fmt.Fprintf(output, "    fy_l_%d = fy_l_%d;\n", operation.DestinationLocal, operand.Local)
		} else {
			value, err := cOperand(operand)
			if err != nil {
				return fmt.Errorf("C string data address in %s: %w", function, err)
			}
			fmt.Fprintf(output, "    fy_l_%d = %s;\n", operation.DestinationLocal, value)
		}
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
	case mir.OperationFieldStore:
		return e.writeFieldStore(output, operation)
	case mir.OperationDerefStore:
		return e.writeDerefStore(output, operation)
	case mir.OperationFieldBorrow:
		return e.writeFieldBorrow(output, operation)
	case mir.OperationFieldRead:
		aggregateType := fieldAggregateType(operation.OwnerType)
		_, _, borrowed := hir.BorrowElement(operation.OwnerType)
		if index, found := e.structByType[aggregateType]; found {
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
			if borrowed {
				fmt.Fprintf(output, "    fy_l_%d = (%s)->%s;\n", operation.DestinationLocal, value, e.structCFieldName(index, fieldIndex))
			} else {
				fmt.Fprintf(output, "    fy_l_%d = (%s).%s;\n", operation.DestinationLocal, value, e.structCFieldName(index, fieldIndex))
			}
			break
		}
		index, found := e.resourceByType[aggregateType]
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
		if borrowed {
			fmt.Fprintf(output, "    fy_l_%d = (%s)->f_%d;\n", operation.DestinationLocal, value, fieldIndex)
		} else {
			fmt.Fprintf(output, "    fy_l_%d = (%s).f_%d;\n", operation.DestinationLocal, value, fieldIndex)
		}
	case mir.OperationViewLength:
		receiver, err := oneOperand(operation.Operands)
		if err != nil {
			return fmt.Errorf("borrowed-view length in %s: %w", function, err)
		}
		value, err := cOperand(receiver)
		if err != nil {
			return fmt.Errorf("borrowed-view length in %s: %w", function, err)
		}
		fmt.Fprintf(output, "    fy_l_%d = (%s).length;\n", operation.DestinationLocal, value)
	case mir.OperationViewEmpty:
		receiver, err := oneOperand(operation.Operands)
		if err != nil {
			return fmt.Errorf("borrowed-view emptiness in %s: %w", function, err)
		}
		value, err := cOperand(receiver)
		if err != nil {
			return fmt.Errorf("borrowed-view emptiness in %s: %w", function, err)
		}
		fmt.Fprintf(output, "    fy_l_%d = ((%s).length == ((uintptr_t)UINT64_C(0))) ? INT32_C(1) : INT32_C(0);\n", operation.DestinationLocal, value)
	case mir.OperationArrayConstruct:
		arrayIndex, found := e.arrayByType[operation.Type]
		if !found {
			return CapabilityError{Function: function, Feature: "fixed-array construction type " + strconv.Quote(string(operation.Type))}
		}
		fmt.Fprintf(output, "    fy_l_%d = (%s){ .data = { ", operation.DestinationLocal, e.arrayCTypeName(arrayIndex))
		for index, operand := range operation.Operands {
			if index != 0 {
				output.WriteString(", ")
			}
			value, err := cOperand(operand)
			if err != nil {
				return fmt.Errorf("fixed-array construction element %d in %s: %w", index, function, err)
			}
			output.WriteString(value)
		}
		output.WriteString(" } };\n")
	case mir.OperationArrayBorrowView:
		receiver, err := oneOperand(operation.Operands)
		if err != nil || receiver.Kind != mir.OperandLocal {
			return fmt.Errorf("fixed-array byte-view borrow in %s has no local receiver", function)
		}
		_, length, array := hir.FixedArrayParts(operation.OwnerType)
		if !array {
			return fmt.Errorf("fixed-array byte-view borrow in %s has type %q", function, operation.OwnerType)
		}
		viewType := "fy_byte_slice"
		if operation.Type == hir.MutableBorrow(hir.TypeByteSlice) {
			viewType = "fy_mut_byte_slice"
		}
		fmt.Fprintf(output, "    fy_l_%d = (%s){ .data = fy_l_%d.data, .length = ((uintptr_t)UINT64_C(%d)) };\n", operation.DestinationLocal, viewType, receiver.Local, length)
	case mir.OperationStorageLayout:
		value, err := e.storageQueryValue(operation)
		if err != nil {
			return err
		}
		fmt.Fprintf(output, "    fy_l_%d = ((uintptr_t)UINT64_C(%d));\n", operation.DestinationLocal, value)
	case mir.OperationStorageTake:
		return e.writeStorageTake(output, operation)
	case mir.OperationStoragePut:
		return e.writeStoragePut(output, operation)
	case mir.OperationStorageBorrow, mir.OperationStorageBorrowMut:
		return e.writeStorageBorrow(output, operation)
	case mir.OperationStorageElementAddress:
		return e.writeStorageElementAddress(output, operation)
	case mir.OperationArrayLength:
		if _, err := oneOperand(operation.Operands); err != nil {
			return fmt.Errorf("fixed-array length in %s: %w", function, err)
		}
		_, length, array := hir.FixedArrayParts(operation.OwnerType)
		if !array {
			return fmt.Errorf("fixed-array length in %s has type %q", function, operation.OwnerType)
		}
		fmt.Fprintf(output, "    fy_l_%d = ((uintptr_t)UINT64_C(%d));\n", operation.DestinationLocal, length)
	case mir.OperationArrayIndex:
		receiver, index, err := twoOperands(operation.Operands)
		if err != nil {
			return fmt.Errorf("fixed-array index in %s: %w", function, err)
		}
		receiverValue, err := cOperand(receiver)
		if err != nil {
			return fmt.Errorf("fixed-array index receiver in %s: %w", function, err)
		}
		indexValue, err := cOperand(index)
		if err != nil {
			return fmt.Errorf("fixed-array index offset in %s: %w", function, err)
		}
		fmt.Fprintf(output, "    fy_l_%d = (%s).data[%s];\n", operation.DestinationLocal, receiverValue, indexValue)
	case mir.OperationArrayIndexStore:
		receiver, index, value, err := threeOperands(operation.Operands)
		if err != nil {
			return fmt.Errorf("fixed-array index store in %s: %w", function, err)
		}
		receiverValue, receiverErr := cOperand(receiver)
		indexValue, indexErr := cOperand(index)
		storedValue, valueErr := cOperand(value)
		if receiverErr != nil || indexErr != nil || valueErr != nil {
			return fmt.Errorf("fixed-array index store in %s has an invalid operand", function)
		}
		fmt.Fprintf(output, "    (%s).data[%s] = %s;\n", receiverValue, indexValue, storedValue)
	case mir.OperationViewIndex:
		receiver, index, err := twoOperands(operation.Operands)
		if err != nil {
			return fmt.Errorf("byte-view index in %s: %w", function, err)
		}
		receiverValue, err := cOperand(receiver)
		if err != nil {
			return fmt.Errorf("byte-view index receiver in %s: %w", function, err)
		}
		indexValue, err := cOperand(index)
		if err != nil {
			return fmt.Errorf("byte-view index offset in %s: %w", function, err)
		}
		fmt.Fprintf(output, "    fy_l_%d = (%s).data[%s];\n", operation.DestinationLocal, receiverValue, indexValue)
	case mir.OperationViewIndexStore:
		receiver, index, value, err := threeOperands(operation.Operands)
		if err != nil {
			return fmt.Errorf("mutable byte-view index store in %s: %w", function, err)
		}
		receiverValue, receiverErr := cOperand(receiver)
		indexValue, indexErr := cOperand(index)
		storedValue, valueErr := cOperand(value)
		if receiverErr != nil || indexErr != nil || valueErr != nil {
			return fmt.Errorf("mutable byte-view index store in %s has an invalid operand", function)
		}
		fmt.Fprintf(output, "    (%s).data[%s] = %s;\n", receiverValue, indexValue, storedValue)
	case mir.OperationViewSlice:
		receiver, start, end, err := threeOperands(operation.Operands)
		if err != nil {
			return fmt.Errorf("immutable byte-view slice in %s: %w", function, err)
		}
		receiverValue, err := cOperand(receiver)
		if err != nil {
			return fmt.Errorf("immutable byte-view slice receiver in %s: %w", function, err)
		}
		startValue, err := cOperand(start)
		if err != nil {
			return fmt.Errorf("immutable byte-view slice start in %s: %w", function, err)
		}
		endValue, err := cOperand(end)
		if err != nil {
			return fmt.Errorf("immutable byte-view slice end in %s: %w", function, err)
		}
		switch {
		case start.Kind == mir.OperandUSize && start.U64 == 0:
			fmt.Fprintf(output, "    fy_l_%d = (fy_byte_slice){ .data = (%s).data, .length = (%s) };\n", operation.DestinationLocal, receiverValue, endValue)
		case start.Kind == mir.OperandUSize:
			fmt.Fprintf(output, "    fy_l_%d = (fy_byte_slice){ .data = (%s).data + (%s), .length = (%s) - (%s) };\n", operation.DestinationLocal, receiverValue, startValue, endValue, startValue)
		default:
			fmt.Fprintf(output, "    fy_l_%d = (fy_byte_slice){ .data = (%s).data, .length = (%s) - (%s) };\n", operation.DestinationLocal, receiverValue, endValue, startValue)
			fmt.Fprintf(output, "    if ((%s) != ((uintptr_t)UINT64_C(0))) { fy_l_%d.data += (%s); }\n", startValue, operation.DestinationLocal, startValue)
		}
	case mir.OperationScalarCast:
		operand, err := oneOperand(operation.Operands)
		if err != nil {
			return fmt.Errorf("value-preserving integer cast in %s: %w", function, err)
		}
		value, err := cOperand(operand)
		if err != nil {
			return fmt.Errorf("value-preserving integer cast in %s: %w", function, err)
		}
		targetType, err := e.cValueType(operation.Type)
		if err != nil {
			return fmt.Errorf("value-preserving integer cast result in %s: %w", function, err)
		}
		fmt.Fprintf(output, "    fy_l_%d = (%s)(%s);\n", operation.DestinationLocal, targetType, value)
	case mir.OperationMove:
		fmt.Fprintf(output, "    (void)fy_l_%d;\n", operation.Symbol)
	case mir.OperationDrop:
		fmt.Fprintf(output, "    (void)fy_l_%d;\n", operation.Symbol)
		if e.instrumentation == InstrumentationDropTraceV0 {
			if err := e.emitDropTrace(output, functionIndex, operation); err != nil {
				return fmt.Errorf("drop trace in %s: %w", function, err)
			}
		}
		if err := e.emitDropGlue(output, functionIndex, operation); err != nil {
			return fmt.Errorf("drop glue in %s: %w", function, err)
		}
	case mir.OperationBurn:
		fmt.Fprintf(output, "    (void)fy_l_%d;\n", operation.Symbol)
		e.features["resource-burn"] = struct{}{}
	case mir.OperationRegionEnter:
		operand, err := oneOperand(operation.Operands)
		if err != nil || operand.Kind != mir.OperandLocal {
			return fmt.Errorf("Region entry in %s has invalid Arena owner", function)
		}
		fmt.Fprintf(output, "    if (fy_std_region_begin_v0((void *)&fy_l_%d, (void *)&fy_region_%d) != UINT32_C(0)) fy_runtime_trap(FY_TRAP_UNREACHABLE);\n", operand.Local, operation.Symbol)
	case mir.OperationRegionPlace:
		operand, err := oneOperand(operation.Operands)
		if err != nil {
			return fmt.Errorf("Region placement in %s: %w", function, err)
		}
		value, err := cOperand(operand)
		if err != nil {
			return fmt.Errorf("Region placement value in %s: %w", function, err)
		}
		ownerType, err := e.cValueType(operation.OwnerType)
		if err != nil {
			return fmt.Errorf("Region placement owner type in %s: %w", function, err)
		}
		taggedIndex, found := e.taggedByType[operation.Type]
		if !found {
			return fmt.Errorf("Region placement in %s has no tagged C type", function)
		}
		_, failureType, _, _ := e.resultCSpec(operation.Type)
		errorIndex, found := e.structByType[failureType]
		if !found {
			return fmt.Errorf("Region placement in %s has no allocation error C type", function)
		}
		errorField := e.structCFieldName(errorIndex, 0)
		fmt.Fprintf(output, "    void *fy_region_storage_%d = NULL;\n", operation.DestinationLocal)
		fmt.Fprintf(output, "    uint32_t fy_region_status_%d = fy_std_region_allocate_zeroed_v0((void *)&fy_region_%d, sizeof(%s), _Alignof(%s), &fy_region_storage_%d);\n", operation.DestinationLocal, operation.Symbol, ownerType, ownerType, operation.DestinationLocal)
		fmt.Fprintf(output, "    if (fy_region_status_%d == UINT32_C(0)) {\n", operation.DestinationLocal)
		fmt.Fprintf(output, "        *((%s *)fy_region_storage_%d) = %s;\n", ownerType, operation.DestinationLocal, value)
		fmt.Fprintf(output, "        fy_l_%d = (fy_tagged_%d){ .tag = FY_RESULT_OK, .p_0 = (const %s *)fy_region_storage_%d };\n", operation.DestinationLocal, taggedIndex, ownerType, operation.DestinationLocal)
		output.WriteString("    } else {\n")
		fmt.Fprintf(output, "        fy_l_%d = (fy_tagged_%d){ .tag = FY_RESULT_ERR, .p_1 = (%s){ .%s = fy_region_status_%d } };\n", operation.DestinationLocal, taggedIndex, e.structCTypeName(errorIndex), errorField, operation.DestinationLocal)
		output.WriteString("    }\n")
	case mir.OperationRegionReset:
		fmt.Fprintf(output, "    if (fy_std_region_reset_v0((void *)&fy_region_%d) != UINT32_C(0)) fy_runtime_trap(FY_TRAP_UNREACHABLE);\n", operation.Symbol)
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
			if operation.OwnerType == hir.TypeI64 {
				fmt.Fprintf(output, "    fy_l_%d = fy_bits_to_i64(UINT64_C(0) - (uint64_t)(%s));\n", operation.DestinationLocal, value)
				break
			}
			fmt.Fprintf(output, "    fy_l_%d = fy_bits_to_i32(UINT32_C(0) - (uint32_t)(%s));\n", operation.DestinationLocal, value)
		case "!":
			fmt.Fprintf(output, "    fy_l_%d = (%s == INT32_C(0)) ? INT32_C(1) : INT32_C(0);\n", operation.DestinationLocal, value)
		case "~":
			if operation.OwnerType == hir.TypeI64 {
				fmt.Fprintf(output, "    fy_l_%d = fy_bits_to_i64(~(uint64_t)(%s));\n", operation.DestinationLocal, value)
				break
			}
			if operation.OwnerType == hir.TypeI32 {
				fmt.Fprintf(output, "    fy_l_%d = fy_bits_to_i32(~(uint32_t)(%s));\n", operation.DestinationLocal, value)
			} else {
				valueType, err := e.cValueType(operation.OwnerType)
				if err != nil {
					return fmt.Errorf("bitwise not in %s: %w", function, err)
				}
				fmt.Fprintf(output, "    fy_l_%d = (%s)(~(%s)(%s));\n", operation.DestinationLocal, valueType, valueType, value)
			}
		case "&", "&mut":
			if operand.Kind != mir.OperandLocal {
				return fmt.Errorf("safe borrow in %s has invalid local operand", function)
			}
			fmt.Fprintf(output, "    fy_l_%d = &fy_l_%d;\n", operation.DestinationLocal, operand.Local)
		case "*":
			fmt.Fprintf(output, "    fy_l_%d = *(%s);\n", operation.DestinationLocal, value)
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
		if operation.OwnerType == hir.TypeI64 {
			emitI64Binary(output, operation, leftValue, rightValue)
			break
		}
		switch operation.Operator {
		case "+", "-", "*":
			if operation.OwnerType == hir.TypeI32 {
				fmt.Fprintf(output, "    fy_l_%d = fy_bits_to_i32((uint32_t)(%s) %s (uint32_t)(%s));\n", operation.DestinationLocal, leftValue, operation.Operator, rightValue)
			} else {
				fmt.Fprintf(output, "    fy_l_%d = (%s) %s (%s);\n", operation.DestinationLocal, leftValue, operation.Operator, rightValue)
			}
		case "/", "%":
			if directUnsignedDivisor(operation) {
				fmt.Fprintf(output, "    fy_l_%d = (%s) %s (%s);\n", operation.DestinationLocal, leftValue, operation.Operator, rightValue)
				break
			}
			divisor, err := checkedDivisorName(operation.OwnerType)
			if err != nil {
				return fmt.Errorf("checked division in %s: %w", function, err)
			}
			output.WriteString("    " + divisor + " = ")
			output.WriteString(rightValue)
			output.WriteString(";\n")
			fmt.Fprintf(output, "    fy_l_%d = (%s) %s %s;\n", operation.DestinationLocal, leftValue, operation.Operator, divisor)
		case ">>":
			if operation.OwnerType == hir.TypeI32 {
				fmt.Fprintf(output, "    fy_l_%d = fy_sar_i32(%s, %s);\n", operation.DestinationLocal, leftValue, rightValue)
			} else {
				fmt.Fprintf(output, "    fy_l_%d = (%s) >> (%s);\n", operation.DestinationLocal, leftValue, rightValue)
			}
		case "<<":
			if operation.OwnerType == hir.TypeI32 {
				fmt.Fprintf(output, "    fy_l_%d = fy_bits_to_i32((uint32_t)(%s) << (uint32_t)(%s));\n", operation.DestinationLocal, leftValue, rightValue)
			} else {
				fmt.Fprintf(output, "    fy_l_%d = (%s) << (%s);\n", operation.DestinationLocal, leftValue, rightValue)
			}
		case "&", "^", "|":
			if operation.OwnerType == hir.TypeI32 {
				fmt.Fprintf(output, "    fy_l_%d = fy_bits_to_i32((uint32_t)(%s) %s (uint32_t)(%s));\n", operation.DestinationLocal, leftValue, operation.Operator, rightValue)
			} else {
				fmt.Fprintf(output, "    fy_l_%d = (%s) %s (%s);\n", operation.DestinationLocal, leftValue, operation.Operator, rightValue)
			}
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
		nullNiche := sharedBorrowOptionNiche(operation.Type, e.structByType, e.resourceByType, e.arrayByType)
		if index, resourceTagged := e.taggedByType[operation.Type]; resourceTagged {
			cType = fmt.Sprintf("fy_tagged_%d", index)
			field = "p_0"
		}
		if operation.Variant == "none" {
			if nullNiche {
				fmt.Fprintf(output, "    fy_l_%d = (%s){ .p_0 = ((void *)0) };\n", operation.DestinationLocal, cType)
			} else {
				fmt.Fprintf(output, "    fy_l_%d = (%s){ .tag = FY_OPTION_NONE };\n", operation.DestinationLocal, cType)
			}
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
		if nullNiche {
			fmt.Fprintf(output, "    fy_l_%d = (%s){ .p_0 = %s };\n", operation.DestinationLocal, cType, payload)
		} else {
			fmt.Fprintf(output, "    fy_l_%d = (%s){ .tag = FY_OPTION_SOME, .%s = %s };\n", operation.DestinationLocal, cType, field, payload)
		}
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
	case mir.OperationUse, mir.OperationArithmeticCheck, mir.OperationCheckedDivision, mir.OperationViewIndexCheck, mir.OperationViewSliceCheck,
		mir.OperationArrayIndexCheck, mir.OperationResultSwitch:
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
		if e.resourceHasNestedFields(function.DropGlueFor) {
			fmt.Fprintf(output, "    return fy_l_%d;\n", function.Parameters[0])
			return nil
		}
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
	case "assert":
		operand, err := oneOperand(terminator.SwitchOperands)
		if err != nil {
			return fmt.Errorf("assert in %s: %w", function.Name, err)
		}
		value, err := cOperand(operand)
		if err != nil {
			return fmt.Errorf("assert condition in %s: %w", function.Name, err)
		}
		fmt.Fprintf(output, "    if (%s == INT32_C(0)) { %sgoto fy_b_%d; }\n", value, e.pendingTrapAssignment(functionIndex, failure, "FY_TRAP_ASSERTION_FAILED"), failure)
		fmt.Fprintf(output, "    goto fy_b_%d;\n", success)
	case "option":
		operand, err := oneOperand(terminator.SwitchOperands)
		if err != nil {
			return fmt.Errorf("Option switch in %s: %w", function.Name, err)
		}
		value, err := cOperand(operand)
		if err != nil {
			return fmt.Errorf("Option switch in %s: %w", function.Name, err)
		}
		valueType, err := e.operandType(functionIndex, operand)
		if err != nil {
			return fmt.Errorf("Option switch in %s: %w", function.Name, err)
		}
		if sharedBorrowOptionNiche(valueType, e.structByType, e.resourceByType, e.arrayByType) {
			fmt.Fprintf(output, "    if ((%s).p_0 != ((void *)0)) goto fy_b_%d;\n", value, success)
			fmt.Fprintf(output, "    goto fy_b_%d;\n", failure)
			break
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
		minimum := "INT32_MIN"
		if terminator.SwitchType == hir.TypeI64 {
			minimum = "INT64_MIN"
		}
		fmt.Fprintf(output, "    if (%s == %s) { %sgoto fy_b_%d; }\n", value, minimum, e.pendingTrapAssignment(functionIndex, failure, "FY_TRAP_INTEGER_OVERFLOW"), failure)
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
		if terminator.SwitchType == hir.TypeI64 {
			check := map[string]string{"+": "add", "-": "sub", "*": "mul"}[terminator.Operator]
			fmt.Fprintf(output, "    if (fy_i64_%s_overflow(%s, %s)) { %sgoto fy_b_%d; }\n", check, leftValue, rightValue, e.pendingTrapAssignment(functionIndex, failure, "FY_TRAP_INTEGER_OVERFLOW"), failure)
		} else if terminator.SwitchType == hir.TypeI32 {
			expression := fmt.Sprintf("((int64_t)(%s) %s (int64_t)(%s))", leftValue, terminator.Operator, rightValue)
			fmt.Fprintf(output, "    if (%s < (int64_t)INT32_MIN || %s > (int64_t)INT32_MAX) { %sgoto fy_b_%d; }\n", expression, expression, e.pendingTrapAssignment(functionIndex, failure, "FY_TRAP_INTEGER_OVERFLOW"), failure)
		} else {
			maximum, err := unsignedCMaximum(terminator.SwitchType)
			if err != nil {
				return fmt.Errorf("checked arithmetic in %s: %w", function.Name, err)
			}
			switch terminator.Operator {
			case "+":
				fmt.Fprintf(output, "    if ((%s) > %s - (%s)) { %sgoto fy_b_%d; }\n", leftValue, maximum, rightValue, e.pendingTrapAssignment(functionIndex, failure, "FY_TRAP_INTEGER_OVERFLOW"), failure)
			case "-":
				fmt.Fprintf(output, "    if ((%s) < (%s)) { %sgoto fy_b_%d; }\n", leftValue, rightValue, e.pendingTrapAssignment(functionIndex, failure, "FY_TRAP_INTEGER_OVERFLOW"), failure)
			case "*":
				fmt.Fprintf(output, "    if ((%s) != 0 && (%s) > %s / (%s)) { %sgoto fy_b_%d; }\n", rightValue, leftValue, maximum, rightValue, e.pendingTrapAssignment(functionIndex, failure, "FY_TRAP_INTEGER_OVERFLOW"), failure)
			}
		}
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
		fmt.Fprintf(output, "    if (%s == 0) { %sgoto fy_b_%d; }\n", rightValue, e.pendingTrapAssignment(functionIndex, failure, "FY_TRAP_DIVISION_BY_ZERO"), failure)
		if terminator.SwitchType == hir.TypeI64 {
			fmt.Fprintf(output, "    if (%s == INT64_MIN && %s == -INT64_C(1)) { %sgoto fy_b_%d; }\n", leftValue, rightValue, e.pendingTrapAssignment(functionIndex, failure, "FY_TRAP_SIGNED_DIVISION_OVERFLOW"), failure)
		}
		if terminator.SwitchType == hir.TypeI32 {
			fmt.Fprintf(output, "    if (%s == INT32_MIN && %s == -INT32_C(1)) { %sgoto fy_b_%d; }\n", leftValue, rightValue, e.pendingTrapAssignment(functionIndex, failure, "FY_TRAP_SIGNED_DIVISION_OVERFLOW"), failure)
		}
		fmt.Fprintf(output, "    goto fy_b_%d;\n", success)
	case "checked-view-index":
		receiver, index, err := twoOperands(terminator.SwitchOperands)
		if err != nil {
			return fmt.Errorf("checked byte-view index in %s: %w", function.Name, err)
		}
		receiverValue, err := cOperand(receiver)
		if err != nil {
			return fmt.Errorf("checked byte-view index receiver in %s: %w", function.Name, err)
		}
		indexValue, err := cOperand(index)
		if err != nil {
			return fmt.Errorf("checked byte-view index offset in %s: %w", function.Name, err)
		}
		fmt.Fprintf(output, "    if (%s >= (%s).length) { %sgoto fy_b_%d; }\n", indexValue, receiverValue, e.pendingTrapAssignment(functionIndex, failure, "FY_TRAP_BOUNDS"), failure)
		fmt.Fprintf(output, "    goto fy_b_%d;\n", success)
	case "checked-array-index":
		receiver, index, err := twoOperands(terminator.SwitchOperands)
		if err != nil {
			return fmt.Errorf("checked fixed-array index in %s: %w", function.Name, err)
		}
		receiverType, err := e.operandType(functionIndex, receiver)
		if err != nil {
			return fmt.Errorf("checked fixed-array index receiver in %s: %w", function.Name, err)
		}
		_, length, array := hir.FixedArrayParts(receiverType)
		if !array {
			return fmt.Errorf("checked fixed-array index receiver in %s has type %q", function.Name, receiverType)
		}
		indexValue, err := cOperand(index)
		if err != nil {
			return fmt.Errorf("checked fixed-array index offset in %s: %w", function.Name, err)
		}
		fmt.Fprintf(output, "    if (%s >= ((uintptr_t)UINT64_C(%d))) { %sgoto fy_b_%d; }\n", indexValue, length, e.pendingTrapAssignment(functionIndex, failure, "FY_TRAP_BOUNDS"), failure)
		fmt.Fprintf(output, "    goto fy_b_%d;\n", success)
	case "checked-view-slice":
		receiver, start, end, err := threeOperands(terminator.SwitchOperands)
		if err != nil {
			return fmt.Errorf("checked byte-view slice in %s: %w", function.Name, err)
		}
		receiverValue, err := cOperand(receiver)
		if err != nil {
			return fmt.Errorf("checked byte-view slice receiver in %s: %w", function.Name, err)
		}
		startValue, err := cOperand(start)
		if err != nil {
			return fmt.Errorf("checked byte-view slice start in %s: %w", function.Name, err)
		}
		endValue, err := cOperand(end)
		if err != nil {
			return fmt.Errorf("checked byte-view slice end in %s: %w", function.Name, err)
		}
		if start.Kind == mir.OperandUSize && start.U64 == 0 {
			fmt.Fprintf(output, "    if ((%s) > (%s).length) { %sgoto fy_b_%d; }\n", endValue, receiverValue, e.pendingTrapAssignment(functionIndex, failure, "FY_TRAP_BOUNDS"), failure)
		} else {
			fmt.Fprintf(output, "    if ((%s) > (%s) || (%s) > (%s).length) { %sgoto fy_b_%d; }\n", startValue, endValue, endValue, receiverValue, e.pendingTrapAssignment(functionIndex, failure, "FY_TRAP_BOUNDS"), failure)
		}
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
		width, err := shiftCWidth(terminator.SwitchType)
		if err != nil {
			return fmt.Errorf("checked shift in %s: %w", function.Name, err)
		}
		if terminator.SwitchType == hir.TypeI32 || terminator.SwitchType == hir.TypeI64 {
			fmt.Fprintf(output, "    if (%s < INT32_C(0) || %s >= %s) { %sgoto fy_b_%d; }\n", amountValue, amountValue, width, e.pendingTrapAssignment(functionIndex, failure, "FY_TRAP_INVALID_SHIFT"), failure)
		} else {
			fmt.Fprintf(output, "    if (%s >= %s) { %sgoto fy_b_%d; }\n", amountValue, width, e.pendingTrapAssignment(functionIndex, failure, "FY_TRAP_INVALID_SHIFT"), failure)
		}
		if terminator.Operator == "<<" {
			value, _, err := twoOperands(terminator.SwitchOperands)
			if err != nil {
				return fmt.Errorf("checked left shift in %s: %w", function.Name, err)
			}
			valueText, err := cOperand(value)
			if err != nil {
				return fmt.Errorf("checked left shift in %s: %w", function.Name, err)
			}
			if terminator.SwitchType == hir.TypeI64 {
				fmt.Fprintf(output, "    if ((%s) > (INT64_MAX >> (uint64_t)(%s)) || (%s) < fy_sar_i64(INT64_MIN, %s)) { %sgoto fy_b_%d; }\n", valueText, amountValue, valueText, amountValue, e.pendingTrapAssignment(functionIndex, failure, "FY_TRAP_INTEGER_OVERFLOW"), failure)
			} else if terminator.SwitchType == hir.TypeI32 {
				expression := fmt.Sprintf("((int64_t)(%s) * (INT64_C(1) << (uint32_t)(%s)))", valueText, amountValue)
				fmt.Fprintf(output, "    if (%s < (int64_t)INT32_MIN || %s > (int64_t)INT32_MAX) { %sgoto fy_b_%d; }\n", expression, expression, e.pendingTrapAssignment(functionIndex, failure, "FY_TRAP_INTEGER_OVERFLOW"), failure)
			} else {
				maximum, err := unsignedCMaximum(terminator.SwitchType)
				if err != nil {
					return fmt.Errorf("checked left shift in %s: %w", function.Name, err)
				}
				fmt.Fprintf(output, "    if ((%s) > (%s >> (%s))) { %sgoto fy_b_%d; }\n", valueText, maximum, amountValue, e.pendingTrapAssignment(functionIndex, failure, "FY_TRAP_INTEGER_OVERFLOW"), failure)
			}
		}
		fmt.Fprintf(output, "    goto fy_b_%d;\n", success)
	default:
		return CapabilityError{Function: function.Name, Feature: "MIR switch " + strconv.Quote(terminator.SwitchKind)}
	}
	return nil
}

func (e *emitter) pendingTrapAssignment(functionIndex int, failure mir.BlockID, category string) string {
	if _, dynamic := e.dynamicTrapTargets[functionIndex][failure]; !dynamic {
		return ""
	}
	return "fy_pending_trap = " + category + "; "
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

func functionRegionSymbols(function mir.Function) []hir.SymbolID {
	found := make(map[hir.SymbolID]struct{})
	for _, block := range function.Blocks {
		for _, operation := range block.Operations {
			if operation.Kind == mir.OperationRegionEnter || operation.Kind == mir.OperationRegionPlace || operation.Kind == mir.OperationRegionReset {
				found[operation.Symbol] = struct{}{}
			}
		}
	}
	result := make([]hir.SymbolID, 0, len(found))
	for symbol := range found {
		if symbol != hir.NoSymbol {
			result = append(result, symbol)
		}
	}
	slices.Sort(result)
	return result
}

func functionCheckedDivisorTypes(function mir.Function) []hir.Type {
	found := make(map[hir.Type]bool)
	for _, block := range function.Blocks {
		for _, operation := range block.Operations {
			if operation.Kind == mir.OperationBinary && (operation.Operator == "/" || operation.Operator == "%") && !directUnsignedDivisor(operation) {
				found[operation.OwnerType] = true
			}
		}
	}
	result := make([]hir.Type, 0, len(found))
	for _, candidate := range []hir.Type{hir.TypeI32, hir.TypeI64, hir.TypeU8, hir.TypeU32, hir.TypeUSize} {
		if found[candidate] {
			result = append(result, candidate)
		}
	}
	return result
}

func operationHasDestination(kind mir.OperationKind) bool {
	switch kind {
	case mir.OperationInitialize, mir.OperationAssign, mir.OperationUnary, mir.OperationBinary,
		mir.OperationResultConstruct, mir.OperationResultPayload,
		mir.OperationOptionConstruct, mir.OperationOptionPayload,
		mir.OperationEnumConstruct, mir.OperationEnumPayload,
		mir.OperationRawAddress, mir.OperationRawBorrowPointer, mir.OperationStrFromRawParts, mir.OperationCStrFromRawParts,
		mir.OperationByteSliceFromRawParts, mir.OperationMutByteSliceFromRawParts,
		mir.OperationStrDataAddress, mir.OperationByteSliceDataAddress, mir.OperationCStrDataAddress,
		mir.OperationStructConstruct, mir.OperationResourceConstruct, mir.OperationFieldRead, mir.OperationFieldBorrow, mir.OperationViewLength, mir.OperationViewEmpty, mir.OperationViewIndex, mir.OperationViewSlice,
		mir.OperationStorageLayout, mir.OperationStorageTake, mir.OperationStorageBorrow, mir.OperationStorageBorrowMut, mir.OperationStorageElementAddress, mir.OperationScalarCast, mir.OperationArrayConstruct, mir.OperationArrayBorrowView, mir.OperationArrayLength, mir.OperationArrayIndex, mir.OperationRegionPlace:
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
	case mir.OperandI64:
		return cI64(operand.I64), nil
	case mir.OperandU8:
		return fmt.Sprintf("UINT8_C(%d)", operand.U64), nil
	case mir.OperandU32:
		return fmt.Sprintf("UINT32_C(%d)", operand.U64), nil
	case mir.OperandUSize:
		return fmt.Sprintf("((uintptr_t)UINT64_C(%d))", operand.U64), nil
	case mir.OperandNull:
		return "((void *)0)", nil
	case mir.OperandByteString:
		if operand.Type != hir.SharedBorrow(hir.TypeByteSlice) {
			return "", fmt.Errorf("byte-string operand has invalid static storage")
		}
		return fmt.Sprintf("(fy_byte_slice){ .data = %s, .length = ((uintptr_t)UINT64_C(%d)) }", byteStringStorageSymbol(operand.StaticData), len(operand.StaticData)), nil
	case mir.OperandString:
		if operand.Type != hir.SharedBorrow(hir.TypeStr) || !utf8.ValidString(operand.StaticData) {
			return "", fmt.Errorf("string operand has invalid static storage")
		}
		return fmt.Sprintf("(fy_str){ .data = %s, .length = ((uintptr_t)UINT64_C(%d)) }", stringStorageSymbol(operand.StaticData), len(operand.StaticData)), nil
	case mir.OperandCString:
		if operand.Type != hir.SharedBorrow(hir.TypeCStr) || !validCStringStorage(operand.StaticData) {
			return "", fmt.Errorf("C string operand has invalid static storage")
		}
		return cStringStorageSymbol(operand.StaticData), nil
	default:
		return "", fmt.Errorf("unsupported scalar operand kind %q", operand.Kind)
	}
}

type staticStorageSpec struct {
	operandKind  mir.OperandKind
	valueType    hir.Type
	label        string
	symbolPrefix string
	valid        func(string) bool
}

func collectStaticStorage(functions []mir.Function, spec staticStorageSpec) (map[string]string, error) {
	result := make(map[string]string)
	record := func(function string, operand mir.Operand) error {
		if operand.Kind != spec.operandKind {
			return nil
		}
		if operand.Type != spec.valueType || spec.valid != nil && !spec.valid(operand.StaticData) {
			return fmt.Errorf("collect %s storage in %s: operand has invalid static storage", spec.label, function)
		}
		symbol := staticStorageSymbol(spec.symbolPrefix, operand.StaticData)
		if previous, found := result[symbol]; found && previous != operand.StaticData {
			return fmt.Errorf("collect %s storage in %s: digest collision for %s", spec.label, function, symbol)
		}
		result[symbol] = operand.StaticData
		return nil
	}
	recordAll := func(function string, operands []mir.Operand) error {
		for _, operand := range operands {
			if err := record(function, operand); err != nil {
				return err
			}
		}
		return nil
	}
	for _, function := range functions {
		for _, block := range function.Blocks {
			for _, operation := range block.Operations {
				if err := recordAll(function.Name, operation.Operands); err != nil {
					return nil, err
				}
			}
			if err := recordAll(function.Name, block.Terminator.Arguments); err != nil {
				return nil, err
			}
			if err := recordAll(function.Name, block.Terminator.SwitchOperands); err != nil {
				return nil, err
			}
			if err := record(function.Name, block.Terminator.ReturnValue); err != nil {
				return nil, err
			}
		}
	}
	return result, nil
}

func staticStorageSymbol(prefix, value string) string {
	digest := sha256.Sum256([]byte(value))
	return fmt.Sprintf("%s%x", prefix, digest)
}

func emitStaticStorage(output *strings.Builder, storage map[string]string, emptySentinel bool) {
	if len(storage) == 0 {
		return
	}
	symbols := make([]string, 0, len(storage))
	for symbol := range storage {
		symbols = append(symbols, symbol)
	}
	slices.Sort(symbols)
	for _, symbol := range symbols {
		value := storage[symbol]
		storageLength := len(value)
		if emptySentinel && storageLength == 0 {
			storageLength = 1
		}
		fmt.Fprintf(output, "static const uint8_t %s[%d] = { ", symbol, storageLength)
		if len(value) == 0 {
			output.WriteString("UINT8_C(0)")
		} else {
			for index := 0; index < len(value); index++ {
				if index != 0 {
					output.WriteString(", ")
				}
				fmt.Fprintf(output, "UINT8_C(%d)", value[index])
			}
		}
		output.WriteString(" };\n")
	}
	output.WriteByte('\n')
}

func collectStringStorage(functions []mir.Function) (map[string]string, error) {
	return collectStaticStorage(functions, staticStorageSpec{
		operandKind: mir.OperandString, valueType: hir.SharedBorrow(hir.TypeStr), label: "UTF-8 string",
		symbolPrefix: "fy_static_str_", valid: utf8.ValidString,
	})
}

func stringStorageSymbol(value string) string {
	return staticStorageSymbol("fy_static_str_", value)
}

func (e *emitter) emitStringStorage(output *strings.Builder) {
	emitStaticStorage(output, e.stringStorage, true)
}

func collectByteStringStorage(functions []mir.Function) (map[string]string, error) {
	return collectStaticStorage(functions, staticStorageSpec{
		operandKind: mir.OperandByteString, valueType: hir.SharedBorrow(hir.TypeByteSlice),
		label: "byte-string", symbolPrefix: "fy_static_bytes_",
	})
}

func byteStringStorageSymbol(value string) string {
	return staticStorageSymbol("fy_static_bytes_", value)
}

func (e *emitter) emitByteStringStorage(output *strings.Builder) {
	emitStaticStorage(output, e.byteStringStorage, true)
}

func collectCStringStorage(functions []mir.Function) (map[string]string, error) {
	return collectStaticStorage(functions, staticStorageSpec{
		operandKind: mir.OperandCString, valueType: hir.SharedBorrow(hir.TypeCStr),
		label: "C string", symbolPrefix: "fy_static_cstr_", valid: validCStringStorage,
	})
}

func validCStringStorage(value string) bool {
	return len(value) > 0 && value[len(value)-1] == 0 && !strings.Contains(value[:len(value)-1], "\x00")
}

func cStringStorageSymbol(value string) string {
	return staticStorageSymbol("fy_static_cstr_", value)
}

func (e *emitter) emitCStringStorage(output *strings.Builder) {
	emitStaticStorage(output, e.cStringStorage, false)
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
	return operator == "+" || operator == "-" || operator == "!" || operator == "~" ||
		operator == "&" || operator == "&mut" || operator == "*"
}

func supportedBinary(operator string) bool {
	switch operator {
	case "+", "-", "*", "/", "%", "<<", ">>", "&", "^", "|", "==", "!=", "<", "<=", ">", ">=":
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

func threeOperands(operands []mir.Operand) (mir.Operand, mir.Operand, mir.Operand, error) {
	if len(operands) != 3 || operands[0].Kind == mir.OperandInvalid || operands[1].Kind == mir.OperandInvalid || operands[2].Kind == mir.OperandInvalid {
		return mir.Operand{}, mir.Operand{}, mir.Operand{}, fmt.Errorf("expected three scalar operands")
	}
	return operands[0], operands[1], operands[2], nil
}

func oneSuccessor(terminator mir.Terminator) (mir.BlockID, error) {
	if len(terminator.Successors) != 1 {
		return mir.NoBlock, fmt.Errorf("expected one successor, got %d", len(terminator.Successors))
	}
	return terminator.Successors[0], nil
}
