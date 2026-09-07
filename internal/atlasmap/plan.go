// Package atlasmap owns deterministic .fymap planning, transport, contained storage, current-source
// validation, and bounded package-local queries. The logical planner itself performs none of the
// filesystem or query operations owned by the later layers in this package.
package atlasmap

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	"github.com/0disoft/fairyc/internal/buildidentity"
	"github.com/0disoft/fairyc/internal/moduleinterface"
	"github.com/0disoft/fairyc/internal/packagecontent"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/semanticsnapshot"
	"github.com/0disoft/fairyc/internal/symbolidentity"
)

const (
	LogicalSchema          = "fairyc.fymap-logical/v1"
	LogicalSchemaV2        = "fairyc.fymap-logical/v2"
	TargetScopeIndependent = "target-independent"
	GenerationExact        = "exact"
	GenerationPartial      = "partial"
)

// Input supplies compatibility facts intentionally absent from SemanticSnapshot.
type Input struct {
	CompilerBuild     buildidentity.Identity
	Interfaces        map[string]moduleinterface.Hash
	GenericInterfaces map[string]moduleinterface.GenericHash
}

type RecordCounts struct {
	Files           int
	Modules         int
	Symbols         int
	Functions       int
	Specializations int
	Relations       int
	Completeness    int
}

// PackageReference supplies the stable namespace mapping required to rederive declaration IDs.
type PackageReference struct {
	UID        string
	Instance   string
	Coordinate string
}

// Header is transport-neutral compatibility and record-count data for one exact package map.
type Header struct {
	LogicalSchema              string
	SnapshotModel              string
	CompilerBuild              string
	CompilerBuildAlgorithm     string
	Language                   string
	PackageUID                 string
	PackageInstance            string
	PackageCoordinate          string
	PackageSource              string
	PackageSourceKind          string
	RuntimeABI                 string
	TargetScope                string
	Target                     string
	PackageInstanceAlgorithm   string
	PackageCoordinateAlgorithm string
	PackageSourceAlgorithm     string
	SourceFileAlgorithm        string
	LineMapAlgorithm           string
	SymbolAlgorithm            string
	InterfaceAlgorithm         string
	InterfaceHash              string
	GenericInterfaceAlgorithm  string
	GenericInterfaceHash       string
	Features                   []string
	Dependencies               []string
	PackageReferences          []PackageReference
	GenerationStatus           string
	Omissions                  []string
	Counts                     RecordCounts
}

// Symbol binds one snapshot declaration key to both collision-checking SymbolId forms.
type Symbol struct {
	Declaration semanticsnapshot.DeclarationKey
	Visible     string
	Full        string
}

type partitionData struct {
	header          Header
	files           []semanticsnapshot.File
	modules         []semanticsnapshot.Module
	symbols         []Symbol
	declarations    []semanticsnapshot.Declaration
	functions       []semanticsnapshot.FunctionSummary
	specializations []semanticsnapshot.SpecializationSummary
	completeness    []semanticsnapshot.Completeness
}

// Partition is an immutable assignment plan for one future whole-package map.
type Partition struct {
	data partitionData
}

func (partition Partition) Header() Header {
	result := partition.data.header
	result.Features = slices.Clone(result.Features)
	result.Dependencies = slices.Clone(result.Dependencies)
	result.PackageReferences = slices.Clone(result.PackageReferences)
	result.Omissions = slices.Clone(result.Omissions)
	return result
}

func (partition Partition) Files() []semanticsnapshot.File {
	return slices.Clone(partition.data.files)
}

func (partition Partition) Modules() []semanticsnapshot.Module {
	return slices.Clone(partition.data.modules)
}

func (partition Partition) Symbols() []Symbol {
	return slices.Clone(partition.data.symbols)
}

func (partition Partition) Declarations() []semanticsnapshot.Declaration {
	return cloneDeclarations(partition.data.declarations)
}

func (partition Partition) Functions() []semanticsnapshot.FunctionSummary {
	return cloneFunctionSummaries(partition.data.functions)
}

