// Package projectmanifestfixture executes versioned human project-manifest fixtures.
package projectmanifestfixture

type manifest struct {
	Schema        string           `json:"$schema"`
	SchemaVersion string           `json:"schemaVersion"`
	Contracts     []string         `json:"contracts"`
	Cases         []caseDefinition `json:"cases"`
}

type caseDefinition struct {
	ID     string      `json:"id"`
	File   string      `json:"file"`
	Expect expectation `json:"expect"`
}

type expectation struct {
	Status       string   `json:"status"`
	Reason       string   `json:"reason,omitempty"`
	Field        string   `json:"field,omitempty"`
	PackageUID   string   `json:"packageUid,omitempty"`
	Members      []string `json:"members,omitempty"`
	Dependencies []string `json:"dependencies,omitempty"`
}

type Failure struct {
	CaseID string
	Detail string
}

type Report struct {
	CasesRun int
	Failures []Failure
}
