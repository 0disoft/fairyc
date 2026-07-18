// Package mirfixture executes the versioned source-to-MIR acceptance manifest.
package mirfixture

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
	Status           string                 `json:"status"`
	CompletedThrough string                 `json:"completedThrough"`
	Observations     []observationAssertion `json:"observations"`
}

type spanAssertion struct {
	Source    string `json:"source"`
	StartByte int    `json:"startByte"`
	EndByte   int    `json:"endByte"`
}

type cleanupActionAssertion struct {
	Kind    string        `json:"kind"`
	Binding spanAssertion `json:"binding"`
}

type observationAssertion struct {
	Kind                     string                   `json:"kind"`
	Function                 string                   `json:"function"`
	Expression               spanAssertion            `json:"expression"`
	Sequence                 []spanAssertion          `json:"sequence"`
	MaterializedBeforeParent bool                     `json:"materializedBeforeParent"`
	Call                     spanAssertion            `json:"call"`
	ArgumentOrder            []spanAssertion          `json:"argumentOrder"`
	NormalSuccessor          bool                     `json:"normalSuccessor"`
	UnwindSuccessor          bool                     `json:"unwindSuccessor"`
	Binding                  spanAssertion            `json:"binding"`
	Operand                  spanAssertion            `json:"operand"`
	Destination              string                   `json:"destination"`
	CleanupTransferred       bool                     `json:"cleanupObligationTransferred"`
	Exit                     spanAssertion            `json:"exit"`
	ExitKind                 string                   `json:"exitKind"`
	Actions                  []cleanupActionAssertion `json:"actions"`
	Question                 spanAssertion            `json:"question"`
	ResultExpression         spanAssertion            `json:"resultExpression"`
	EvaluationCount          int                      `json:"evaluationCount"`
	SuccessPayload           string                   `json:"successPayload"`
	ErrorPayload             string                   `json:"errorPayload"`
	ResultTemporaryDeadOn    []string                 `json:"resultTemporaryDeadOn"`
	ErrorCleanupActions      []cleanupActionAssertion `json:"errorCleanupActions"`
	Operation                spanAssertion            `json:"operation"`
	Categories               []string                 `json:"categories"`
	CleanupActions           []cleanupActionAssertion `json:"cleanupActions"`
	HasSuccessor             bool                     `json:"hasSuccessor"`
	Form                     string                   `json:"form"`
	RemainingCount           int                      `json:"remainingCount"`
}

// Failure identifies one manifest assertion mismatch.
type Failure struct {
	CaseID string
	Detail string
}

// Report is deterministic source-to-MIR fixture execution evidence.
type Report struct {
	CasesRun int
	Failures []Failure
}
