package genericplan

import "slices"

// CatalogLimits bounds compiler memory and validation work before planning.
type CatalogLimits struct {
	MaxTypes           int
	MaxCapabilities    int
	MaxImplementations int
	MaxDeclarations    int
	MaxTypeParameters  int
	MaxConstraints     int
	MaxDependencies    int
}

// DefaultCatalogLimits returns deliberately bounded but practical limits for an
// initial compiler-owned SDK and application package graph.
func DefaultCatalogLimits() CatalogLimits {
	return CatalogLimits{
		MaxTypes:           4096,
		MaxCapabilities:    512,
		MaxImplementations: 16384,
		MaxDeclarations:    4096,
		MaxTypeParameters:  8,
		MaxConstraints:     32,
		MaxDependencies:    32,
	}
}

// Builder collects mutable input and publishes one immutable Catalog on Build.
type Builder struct {
	limits          CatalogLimits
	types           map[TypeID]TypeDescriptor
	capabilities    map[CapabilityID]CapabilityDescriptor
	implementations []Implementation
	declarations    map[DeclarationID]Declaration
}

// NewBuilder validates limits before accepting catalog state.
func NewBuilder(limits CatalogLimits) (*Builder, error) {
	if limits.MaxTypes <= 0 || limits.MaxCapabilities <= 0 || limits.MaxImplementations <= 0 ||
		limits.MaxDeclarations <= 0 || limits.MaxTypeParameters <= 0 || limits.MaxConstraints <= 0 ||
		limits.MaxDependencies <= 0 {
		return nil, fail(CodeCatalogBudgetExceeded, "every catalog limit must be positive")
	}
	return &Builder{
		limits:       limits,
		types:        make(map[TypeID]TypeDescriptor),
		capabilities: make(map[CapabilityID]CapabilityDescriptor),
		declarations: make(map[DeclarationID]Declaration),
	}, nil
}

// AddType adds one concrete type after canonicalizing its ability closure.
func (b *Builder) AddType(value TypeDescriptor) error {
	if b == nil {
		return fail(CodeInvalidDeclaration, "catalog builder is nil")
	}
	if !validPath(string(value.ID)) || !validPath(string(value.Owner)) {
		return fail(CodeInvalidIdentifier, "type %q or owner %q is not a canonical path", value.ID, value.Owner)
	}
	if _, duplicate := b.types[value.ID]; duplicate {
		return fail(CodeDuplicateCatalogItem, "type %q is declared more than once", value.ID)
	}
	if len(b.types) >= b.limits.MaxTypes {
		return fail(CodeCatalogBudgetExceeded, "type count exceeds %d", b.limits.MaxTypes)
	}
	if value.Definition.IsZero() {
		return fail(CodeInvalidDeclaration, "type %q has no definition digest", value.ID)
	}
	abilities, err := canonicalAbilities(value.Abilities)
	if err != nil {
		return err
	}
	value.Abilities = abilities
	b.types[value.ID] = value
	return nil
}

// AddCapability adds one exact static capability family.
func (b *Builder) AddCapability(value CapabilityDescriptor) error {
	if b == nil {
		return fail(CodeInvalidDeclaration, "catalog builder is nil")
	}
	if !validPath(string(value.ID)) || !validPath(string(value.Owner)) {
		return fail(CodeInvalidIdentifier, "capability %q or owner %q is not a canonical path", value.ID, value.Owner)
	}
	if value.Definition.IsZero() {
		return fail(CodeInvalidDeclaration, "capability %q has no definition digest", value.ID)
	}
	if value.Arity <= 0 || value.Arity > b.limits.MaxTypeParameters {
		return fail(CodeInvalidDeclaration, "capability %q arity %d is outside 1..%d", value.ID, value.Arity, b.limits.MaxTypeParameters)
	}
	if _, duplicate := b.capabilities[value.ID]; duplicate {
		return fail(CodeDuplicateCatalogItem, "capability %q is declared more than once", value.ID)
	}
	if len(b.capabilities) >= b.limits.MaxCapabilities {
		return fail(CodeCatalogBudgetExceeded, "capability count exceeds %d", b.limits.MaxCapabilities)
	}
	b.capabilities[value.ID] = value
	return nil
}

// AddImplementation records one exact provider. Cross-reference and coherence
// validation occurs atomically in Build so declaration order is irrelevant.
func (b *Builder) AddImplementation(value Implementation) error {
	if b == nil {
		return fail(CodeInvalidDeclaration, "catalog builder is nil")
	}
	if !validPath(string(value.Capability)) || !validPath(string(value.Owner)) || !validPath(string(value.Provider)) {
		return fail(CodeInvalidIdentifier, "implementation identifiers are not canonical paths")
	}
	if value.ProviderDigest.IsZero() {
		return fail(CodeInvalidDeclaration, "implementation provider %q has no digest", value.Provider)
	}
	if len(value.Arguments) == 0 {
		return fail(CodeInvalidDeclaration, "implementation for %q has no type arguments", value.Capability)
	}
	for _, argument := range value.Arguments {
		if !validPath(string(argument)) {
			return fail(CodeInvalidIdentifier, "implementation type %q is not a canonical path", argument)
		}
	}
	if len(b.implementations) >= b.limits.MaxImplementations {
		return fail(CodeCatalogBudgetExceeded, "implementation count exceeds %d", b.limits.MaxImplementations)
	}
	value.Arguments = slices.Clone(value.Arguments)
	b.implementations = append(b.implementations, value)
	return nil
}

// AddDeclaration records one generic declaration. Cross-reference validation is
// delayed until Build to permit forward dependency declarations.
func (b *Builder) AddDeclaration(value Declaration) error {
	if b == nil {
		return fail(CodeInvalidDeclaration, "catalog builder is nil")
	}
	if !validPath(string(value.ID)) || !validPath(string(value.Owner)) {
		return fail(CodeInvalidIdentifier, "declaration %q or owner %q is not a canonical path", value.ID, value.Owner)
	}
	if _, duplicate := b.declarations[value.ID]; duplicate {
		return fail(CodeDuplicateCatalogItem, "declaration %q is declared more than once", value.ID)
	}
	if len(b.declarations) >= b.limits.MaxDeclarations {
		return fail(CodeCatalogBudgetExceeded, "declaration count exceeds %d", b.limits.MaxDeclarations)
	}
	normalized, err := canonicalDeclaration(value, b.limits)
	if err != nil {
		return err
	}
	b.declarations[value.ID] = normalized
	return nil
}
