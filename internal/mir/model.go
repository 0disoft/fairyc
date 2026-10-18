// Package mir defines and verifies the executable Stage 0 MIR boundary.
package mir

import (
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/0disoft/fairyc/internal/hir"
)

// BlockID identifies one function-local basic block.
type BlockID int

// NoBlock denotes the absence of a control-flow successor.
const NoBlock BlockID = -1

// typeProperties provides the compiler-known type facts needed during lowering.
type typeProperties struct {
	resources          map[hir.Type]struct{}
	structs            map[hir.Type]*StructType
	enums              map[hir.Type]*EnumType
	constructionFields map[hir.Type][]string
}

func newTypeProperties(resources []ResourceType, structs []StructType, enums []EnumType) typeProperties {
	constructionFields := make(map[hir.Type][]string, len(resources)+len(structs))
	resourceValues := make(map[hir.Type]struct{}, len(resources))
	for _, value := range resources {
		resourceValues[value.Type] = struct{}{}
		constructionFields[value.Type] = resourceFieldNames(value.Fields)
	}
	structValues := make(map[hir.Type]*StructType, len(structs))
	for index := range structs {
		structValues[structs[index].Type] = &structs[index]
		constructionFields[structs[index].Type] = structFieldNames(structs[index].Fields)
	}
	enumValues := make(map[hir.Type]*EnumType, len(enums))
	for index := range enums {
		enumValues[enums[index].Type] = &enums[index]
	}
	return typeProperties{resources: resourceValues, structs: structValues, enums: enumValues, constructionFields: constructionFields}
}

func sameSpanSource(left, right hir.Span) bool {
	return left.Package == right.Package && left.Source == right.Source
}

func (p typeProperties) IsResource(value hir.Type) bool {
	_, found := p.resources[value]
	return found
}

func (p typeProperties) IsStruct(value hir.Type) bool {
	_, found := p.structs[value]
	return found
}

func (p typeProperties) Struct(value hir.Type) *StructType {
	return p.structs[value]
}

func (p typeProperties) Enum(value hir.Type) *EnumType {
	return p.enums[value]
}

// NeedsDrop reports whether a value can own an active cleanup obligation.
func (p typeProperties) NeedsDrop(value hir.Type) bool {
	if _, found := p.resources[value]; found {
		return true
	}
	if element, option := hir.OptionElement(value); option {
		return p.NeedsDrop(element)
	}
	success, failure, result := hir.ResultElements(value)
	return result && (p.NeedsDrop(success) || p.NeedsDrop(failure))
}

// Program is verified before it is handed to an interpreter or backend.
type Program struct {
	StructTypes      []StructType
	EnumTypes        []EnumType
	ResourceTypes    []ResourceType
	ForeignFunctions []ForeignFunction
	Functions        []Function
}

// EnumType is one verified restricted inert tagged definition used by executable MIR.
type EnumType struct {
	Type     hir.Type
	Name     string
	Span     hir.Span
	Entity   string
	Variants []EnumVariant
}

// EnumVariant preserves declaration order and its named payload fields.
type EnumVariant struct {
	Name   string
	Span   hir.Span
	Fields []EnumField
}

// EnumField preserves one named inert payload field.
type EnumField struct {
	Name string
	Type hir.Type
	Span hir.Span
}

// StructType is one verified inert aggregate definition used by executable MIR.
type StructType struct {
	Type   hir.Type
	Name   string
	Span   hir.Span
	Entity string
	ReprC  bool
	Fields []StructField
}

// StructField preserves source identity, declaration order, and the resolved field type.
type StructField struct {
	Name string
	Type hir.Type
	Span hir.Span
}

// ForeignFunction is one verified raw external declaration consumed by a backend, not the interpreter.
type ForeignFunction struct {
	Symbol         hir.SymbolID
	Name           string
	LinkName       string
	ABI            string
	Span           hir.Span
	ParameterTypes []hir.Type
	ResultType     hir.Type
	Entity         string
}

// CompatibleForeignSignature reports whether two declarations can name one external ABI symbol.
func CompatibleForeignSignature(left, right ForeignFunction) bool {
	return left.ABI == right.ABI && left.ResultType == right.ResultType && slices.Equal(left.ParameterTypes, right.ParameterTypes)
}

// ResourceType is one verified move-only aggregate definition used by executable MIR.
type ResourceType struct {
	Type        hir.Type
	Span        hir.Span
	Entity      string
	Fields      []ResourceField
	DropKind    hir.ResourceDropKind
	DropSpan    hir.Span
	DropSymbol  hir.SymbolID
	DropEffects []hir.Effect
}

