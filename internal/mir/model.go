// Package mir defines and verifies the executable Stage 0 MIR boundary.
package mir

import "github.com/0disoft/fairyc/internal/hir"

// BlockID identifies one function-local basic block.
type BlockID int

// NoBlock denotes the absence of a control-flow successor.
const NoBlock BlockID = -1

// TypeProperties provides the compiler-known type facts needed during lowering.
type TypeProperties struct {
	resources map[hir.Type]struct{}
}

// NewTypeProperties constructs an immutable resource-type lookup.
func NewTypeProperties(resources []hir.Type) TypeProperties {
	values := make(map[hir.Type]struct{}, len(resources))
	for _, value := range resources {
		values[value] = struct{}{}
	}
	return TypeProperties{resources: values}
}

// NeedsDrop reports whether a value can own an active cleanup obligation.
func (p TypeProperties) NeedsDrop(value hir.Type) bool {
	if _, found := p.resources[value]; found {
		return true
	}
	success, failure, result := hir.ResultElements(value)
	return result && (p.NeedsDrop(success) || p.NeedsDrop(failure))
}

// Program is verified before it is handed to an interpreter or backend.
type Program struct {
	Functions []Function
}

// Function is one mutable-local control-flow graph and its source-linked facts.
type Function struct {
	Symbol             hir.SymbolID
	Name               string
	Span               hir.Span
	Parameters         []LocalID
	ResultType         hir.Type
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
	CleanupPaths       []CleanupPathFact
	ResultBranches     []ResultBranchFact
	TrapPaths          []TrapPathFact
	EliminatedForms    []EliminatedFormFact
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

// OperationKind identifies an already-normalized MIR operation.
type OperationKind string

const (
	OperationInitialize      OperationKind = "initialize"
	OperationMove            OperationKind = "move"
	OperationDrop            OperationKind = "drop"
	OperationCheckedDivision OperationKind = "checked-division"
	OperationResultSwitch    OperationKind = "result-switch"
	OperationUse             OperationKind = "use"
	OperationDropFlagInit    OperationKind = "drop-flag-init"
	OperationDropFlagClear   OperationKind = "drop-flag-clear"
	OperationAssign          OperationKind = "assign"
	OperationUnary           OperationKind = "unary"
	OperationBinary          OperationKind = "binary"
)

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
)

// Operand is an already-evaluated scalar value source.
type Operand struct {
	Kind  OperandKind
	Local LocalID
	I32   int32
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
	Operator         string
	TrapCategories   []string
	FlagClearedFirst bool
	SourceForm       string
}

// TerminatorKind identifies the only legal block-ending control transfers.
type TerminatorKind string

const (
	TerminatorGoto   TerminatorKind = "goto"
	TerminatorCall   TerminatorKind = "call"
	TerminatorSwitch TerminatorKind = "switch"
	TerminatorReturn TerminatorKind = "return"
	TerminatorTrap   TerminatorKind = "trap"
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
	Arguments        []Operand
	DestinationLocal LocalID
	ReturnValue      Operand
	SwitchKind       string
	SwitchOperands   []Operand
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
}

// MoveFact records a complete-root ownership transfer.
type MoveFact struct {
	Binding                      hir.Span
	Operand                      hir.Span
	Destination                  string
	CleanupObligationTransferred bool
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
