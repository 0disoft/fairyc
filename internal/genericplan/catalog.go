package genericplan

import "slices"

// Catalog is an immutable set of concrete types, static capabilities, exact
// implementations, and bounded generic declarations.
type Catalog struct {
	digest          Digest
	types           map[TypeID]TypeDescriptor
	capabilities    map[CapabilityID]CapabilityDescriptor
	implementations map[string]Implementation
	declarations    map[DeclarationID]Declaration
}

// Digest returns the deterministic complete catalog identity. Specialization
// keys do not include this broad digest; they include only reachable definitions.
func (c *Catalog) Digest() Digest {
	if c == nil {
		return Digest{}
	}
	return c.digest
}

// Type returns a defensive copy of one concrete type descriptor.
func (c *Catalog) Type(id TypeID) (TypeDescriptor, bool) {
	if c == nil {
		return TypeDescriptor{}, false
	}
	value, found := c.types[id]
	if !found {
		return TypeDescriptor{}, false
	}
	value.Abilities = slices.Clone(value.Abilities)
	return value, true
}

// Capability returns one static capability descriptor.
func (c *Catalog) Capability(id CapabilityID) (CapabilityDescriptor, bool) {
	if c == nil {
		return CapabilityDescriptor{}, false
	}
	value, found := c.capabilities[id]
	return value, found
}

// Declaration returns a defensive copy of one generic declaration.
func (c *Catalog) Declaration(id DeclarationID) (Declaration, bool) {
	if c == nil {
		return Declaration{}, false
	}
	value, found := c.declarations[id]
	if !found {
		return Declaration{}, false
	}
	return cloneDeclarationMap(map[DeclarationID]Declaration{id: value})[id], true
}
