package atlasmap

import (
	"cmp"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/0disoft/fairyc/internal/buildidentity"
	"github.com/0disoft/fairyc/internal/moduleinterface"
	"github.com/0disoft/fairyc/internal/packagecontent"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/semanticsnapshot"
	"github.com/0disoft/fairyc/internal/source"
	"github.com/0disoft/fairyc/internal/symbolidentity"
)

const maxTransportTypeDepth = 64

func validateTransportPartition(partition Partition) error {
	return validatePartitionVersion(partition, LogicalSchema)
}

func validatePartitionVersion(partition Partition, schema string) error {
	generic := schema == LogicalSchemaV2
	if !generic && len(partition.data.specializations) != 0 {
		return fmt.Errorf("Atlas map v1 cannot preserve generic specializations")
	}
	header := partition.Header()
	instanceUIDs, coordinateUIDs, err := validateTransportHeader(header, schema)
	if err != nil {
		return err
	}
	files := partition.Files()
	modules := partition.Modules()
	symbols := partition.Symbols()
	declarations := partition.Declarations()
	functions := partition.Functions()
	completeness := partition.Completeness()
	relations := countRelations(functions)
	for _, item := range partition.data.specializations {
		relations += relationCount(item.Function)
	}
	if header.Counts != (RecordCounts{
		Files: len(files), Modules: len(modules), Symbols: len(symbols), Functions: len(functions),
		Relations: relations, Completeness: len(completeness), Specializations: len(partition.data.specializations),
	}) {
		return fmt.Errorf("header record counts do not match payload")
	}
	if len(files) == 0 || len(modules) == 0 || len(declarations) == 0 || len(symbols) == 0 {
		return fmt.Errorf("transport partition has an empty required record set")
	}

	filePaths := make(map[string]struct{}, len(files))
	for index, file := range files {
		if file.PackageInstance != header.PackageInstance {
			return fmt.Errorf("file %q belongs to another package", file.Path)
		}
		if index > 0 && files[index-1].Path >= file.Path {
			return fmt.Errorf("files are not canonical")
		}
		if _, err := source.NewFile(file.Path, nil).ExpectedModule(); err != nil || strings.Contains(file.Path, "\\") {
			return fmt.Errorf("file %q has a noncanonical source path", file.Path)
		}
		fileDigest, err := packagecontent.ParseFileDigest(file.Digest)
		if err != nil {
			return fmt.Errorf("file %q digest: %w", file.Path, err)
		}
		if fileDigest == (packagecontent.FileDigest{}) {
			return fmt.Errorf("file %q digest is zero", file.Path)
		}
		lineMap, err := packagecontent.ParseLineMapID(file.LineMap)
		if err != nil {
			return fmt.Errorf("file %q line map: %w", file.Path, err)
		}
		if lineMap == (packagecontent.LineMapID{}) {
			return fmt.Errorf("file %q line map is zero", file.Path)
		}
		if file.Kind != semanticsnapshot.FileKindSource && file.Kind != semanticsnapshot.FileKindTest {
			return fmt.Errorf("file %q has invalid kind %q", file.Path, file.Kind)
		}
		filePaths[file.Path] = struct{}{}
	}

	modulePaths := make(map[string]struct{}, len(modules))
	moduleSources := make(map[string]struct{}, len(modules))
	for index, module := range modules {
		if module.PackageInstance != header.PackageInstance {
			return fmt.Errorf("module %q belongs to another package", module.Path)
		}
		if index > 0 && modules[index-1].Path >= module.Path {
			return fmt.Errorf("modules are not canonical")
		}
		expected, err := source.NewFile(module.Source, nil).ExpectedModule()
		if err != nil || expected != module.Path {
			return fmt.Errorf("module %q does not match source %q", module.Path, module.Source)
		}
		if _, found := filePaths[module.Source]; !found {
			return fmt.Errorf("module %q references unknown file %q", module.Path, module.Source)
		}
		if _, duplicate := moduleSources[module.Source]; duplicate {
			return fmt.Errorf("multiple modules reference file %q", module.Source)
		}
		modulePaths[module.Path] = struct{}{}
		moduleSources[module.Source] = struct{}{}
	}
	if len(moduleSources) != len(filePaths) {
		return fmt.Errorf("file and module records are not bijective")
	}

	declarationSet := make(map[semanticsnapshot.DeclarationKey]struct{}, len(declarations))
	for index, declaration := range declarations {
		if index > 0 && compareDeclarationKey(declarations[index-1].Key, declaration.Key) >= 0 {
			return fmt.Errorf("declarations are not canonical")
		}
		if err := validateDeclaration(declaration, header.PackageInstance, filePaths, modulePaths, generic); err != nil {
			return err
		}
		declarationSet[declaration.Key] = struct{}{}
	}

	ids := make([]symbolidentity.ID, 0, len(symbols))
	symbolDeclarations := make(map[semanticsnapshot.DeclarationKey]struct{}, len(symbols))
	for index, symbol := range symbols {
		if index > 0 && compareSymbol(symbols[index-1], symbol) >= 0 {
			return fmt.Errorf("symbols are not canonical")
		}
		if symbol.Declaration.PackageInstance != header.PackageInstance {
			return fmt.Errorf("symbol %+v belongs to another package", symbol.Declaration)
		}
		if _, found := declarationSet[symbol.Declaration]; !found {
			return fmt.Errorf("symbol %+v has no declaration", symbol.Declaration)
		}
		if _, duplicate := symbolDeclarations[symbol.Declaration]; duplicate {
			return fmt.Errorf("declaration %+v has multiple symbols", symbol.Declaration)
		}
		visible, err := symbolidentity.ParseVisibleID(symbol.Visible)
		if err != nil {
			return fmt.Errorf("symbol %+v visible identity: %w", symbol.Declaration, err)
		}
		full, err := symbolidentity.ParseFullDigest(symbol.Full)
		if err != nil {
			return fmt.Errorf("symbol %+v full identity: %w", symbol.Declaration, err)
		}
		ids = append(ids, symbolidentity.ID{Visible: visible, Full: full})
		symbolDeclarations[symbol.Declaration] = struct{}{}
	}
	if len(symbolDeclarations) != len(declarationSet) {
		return fmt.Errorf("declaration and symbol records are not bijective")
	}
	if err := symbolidentity.ValidateCollisionSet(ids); err != nil {
		return err
	}
	recomputed, err := symbolidentity.FromDeclarations(declarations, instanceUIDs, coordinateUIDs)
	if err != nil {
		return fmt.Errorf("rederive transport SymbolIds: %w", err)
	}
	if len(recomputed) != len(symbols) {
		return fmt.Errorf("rederived SymbolId count does not match payload")
	}
	for index, expected := range recomputed {
		actual := symbols[index]
		if actual.Declaration != expected.Declaration || actual.Visible != expected.ID.Visible.String() || actual.Full != expected.ID.Full.String() {
			return fmt.Errorf("symbol %+v does not match its declaration", actual.Declaration)
		}
	}

	functionSet := make(map[semanticsnapshot.DeclarationKey]struct{}, len(functions))
	for index, function := range functions {
		if index > 0 && compareDeclarationKey(functions[index-1].Function, function.Function) >= 0 {
			return fmt.Errorf("function summaries are not canonical")
		}
		if function.Function.PackageInstance != header.PackageInstance || function.Function.Kind != semanticsnapshot.DeclarationFunction {
			return fmt.Errorf("function summary %+v has invalid ownership or kind", function.Function)
		}
		if _, found := declarationSet[function.Function]; !found {
			return fmt.Errorf("function summary %+v has no declaration", function.Function)
		}
		if err := validateFunction(function, header.PackageInstance, filePaths, declarationSet, instanceUIDs, generic); err != nil {
			return err
		}
		functionSet[function.Function] = struct{}{}
	}
	for _, declaration := range declarations {
		if declaration.Key.Kind == semanticsnapshot.DeclarationFunction {
			if len(declaration.TypeParameters) != 0 {
				if _, found := functionSet[declaration.Key]; found {
					return fmt.Errorf("generic declaration has an ordinary executable summary")
				}
				continue
			}
			if _, found := functionSet[declaration.Key]; !found {
				return fmt.Errorf("source function %+v has no summary", declaration.Key)
			}
		}
	}
	if generic {
		if err := validateGenericPartition(partition, filePaths, declarationSet, instanceUIDs); err != nil {
			return err
		}
	}

	status, omissions, err := generationCompleteness(completeness)
	if err != nil {
		return err
	}
	if status != header.GenerationStatus || !slices.Equal(omissions, header.Omissions) {
		return fmt.Errorf("header generation completeness does not match records")
	}
	return nil
}

