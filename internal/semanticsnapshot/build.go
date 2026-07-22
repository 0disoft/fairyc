package semanticsnapshot

import (
	"cmp"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
)

// Build validates and detaches compiler facts from one completed semantic result and its verified
// MIR. It performs no filesystem, network, cache, backend, toolchain, or source-text access.
func Build(semanticResult semantic.Result, program mir.Program, packages []PackageInput) (Snapshot, error) {
	if semanticResult.CompletedThrough != semantic.PhaseFlow || semanticResult.FailedPhase != semantic.PhaseNone ||
		len(semanticResult.Diagnostics) != 0 || len(semanticResult.Unsupported) != 0 {
		return Snapshot{}, fmt.Errorf("semantic snapshot requires a successful complete Flow result")
	}
	if failures := mir.Verify(program); len(failures) != 0 {
		return Snapshot{}, fmt.Errorf("semantic snapshot requires verified MIR: %s", failures[0])
	}

	metadata, stablePackages, files, err := normalizePackages(packages)
	if err != nil {
		return Snapshot{}, err
	}
	modules, declarations, symbols, foreignSymbols, privateSymbols, bounds, err := buildDeclarations(semanticResult.HIR, metadata)
	if err != nil {
		return Snapshot{}, err
	}
	if err := validatePackageCoverage(stablePackages, files, modules); err != nil {
		return Snapshot{}, err
	}
	functions, resolvedCalls, externalCalls, err := buildFunctionSummaries(
		program, semanticResult.Borrows, metadata, symbols, foreignSymbols, privateSymbols, bounds,
	)
	if err != nil {
		return Snapshot{}, err
	}
	completeness := buildCompleteness(resolvedCalls, externalCalls)
	return Snapshot{data: snapshotData{
		packages: stablePackages, files: files, modules: modules, declarations: declarations,
		functions: functions, completeness: completeness,
	}}, nil
}

func validatePackageCoverage(packages []Package, files []File, modules []Module) error {
	covered := make(map[string]struct{}, len(modules))
	moduleFiles := make(map[string]struct{}, len(modules))
	for _, module := range modules {
		covered[module.PackageInstance] = struct{}{}
		key := module.PackageInstance + "\x00" + module.Source
		if _, duplicate := moduleFiles[key]; duplicate {
			return fmt.Errorf("semantic snapshot source file %q has multiple typed modules", module.Source)
		}
		moduleFiles[key] = struct{}{}
	}
	for _, packageValue := range packages {
		if _, found := covered[packageValue.Instance]; !found {
			return fmt.Errorf("semantic snapshot package %s has no typed source module", packageValue.Instance)
		}
	}
	for _, file := range files {
		key := file.PackageInstance + "\x00" + file.Path
		if _, found := moduleFiles[key]; !found {
			return fmt.Errorf("semantic snapshot file %q has no typed source module", file.Path)
		}
		delete(moduleFiles, key)
	}
	if len(moduleFiles) != 0 {
		return fmt.Errorf("semantic snapshot typed module has no exact file identity")
	}
	return nil
}

type packageMetadata struct {
	input    PackageInput
	instance string
	coord    string
}

