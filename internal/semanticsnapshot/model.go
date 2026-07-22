// Package semanticsnapshot owns the immutable compiler-fact boundary consumed by Fairy Atlas.
// It contains no map encoding, query behavior, source text, or host authority.
package semanticsnapshot

import (
	"slices"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/packagecontent"
	"github.com/0disoft/fairyc/internal/packagegraph"
)

const ModelVersion = "fairyc.semantic-snapshot/v0"

type Provenance string

const (
	ProvenanceCompilerDeclared Provenance = "compiler-declared"
	ProvenanceCompilerObserved Provenance = "compiler-observed"
)

type CompletenessStatus string

const (
	CompletenessExact       CompletenessStatus = "exact"
	CompletenessPartial     CompletenessStatus = "partial"
	CompletenessUnavailable CompletenessStatus = "unavailable"
)

type CallTargetKind string

const (
	CallTargetResolved CallTargetKind = "resolved"
	CallTargetExternal CallTargetKind = "external"
)

type Certainty string

const (
	CertaintyExact  Certainty = "exact"
	CertaintyOpaque Certainty = "opaque"
)

type TypeKind string

const (
	TypeBuiltin    TypeKind = "builtin"
	TypeNominal    TypeKind = "nominal"
	TypeBorrow     TypeKind = "borrow"
	TypePointer    TypeKind = "pointer"
	TypeFixedArray TypeKind = "fixed-array"
	TypeOption     TypeKind = "option"
	TypeResult     TypeKind = "result"
)

type DeclarationKind string

const (
	DeclarationStruct          DeclarationKind = "struct"
	DeclarationEnum            DeclarationKind = "enum"
	DeclarationResource        DeclarationKind = "resource"
	DeclarationFunction        DeclarationKind = "function"
	DeclarationForeignFunction DeclarationKind = "foreign-function"
)

// PackageInput binds disposable compiler package numbers to exact stable package metadata. BuildID
// and Dependencies are consumed during construction and never appear in a Snapshot.
type PackageInput struct {
	BuildID      hir.PackageID
	UID          string
	Name         string
	Version      string
	Language     string
	Instance     packagegraph.InstanceID
	Coordinate   packagegraph.CoordinateID
	Source       packagecontent.Digest
	SourceKind   string
	RuntimeABI   string
	Role         string
	Features     []string
	Dependencies []hir.PackageID
}

// Package is one path-free exact package record in the analyzed closure.
type Package struct {
	UID          string
	Name         string
	Version      string
	Language     string
	Instance     string
	Coordinate   string
	Source       string
	SourceKind   string
	RuntimeABI   string
	Role         string
	Features     []string
	Dependencies []string
}

// Span identifies package-relative original source bytes without storing source contents or a
// build-local package number.
type Span struct {
	PackageInstance string
	Source          string
	StartByte       int
	EndByte         int
}

// Module records one source module and the source file that owns it.
type Module struct {
	PackageInstance string
	Path            string
	Source          string
}

// DeclarationKey is an exact-snapshot join key, not the future stable Atlas SymbolId. It includes
// PackageInstanceId so feature- or source-distinct packages cannot collapse inside one snapshot.
type DeclarationKey struct {
	PackageInstance string
	Module          string
	Kind            DeclarationKind
	Name            string
}

// TypeRef removes disposable HIR package numbers while retaining the semantic type structure.
// Nominal types carry PackageCoordinateId because their source-independent namespace is distinct
// from the exact package instance that owns a snapshot record.
type TypeRef struct {
	Kind              TypeKind
	Name              string
	PackageCoordinate string
	Mutable           bool
	Length            uint64
	Arguments         []TypeRef
}

type Parameter struct {
	Name    string
	Type    TypeRef
	Mutable bool
	Span    Span
}

type Field struct {
	Name string
	Type TypeRef
	Span Span
}

type Variant struct {
	Name   string
	Span   Span
	Fields []Field
}

// Declaration contains source-declared facts only. Compiler-created drop glue is represented by
// its owning resource facts and is deliberately absent from this declaration set.
type Declaration struct {
	Key         DeclarationKey
	Span        Span
	Provenance  Provenance
	Public      bool
	Unsafe      bool
	Foreign     bool
	ABI         string
	LinkName    string
	Type        TypeRef
	Owner       *TypeRef
	Receiver    string
	Parameters  []Parameter
	Effects     []string
	Fields      []Field
	Variants    []Variant
	ReprC       bool
	DropKind    string
	DropEffects []string
}

