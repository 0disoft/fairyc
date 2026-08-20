package genericplan

// PackageID identifies the package that owns a declaration, capability, type,
// or implementation for coherence checks.
type PackageID string

// TypeID is one exact canonical concrete type identity.
type TypeID string

// CapabilityID is one exact static capability identity such as sdk::Codec.
type CapabilityID string

// DeclarationID is one exact generic declaration identity.
type DeclarationID string

// ProviderID identifies the concrete function or compiler primitive satisfying
// one static capability implementation.
type ProviderID string

// SpecializationID is a deterministic monomorphization identity.
type SpecializationID string

// TypeDescriptor records only properties needed by generic planning. Layout,
// ownership Flow, ABI, and backend representation remain owned elsewhere.
type TypeDescriptor struct {
	ID         TypeID
	Owner      PackageID
	Abilities  []Ability
	Definition Digest
}

// CapabilityDescriptor declares one exact, statically resolved operation family.
// The first kernel stores arity and identity only; method sets, associated types,
// defaults, inheritance, and dynamic values are deliberately absent.
type CapabilityDescriptor struct {
	ID         CapabilityID
	Owner      PackageID
	Arity      int
	Definition Digest
}

// Implementation binds one exact concrete type tuple to one exact provider.
// Partial, blanket, negative, and specialized implementations are unsupported.
type Implementation struct {
	Capability     CapabilityID
	Arguments      []TypeID
	Owner          PackageID
	Provider       ProviderID
	ProviderDigest Digest
}

// TypeParameter is one declaration-local type variable with compiler-known
// ability requirements.
type TypeParameter struct {
	Name      string
	Abilities []Ability
}

// TypeExpression is either one declaration-local parameter reference or one
// exact concrete type. Composite inference and higher-kinded parameters are not
// part of the first kernel.
type TypeExpression struct {
	Parameter string
	Concrete  TypeID
}

// ParameterType references one declaration-local type parameter.
func ParameterType(name string) TypeExpression {
	return TypeExpression{Parameter: name}
}

// ConcreteType references one exact catalog type.
func ConcreteType(value TypeID) TypeExpression {
	return TypeExpression{Concrete: value}
}

// CapabilityConstraint requires one exact capability implementation after type
// substitution.
type CapabilityConstraint struct {
	Capability CapabilityID
	Arguments  []TypeExpression
}

// Dependency requests another generic specialization after substituting this
// declaration's bound type arguments.
type Dependency struct {
	Declaration DeclarationID
	Arguments   []TypeExpression
}

// Declaration is one bounded generic callable or aggregate planning contract.
// Definition must change whenever executable semantics or lowering inputs change.
type Declaration struct {
	ID           DeclarationID
	Owner        PackageID
	Parameters   []TypeParameter
	Constraints  []CapabilityConstraint
	Dependencies []Dependency
	Definition   Digest
}

// Request asks for one exact specialization. There is no type inference.
type Request struct {
	Declaration DeclarationID
	Arguments   []TypeID
}

// Binding records one parameter-to-concrete-type substitution.
type Binding struct {
	Parameter string
	Type      TypeID
}

// ResolvedConstraint records the exact provider selected for one specialization.
type ResolvedConstraint struct {
	Capability     CapabilityID
	Arguments      []TypeID
	Provider       ProviderID
	ProviderDigest Digest
}

// Specialization is one fully concrete, dependency-closed monomorphization unit.
type Specialization struct {
	ID           SpecializationID
	Declaration  DeclarationID
	Bindings     []Binding
	Constraints  []ResolvedConstraint
	Dependencies []SpecializationID
}

// Plan is deterministic dependency-first output. Consumers may emit each entry
// in order without runtime lookup tables.
type Plan struct {
	CatalogDigest   Digest
	Roots           []SpecializationID
	Specializations []Specialization
}
