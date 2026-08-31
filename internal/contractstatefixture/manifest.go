// Package contractstatefixture executes the versioned contract-state transaction corpus.
package contractstatefixture

type manifest struct {
	Schema        string           `json:"$schema"`
	SchemaVersion string           `json:"schemaVersion"`
	Deferred      []string         `json:"deferred"`
	Contracts     []string         `json:"contracts"`
	Layout        layoutDefinition `json:"layout"`
	Limits        limitsDefinition `json:"limits"`
	Cases         []caseDefinition `json:"cases"`
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

type limitsDefinition struct {
	MaxWrites      int    `json:"maxWrites"`
	MaxWriteBytes  uint64 `json:"maxWriteBytes"`
	MaxValueBytes  int    `json:"maxValueBytes"`
	MaxMapKeyBytes int    `json:"maxMapKeyBytes"`
}

type caseDefinition struct {
	ID           string            `json:"id"`
	Operations   []operation       `json:"operations"`
	Exit         string            `json:"exit"`
	TrapCategory string            `json:"trapCategory,omitempty"`
	Limits       *limitsDefinition `json:"limits,omitempty"`
	Expect       expectation       `json:"expect"`
}

type operation struct {
	Kind      string `json:"kind"`
	Field     string `json:"field"`
	KeyType   string `json:"keyTypeID,omitempty"`
	KeyHex    string `json:"keyHex,omitempty"`
	ValueType string `json:"valueTypeID,omitempty"`
	ValueHex  string `json:"valueHex,omitempty"`
}

type expectation struct {
	Status        string          `json:"status"`
	Exit          string          `json:"exit,omitempty"`
	TrapCategory  string          `json:"trapCategory,omitempty"`
	ErrorContains string          `json:"errorContains,omitempty"`
	Version       uint64          `json:"version"`
	Writes        int             `json:"writes"`
	Reads         []expectedRead  `json:"reads"`
	Store         []expectedEntry `json:"store"`
}

type expectedRead struct {
	Operation int    `json:"operation"`
	Found     bool   `json:"found"`
	TypeID    string `json:"typeID"`
	ValueHex  string `json:"valueHex"`
}

type expectedEntry struct {
	Field    string `json:"field"`
	ValueHex string `json:"valueHex"`
}

// Failure identifies one malformed or behaviorally mismatched fixture case.
type Failure struct {
	CaseID string
	Detail string
}

// Report is deterministic contract-state execution evidence.
type Report struct {
	CasesRun int
	Failures []Failure
}
