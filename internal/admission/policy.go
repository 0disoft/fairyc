// Package admission describes compiler-private feature admission. A zero policy
// admits only the ordinary scalar generic pipeline; it grants no SDK authority.
package admission

import "fmt"

type StorageMode uint8

const (
	StorageDisabled StorageMode = iota
	StorageQueries
	StorageTransfers
)

type ArgumentMode uint8

const (
	ScalarArguments ArgumentMode = iota
	NominalArguments
)

type TemplateMode uint8

const (
	OrdinaryTemplates TemplateMode = iota
	SDKOwnerTemplates
)

type VectorMode uint8

const (
	ReleasedVectors VectorMode = iota
	CandidateVectors
)

// Capabilities travels unchanged from source preparation to MIR verification.
// StorageTransfers includes queries; SDK templates also require nominal arguments.
type Capabilities struct {
	Storage   StorageMode
	Arguments ArgumentMode
	Templates TemplateMode
	Vectors   VectorMode
}

func (c Capabilities) Validate() error {
	if c.Storage > StorageTransfers || c.Arguments > NominalArguments || c.Templates > SDKOwnerTemplates || c.Vectors > CandidateVectors {
		return fmt.Errorf("unknown compiler capability")
	}
	if c.Templates == SDKOwnerTemplates && (c.Storage != StorageTransfers || c.Arguments != NominalArguments) {
		return fmt.Errorf("SDK templates require nominal storage transfers")
	}
	if c.Vectors == CandidateVectors && c.Templates != SDKOwnerTemplates {
		return fmt.Errorf("candidate vectors require SDK owner templates")
	}
	return nil
}

func (c Capabilities) AllowsStorageQueries() bool {
	return c.Storage == StorageQueries || c.Storage == StorageTransfers
}
func (c Capabilities) AllowsStorageTransfers() bool { return c.Storage == StorageTransfers }
func (c Capabilities) AllowsNominalArguments() bool { return c.Arguments == NominalArguments }
func (c Capabilities) SDKTemplates() bool           { return c.Templates == SDKOwnerTemplates }
func (c Capabilities) VectorCandidate() bool        { return c.Vectors == CandidateVectors }

// Policy adds the exact, caller-authenticated storage owner scope. Nil means
// unrestricted private admission; an explicit empty scope is rejected. No source
// spelling or package label authenticates an owner. Owner IDs stay layer-typed.
type Policy[Owner comparable] struct {
	Capabilities  Capabilities
	StorageOwners []Owner
}

func (p Policy[Owner]) Validate() error {
	if err := p.Capabilities.Validate(); err != nil {
		return err
	}
	if p.StorageOwners != nil || p.Capabilities.SDKTemplates() {
		if len(p.StorageOwners) == 0 {
			return fmt.Errorf("storage operation owners must be explicit")
		}
		if p.Capabilities.Storage == StorageDisabled {
			return fmt.Errorf("storage owners require storage capabilities")
		}
		seen := make(map[Owner]bool, len(p.StorageOwners))
		for _, owner := range p.StorageOwners {
			if seen[owner] {
				return fmt.Errorf("storage operation owner is duplicated")
			}
			seen[owner] = true
		}
	}
	return nil
}
