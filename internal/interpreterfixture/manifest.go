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
	ID             string                    `json:"id"`
	SourceRoot     string                    `json:"sourceRoot"`
	Sources        []string                  `json:"sources"`
	Entry          string                    `json:"entry"`
	TestIntrinsics []testIntrinsicDefinition `json:"testIntrinsics,omitempty"`
	OutputLimit    int                       `json:"outputLimit,omitempty"`
	Expect         expectation               `json:"expect"`
}

type testIntrinsicDefinition struct {
	Function string `json:"function"`
	Kind     string `json:"kind"`
}

type expectation struct {
	Status           string              `json:"status"`
	CompletedThrough string              `json:"completedThrough"`
	Outcome          string              `json:"outcome"`
	Value            *int32              `json:"value,omitempty"`
	Category         *string             `json:"category,omitempty"`
	Drops            []dropExpectation   `json:"drops,omitempty"`
	Regions          []regionExpectation `json:"regions,omitempty"`
	StdoutBytes      []byte              `json:"stdoutBytes,omitempty"`
	StderrBytes      []byte              `json:"stderrBytes,omitempty"`
	IOTrace          []ioExpectation     `json:"ioTrace,omitempty"`
}

type ioExpectation struct {
	Stream string `json:"stream"`
	Byte   byte   `json:"byte"`
}

type dropExpectation struct {
	ResourceType string          `json:"resourceType"`
	Binding      spanExpectation `json:"binding"`
}

type regionExpectation struct {
	Region               spanExpectation `json:"region"`
	ExitKind             string          `json:"exitKind"`
	Placements           int             `json:"placements"`
	DropCountBeforeReset int             `json:"dropCountBeforeReset"`
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