func normalizePackages(inputs []PackageInput) (map[hir.PackageID]packageMetadata, []Package, []File, error) {
	if len(inputs) == 0 {
		return nil, nil, nil, fmt.Errorf("semantic snapshot requires package metadata")
	}
	metadata := make(map[hir.PackageID]packageMetadata, len(inputs))
	instances := make(map[string]hir.PackageID, len(inputs))
	for _, input := range inputs {
		if _, duplicate := metadata[input.BuildID]; duplicate {
			return nil, nil, nil, fmt.Errorf("semantic snapshot package id %d is duplicated", input.BuildID)
		}
		if !validMetadataText(input.UID) || !validMetadataText(input.Name) || !validMetadataText(input.Version) ||
			!validMetadataText(input.Language) || !validMetadataText(input.SourceKind) || !validMetadataText(input.Role) ||
			input.RuntimeABI != "" && !validMetadataText(input.RuntimeABI) {
			return nil, nil, nil, fmt.Errorf("semantic snapshot package %d has invalid text metadata", input.BuildID)
		}
		if input.Instance == ([32]byte{}) || input.Coordinate == ([32]byte{}) || input.Source == ([32]byte{}) {
			return nil, nil, nil, fmt.Errorf("semantic snapshot package %d has an empty stable identity", input.BuildID)
		}
		if !slices.IsSorted(input.Features) {
			return nil, nil, nil, fmt.Errorf("semantic snapshot package %d features are not canonical", input.BuildID)
		}
		for index, feature := range input.Features {
			if !validMetadataText(feature) || index > 0 && input.Features[index-1] == feature {
				return nil, nil, nil, fmt.Errorf("semantic snapshot package %d features are invalid or duplicated", input.BuildID)
			}
		}
		dependencySet := make(map[hir.PackageID]struct{}, len(input.Dependencies))
		for _, dependency := range input.Dependencies {
			if dependency == input.BuildID {
				return nil, nil, nil, fmt.Errorf("semantic snapshot package %d depends on itself", input.BuildID)
			}
			if _, duplicate := dependencySet[dependency]; duplicate {
				return nil, nil, nil, fmt.Errorf("semantic snapshot package %d repeats dependency %d", input.BuildID, dependency)
			}
			dependencySet[dependency] = struct{}{}
		}
		instance := input.Instance.String()
		if previous, duplicate := instances[instance]; duplicate {
			return nil, nil, nil, fmt.Errorf("semantic snapshot packages %d and %d share PackageInstanceId %s", previous, input.BuildID, instance)
		}
		if len(input.Files) == 0 {
			return nil, nil, nil, fmt.Errorf("semantic snapshot package %d has no file identities", input.BuildID)
		}
		input.Files = slices.Clone(input.Files)
		slices.SortFunc(input.Files, func(left, right FileInput) int { return cmp.Compare(left.Path, right.Path) })
		for index, file := range input.Files {
			if err := validateFileInput(file); err != nil {
				return nil, nil, nil, fmt.Errorf("semantic snapshot package %d: %w", input.BuildID, err)
			}
			if index > 0 && input.Files[index-1].Path == file.Path {
				return nil, nil, nil, fmt.Errorf("semantic snapshot package %d repeats file %q", input.BuildID, file.Path)
			}
		}
		instances[instance] = input.BuildID
		metadata[input.BuildID] = packageMetadata{input: input, instance: instance, coord: input.Coordinate.String()}
	}

	packages := make([]Package, 0, len(inputs))
	files := make([]File, 0)
	for _, input := range inputs {
		item := metadata[input.BuildID]
		dependencies := make([]string, 0, len(input.Dependencies))
		for _, dependency := range input.Dependencies {
			dependencyMetadata, found := metadata[dependency]
			if !found {
				return nil, nil, nil, fmt.Errorf("semantic snapshot package %d references missing dependency %d", input.BuildID, dependency)
			}
			dependencies = append(dependencies, dependencyMetadata.instance)
		}
		slices.Sort(dependencies)
		packages = append(packages, Package{
			UID: input.UID, Name: input.Name, Version: input.Version, Language: input.Language,
			Instance: item.instance, Coordinate: item.coord, Source: input.Source.String(),
			SourceKind: input.SourceKind, RuntimeABI: input.RuntimeABI, Role: input.Role,
			Features: slices.Clone(input.Features), Dependencies: dependencies,
		})
		for _, file := range item.input.Files {
			files = append(files, File{
				PackageInstance: item.instance, Path: file.Path, Digest: file.Digest.String(),
				LineMap: file.LineMap.String(), Kind: file.Kind,
			})
		}
	}
	slices.SortFunc(packages, func(left, right Package) int { return cmp.Compare(left.Instance, right.Instance) })
	slices.SortFunc(files, compareFile)
	return metadata, packages, files, nil
}

func validateFileInput(file FileInput) error {
	if file.Digest == ([32]byte{}) || file.LineMap == ([32]byte{}) {
		return fmt.Errorf("file %q has an empty content or line-map identity", file.Path)
	}
	if file.Kind != FileKindSource && file.Kind != FileKindTest {
		return fmt.Errorf("file %q has invalid kind %q", file.Path, file.Kind)
	}
	if _, err := source.NewFile(file.Path, nil).ExpectedModule(); err != nil || strings.Contains(file.Path, "\\") {
		return fmt.Errorf("file %q has a noncanonical source path", file.Path)
	}
	return nil
}

func validMetadataText(value string) bool {
	if value == "" || !utf8.ValidString(value) {
		return false
	}
	for _, character := range value {
		if character < 0x20 || character == 0x7f {
			return false
		}
	}
	return true
}

type declarationBuild struct {
	modules        []Module
	declarations   []Declaration
	symbols        map[hir.SymbolID]DeclarationKey
	foreignSymbols map[hir.SymbolID]Declaration
	privateSymbols map[hir.SymbolID]struct{}
	bounds         map[DeclarationKey]Span
}

