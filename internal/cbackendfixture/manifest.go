// Package cbackendfixture executes the versioned source-to-C-emission manifest.
package cbackendfixture

type manifest struct {
	Schema        string           `json:"$schema"`
	SchemaVersion string           `json:"schemaVersion"`
	Deferred      []string         `json:"deferred"`
	Contracts     []string         `json:"contracts"`
	Cases         []caseDefinition `json:"cases"`
}

type caseDefinition struct {
	ID         string      `json:"id"`
	SourceRoot string      `json:"sourceRoot"`
	Sources    []string    `json:"sources"`
	Entry      string      `json:"entry"`
	Expect     expectation `json:"expect"`
}

type expectation struct {
	Status           string   `json:"status"`
	CompletedThrough string   `json:"completedThrough"`
	Outcome          string   `json:"outcome"`
	Value            *int32   `json:"value,omitempty"`
	Category         *string  `json:"category,omitempty"`
	Target           string   `json:"target"`
	Dialect          string   `json:"dialect"`
	Functions        []string `json:"functions"`
	Features         []string `json:"features"`
}

// Failure identifies one source-to-C-emission mismatch.
type Failure struct {
	CaseID string
	Detail string
}

// Report is deterministic C-emission fixture evidence.
type Report struct {
	CasesRun int
	Failures []Failure
}