func validateTransportHeader(header Header, schema string) (map[string]string, map[string]string, error) {
	if header.GenericInterfaceAlgorithm != "" || header.GenericInterfaceHash != "" {
		if schema != LogicalSchemaV2 || header.GenericInterfaceAlgorithm != moduleinterface.GenericHashEncoding {
			return nil, nil, fmt.Errorf("generic interface algorithm is incompatible")
		}
		hash, err := moduleinterface.ParseGenericHash(header.GenericInterfaceHash)
		if err != nil || hash == (moduleinterface.GenericHash{}) {
			return nil, nil, fmt.Errorf("generic interface hash is invalid or zero")
		}
	}
	if header.LogicalSchema != schema || header.SnapshotModel != semanticsnapshot.ModelVersion {
		return nil, nil, fmt.Errorf("logical schema or snapshot model is incompatible")
	}
	if header.CompilerBuildAlgorithm != buildidentity.Encoding {
		return nil, nil, fmt.Errorf("compiler build algorithm is incompatible")
	}
	if _, err := buildidentity.Parse(header.CompilerBuild); err != nil {
		return nil, nil, err
	}
	if !validTransportText(header.Language) || !validTransportText(header.PackageUID) ||
		!validTransportText(header.PackageSourceKind) {
		return nil, nil, fmt.Errorf("header contains invalid required text")
	}
	instance, err := packagegraph.ParseInstanceID(header.PackageInstance)
	if err != nil {
		return nil, nil, err
	}
	if instance == (packagegraph.InstanceID{}) {
		return nil, nil, fmt.Errorf("header package instance is zero")
	}
	coordinate, err := packagegraph.ParseCoordinateID(header.PackageCoordinate)
	if err != nil {
		return nil, nil, err
	}
	if coordinate == (packagegraph.CoordinateID{}) {
		return nil, nil, fmt.Errorf("header package coordinate is zero")
	}
	sourceDigest, err := packagecontent.ParseDigest(header.PackageSource)
	if err != nil {
		return nil, nil, err
	}
	if sourceDigest == (packagecontent.Digest{}) {
		return nil, nil, fmt.Errorf("header package source digest is zero")
	}
	interfaceHash, err := moduleinterface.ParseHash(header.InterfaceHash)
	if err != nil {
		return nil, nil, err
	}
	if interfaceHash == (moduleinterface.Hash{}) {
		return nil, nil, fmt.Errorf("header interface hash is zero")
	}
	if header.RuntimeABI != "" && !validTransportText(header.RuntimeABI) {
		return nil, nil, fmt.Errorf("header runtime ABI is invalid")
	}
	if header.TargetScope != TargetScopeIndependent || header.Target != "" {
		return nil, nil, fmt.Errorf("transport v1 requires target-independent facts")
	}
	if header.PackageInstanceAlgorithm != packagegraph.InstanceIDEncoding ||
		header.PackageCoordinateAlgorithm != packagegraph.CoordinateIDEncoding ||
		header.PackageSourceAlgorithm != packagecontent.DigestEncoding ||
		header.SourceFileAlgorithm != packagecontent.FileDigestEncoding ||
		header.LineMapAlgorithm != packagecontent.LineMapEncoding ||
		header.SymbolAlgorithm != symbolidentity.Algorithm ||
		header.InterfaceAlgorithm != moduleinterface.HashEncoding {
		return nil, nil, fmt.Errorf("header identity algorithm is incompatible")
	}
	if err := validateSortedText("features", header.Features, false); err != nil {
		return nil, nil, err
	}
	if err := validateSortedText("dependencies", header.Dependencies, false); err != nil {
		return nil, nil, err
	}
	for _, dependency := range header.Dependencies {
		if dependency == header.PackageInstance {
			return nil, nil, fmt.Errorf("header package depends on itself")
		}
		if _, err := packagegraph.ParseInstanceID(dependency); err != nil {
			return nil, nil, fmt.Errorf("header dependency: %w", err)
		}
	}
	instanceUIDs := make(map[string]string, len(header.PackageReferences))
	coordinateUIDs := make(map[string]string, len(header.PackageReferences))
	referenceInstances := make(map[string]struct{}, len(header.PackageReferences))
	for index, reference := range header.PackageReferences {
		if index > 0 && header.PackageReferences[index-1].Instance >= reference.Instance {
			return nil, nil, fmt.Errorf("package references are not canonical")
		}
		if !validTransportText(reference.UID) {
			return nil, nil, fmt.Errorf("package reference has invalid UID")
		}
		referenceInstance, err := packagegraph.ParseInstanceID(reference.Instance)
		if err != nil {
			return nil, nil, fmt.Errorf("package reference instance: %w", err)
		}
		if referenceInstance == (packagegraph.InstanceID{}) {
			return nil, nil, fmt.Errorf("package reference instance is zero")
		}
		referenceCoordinate, err := packagegraph.ParseCoordinateID(reference.Coordinate)
		if err != nil {
			return nil, nil, fmt.Errorf("package reference coordinate: %w", err)
		}
		if referenceCoordinate == (packagegraph.CoordinateID{}) {
			return nil, nil, fmt.Errorf("package reference coordinate is zero")
		}
		if previous, duplicate := coordinateUIDs[reference.Coordinate]; duplicate && previous != reference.UID {
			return nil, nil, fmt.Errorf("package coordinate %q has conflicting UIDs", reference.Coordinate)
		}
		instanceUIDs[reference.Instance] = reference.UID
		coordinateUIDs[reference.Coordinate] = reference.UID
		referenceInstances[reference.Instance] = struct{}{}
	}
	if len(header.PackageReferences) != len(header.Dependencies)+1 ||
		instanceUIDs[header.PackageInstance] != header.PackageUID {
		return nil, nil, fmt.Errorf("package references do not contain the exact self and dependency set")
	}
	selfReferenceFound := false
	for _, reference := range header.PackageReferences {
		if reference.Instance == header.PackageInstance {
			selfReferenceFound = reference.Coordinate == header.PackageCoordinate && reference.UID == header.PackageUID
			continue
		}
		if _, directDependency := slices.BinarySearch(header.Dependencies, reference.Instance); !directDependency {
			return nil, nil, fmt.Errorf("package reference %q is not a direct dependency", reference.Instance)
		}
	}
	if !selfReferenceFound {
		return nil, nil, fmt.Errorf("package self reference contradicts header identity")
	}
	for _, dependency := range header.Dependencies {
		if _, found := referenceInstances[dependency]; !found {
			return nil, nil, fmt.Errorf("dependency %q has no package reference", dependency)
		}
	}
	if err := validateSortedText("omissions", header.Omissions, false); err != nil {
		return nil, nil, err
	}
	if header.GenerationStatus != GenerationExact && header.GenerationStatus != GenerationPartial {
		return nil, nil, fmt.Errorf("header has invalid generation status %q", header.GenerationStatus)
	}
	return instanceUIDs, coordinateUIDs, nil
}