func buildDeclarations(hirPackage hir.Package, metadata map[hir.PackageID]packageMetadata) (
	[]Module,
	[]Declaration,
	map[hir.SymbolID]DeclarationKey,
	map[hir.SymbolID]Declaration,
	map[hir.SymbolID]struct{},
	map[DeclarationKey]Span,
	error,
) {
	result := declarationBuild{
		symbols: make(map[hir.SymbolID]DeclarationKey), foreignSymbols: make(map[hir.SymbolID]Declaration),
		privateSymbols: make(map[hir.SymbolID]struct{}), bounds: make(map[DeclarationKey]Span),
	}
	keys := make(map[DeclarationKey]struct{})
	moduleKeys := make(map[Module]struct{})
	for _, module := range hirPackage.Modules {
		packageInfo, found := metadata[module.Package]
		if !found {
			return nil, nil, nil, nil, nil, nil, fmt.Errorf("module %q references package %d without metadata", module.Path, module.Package)
		}
		if err := validateModule(module.Path, module.SourcePath); err != nil {
			return nil, nil, nil, nil, nil, nil, err
		}
		moduleRecord := Module{PackageInstance: packageInfo.instance, Path: module.Path, Source: module.SourcePath}
		if _, duplicate := moduleKeys[moduleRecord]; duplicate {
			return nil, nil, nil, nil, nil, nil, fmt.Errorf("semantic snapshot module %+v is duplicated", moduleRecord)
		}
		moduleKeys[moduleRecord] = struct{}{}
		result.modules = append(result.modules, moduleRecord)
		for _, structure := range module.Structs {
			declaration, err := structureDeclaration(module, structure, metadata)
			if err != nil {
				return nil, nil, nil, nil, nil, nil, err
			}
			if err := appendDeclaration(&result, keys, declaration); err != nil {
				return nil, nil, nil, nil, nil, nil, err
			}
		}
		for _, enumeration := range module.Enums {
			declaration, err := enumDeclaration(module, enumeration, metadata)
			if err != nil {
				return nil, nil, nil, nil, nil, nil, err
			}
			if err := appendDeclaration(&result, keys, declaration); err != nil {
				return nil, nil, nil, nil, nil, nil, err
			}
		}
		for _, resource := range module.Resources {
			declaration, err := resourceDeclaration(module, resource, metadata)
			if err != nil {
				return nil, nil, nil, nil, nil, nil, err
			}
			if err := appendDeclaration(&result, keys, declaration); err != nil {
				return nil, nil, nil, nil, nil, nil, err
			}
			if resource.DropSymbol != hir.NoSymbol {
				result.privateSymbols[resource.DropSymbol] = struct{}{}
			}
		}
		for _, function := range module.Functions {
			if function.DropGlueFor != "" {
				if function.Symbol == hir.NoSymbol {
					return nil, nil, nil, nil, nil, nil, fmt.Errorf("compiler-private drop function in %q has no symbol", module.Path)
				}
				result.privateSymbols[function.Symbol] = struct{}{}
				continue
			}
			declaration, err := functionDeclaration(module, function, metadata)
			if err != nil {
				return nil, nil, nil, nil, nil, nil, err
			}
			if err := appendDeclaration(&result, keys, declaration); err != nil {
				return nil, nil, nil, nil, nil, nil, err
			}
			if err := recordSymbol(result.symbols, function.Symbol, declaration.Key); err != nil {
				return nil, nil, nil, nil, nil, nil, err
			}
			result.bounds[declaration.Key] = declaration.Span
		}
		for _, function := range module.ForeignFunctions {
			declaration, err := foreignDeclaration(module, function, metadata)
			if err != nil {
				return nil, nil, nil, nil, nil, nil, err
			}
			if err := appendDeclaration(&result, keys, declaration); err != nil {
				return nil, nil, nil, nil, nil, nil, err
			}
			if err := recordSymbol(result.symbols, function.Symbol, declaration.Key); err != nil {
				return nil, nil, nil, nil, nil, nil, err
			}
			result.foreignSymbols[function.Symbol] = declaration
		}
	}
	slices.SortFunc(result.modules, compareModule)
	slices.SortFunc(result.declarations, func(left, right Declaration) int { return compareDeclarationKey(left.Key, right.Key) })
	return result.modules, result.declarations, result.symbols, result.foreignSymbols, result.privateSymbols, result.bounds, nil
}

func validateModule(modulePath, sourcePath string) error {
	if modulePath == "" || sourcePath == "" || strings.Contains(sourcePath, "\\") {
		return fmt.Errorf("semantic snapshot module has invalid path metadata")
	}
	expected, err := source.NewFile(sourcePath, nil).ExpectedModule()
	if err != nil || expected != modulePath {
		return fmt.Errorf("semantic snapshot module %q does not match source path %q", modulePath, sourcePath)
	}
	return nil
}

