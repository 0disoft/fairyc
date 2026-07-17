// Package parserfixture executes the versioned parser acceptance manifest.
package parserfixture

import "github.com/0disoft/fairyc/internal/diagnostic"

type manifest struct {
	Schema        string           `json:"$schema"`
	SchemaVersion string           `json:"schemaVersion"`
	Grammar       string           `json:"grammar"`
	Cases         []caseDefinition `json:"cases"`
}

type caseDefinition struct {
	ID         string      `json:"id"`
	SourceRoot string      `json:"sourceRoot"`
	Source     string      `json:"source"`
	Expect     expectation `json:"expect"`
}

type expectation struct {
	Status      string                `json:"status"`
	Module      string                `json:"module"`
	Expressions []expressionAssertion `json:"expressions"`
	Literals    []literalAssertion    `json:"literals"`
	Diagnostics []diagnosticAssertion `json:"diagnostics"`
}

type expressionAssertion struct {
	Binding string `json:"binding"`
	Tree    string `json:"tree"`
}

type literalAssertion struct {
	Raw    string `json:"raw"`
	Kind   string `json:"kind"`
	Cooked string `json:"cooked"`
}

type diagnosticAssertion struct {
	Code   string          `json:"code"`
	Span   diagnostic.Span `json:"span"`
	Repair string          `json:"repair"`
	Token  string          `json:"token"`
}

// Failure identifies one manifest assertion mismatch.
type Failure struct {
	CaseID string
	Detail string
}

// Report is deterministic fixture execution evidence.
type Report struct {
	CasesRun int
	Failures []Failure
}