func validateDeclaration(value semanticsnapshot.Declaration, instance string, files, modules map[string]struct{}, generic bool) error {
	if !generic && len(value.TypeParameters) != 0 {
		return fmt.Errorf("Atlas map v1 cannot preserve a generic declaration")
	}
	if err := validateLocalKey(value.Key, instance, modules); err != nil {
		return err
	}
	if value.Provenance != semanticsnapshot.ProvenanceCompilerDeclared {
		return fmt.Errorf("declaration %+v has invalid provenance %q", value.Key, value.Provenance)
	}
	if err := validateSpan(value.Span, instance, files); err != nil {
		return fmt.Errorf("declaration %+v: %w", value.Key, err)
	}
	if err := validateBoundTypeRef(value.Type, 0, len(value.TypeParameters)); err != nil {
		return fmt.Errorf("declaration %+v type: %w", value.Key, err)
	}
	if value.Owner != nil {
		if err := validateTypeRef(*value.Owner, 0); err != nil {
			return fmt.Errorf("declaration %+v owner: %w", value.Key, err)
		}
	}
	if err := validateSortedText("declaration effects", value.Effects, false); err != nil {
		return err
	}
	if err := validateSortedText("declaration drop effects", value.DropEffects, false); err != nil {
		return err
	}
	for _, parameter := range value.Parameters {
		if !source.ValidIdentifier(parameter.Name) {
			return fmt.Errorf("declaration %+v has invalid parameter %q", value.Key, parameter.Name)
		}
		if err := validateBoundTypeRef(parameter.Type, 0, len(value.TypeParameters)); err != nil {
			return err
		}
		if err := validateSpan(parameter.Span, instance, files); err != nil {
			return err
		}
	}
	if err := validateFields(value.Fields, instance, files); err != nil {
		return err
	}
	variantNames := make(map[string]struct{}, len(value.Variants))
	for _, variant := range value.Variants {
		if !source.ValidIdentifier(variant.Name) {
			return fmt.Errorf("declaration %+v has invalid variant %q", value.Key, variant.Name)
		}
		if _, duplicate := variantNames[variant.Name]; duplicate {
			return fmt.Errorf("declaration %+v repeats variant %q", value.Key, variant.Name)
		}
		variantNames[variant.Name] = struct{}{}
		if err := validateSpan(variant.Span, instance, files); err != nil {
			return err
		}
		if err := validateFields(variant.Fields, instance, files); err != nil {
			return err
		}
	}
	return nil
}

