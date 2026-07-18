// Package resourcefixture executes the versioned resource Flow acceptance manifest.
package resourcefixture

type manifest struct {
	Schema        string           `json:"$schema"`
	SchemaVersion string           `json:"schemaVersion"`
	Contracts     []string         `json:"contracts"`
	Cases         []caseDefinition `json:"cases"`
}

type caseDefinition struct {
	ID         string      `json:"id"`
	SourceRoot string      `json:"sourceRoot"`
	Sources    []string    `json:"sources"`
	Expect     expectation `json:"expect"`
}

type expectation struct {
	Status      string                `json:"status"`
	States      []stateAssertion      `json:"states"`
	Diagnostics []diagnosticAssertion `json:"diagnostics"`
}

type spanAssertion struct {
	Source    string `json:"source"`
	StartByte int    `json:"startByte"`
	EndByte   int    `json:"endByte"`
}

type stateAssertion struct {
	Binding            spanAssertion `json:"binding"`
	At                 spanAssertion `json:"at"`
	Observation        string        `json:"observation"`
	State              string        `json:"state"`
	CleanupObligations *int          `json:"cleanupObligations"`
}

type diagnosticAssertion struct {
	Code    string                 `json:"code"`
	Primary spanAssertion          `json:"primary"`
	Related []relatedSpanAssertion `json:"related"`
}

type relatedSpanAssertion struct {
	Role string        `json:"role"`
	Span spanAssertion `json:"span"`
}

// Failure identifies one manifest assertion mismatch.
type Failure struct {
	CaseID string
	Detail string
}

// Report is deterministic resource fixture execution evidence.
type Report struct {
	CasesRun int
	Failures []Failure
}
