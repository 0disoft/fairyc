// Package packageidentityfixture executes the versioned PackageInstanceId acceptance manifest.
package packageidentityfixture

type manifest struct {
	Schema        string           `json:"$schema"`
	SchemaVersion string           `json:"schemaVersion"`
	Contracts     []string         `json:"contracts"`
	Cases         []caseDefinition `json:"cases"`
}

type caseDefinition struct {
	ID       string      `json:"id"`
	Identity identity    `json:"identity"`
	Expect   expectation `json:"expect"`
}

type identity struct {
	PackageUID      string   `json:"packageUid"`
	SourceIdentity  string   `json:"sourceIdentity"`
	VersionIdentity string   `json:"versionIdentity"`
	Features        []string `json:"features"`
}

type expectation struct {
	Status      string `json:"status"`
	EncodingHex string `json:"encodingHex,omitempty"`
	InstanceID  string `json:"instanceId,omitempty"`
	Reason      string `json:"reason,omitempty"`
}

// Failure identifies one manifest assertion or contract-integrity mismatch.
type Failure struct {
	CaseID string
	Detail string
}

// Report is deterministic PackageInstanceId fixture execution evidence.
type Report struct {
	CasesRun int
	Failures []Failure
}