func validateFields(values []semanticsnapshot.Field, instance string, files map[string]struct{}) error {
	names := make(map[string]struct{}, len(values))
	for _, field := range values {
		if !source.ValidIdentifier(field.Name) {
			return fmt.Errorf("invalid field name %q", field.Name)
		}
		if _, duplicate := names[field.Name]; duplicate {
			return fmt.Errorf("duplicate field name %q", field.Name)
		}
		names[field.Name] = struct{}{}
		if err := validateTypeRef(field.Type, 0); err != nil {
			return err
		}
		if err := validateSpan(field.Span, instance, files); err != nil {
			return err
		}
	}
	return nil
}

func validateFunction(
	value semanticsnapshot.FunctionSummary,
	instance string,
	files map[string]struct{},
	declarations map[semanticsnapshot.DeclarationKey]struct{},
	packageInstances map[string]string,
	generic bool,
) error {
	if value.Provenance != semanticsnapshot.ProvenanceCompilerObserved {
		return fmt.Errorf("function %+v has invalid provenance %q", value.Function, value.Provenance)
	}
	if err := validateSortedText("trap categories", value.TrapCategories, false); err != nil {
		return err
	}
	for index, call := range value.Calls {
		if !generic && call.Specialization != "" {
			return fmt.Errorf("Atlas map v1 cannot preserve a specialization call")
		}
		if index > 0 && compareCall(value.Calls[index-1], call) >= 0 {
			return fmt.Errorf("function %+v calls are not canonical", value.Function)
		}
		if err := validateSpan(call.Span, instance, files); err != nil {
			return err
		}
		if err := validateCallTarget(call.Target); err != nil {
			return err
		}
		if _, found := packageInstances[call.Target.PackageInstance]; !found {
			return fmt.Errorf("call target %+v is outside the package reference closure", call.Target)
		}
		switch call.TargetKind {
		case semanticsnapshot.CallTargetResolved:
			if call.Certainty != semanticsnapshot.CertaintyExact || call.ABI != "" || call.LinkName != "" {
				return fmt.Errorf("resolved call has incompatible certainty or foreign metadata")
			}
			if call.Target.Kind != semanticsnapshot.DeclarationFunction {
				return fmt.Errorf("resolved call target %+v is not an ordinary function", call.Target)
			}
			if call.Target.PackageInstance == instance {
				if _, found := declarations[call.Target]; !found {
					return fmt.Errorf("resolved local call target %+v has no declaration", call.Target)
				}
			}
		case semanticsnapshot.CallTargetExternal:
			if call.Certainty != semanticsnapshot.CertaintyOpaque || !validTransportText(call.ABI) || !validTransportText(call.LinkName) {
				return fmt.Errorf("external call has incomplete ABI metadata")
			}
			if call.Target.PackageInstance != instance || call.Target.Kind != semanticsnapshot.DeclarationForeignFunction {
				return fmt.Errorf("external call target %+v is not a local foreign declaration", call.Target)
			}
			if _, found := declarations[call.Target]; !found {
				return fmt.Errorf("external call target %+v has no declaration", call.Target)
			}
		default:
			return fmt.Errorf("call has invalid target kind %q", call.TargetKind)
		}
	}
	for _, group := range []struct {
		name   string
		values []semanticsnapshot.Span
	}{{"unsafe", value.UnsafeSites}, {"move", value.MoveSites}, {"drop", value.DropSites}, {"allocation", value.AllocationSites}, {"return-error", value.ReturnErrorSites}} {
		if err := validateCanonicalSpans(group.name, group.values, instance, files); err != nil {
			return err
		}
	}
	for index, borrow := range value.Borrows {
		if index > 0 && compareBorrow(value.Borrows[index-1], borrow) >= 0 {
			return fmt.Errorf("function %+v borrows are not canonical", value.Function)
		}
		if borrow.Kind != "shared" && borrow.Kind != "mutable" {
			return fmt.Errorf("function %+v has invalid borrow kind %q", value.Function, borrow.Kind)
		}
		if err := validateSpan(borrow.Span, instance, files); err != nil {
			return err
		}
	}
	return nil
}

