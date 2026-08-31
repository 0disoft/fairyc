// Package contractstateconflictfixture executes versioned overlapping-transaction schedules.
package contractstateconflictfixture

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
	Kind        string `json:"kind,omitempty"`
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
	ID     string           `json:"id"`
	Steps  []stepDefinition `json:"steps"`
	Expect finalExpectation `json:"expect"`
}

type stepDefinition struct {
	ID          string          `json:"id"`
	Transaction string          `json:"transaction"`
	Action      string          `json:"action"`
	Field       string          `json:"field,omitempty"`
	KeyTypeID   string          `json:"keyTypeID,omitempty"`
	KeyHex      string          `json:"keyHex,omitempty"`
	ValueTypeID string          `json:"valueTypeID,omitempty"`
	ValueHex    string          `json:"valueHex,omitempty"`
	Expect      stepExpectation `json:"expect"`
}

type stepExpectation struct {
	Status   string `json:"status"`
	Found    *bool  `json:"found,omitempty"`
	ValueHex string `json:"valueHex,omitempty"`
	Version  uint64 `json:"version,omitempty"`
	Writes   int    `json:"writes,omitempty"`
}

type finalExpectation struct {
	Version uint64          `json:"version"`
	Store   []expectedEntry `json:"store"`
}

type expectedEntry struct {
	Field         string `json:"field"`
	StorageKeyHex string `json:"storageKeyHex,omitempty"`
	ValueHex      string `json:"valueHex"`
}

// Failure identifies one malformed or behaviorally mismatched schedule.
type Failure struct {
	CaseID string
	Detail string
}

// Report is deterministic overlapping-transaction evidence.
type Report struct {
	CasesRun int
	Failures []Failure
}