func (partition Partition) Completeness() []semanticsnapshot.Completeness {
	result := slices.Clone(partition.data.completeness)
	for index := range result {
		result[index].Reasons = slices.Clone(result[index].Reasons)
	}
	return result
}

type partitionBuilder struct {
	packageValue semanticsnapshot.Package
	data         partitionData
}

// BuildPlan validates every compatibility input and assigns every snapshot record to exactly one
// whole-package logical partition. It performs no filesystem, source, executable, target, or map I/O.
func BuildPlan(snapshot semanticsnapshot.Snapshot, input Input) ([]Partition, error) {
	return buildPlan(snapshot, input, LogicalSchema)
}

// BuildPlanV2 retains original generic declarations and separate concrete summaries.
// It is a logical planning boundary; the v1 codec cannot encode these partitions.
func BuildPlanV2(snapshot semanticsnapshot.Snapshot, input Input) ([]Partition, error) {
	return buildPlan(snapshot, input, LogicalSchemaV2)
}

func buildPlan(snapshot semanticsnapshot.Snapshot, input Input, schema string) ([]Partition, error) {
	if schema == LogicalSchema && len(snapshot.Specializations()) != 0 {
		return nil, fmt.Errorf("Atlas map v1 cannot preserve generic specializations")
	}
	if snapshot.Version() != semanticsnapshot.ModelVersion {
		return nil, fmt.Errorf("Atlas map plan requires snapshot model %q", semanticsnapshot.ModelVersion)
	}
	if input.CompilerBuild == ([32]byte{}) {
		return nil, fmt.Errorf("Atlas map plan requires exact compiler build identity")
	}
	packages := snapshot.Packages()
	if len(packages) == 0 {
		return nil, fmt.Errorf("Atlas map plan requires at least one package")
	}
	if len(input.Interfaces) != len(packages) {
		return nil, fmt.Errorf("Atlas map plan requires one public-interface hash per package")
	}
	hasGeneric := len(snapshot.Specializations()) != 0
	for _, declaration := range snapshot.Declarations() {
		hasGeneric = hasGeneric || len(declaration.TypeParameters) != 0
	}
	if schema == LogicalSchema && len(input.GenericInterfaces) != 0 {
		return nil, fmt.Errorf("Atlas map v1 cannot preserve generic interface identities")
	}
	if schema == LogicalSchemaV2 && (hasGeneric || len(input.GenericInterfaces) != 0) && len(input.GenericInterfaces) != len(packages) {
		return nil, fmt.Errorf("Atlas map plan requires one generic source interface per package")
	}

	builders := make(map[string]*partitionBuilder, len(packages))
	for _, packageValue := range packages {
		if err := validatePackage(packageValue); err != nil {
			return nil, err
		}
		if _, duplicate := builders[packageValue.Instance]; duplicate {
			return nil, fmt.Errorf("Atlas map plan repeats package instance %q", packageValue.Instance)
		}
		if len(input.GenericInterfaces) != 0 && input.GenericInterfaces[packageValue.Instance] == (moduleinterface.GenericHash{}) {
			return nil, fmt.Errorf("package %q has no exact generic source interface", packageValue.Instance)
		}
		interfaceHash, found := input.Interfaces[packageValue.Instance]
		if !found || interfaceHash == ([32]byte{}) {
			return nil, fmt.Errorf("package %q has no exact public-interface hash", packageValue.Instance)
		}
		builders[packageValue.Instance] = &partitionBuilder{packageValue: packageValue, data: partitionData{header: Header{
			LogicalSchema: schema, SnapshotModel: snapshot.Version(), CompilerBuild: input.CompilerBuild.String(),
			CompilerBuildAlgorithm: buildidentity.Encoding,
			Language:               packageValue.Language, PackageUID: packageValue.UID, PackageInstance: packageValue.Instance,
			PackageCoordinate: packageValue.Coordinate, PackageSource: packageValue.Source,
			PackageSourceKind: packageValue.SourceKind, RuntimeABI: packageValue.RuntimeABI,
			TargetScope:                TargetScopeIndependent,
			PackageInstanceAlgorithm:   packagegraph.InstanceIDEncoding,
			PackageCoordinateAlgorithm: packagegraph.CoordinateIDEncoding,
			PackageSourceAlgorithm:     packagecontent.DigestEncoding,
			SourceFileAlgorithm:        packagecontent.FileDigestEncoding, LineMapAlgorithm: packagecontent.LineMapEncoding,
			SymbolAlgorithm: symbolidentity.Algorithm, InterfaceAlgorithm: moduleinterface.HashEncoding,
			InterfaceHash: interfaceHash.String(), Features: slices.Clone(packageValue.Features),
			Dependencies: slices.Clone(packageValue.Dependencies),
		}}}
	}
	for instance := range input.Interfaces {
		if _, found := builders[instance]; !found {
			return nil, fmt.Errorf("public-interface hash references unknown package %q", instance)
		}
	}
	for _, builder := range builders {
		for _, dependency := range builder.packageValue.Dependencies {
			if _, found := builders[dependency]; !found {
				return nil, fmt.Errorf("package %q references unknown dependency %q", builder.packageValue.Instance, dependency)
			}
		}
		references := make([]PackageReference, 0, len(builder.packageValue.Dependencies)+1)
		instances := append([]string{builder.packageValue.Instance}, builder.packageValue.Dependencies...)
		for _, instance := range instances {
			packageValue := builders[instance].packageValue
			references = append(references, PackageReference{UID: packageValue.UID, Instance: packageValue.Instance, Coordinate: packageValue.Coordinate})
		}
		slices.SortFunc(references, func(left, right PackageReference) int { return cmp.Compare(left.Instance, right.Instance) })
		builder.data.header.PackageReferences = references
	}

	for _, file := range snapshot.Files() {
		builder, found := builders[file.PackageInstance]
		if !found {
			return nil, fmt.Errorf("file %q references unknown package %q", file.Path, file.PackageInstance)
		}
		if _, err := packagecontent.ParseFileDigest(file.Digest); err != nil {
			return nil, fmt.Errorf("file %q content identity: %w", file.Path, err)
		}
		if _, err := packagecontent.ParseLineMapID(file.LineMap); err != nil {
			return nil, fmt.Errorf("file %q line-map identity: %w", file.Path, err)
		}
		builder.data.files = append(builder.data.files, file)
	}
	for _, module := range snapshot.Modules() {
		builder, found := builders[module.PackageInstance]
		if !found {
			return nil, fmt.Errorf("module %q references unknown package %q", module.Path, module.PackageInstance)
		}
		builder.data.modules = append(builder.data.modules, module)
	}

	identitySymbols, err := symbolidentity.FromSnapshot(snapshot)
	if err != nil {
		return nil, err
	}
	symbolsByDeclaration := make(map[semanticsnapshot.DeclarationKey]Symbol, len(identitySymbols))
	for _, value := range identitySymbols {
		symbol := Symbol{Declaration: value.Declaration, Visible: value.ID.Visible.String(), Full: value.ID.Full.String()}
		if _, duplicate := symbolsByDeclaration[value.Declaration]; duplicate {
			return nil, fmt.Errorf("Atlas map plan repeats symbol declaration %+v", value.Declaration)
		}
		symbolsByDeclaration[value.Declaration] = symbol
	}
	for _, declaration := range snapshot.Declarations() {
		if schema == LogicalSchema && len(declaration.TypeParameters) != 0 {
			return nil, fmt.Errorf("Atlas map v1 cannot preserve a generic declaration")
		}
		builder, found := builders[declaration.Key.PackageInstance]
		if !found {
			return nil, fmt.Errorf("declaration %+v references an unknown package", declaration.Key)
		}
		symbol, found := symbolsByDeclaration[declaration.Key]
		if !found {
			return nil, fmt.Errorf("declaration %+v has no SymbolId", declaration.Key)
		}
		builder.data.declarations = append(builder.data.declarations, declaration)
		builder.data.symbols = append(builder.data.symbols, symbol)
		delete(symbolsByDeclaration, declaration.Key)
	}
	if len(symbolsByDeclaration) != 0 {
		return nil, fmt.Errorf("Atlas map plan has SymbolId records without declarations")
	}
	for _, function := range snapshot.Functions() {
		builder, found := builders[function.Function.PackageInstance]
		if !found {
			return nil, fmt.Errorf("function %+v references an unknown package", function.Function)
		}
		builder.data.functions = append(builder.data.functions, function)
		builder.data.header.Counts.Relations += relationCount(function)
	}
	for _, specialization := range snapshot.Specializations() {
		builder, found := builders[specialization.Declaration.PackageInstance]
		if !found {
			return nil, fmt.Errorf("specialization %q references an unknown package", specialization.ID)
		}
		builder.data.specializations = append(builder.data.specializations, specialization)
		builder.data.header.Counts.Relations += relationCount(specialization.Function)
	}

	completeness := snapshot.Completeness()
	generationStatus, omissions, err := generationCompleteness(completeness)
	if err != nil {
		return nil, err
	}
	result := make([]Partition, 0, len(packages))
	for _, packageValue := range packages {
		builder := builders[packageValue.Instance]
		if len(input.GenericInterfaces) != 0 {
			builder.data.header.GenericInterfaceAlgorithm = moduleinterface.GenericHashEncoding
			builder.data.header.GenericInterfaceHash = input.GenericInterfaces[packageValue.Instance].String()
		}
		builder.data.completeness = cloneCompleteness(completeness)
		builder.data.header.GenerationStatus = generationStatus
		builder.data.header.Omissions = slices.Clone(omissions)
		builder.data.header.Counts.Files = len(builder.data.files)
		builder.data.header.Counts.Modules = len(builder.data.modules)
		builder.data.header.Counts.Symbols = len(builder.data.symbols)
		builder.data.header.Counts.Functions = len(builder.data.functions)
		builder.data.header.Counts.Specializations = len(builder.data.specializations)
		builder.data.header.Counts.Completeness = len(completeness)
		if builder.data.header.Counts.Files == 0 || builder.data.header.Counts.Modules == 0 {
			return nil, fmt.Errorf("package %q has no file or module partition", packageValue.Instance)
		}
		slices.SortFunc(builder.data.symbols, compareSymbol)
		result = append(result, Partition{data: builder.data})
	}
	return result, nil
}