func validateTypeRef(value semanticsnapshot.TypeRef, depth int) error {
	return validateBoundTypeRef(value, depth, 0)
}

func validateBoundTypeRef(value semanticsnapshot.TypeRef, depth, arity int) error {
	if depth >= maxTransportTypeDepth {
		return fmt.Errorf("type nesting exceeds %d", maxTransportTypeDepth)
	}
	if value.Mutable && value.Kind != semanticsnapshot.TypeBorrow && value.Kind != semanticsnapshot.TypePointer {
		return fmt.Errorf("type kind %q cannot be mutable", value.Kind)
	}
	if value.Length != 0 && value.Kind != semanticsnapshot.TypeFixedArray {
		return fmt.Errorf("type kind %q cannot carry a length", value.Kind)
	}
	switch value.Kind {
	case semanticsnapshot.TypeParameter:
		index, err := strconv.Atoi(value.Name)
		if err != nil || index < 0 || index >= arity || strconv.Itoa(index) != value.Name || value.PackageCoordinate != "" || len(value.Arguments) != 0 {
			return fmt.Errorf("invalid or unbound type parameter")
		}
	case semanticsnapshot.TypeBuiltin:
		if !validTransportText(value.Name) || value.PackageCoordinate != "" || len(value.Arguments) != 0 {
			return fmt.Errorf("invalid builtin type")
		}
	case semanticsnapshot.TypeNominal:
		if !validQualifiedPath(value.Name) || len(value.Arguments) != 0 {
			return fmt.Errorf("invalid nominal type %q", value.Name)
		}
		if _, err := packagegraph.ParseCoordinateID(value.PackageCoordinate); err != nil {
			return err
		}
	case semanticsnapshot.TypeBorrow, semanticsnapshot.TypePointer, semanticsnapshot.TypeFixedArray, semanticsnapshot.TypeOption:
		if value.Name != "" || value.PackageCoordinate != "" || len(value.Arguments) != 1 {
			return fmt.Errorf("invalid unary type kind %q", value.Kind)
		}
	case semanticsnapshot.TypeResult:
		if value.Name != "" || value.PackageCoordinate != "" || len(value.Arguments) != 2 {
			return fmt.Errorf("invalid result type")
		}
	default:
		return fmt.Errorf("unknown type kind %q", value.Kind)
	}
	for _, argument := range value.Arguments {
		if err := validateBoundTypeRef(argument, depth+1, arity); err != nil {
			return err
		}
	}
	return nil
}

