// Package regionfixture executes the versioned Region acceptance manifest.
package regionfixture

type manifest struct {
	Schema        string           `json:"$schema"`
	SchemaVersion string           `json:"schemaVersion"`
	Deferred      []string         `json:"deferred"`
	Contracts     []string         `json:"contracts"`
	Cases         []caseDefinition `json:"cases"`
}

type caseDefinition struct {
	ID           string                 `json:"id"`
	SourceRoot   string                 `json:"sourceRoot"`
	Sources      []string               `json:"sources"`
	PackageGraph *packageGraphReference `json:"packageGraph,omitempty"`
	Expect       expectation            `json:"expect"`
}

type packageGraphReference struct {
	Manifest string `json:"manifest"`
	CaseID   string `json:"caseId"`
	RootNode string `json:"rootNode"`
}

type expectation struct {
	Status           string                `json:"status"`
	CompletedThrough string                `json:"completedThrough,omitempty"`
	Phase            string                `json:"phase,omitempty"`
	Origins          []originAssertion     `json:"origins,omitempty"`
	Diagnostics      []diagnosticAssertion `json:"diagnostics,omitempty"`
}

type spanAssertion struct {
	Source    string `json:"source"`
	StartByte int    `json:"startByte"`
	EndByte   int    `json:"endByte"`
}

type originAssertion struct {
	Operation  string         `json:"operation"`
	Allocation spanAssertion  `json:"allocation"`
	Origin     string         `json:"origin"`
	LastUse    *spanAssertion `json:"lastUse,omitempty"`
	Escapes    bool           `json:"escapes"`
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

// Report is deterministic Region fixture execution evidence.
type Report struct {
	CasesRun int
	Failures []Failure
}
