// Package contractstatelayoutfixture executes versioned contract-state layout
// publication schedules against the in-memory semantic oracle.
package contractstatelayoutfixture

type manifest struct {
	Schema        string            `json:"$schema"`
	SchemaVersion string            `json:"schemaVersion"`
	Contracts     []string          `json:"contracts"`
	Limits        limitsDefinition  `json:"limits"`
	BaseLayout    layoutDefinition  `json:"baseLayout"`
	Seed          []writeDefinition `json:"seed"`
	Pending       []writeDefinition `json:"pending"`
	Cases         []caseDefinition  `json:"cases"`
}

type limitsDefinition struct {
	MaxWrites      int    `json:"maxWrites"`
	MaxWriteBytes  uint64 `json:"maxWriteBytes"`
	MaxValueBytes  int    `json:"maxValueBytes"`
	MaxMapKeyBytes int    `json:"maxMapKeyBytes"`
}

type layoutDefinition struct {
	ContractUID string            `json:"contractUID"`
	Fields      []fieldDefinition `json:"fields"`
}

type fieldDefinition struct {
	Name        string `json:"name"`
	Kind        string `json:"kind"`
	KeyTypeID   string `json:"keyTypeID,omitempty"`
	ValueTypeID string `json:"valueTypeID"`
}

type writeDefinition struct {
	Field    string `json:"field"`
	KeyHex   string `json:"keyHex,omitempty"`
	ValueHex string `json:"valueHex"`
}

type caseDefinition struct {
	ID        string           `json:"id"`
	Candidate layoutDefinition `json:"candidate"`
	Expect    string           `json:"expect"`
}

// Failure identifies one malformed or behaviorally mismatched layout case.
type Failure struct {
	CaseID string
	Detail string
}

// Report is deterministic layout-publication evidence.
type Report struct {
	CasesRun int
	Failures []Failure
}