func validateLocalKey(value semanticsnapshot.DeclarationKey, instance string, modules map[string]struct{}) error {
	if value.PackageInstance != instance {
		return fmt.Errorf("declaration %+v belongs to another package", value)
	}
	if _, found := modules[value.Module]; !found {
		return fmt.Errorf("declaration %+v references unknown module", value)
	}
	if !validDeclarationKind(value.Kind) || !source.ValidIdentifier(value.Name) {
		return fmt.Errorf("declaration %+v has invalid kind or name", value)
	}
	return nil
}

func validateCallTarget(value semanticsnapshot.DeclarationKey) error {
	if _, err := packagegraph.ParseInstanceID(value.PackageInstance); err != nil {
		return fmt.Errorf("call target package: %w", err)
	}
	if !validQualifiedPath(value.Module) || !validDeclarationKind(value.Kind) || !source.ValidIdentifier(value.Name) {
		return fmt.Errorf("call target %+v is invalid", value)
	}
	return nil
}

func validateSpan(value semanticsnapshot.Span, instance string, files map[string]struct{}) error {
	if value.PackageInstance != instance || value.StartByte < 0 || value.EndByte < value.StartByte {
		return fmt.Errorf("span has invalid package or byte bounds")
	}
	if _, found := files[value.Source]; !found {
		return fmt.Errorf("span references unknown source %q", value.Source)
	}
	return nil
}