func appendDeclaration(result *declarationBuild, keys map[DeclarationKey]struct{}, declaration Declaration) error {
	if _, duplicate := keys[declaration.Key]; duplicate {
		return fmt.Errorf("semantic snapshot declaration %+v is duplicated", declaration.Key)
	}
	keys[declaration.Key] = struct{}{}
	result.declarations = append(result.declarations, declaration)
	return nil
}

func recordSymbol(symbols map[hir.SymbolID]DeclarationKey, symbol hir.SymbolID, key DeclarationKey) error {
	if symbol == hir.NoSymbol {
		return fmt.Errorf("semantic snapshot declaration %+v has no HIR symbol", key)
	}
	if previous, duplicate := symbols[symbol]; duplicate {
		return fmt.Errorf("semantic snapshot declarations %+v and %+v share HIR symbol %d", previous, key, symbol)
	}
	symbols[symbol] = key
	return nil
}

func declarationBase(module hir.Module, kind DeclarationKind, name string, span hir.Span, metadata map[hir.PackageID]packageMetadata) (Declaration, error) {
	packageInfo, found := metadata[module.Package]
	if !found {
		return Declaration{}, fmt.Errorf("declaration %q references package %d without metadata", name, module.Package)
	}
	stableSpan, err := normalizeSpan(span, metadata)
	if err != nil {
		return Declaration{}, err
	}
	return Declaration{
		Key:  DeclarationKey{PackageInstance: packageInfo.instance, Module: module.Path, Kind: kind, Name: name},
		Span: stableSpan, Provenance: ProvenanceCompilerDeclared,
	}, nil
}

func structureDeclaration(module hir.Module, value hir.Struct, metadata map[hir.PackageID]packageMetadata) (Declaration, error) {
	result, err := declarationBase(module, DeclarationStruct, value.Name, value.Span, metadata)
	if err != nil {
		return Declaration{}, err
	}
	result.Public = value.Public
	result.ReprC = value.ReprC
	result.Type, err = normalizeType(value.Type, metadata)
	if err != nil {
		return Declaration{}, err
	}
	result.Fields, err = normalizeStructFields(value.Fields, metadata)
	return result, err
}

func enumDeclaration(module hir.Module, value hir.Enum, metadata map[hir.PackageID]packageMetadata) (Declaration, error) {
	result, err := declarationBase(module, DeclarationEnum, value.Name, value.Span, metadata)
	if err != nil {
		return Declaration{}, err
	}
	result.Public = value.Public
	result.Type, err = normalizeType(value.Type, metadata)
	if err != nil {
		return Declaration{}, err
	}
	for _, variant := range value.Variants {
		span, spanErr := normalizeSpan(variant.Span, metadata)
		if spanErr != nil {
			return Declaration{}, spanErr
		}
		item := Variant{Name: variant.Name, Span: span}
		for _, field := range variant.Fields {
			normalized, fieldErr := normalizeField(field.Name, field.Type, field.Span, metadata)
			if fieldErr != nil {
				return Declaration{}, fieldErr
			}
			item.Fields = append(item.Fields, normalized)
		}
		result.Variants = append(result.Variants, item)
	}
	return result, nil
}

func resourceDeclaration(module hir.Module, value hir.Resource, metadata map[hir.PackageID]packageMetadata) (Declaration, error) {
	result, err := declarationBase(module, DeclarationResource, value.Name, value.Span, metadata)
	if err != nil {
		return Declaration{}, err
	}
	result.Public = value.Public
	result.DropKind = string(value.DropKind)
	result.Type, err = normalizeType(value.Type, metadata)
	if err != nil {
		return Declaration{}, err
	}
	for _, effect := range value.DropEffects {
		result.DropEffects = append(result.DropEffects, string(effect))
	}
	if !slices.IsSorted(result.DropEffects) {
		return Declaration{}, fmt.Errorf("resource %q drop effects are not canonical", value.Name)
	}
	for _, field := range value.Fields {
		normalized, fieldErr := normalizeField(field.Name, field.Type, field.Span, metadata)
		if fieldErr != nil {
			return Declaration{}, fieldErr
		}
		result.Fields = append(result.Fields, normalized)
	}
	return result, nil
}

