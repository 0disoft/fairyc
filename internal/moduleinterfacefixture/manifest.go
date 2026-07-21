// Package moduleinterfacefixture executes the versioned source-to-interface-hash manifest.
package moduleinterfacefixture

type manifest struct {
	Schema        string           `json:"$schema"`
	SchemaVersion string           `json:"schemaVersion"`
	Contracts     []string         `json:"contracts"`
	Identity      identity         `json:"identity"`
	Cases         []caseDefinition `json:"cases"`
}

type identity struct {
	PackageUID      string   `json:"packageUid"`
	SourceIdentity  string   `json:"sourceIdentity"`
	VersionIdentity string   `json:"versionIdentity"`
	Features        []string `json:"features"`
}

type caseDefinition struct {
	ID         string `json:"id"`
	Source     string `json:"source"`
	CompareTo  string `json:"compareTo,omitempty"`
	Relation   string `json:"relation"`
	ExpectHash string `json:"expectHash,omitempty"`
}

// Failure identifies one interface invalidation or fixture-integrity mismatch.
type Failure struct {
	CaseID string
	Detail string
}

// Report is deterministic source-to-interface-hash evidence.
type Report struct {
	CasesRun int
	Failures []Failure
}