// ResourceField preserves declaration order and the resolved semantic field type.
type ResourceField struct {
	Name string
	Type hir.Type
	Span hir.Span
}

// Function is one mutable-local control-flow graph and its source-linked facts.
type Function struct {
	Symbol             hir.SymbolID
	Name               string
	Span               hir.Span
	Parameters         []LocalID
	ParameterTypes     []hir.Type
	ResultType         hir.Type
	Effects            []hir.Effect
	Entry              BlockID
	Blocks             []Block
	Resources          []ResourceLocal
	DropFlags          []DropFlag
	Loans              []Loan
	ReturnOrigins      []ReturnOrigin
	RegionValues       []RegionValue
	UnsafeOperations   []UnsafeOperation
	ReturnEntity       string
	Entities           []string
	OrderedEvaluations []OrderedEvaluationFact
	Calls              []CallFact
	Moves              []MoveFact
	Reassignments      []ReassignmentFact
	RegionScopes       []RegionScopeFact
	CleanupPaths       []CleanupPathFact
	ResultBranches     []ResultBranchFact
	TrapPaths          []TrapPathFact
	EliminatedForms    []EliminatedFormFact
	DropGlueFor        hir.Type
}

// ResourceLocal is one parameter or local that owns a cleanup obligation while live.
type ResourceLocal struct {
	Symbol    hir.SymbolID
	Type      hir.Type
	Binding   hir.Span
	Parameter bool
	Entity    string
}

// DropFlag is one compiler-owned conditional cleanup flag.
type DropFlag struct {
	Entity string
	Symbol hir.SymbolID
}

// Loan records the verified origin extent of one safe borrow.
type Loan struct {
	Entity         string
	OriginEntity   string
	OutlivesOrigin bool
}

// ReturnOrigin is one candidate origin for a safe-borrow return.
type ReturnOrigin struct {
	Entity   string
	Returned bool
}

// RegionValue records one value placed in a plain lexical region.
type RegionValue struct {
	Entity          string
	OperationEntity string
	RegionEntity    string
	Escapes         bool
	NeedsDrop       bool
}

// UnsafeOperation records the provenance required for an unsafe instruction.
type UnsafeOperation struct {
	Entity           string
	SourceSpanEntity string
	ScopeEntity      string
}

// Block is a straight-line operation sequence ending in exactly one terminator.
type Block struct {
	ID         BlockID
	Entity     string
	Operations []Operation
	Terminator Terminator
}

// LocalID identifies one source local, parameter, or compiler temporary.
type LocalID int

// NoLocal denotes an operation without a value destination.
const NoLocal LocalID = -1

// OperandKind identifies a scalar MIR operand.
type OperandKind string

const (
	OperandInvalid OperandKind = ""
	OperandLocal   OperandKind = "local"
	OperandI32     OperandKind = "i32"
	OperandI64     OperandKind = "i64"
	OperandU8      OperandKind = "u8"
	OperandU32     OperandKind = "u32"
	OperandUSize   OperandKind = "usize"
	OperandNull    OperandKind = "null"
	OperandCString OperandKind = "c-string"
)

const OperandByteString OperandKind = "byte-string"

const OperandString OperandKind = "string"

// Operand is an already-evaluated scalar value source.
type Operand struct {
	Kind  OperandKind
	Local LocalID
	I32   int32
	I64   int64 `json:",omitempty"`
	U64   uint64
	Type  hir.Type
	// StaticData stores exact compiler-owned bytes in a comparable form. Byte strings retain
	// their exact length; C strings include their single trailing NUL.
	StaticData string
}

func validCStringStaticData(value string) bool {
	return len(value) > 0 && value[len(value)-1] == 0 && !strings.Contains(value[:len(value)-1], "\x00")
}

func validUTF8StringStaticData(value string) bool {
	return utf8.ValidString(value)
}

// Operation carries explicit ownership, trap, and normalization information.
type Operation struct {
	Kind             OperationKind
	Entity           string
	Span             hir.Span
	Symbol           hir.SymbolID
	Destination      string
	ProjectionEntity string
	DropFlagEntity   string
	DestinationLocal LocalID
	Operands         []Operand
	Fields           []string
	Type             hir.Type
	OwnerType        hir.Type
	IndexType        hir.Type
	Field            string
	Operator         string
	UnsafeScope      hir.Span
	Variant          string
	MovePayload      bool
	TrapCategories   []string
	FlagClearedFirst bool
	SourceForm       string
	ExitKind         string
}

