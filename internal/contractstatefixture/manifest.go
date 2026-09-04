// Package contractstatefixture executes the versioned contract-state transaction corpus.
package contractstatefixture

type manifest struct {
	Schema        string               `json:"$schema"`
	SchemaVersion string               `json:"schemaVersion"`
	Deferred      []string             `json:"deferred"`
	Contracts     []string             `json:"contracts"`
	Layout        layoutDefinition     `json:"layout"`
	Limits        limitsDefinition     `json:"limits"`
	Cases         []caseDefinition     `json:"cases"`
	Mutations     []mutationDefinition `json:"mutations,omitempty"`
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
	Entry        uint32            `json:"entry,omitempty"`
	Blocks       []blockDefinition `json:"blocks,omitempty"`
}

type operation struct {
	Kind       string `json:"kind"`
	Field      string `json:"field"`
	KeyType    string `json:"keyTypeID,omitempty"`
	KeyHex     string `json:"keyHex,omitempty"`
	ValueType  string `json:"valueTypeID,omitempty"`
	ValueHex   string `json:"valueHex,omitempty"`
	KeyInput   uint32 `json:"keyInput,omitempty"`
	ValueInput uint32 `json:"valueInput,omitempty"`
	Result     uint32 `json:"result,omitempty"`
	Found      uint32 `json:"found,omitempty"`
	Cost       uint32 `json:"cost,omitempty"`
}

type blockDefinition struct {
	ID         uint32               `json:"id"`
	Parameters []blockParameter     `json:"parameters"`
	Operations []operation          `json:"operations"`
	Terminator terminatorDefinition `json:"terminator"`
}

type blockParameter struct {
	ID     uint32 `json:"id"`
	TypeID string `json:"typeID"`
}

type terminatorDefinition struct {
	Kind           string   `json:"kind"`
	Condition      uint32   `json:"condition,omitempty"`
	Target         uint32   `json:"target,omitempty"`
	Arguments      []uint32 `json:"arguments"`
	TrueTarget     uint32   `json:"trueTarget,omitempty"`
	TrueArguments  []uint32 `json:"trueArguments"`
	FalseTarget    uint32   `json:"falseTarget,omitempty"`
	FalseArguments []uint32 `json:"falseArguments"`
	TrapCategory   string   `json:"trapCategory,omitempty"`
	Cost           uint32   `json:"cost"`
}

type mutationDefinition struct {
	ID            string `json:"id"`
	Case          string `json:"case"`
	Kind          string `json:"kind"`
	ErrorContains string `json:"errorContains"`
}

type expectation struct {
	Status        string          `json:"status"`
	Exit          string          `json:"exit,omitempty"`
	TrapCategory  string          `json:"trapCategory,omitempty"`
	ErrorContains string          `json:"errorContains,omitempty"`
	Version       uint64          `json:"version"`
	Writes        int             `json:"writes"`
	Cost          uint64          `json:"cost,omitempty"`
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
	Field         string `json:"field"`
	StorageKeyHex string `json:"storageKeyHex,omitempty"`
	ValueHex      string `json:"valueHex"`
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
