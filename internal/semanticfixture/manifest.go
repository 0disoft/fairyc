// Package semanticfixture executes the versioned semantic acceptance manifest.
package semanticfixture

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
	Status           string                `json:"status"`
	CompletedThrough string                `json:"completedThrough"`
	Phase            string                `json:"phase"`
	Resolutions      []resolutionAssertion `json:"resolutions"`
	Types            []typeAssertion       `json:"types"`
	Borrows          []borrowAssertion     `json:"borrows"`
	Diagnostics      []diagnosticAssertion `json:"diagnostics"`
}

type spanAssertion struct {
	Source    string `json:"source"`
	StartByte int    `json:"startByte"`
	EndByte   int    `json:"endByte"`
}

type resolutionAssertion struct {
	Use    spanAssertion    `json:"use"`
	Target symbolCoordinate `json:"target"`
}

type symbolCoordinate struct {
	Module      string        `json:"module"`
	Name        string        `json:"name"`
	Kind        string        `json:"kind"`
	Declaration spanAssertion `json:"declaration"`
}

type typeAssertion struct {
	Expression    spanAssertion `json:"expression"`
	CanonicalType string        `json:"canonicalType"`
}

type borrowAssertion struct {
	Borrow  spanAssertion  `json:"borrow"`
	Kind    string         `json:"kind"`
	Origin  string         `json:"origin"`
	LastUse *spanAssertion `json:"lastUse"`
	Escapes bool           `json:"escapes"`
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

// Deferred identifies a parser-valid case whose owning semantic phase is not implemented yet.
type Deferred struct {
	CaseID string
	Phase  string
}

// Report is deterministic fixture execution evidence.
type Report struct {
	CasesParsed int
	CasesRun    int
	Deferred    []Deferred
	Failures    []Failure
}