func functionDeclaration(module hir.Module, value hir.Function, metadata map[hir.PackageID]packageMetadata) (Declaration, error) {
	name := value.Name
	var owner *TypeRef
	if value.OwnerType != "" {
		normalized, err := normalizeType(value.OwnerType, metadata)
		if err != nil {
			return Declaration{}, err
		}
		owner = &normalized
		ownerName := normalized.Name
		prefix := module.Path + "::"
		if normalized.Kind == TypeNominal && strings.HasPrefix(ownerName, prefix) {
			ownerName = strings.TrimPrefix(ownerName, prefix)
		}
		name = ownerName + "::" + value.Name
	}
	result, err := declarationBase(module, DeclarationFunction, name, value.Span, metadata)
	if err != nil {
		return Declaration{}, err
	}
	result.Public = value.Public
	result.Unsafe = value.Unsafe
	result.Owner = owner
	result.Receiver = value.Receiver
	result.Type, err = normalizeType(value.ResultType, metadata)
	if err != nil {
		return Declaration{}, err
	}
	result.Parameters, err = normalizeParameters(value.Parameters, metadata)
	if err != nil {
		return Declaration{}, err
	}
	for _, effect := range value.Effects {
		result.Effects = append(result.Effects, string(effect))
	}
	if !slices.IsSorted(result.Effects) {
		return Declaration{}, fmt.Errorf("function %q effects are not canonical", value.Name)
	}
	return result, nil
}

func foreignDeclaration(module hir.Module, value hir.ForeignFunction, metadata map[hir.PackageID]packageMetadata) (Declaration, error) {
	result, err := declarationBase(module, DeclarationForeignFunction, value.Name, value.Span, metadata)
	if err != nil {
		return Declaration{}, err
	}
	result.Public = value.Public
	result.Unsafe = true
	result.Foreign = true
	result.ABI = value.ABI
	result.LinkName = value.LinkName
	result.Type, err = normalizeType(value.ResultType, metadata)
	if err != nil {
		return Declaration{}, err
	}
	for _, parameter := range value.Parameters {
		typeRef, typeErr := normalizeType(parameter.Type, metadata)
		if typeErr != nil {
			return Declaration{}, typeErr
		}
		span, spanErr := normalizeSpan(parameter.Span, metadata)
		if spanErr != nil {
			return Declaration{}, spanErr
		}
		result.Parameters = append(result.Parameters, Parameter{Name: parameter.Name, Type: typeRef, Span: span})
	}
	for _, effect := range value.Effects {
		result.Effects = append(result.Effects, string(effect))
	}
	if !slices.IsSorted(result.Effects) {
		return Declaration{}, fmt.Errorf("foreign function %q effects are not canonical", value.Name)
	}
	return result, nil
}

func normalizeStructFields(values []hir.StructField, metadata map[hir.PackageID]packageMetadata) ([]Field, error) {
	result := make([]Field, 0, len(values))
	for _, value := range values {
		field, err := normalizeField(value.Name, value.Type, value.Span, metadata)
		if err != nil {
			return nil, err
		}
		result = append(result, field)
	}
	return result, nil
}

func normalizeField(name string, valueType hir.Type, valueSpan hir.Span, metadata map[hir.PackageID]packageMetadata) (Field, error) {
	typeRef, err := normalizeType(valueType, metadata)
	if err != nil {
		return Field{}, err
	}
	span, err := normalizeSpan(valueSpan, metadata)
	if err != nil {
		return Field{}, err
	}
	return Field{Name: name, Type: typeRef, Span: span}, nil
}

func normalizeParameters(values []hir.Parameter, metadata map[hir.PackageID]packageMetadata) ([]Parameter, error) {
	result := make([]Parameter, 0, len(values))
	for _, value := range values {
		typeRef, err := normalizeType(value.Type, metadata)
		if err != nil {
			return nil, err
		}
		span, err := normalizeSpan(value.Span, metadata)
		if err != nil {
			return nil, err
		}
		result = append(result, Parameter{Name: value.Name, Type: typeRef, Mutable: value.Mutable, Span: span})
	}
	return result, nil
}

func normalizeSpan(value hir.Span, metadata map[hir.PackageID]packageMetadata) (Span, error) {
	packageInfo, found := metadata[value.Package]
	if !found {
		return Span{}, fmt.Errorf("source span references package %d without metadata", value.Package)
	}
	if value.StartByte < 0 || value.EndByte < value.StartByte || value.Source == "" || strings.Contains(value.Source, "\\") {
		return Span{}, fmt.Errorf("source span in package %d has invalid coordinates", value.Package)
	}
	if _, err := source.NewFile(value.Source, nil).ExpectedModule(); err != nil {
		return Span{}, fmt.Errorf("source span path %q is not canonical: %w", value.Source, err)
	}
	return Span{PackageInstance: packageInfo.instance, Source: value.Source, StartByte: value.StartByte, EndByte: value.EndByte}, nil
}