func validatePackage(value semanticsnapshot.Package) error {
	if _, err := packagegraph.ParseInstanceID(value.Instance); err != nil {
		return fmt.Errorf("package %q instance identity: %w", value.UID, err)
	}
	if _, err := packagegraph.ParseCoordinateID(value.Coordinate); err != nil {
		return fmt.Errorf("package %q coordinate identity: %w", value.UID, err)
	}
	if _, err := packagecontent.ParseDigest(value.Source); err != nil {
		return fmt.Errorf("package %q source identity: %w", value.UID, err)
	}
	if value.UID == "" || value.Language == "" || value.SourceKind == "" {
		return fmt.Errorf("package %q has incomplete map header metadata", value.Instance)
	}
	for _, dependency := range value.Dependencies {
		if _, err := packagegraph.ParseInstanceID(dependency); err != nil {
			return fmt.Errorf("package %q dependency identity: %w", value.Instance, err)
		}
	}
	return nil
}

func generationCompleteness(values []semanticsnapshot.Completeness) (string, []string, error) {
	if len(values) == 0 {
		return "", nil, fmt.Errorf("Atlas map plan requires relation completeness records")
	}
	status := GenerationExact
	omissions := make([]string, 0)
	for index, value := range values {
		if value.Relation == "" || index > 0 && values[index-1].Relation >= value.Relation {
			return "", nil, fmt.Errorf("Atlas completeness records are not canonical")
		}
		switch value.Status {
		case semanticsnapshot.CompletenessExact:
			if len(value.Reasons) != 0 {
				return "", nil, fmt.Errorf("exact relation %q has omission reasons", value.Relation)
			}
		case semanticsnapshot.CompletenessPartial, semanticsnapshot.CompletenessUnavailable:
			status = GenerationPartial
			if len(value.Reasons) == 0 {
				return "", nil, fmt.Errorf("incomplete relation %q has no reason", value.Relation)
			}
			for _, reason := range value.Reasons {
				if reason == "" || strings.ContainsAny(reason, "\r\n\x00") {
					return "", nil, fmt.Errorf("relation %q has invalid omission reason", value.Relation)
				}
				omissions = append(omissions, value.Relation+":"+string(value.Status)+":"+reason)
			}
		default:
			return "", nil, fmt.Errorf("relation %q has unknown completeness %q", value.Relation, value.Status)
		}
	}
	slices.Sort(omissions)
	return status, omissions, nil
}

