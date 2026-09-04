package hir

// StateFieldKind separates one direct storage cell from one key-derived map
// namespace without exposing the implementation-side transaction package.
type StateFieldKind string

const (
	StateFieldCell StateFieldKind = "cell"
	StateFieldMap  StateFieldKind = "map"
)

// StateLayout is one package-owned persistent-state schema. Language 13 source declarations and
// compiler-private direct fixtures both construct this same HIR representation.
type StateLayout struct {
	Package     PackageID
	ContractUID string
	Span        Span
	Fields      []StateField
}

// StateField preserves declaration order and canonical storage codec identity.
type StateField struct {
	Name        string
	Kind        StateFieldKind
	KeyTypeID   string
	ValueTypeID string
	Span        Span
}

// StateAccessKind is one source-independent persistent-state action. Ordinary
// Fairy expressions do not construct these operations yet.
type StateAccessKind string

const (
	StateAccessCellRead   StateAccessKind = "cell-read"
	StateAccessCellWrite  StateAccessKind = "cell-write"
	StateAccessCellDelete StateAccessKind = "cell-delete"
	StateAccessMapRead    StateAccessKind = "map-read"
	StateAccessMapWrite   StateAccessKind = "map-write"
	StateAccessMapDelete  StateAccessKind = "map-delete"
	StateAccessCellBorrow StateAccessKind = "cell-borrow"
	StateAccessMapBorrow  StateAccessKind = "map-borrow"
	StateAccessBorrowView StateAccessKind = "borrow-view"
	StateAccessBorrowCopy StateAccessKind = "borrow-copy"
	StateAccessCall       StateAccessKind = "call"
)

// StateValue carries one storage-codec identity and its canonical encoded
// bytes. The contract-state MIR verifier remains the canonicality authority.
type StateValue struct {
	TypeID  string
	Encoded []byte
}

// StateValueID is one transaction-local SSA value identity. Zero means that
// an access uses its canonical literal operand instead of a prior value.
type StateValueID uint32

// StateBorrowID is one transaction-unique owner-bound state view. Borrow identities
// cannot enter ordinary value edges or block parameters, and cannot cross blocks.
type StateBorrowID uint32

// StateBlockID is one transaction-local basic-block identity. IDs are dense
// and source ordered so stable transport never depends on map iteration.
type StateBlockID uint32

// StateBlockParameter is an explicit join value. Every predecessor must pass
// one same-typed argument for each parameter.
type StateBlockParameter struct {
	ID     StateValueID
	TypeID string
}

// StateAccess names a package-owned field rather than a physical field index.
// Lowering resolves the name through the exact StateLayout owned by Package.
type StateAccess struct {
	Kind         StateAccessKind
	Field        string
	Key          StateValue
	Value        StateValue
	KeyInput     StateValueID
	ValueInput   StateValueID
	Result       StateValueID
	FoundResult  StateValueID
	BorrowInput  StateBorrowID
	BorrowResult StateBorrowID
	ViewOffset   uint32
	ViewLength   uint32
	Callee       SymbolID
	Cost         uint32
	Span         Span
}

type StateTerminatorKind string

const (
	StateTerminatorJump   StateTerminatorKind = "jump"
	StateTerminatorBranch StateTerminatorKind = "branch"
	StateTerminatorCommit StateTerminatorKind = "commit"
	StateTerminatorReject StateTerminatorKind = "reject"
	StateTerminatorTrap   StateTerminatorKind = "trap"
	StateTerminatorReturn StateTerminatorKind = "return"
)

// StateTerminator closes every block. Branch conditions must be bool/v1 SSA
// values; jumps and branch edges pass explicit arguments to join parameters.
type StateTerminator struct {
	Kind           StateTerminatorKind
	Condition      StateValueID
	Target         StateBlockID
	Arguments      []StateValueID
	TrueTarget     StateBlockID
	TrueArguments  []StateValueID
	FalseTarget    StateBlockID
	FalseArguments []StateValueID
	TrapCategory   string
	Cost           uint32
}

// StateBlock is one acyclic transaction block. Accesses execute in order and
// may define typed SSA values consumed later in the block or passed on edges.
type StateBlock struct {
	ID         StateBlockID
	Parameters []StateBlockParameter
	Accesses   []StateAccess
	Terminator StateTerminator
}

// StateTransactionExit is one statically selected transaction outcome. Branch
// and data-dependent exit lowering remain outside this source-independent cut.
type StateTransactionExit string

const (
	StateTransactionCommit StateTransactionExit = "commit"
	StateTransactionReject StateTransactionExit = "reject"
	StateTransactionTrap   StateTransactionExit = "trap"
	StateTransactionReturn StateTransactionExit = "return"
)

// StateTransaction attaches either legacy straight-line accesses or one
// bounded acyclic state graph to an existing HIR function. The function owns
// the declared effects; this side table supplies no second effect authority.
type StateTransaction struct {
	Package      PackageID
	Function     SymbolID
	Accesses     []StateAccess
	Exit         StateTransactionExit
	TrapCategory string
	Entry        StateBlockID
	Blocks       []StateBlock
	Span         Span
}