func normalizeType(value hir.Type, metadata map[hir.PackageID]packageMetadata) (TypeRef, error) {
	if builtinType(value) {
		return TypeRef{Kind: TypeBuiltin, Name: string(value)}, nil
	}
	if element, mutable, ok := hir.BorrowElement(value); ok {
		argument, err := normalizeType(element, metadata)
		return TypeRef{Kind: TypeBorrow, Mutable: mutable, Arguments: []TypeRef{argument}}, err
	}
	if element, mutable, ok := hir.RawPointerElement(value); ok {
		argument, err := normalizeType(element, metadata)
		return TypeRef{Kind: TypePointer, Mutable: mutable, Arguments: []TypeRef{argument}}, err
	}
	if element, length, ok := hir.FixedArrayParts(value); ok {
		argument, err := normalizeType(element, metadata)
		return TypeRef{Kind: TypeFixedArray, Length: length, Arguments: []TypeRef{argument}}, err
	}
	if element, ok := hir.OptionElement(value); ok {
		argument, err := normalizeType(element, metadata)
		return TypeRef{Kind: TypeOption, Arguments: []TypeRef{argument}}, err
	}
	if success, failure, ok := hir.ResultElements(value); ok {
		left, err := normalizeType(success, metadata)
		if err != nil {
			return TypeRef{}, err
		}
		right, err := normalizeType(failure, metadata)
		return TypeRef{Kind: TypeResult, Arguments: []TypeRef{left, right}}, err
	}
	packageID, logicalName, ok := nominalTypeParts(value)
	if !ok {
		return TypeRef{}, fmt.Errorf("semantic snapshot cannot normalize type %q", value)
	}
	packageInfo, found := metadata[packageID]
	if !found {
		return TypeRef{}, fmt.Errorf("nominal type %q references package %d without metadata", value, packageID)
	}
	for _, segment := range strings.Split(logicalName, "::") {
		if !source.ValidIdentifier(segment) {
			return TypeRef{}, fmt.Errorf("nominal type %q has an invalid logical path", value)
		}
	}
	return TypeRef{Kind: TypeNominal, Name: logicalName, PackageCoordinate: packageInfo.coord}, nil
}

func nominalTypeParts(value hir.Type) (hir.PackageID, string, bool) {
	text := string(value)
	if strings.HasPrefix(text, "$p") {
		separator := strings.Index(text, "::")
		if separator <= 2 || separator+2 >= len(text) {
			return 0, "", false
		}
		parsed, err := strconv.ParseUint(text[2:separator], 10, 32)
		if err != nil || parsed == 0 {
			return 0, "", false
		}
		return hir.PackageID(parsed), text[separator+2:], true
	}
	if !strings.Contains(text, "::") {
		return 0, "", false
	}
	return hir.DefaultPackage, text, true
}

func builtinType(value hir.Type) bool {
	return value == hir.TypeVoid || value == hir.TypeBool || value == hir.TypeU8 || value == hir.TypeI32 ||
		value == hir.TypeU32 || value == hir.TypeUSize || value == hir.TypeByteSlice || value == hir.TypeStr ||
		value == hir.TypeCStr || value == hir.TypeRegion || hir.IsCScalar(value)
}