func relationCount(value semanticsnapshot.FunctionSummary) int {
	return len(value.Calls) + len(value.TrapCategories) + len(value.UnsafeSites) + len(value.MoveSites) +
		len(value.DropSites) + len(value.AllocationSites) + len(value.ReturnErrorSites) + len(value.Borrows)
}

func cloneCompleteness(values []semanticsnapshot.Completeness) []semanticsnapshot.Completeness {
	result := slices.Clone(values)
	for index := range result {
		result[index].Reasons = slices.Clone(result[index].Reasons)
	}
	return result
}

func cloneDeclarations(values []semanticsnapshot.Declaration) []semanticsnapshot.Declaration {
	result := slices.Clone(values)
	for index := range result {
		result[index].TypeParameters = slices.Clone(result[index].TypeParameters)
		result[index].Type = cloneTypeRef(result[index].Type)
		if result[index].Owner != nil {
			owner := cloneTypeRef(*result[index].Owner)
			result[index].Owner = &owner
		}
		result[index].Parameters = slices.Clone(result[index].Parameters)
		for parameterIndex := range result[index].Parameters {
			result[index].Parameters[parameterIndex].Type = cloneTypeRef(result[index].Parameters[parameterIndex].Type)
		}
		result[index].Effects = slices.Clone(result[index].Effects)
		result[index].Fields = cloneFields(result[index].Fields)
		result[index].Variants = slices.Clone(result[index].Variants)
		for variantIndex := range result[index].Variants {
			result[index].Variants[variantIndex].Fields = cloneFields(result[index].Variants[variantIndex].Fields)
		}
		result[index].DropEffects = slices.Clone(result[index].DropEffects)
	}
	return result
}

