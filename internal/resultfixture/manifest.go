// Package resultfixture executes the versioned Result acceptance manifest.
package resultfixture

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
	Expect     expectation `json:"expect"`
}

type expectation struct {
	Status           string                `json:"status"`
	CompletedThrough string                `json:"completedThrough,omitempty"`
	Phase            string                `json:"phase,omitempty"`
	Handling         []handlingAssertion   `json:"handling,omitempty"`
	Diagnostics      []diagnosticAssertion `json:"diagnostics,omitempty"`
}

type spanAssertion struct {
	Source    string `json:"source"`
	StartByte int    `json:"startByte"`
	EndByte   int    `json:"endByte"`
}

type handlingAssertion struct {
	Expression      spanAssertion `json:"expression"`
	Mode            string        `json:"mode"`
	ResultType      string        `json:"resultType"`
	SuccessType     string        `json:"successType"`
	ErrorType       string        `json:"errorType"`
	EvaluationCount int           `json:"evaluationCount"`
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

// Report is deterministic Result fixture execution evidence.
type Report struct {
	CasesRun int
	Failures []Failure
}