func buildFunctionSummaries(
	program mir.Program,
	borrowFacts []semantic.BorrowFact,
	metadata map[hir.PackageID]packageMetadata,
	symbols map[hir.SymbolID]DeclarationKey,
	foreignSymbols map[hir.SymbolID]Declaration,
	privateSymbols map[hir.SymbolID]struct{},
	bounds map[DeclarationKey]Span,
) ([]FunctionSummary, int, int, error) {
	result := make([]FunctionSummary, 0, len(program.Functions))
	indexes := make(map[DeclarationKey]int)
	resolvedCalls := 0
	externalCalls := 0
	for _, function := range program.Functions {
		if function.DropGlueFor != "" {
			if _, known := privateSymbols[function.Symbol]; !known {
				return nil, 0, 0, fmt.Errorf("MIR drop function %q has no HIR private-symbol owner", function.Name)
			}
			continue
		}
		key, found := symbols[function.Symbol]
		if !found {
			return nil, 0, 0, fmt.Errorf("MIR function %q has no source declaration", function.Name)
		}
		if key.Kind != DeclarationFunction {
			return nil, 0, 0, fmt.Errorf("MIR function %q resolves to non-function declaration %+v", function.Name, key)
		}
		if _, duplicate := indexes[key]; duplicate {
			return nil, 0, 0, fmt.Errorf("MIR function summary %+v is duplicated", key)
		}
		summary := FunctionSummary{Function: key, Provenance: ProvenanceCompilerObserved}
		entitySpans := make(map[string]hir.Span)
		trapSet := make(map[string]struct{})
		for _, block := range function.Blocks {
			for _, operation := range block.Operations {
				if err := recordEntitySpan(entitySpans, operation.Entity, operation.Span); err != nil {
					return nil, 0, 0, err
				}
				for _, category := range operation.TrapCategories {
					trapSet[category] = struct{}{}
				}
				switch operation.Kind {
				case mir.OperationDrop:
					span, err := normalizeSpan(operation.Span, metadata)
					if err != nil {
						return nil, 0, 0, err
					}
					summary.DropSites = append(summary.DropSites, span)
				case mir.OperationRegionPlace:
					span, err := normalizeSpan(operation.Span, metadata)
					if err != nil {
						return nil, 0, 0, err
					}
					summary.AllocationSites = append(summary.AllocationSites, span)
				}
			}
			terminator := block.Terminator
			if err := recordEntitySpan(entitySpans, terminator.Entity, terminator.Span); err != nil {
				return nil, 0, 0, err
			}
			for _, category := range terminator.TrapCategories {
				trapSet[category] = struct{}{}
			}
			if terminator.Kind != mir.TerminatorCall {
				continue
			}
			span, err := normalizeSpan(terminator.Span, metadata)
			if err != nil {
				return nil, 0, 0, err
			}
			target, targetFound := symbols[terminator.Callee]
			if !targetFound {
				return nil, 0, 0, fmt.Errorf("MIR call in %q has no source target for symbol %d", function.Name, terminator.Callee)
			}
			call := Call{Span: span, Target: target}
			if foreign, isForeign := foreignSymbols[terminator.Callee]; isForeign {
				call.TargetKind = CallTargetExternal
				call.Certainty = CertaintyOpaque
				call.ABI = foreign.ABI
				call.LinkName = foreign.LinkName
				externalCalls++
			} else {
				call.TargetKind = CallTargetResolved
				call.Certainty = CertaintyExact
				resolvedCalls++
			}
			summary.Calls = append(summary.Calls, call)
		}
		for _, path := range function.TrapPaths {
			for _, category := range path.Categories {
				trapSet[category] = struct{}{}
			}
		}
		for category := range trapSet {
			if !validMetadataText(category) {
				return nil, 0, 0, fmt.Errorf("MIR function %q has an invalid trap category", function.Name)
			}
			summary.TrapCategories = append(summary.TrapCategories, category)
		}
		slices.Sort(summary.TrapCategories)
		for _, unsafeOperation := range function.UnsafeOperations {
			span, found := entitySpans[unsafeOperation.Entity]
			if !found {
				return nil, 0, 0, fmt.Errorf("MIR unsafe operation %q has no matching executable entity", unsafeOperation.Entity)
			}
			normalized, err := normalizeSpan(span, metadata)
			if err != nil {
				return nil, 0, 0, err
			}
			summary.UnsafeSites = append(summary.UnsafeSites, normalized)
		}
		for _, move := range function.Moves {
			span, err := normalizeSpan(move.Operand, metadata)
			if err != nil {
				return nil, 0, 0, err
			}
			summary.MoveSites = append(summary.MoveSites, span)
		}
		for _, branch := range function.ResultBranches {
			span, err := normalizeSpan(branch.Question, metadata)
			if err != nil {
				return nil, 0, 0, err
			}
			summary.ReturnErrorSites = append(summary.ReturnErrorSites, span)
		}
		slices.SortFunc(summary.Calls, compareCall)
		summary.UnsafeSites = canonicalSpans(summary.UnsafeSites)
		summary.MoveSites = canonicalSpans(summary.MoveSites)
		summary.DropSites = canonicalSpans(summary.DropSites)
		summary.AllocationSites = canonicalSpans(summary.AllocationSites)
		summary.ReturnErrorSites = canonicalSpans(summary.ReturnErrorSites)
		indexes[key] = len(result)
		result = append(result, summary)
	}

	for _, fact := range borrowFacts {
		span, err := normalizeSpan(fact.Borrow, metadata)
		if err != nil {
			return nil, 0, 0, err
		}
		owner := -1
		for key, index := range indexes {
			if containsSpan(bounds[key], span) {
				if owner != -1 {
					return nil, 0, 0, fmt.Errorf("borrow at %+v belongs to multiple functions", span)
				}
				owner = index
			}
		}
		if owner == -1 {
			return nil, 0, 0, fmt.Errorf("borrow at %+v belongs to no source function", span)
		}
		if fact.Kind != "shared" && fact.Kind != "mutable" {
			return nil, 0, 0, fmt.Errorf("borrow at %+v has unknown kind %q", span, fact.Kind)
		}
		result[owner].Borrows = append(result[owner].Borrows, Borrow{Span: span, Kind: fact.Kind, Escapes: fact.Escapes})
	}
	for index := range result {
		slices.SortFunc(result[index].Borrows, compareBorrow)
	}
	if len(indexes) != len(bounds) {
		for key := range bounds {
			if _, found := indexes[key]; !found {
				return nil, 0, 0, fmt.Errorf("source function %+v has no verified MIR summary", key)
			}
		}
		return nil, 0, 0, fmt.Errorf("semantic snapshot function-summary cardinality does not match source declarations")
	}
	slices.SortFunc(result, func(left, right FunctionSummary) int { return compareDeclarationKey(left.Function, right.Function) })
	return result, resolvedCalls, externalCalls, nil
}

