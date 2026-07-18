// Package mirverifyfixture executes the versioned MIR verifier mutation manifest.
package mirverifyfixture

type manifest struct {
	Schema        string           `json:"$schema"`
	SchemaVersion string           `json:"schemaVersion"`
	Deferred      []string         `json:"deferred"`
	Contracts     []string         `json:"contracts"`
	Seeds         []seedDefinition `json:"seeds"`
	Cases         []caseDefinition `json:"cases"`
}

type seedDefinition struct {
	ID       string   `json:"id"`
	Baseline string   `json:"baseline"`
	Entities []string `json:"entities"`
}

type caseDefinition struct {
	ID       string      `json:"id"`
	Seed     string      `json:"seed"`
	Mutation mutation    `json:"mutation"`
	Expect   expectation `json:"expect"`
}

type mutation struct {
	Kind             string `json:"kind"`
	Target           string `json:"target"`
	SupportingEntity string `json:"supportingEntity,omitempty"`
}

type expectation struct {
	Status        string   `json:"status"`
	Codes         []string `json:"codes"`
	PrimaryEntity string   `json:"primaryEntity"`
}

// Failure identifies one seed, mutation, or verifier mismatch.
type Failure struct {
	CaseID string
	Detail string
}

// Report is deterministic MIR verifier mutation evidence.
type Report struct {
	SeedsRun int
	CasesRun int
	Failures []Failure
}
