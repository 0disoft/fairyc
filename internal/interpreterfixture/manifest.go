// Package interpreterfixture executes the versioned source-to-interpreter manifest.
package interpreterfixture

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
	Status           string            `json:"status"`
	CompletedThrough string            `json:"completedThrough"`
	Outcome          string            `json:"outcome"`
	Value            *int32            `json:"value,omitempty"`
	Category         *string           `json:"category,omitempty"`
	Drops            []dropExpectation `json:"drops,omitempty"`
}

type dropExpectation struct {
	ResourceType string          `json:"resourceType"`
	Binding      spanExpectation `json:"binding"`
}

type spanExpectation struct {
	Source    string `json:"source"`
	StartByte int    `json:"startByte"`
	EndByte   int    `json:"endByte"`
}

// Failure identifies one source-to-interpreter mismatch.
type Failure struct {
	CaseID string
	Detail string
}

// Report is deterministic interpreter fixture evidence.
type Report struct {
	CasesRun int
	Failures []Failure
}