func recordEntitySpan(values map[string]hir.Span, entity string, span hir.Span) error {
	if entity == "" {
		return nil
	}
	if previous, found := values[entity]; found && previous != span {
		return fmt.Errorf("MIR entity %q has conflicting source spans", entity)
	}
	values[entity] = span
	return nil
}

func containsSpan(container, value Span) bool {
	return container.PackageInstance == value.PackageInstance && container.Source == value.Source &&
		container.StartByte <= value.StartByte && container.EndByte >= value.EndByte
}

func canonicalSpans(values []Span) []Span {
	slices.SortFunc(values, compareSpan)
	return slices.Compact(values)
}

func buildCompleteness(resolvedCalls, externalCalls int) []Completeness {
	calls := Completeness{Relation: "calls", Status: CompletenessExact, Resolved: resolvedCalls, External: externalCalls}
	if externalCalls != 0 {
		calls.Status = CompletenessPartial
		calls.Reasons = []string{"external-ffi"}
	}
	result := []Completeness{
		{Relation: "allocates", Status: CompletenessExact},
		{Relation: "borrows", Status: CompletenessExact},
		calls,
		{Relation: "drops", Status: CompletenessExact},
		{Relation: "effects", Status: CompletenessExact},
		{Relation: "may-trap", Status: CompletenessExact},
		{Relation: "moves", Status: CompletenessExact},
		{Relation: "reads", Status: CompletenessUnavailable, Reasons: []string{"not-implemented"}},
		{Relation: "returns-error", Status: CompletenessExact},
		{Relation: "test-calls", Status: CompletenessUnavailable, Reasons: []string{"test-metadata-not-retained"}},
		{Relation: "uses-unsafe", Status: CompletenessExact},
		{Relation: "writes", Status: CompletenessUnavailable, Reasons: []string{"not-implemented"}},
	}
	slices.SortFunc(result, func(left, right Completeness) int { return cmp.Compare(left.Relation, right.Relation) })
	return result
}

func compareDeclarationKey(left, right DeclarationKey) int {
	if value := cmp.Compare(left.PackageInstance, right.PackageInstance); value != 0 {
		return value
	}
	if value := cmp.Compare(left.Module, right.Module); value != 0 {
		return value
	}
	if value := cmp.Compare(left.Kind, right.Kind); value != 0 {
		return value
	}
	return cmp.Compare(left.Name, right.Name)
}

func compareModule(left, right Module) int {
	if value := cmp.Compare(left.PackageInstance, right.PackageInstance); value != 0 {
		return value
	}
	return cmp.Compare(left.Path, right.Path)
}

func compareFile(left, right File) int {
	if value := cmp.Compare(left.PackageInstance, right.PackageInstance); value != 0 {
		return value
	}
	return cmp.Compare(left.Path, right.Path)
}

func compareSpan(left, right Span) int {
	if value := cmp.Compare(left.PackageInstance, right.PackageInstance); value != 0 {
		return value
	}
	if value := cmp.Compare(left.Source, right.Source); value != 0 {
		return value
	}
	if value := cmp.Compare(left.StartByte, right.StartByte); value != 0 {
		return value
	}
	return cmp.Compare(left.EndByte, right.EndByte)
}

func compareCall(left, right Call) int {
	if value := compareSpan(left.Span, right.Span); value != 0 {
		return value
	}
	return compareDeclarationKey(left.Target, right.Target)
}

func compareBorrow(left, right Borrow) int {
	if value := compareSpan(left.Span, right.Span); value != 0 {
		return value
	}
	if value := cmp.Compare(left.Kind, right.Kind); value != 0 {
		return value
	}
	if left.Escapes == right.Escapes {
		return 0
	}
	if !left.Escapes {
		return -1
	}
	return 1
}
