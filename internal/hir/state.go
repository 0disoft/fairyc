package hir

// StateFieldKind separates one direct storage cell from one key-derived map
// namespace without exposing the implementation-side transaction package.
type StateFieldKind string

const (
	StateFieldCell StateFieldKind = "cell"
	StateFieldMap  StateFieldKind = "map"
)

// StateLayout is one source-independent package-owned persistent-state schema.
// No Stage 0 syntax constructs this HIR yet.
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
