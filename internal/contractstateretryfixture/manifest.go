// Package contractstateretryfixture executes versioned caller-visible retry
// schedules against the contract-state transaction oracle.
package contractstateretryfixture

type manifest struct {
	Schema        string           `json:"$schema"`
	SchemaVersion string           `json:"schemaVersion"`
	Contracts     []string         `json:"contracts"`
	Limits        limitsDefinition `json:"limits"`
	Cases         []caseDefinition `json:"cases"`
}

type limitsDefinition struct {
	MaxWrites      int    `json:"maxWrites"`
	MaxWriteBytes  uint64 `json:"maxWriteBytes"`
	MaxValueBytes  int    `json:"maxValueBytes"`
	MaxMapKeyBytes int    `json:"maxMapKeyBytes"`
}

type caseDefinition struct {
	ID                string          `json:"id"`
	MaxAttempts       int             `json:"maxAttempts"`
	InjectedConflicts int             `json:"injectedConflicts"`
	Expect            caseExpectation `json:"expect"`
}

type caseExpectation struct {
	Status   string `json:"status"`
	Attempts int    `json:"attempts"`
	Version  uint64 `json:"version"`
	Value    uint64 `json:"value"`
}

// Failure identifies one malformed or behaviorally mismatched retry case.
type Failure struct {
	CaseID string
	Detail string
}

// Report is deterministic explicit-retry evidence.
type Report struct {
	CasesRun int
	Failures []Failure
}
