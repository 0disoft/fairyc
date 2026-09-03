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
)

// StateValue carries one storage-codec identity and its canonical encoded
// bytes. The contract-state MIR verifier remains the canonicality authority.
type StateValue struct {
	TypeID  string
	Encoded []byte
}

// StateAccess names a package-owned field rather than a physical field index.
// Lowering resolves the name through the exact StateLayout owned by Package.
type StateAccess struct {
	Kind  StateAccessKind
	Field string
	Key   StateValue
	Value StateValue
	Span  Span
}

// StateTransactionExit is one statically selected transaction outcome. Branch
// and data-dependent exit lowering remain outside this source-independent cut.
type StateTransactionExit string

const (
	StateTransactionCommit StateTransactionExit = "commit"
	StateTransactionReject StateTransactionExit = "reject"
	StateTransactionTrap   StateTransactionExit = "trap"
)

// StateTransaction attaches straight-line state accesses to one existing HIR
// function. The function owns the declared effects; this side table supplies no
// second effect authority and is not part of a stable artifact.
type StateTransaction struct {
	Package      PackageID
	Function     SymbolID
	Accesses     []StateAccess
	Exit         StateTransactionExit
	TrapCategory string
	Span         Span
}