func structTypeEntity(value hir.Type) string {
	return aggregateTypeEntity("struct-type", value)
}

func enumTypeEntity(value hir.Type) string {
	return aggregateTypeEntity("enum-type", value)
}

func resourceTypeEntity(value hir.Type) string {
	return aggregateTypeEntity("resource-type", value)
}

func aggregateTypeEntity(prefix string, value hir.Type) string {
	var name strings.Builder
	separator := false
	for _, value := range strings.ToLower(value.String()) {
		if value >= 'a' && value <= 'z' || value >= '0' && value <= '9' {
			if separator && name.Len() > 0 {
				name.WriteByte('-')
			}
			name.WriteRune(value)
			separator = false
			continue
		}
		separator = name.Len() > 0
	}
	if name.Len() == 0 {
		return prefix + ":invalid"
	}
	return prefix + ":" + name.String()
}

// TerminatorKind identifies the only legal block-ending control transfers.
type TerminatorKind string

const (
	TerminatorGoto        TerminatorKind = "goto"
	TerminatorCall        TerminatorKind = "call"
	TerminatorSwitch      TerminatorKind = "switch"
	TerminatorReturn      TerminatorKind = "return"
	TerminatorTrap        TerminatorKind = "trap"
	TerminatorUnreachable TerminatorKind = "unreachable"
)

// Terminator ends one block. Calls never have an unwind successor in Stage 0.
type Terminator struct {
	Kind             TerminatorKind
	Entity           string
	Span             hir.Span
	Successors       []BlockID
	UnwindSuccessor  BlockID
	TrapCategories   []string
	Callee           hir.SymbolID
	ForeignABI       string
	UnsafeScope      hir.Span
	Arguments        []Operand
	DestinationLocal LocalID
	ReturnValue      Operand
	SwitchKind       string
	SwitchType       hir.Type
	SwitchVariants   []string
	SwitchOperands   []Operand
	Operator         string
}

// CleanupAction is one explicit normal-exit cleanup operation.
type CleanupAction struct {
	Kind    string
	Binding hir.Span
}

// OrderedEvaluationFact records child evaluation before a parent operation.
type OrderedEvaluationFact struct {
	Expression               hir.Span
	Sequence                 []hir.Span
	MaterializedBeforeParent bool
}

// CallFact records a non-unwinding call terminator.
type CallFact struct {
	Call            hir.Span
	ArgumentOrder   []hir.Span
	NormalSuccessor bool
	UnwindSuccessor bool
	ForeignABI      string
	UnsafeScope     hir.Span
}

// MoveFact records a complete-root ownership transfer.
type MoveFact struct {
	Binding                      hir.Span
	Operand                      hir.Span
	Destination                  string
	CleanupObligationTransferred bool
}

// ReassignmentFact records the success-path ordering of one whole resource replacement.
type ReassignmentFact struct {
	Binding         hir.Span
	Target          hir.Span
	Value           hir.Span
	ValueBeforeDrop bool
	DroppedOld      bool
	InstalledNew    bool
}

// RegionScopeFact records source-linked placement and normal reset ordering for one lexical region.
type RegionScopeFact struct {
	Region             hir.Span
	Placements         []hir.Span
	CleanupBeforeReset []CleanupAction
	Reset              hir.Span
	EarlyResets        []RegionResetFact
}

// RegionResetFact records one non-fallthrough exit that cleans and resets a lexical region.
type RegionResetFact struct {
	Exit               hir.Span
	ExitKind           string
	CleanupBeforeReset []CleanupAction
	Reset              hir.Span
}

// CleanupPathFact records the explicit cleanup chain for one normal exit.
type CleanupPathFact struct {
	Exit     hir.Span
	ExitKind string
	Actions  []CleanupAction
}

// ResultBranchFact records the normalized success and error paths of `?`.
type ResultBranchFact struct {
	Question              hir.Span
	ResultExpression      hir.Span
	EvaluationCount       int
	SuccessPayload        string
	ErrorPayload          string
	ResultTemporaryDeadOn []string
	ErrorCleanupActions   []CleanupAction
}

// TrapPathFact records a non-returning semantic trap path.
type TrapPathFact struct {
	Operation      hir.Span
	Categories     []string
	CleanupActions []CleanupAction
	HasSuccessor   bool
}

// EliminatedFormFact proves that a source-only form is absent from MIR.
type EliminatedFormFact struct {
	Form           string
	RemainingCount int
}