func validateCanonicalSpans(name string, values []semanticsnapshot.Span, instance string, files map[string]struct{}) error {
	for index, value := range values {
		if index > 0 && compareSpan(values[index-1], value) >= 0 {
			return fmt.Errorf("%s spans are not canonical", name)
		}
		if err := validateSpan(value, instance, files); err != nil {
			return err
		}
	}
	return nil
}

func validateSortedText(name string, values []string, allowEmpty bool) error {
	if !slices.IsSorted(values) {
		return fmt.Errorf("%s are not sorted", name)
	}
	for index, value := range values {
		if (!allowEmpty && !validTransportText(value)) || index > 0 && values[index-1] == value {
			return fmt.Errorf("%s contain invalid or duplicate text", name)
		}
	}
	return nil
}

func validTransportText(value string) bool {
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

func validQualifiedPath(value string) bool {
	if value == "" {
		return false
	}
	for _, segment := range strings.Split(value, "::") {
		if !source.ValidIdentifier(segment) {
			return false
		}
	}
	return true
}

func validDeclarationKind(value semanticsnapshot.DeclarationKind) bool {
	switch value {
	case semanticsnapshot.DeclarationStruct, semanticsnapshot.DeclarationEnum, semanticsnapshot.DeclarationResource,
		semanticsnapshot.DeclarationFunction, semanticsnapshot.DeclarationForeignFunction:
		return true
	default:
		return false
	}
}

func countRelations(values []semanticsnapshot.FunctionSummary) int {
	result := 0
	for _, value := range values {
		result += relationCount(value)
	}
	return result
}

func compareDeclarationKey(left, right semanticsnapshot.DeclarationKey) int {
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

func compareSpan(left, right semanticsnapshot.Span) int {
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

func compareCall(left, right semanticsnapshot.Call) int {
	if value := compareSpan(left.Span, right.Span); value != 0 {
		return value
	}
	if value := compareDeclarationKey(left.Target, right.Target); value != 0 {
		return value
	}
	return cmp.Compare(left.Specialization, right.Specialization)
}

func compareBorrow(left, right semanticsnapshot.Borrow) int {
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