type Call struct {
	Span       Span
	TargetKind CallTargetKind
	Target     DeclarationKey
	Certainty  Certainty
	ABI        string
	LinkName   string
}

type Borrow struct {
	Span    Span
	Kind    string
	Escapes bool
}

// FunctionSummary contains verified executable facts for one source-declared function.
type FunctionSummary struct {
	Function         DeclarationKey
	Provenance       Provenance
	Calls            []Call
	TrapCategories   []string
	UnsafeSites      []Span
	MoveSites        []Span
	DropSites        []Span
	AllocationSites  []Span
	ReturnErrorSites []Span
	Borrows          []Borrow
}

// Completeness is relation-specific so a foreign call cannot silently degrade unrelated local
// ownership or trap facts.
type Completeness struct {
	Relation string
	Status   CompletenessStatus
	Resolved int
	External int
	Unknown  int
	Reasons  []string
}

type snapshotData struct {
	packages     []Package
	modules      []Module
	declarations []Declaration
	functions    []FunctionSummary
	completeness []Completeness
}

// Snapshot is immutable by construction. Accessors return deep detached copies.
type Snapshot struct {
	data snapshotData
}

func (Snapshot) Version() string {
	return ModelVersion
}

func (snapshot Snapshot) Packages() []Package {
	result := slices.Clone(snapshot.data.packages)
	for index := range result {
		result[index].Features = slices.Clone(result[index].Features)
		result[index].Dependencies = slices.Clone(result[index].Dependencies)
	}
	return result
}

func (snapshot Snapshot) Modules() []Module {
	return slices.Clone(snapshot.data.modules)
}

func (snapshot Snapshot) Declarations() []Declaration {
	result := make([]Declaration, len(snapshot.data.declarations))
	for index, declaration := range snapshot.data.declarations {
		result[index] = cloneDeclaration(declaration)
	}
	return result
}

func (snapshot Snapshot) Functions() []FunctionSummary {
	result := make([]FunctionSummary, len(snapshot.data.functions))
	for index, function := range snapshot.data.functions {
		result[index] = cloneFunctionSummary(function)
	}
	return result
}

func (snapshot Snapshot) Completeness() []Completeness {
	result := slices.Clone(snapshot.data.completeness)
	for index := range result {
		result[index].Reasons = slices.Clone(result[index].Reasons)
	}
	return result
}

func cloneDeclaration(value Declaration) Declaration {
	result := value
	result.Type = cloneType(value.Type)
	if value.Owner != nil {
		owner := cloneType(*value.Owner)
		result.Owner = &owner
	}
	result.Parameters = make([]Parameter, len(value.Parameters))
	for index, parameter := range value.Parameters {
		result.Parameters[index] = parameter
		result.Parameters[index].Type = cloneType(parameter.Type)
	}
	result.Effects = slices.Clone(value.Effects)
	result.Fields = cloneFields(value.Fields)
	result.Variants = make([]Variant, len(value.Variants))
	for index, variant := range value.Variants {
		result.Variants[index] = variant
		result.Variants[index].Fields = cloneFields(variant.Fields)
	}
	result.DropEffects = slices.Clone(value.DropEffects)
	return result
}

func cloneFields(values []Field) []Field {
	result := make([]Field, len(values))
	for index, field := range values {
		result[index] = field
		result[index].Type = cloneType(field.Type)
	}
	return result
}

func cloneType(value TypeRef) TypeRef {
	result := value
	result.Arguments = make([]TypeRef, len(value.Arguments))
	for index, argument := range value.Arguments {
		result.Arguments[index] = cloneType(argument)
	}
	return result
}

func cloneFunctionSummary(value FunctionSummary) FunctionSummary {
	result := value
	result.Calls = slices.Clone(value.Calls)
	result.TrapCategories = slices.Clone(value.TrapCategories)
	result.UnsafeSites = slices.Clone(value.UnsafeSites)
	result.MoveSites = slices.Clone(value.MoveSites)
	result.DropSites = slices.Clone(value.DropSites)
	result.AllocationSites = slices.Clone(value.AllocationSites)
	result.ReturnErrorSites = slices.Clone(value.ReturnErrorSites)
	result.Borrows = slices.Clone(value.Borrows)
	return result
}