func cloneFields(values []semanticsnapshot.Field) []semanticsnapshot.Field {
	result := slices.Clone(values)
	for index := range result {
		result[index].Type = cloneTypeRef(result[index].Type)
	}
	return result
}

func cloneTypeRef(value semanticsnapshot.TypeRef) semanticsnapshot.TypeRef {
	value.Arguments = slices.Clone(value.Arguments)
	for index := range value.Arguments {
		value.Arguments[index] = cloneTypeRef(value.Arguments[index])
	}
	return value
}

func cloneFunctionSummaries(values []semanticsnapshot.FunctionSummary) []semanticsnapshot.FunctionSummary {
	result := slices.Clone(values)
	for index := range result {
		result[index].Calls = slices.Clone(result[index].Calls)
		result[index].TrapCategories = slices.Clone(result[index].TrapCategories)
		result[index].UnsafeSites = slices.Clone(result[index].UnsafeSites)
		result[index].MoveSites = slices.Clone(result[index].MoveSites)
		result[index].DropSites = slices.Clone(result[index].DropSites)
		result[index].AllocationSites = slices.Clone(result[index].AllocationSites)
		result[index].ReturnErrorSites = slices.Clone(result[index].ReturnErrorSites)
		result[index].Borrows = slices.Clone(result[index].Borrows)
	}
	return result
}

func compareSymbol(left, right Symbol) int {
	if value := cmp.Compare(left.Full, right.Full); value != 0 {
		return value
	}
	if value := cmp.Compare(left.Declaration.Module, right.Declaration.Module); value != 0 {
		return value
	}
	if value := cmp.Compare(left.Declaration.Kind, right.Declaration.Kind); value != 0 {
		return value
	}
	return cmp.Compare(left.Declaration.Name, right.Declaration.Name)
}
