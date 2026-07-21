// Package packagecontentfixture executes the versioned package-source digest manifest.
package packagecontentfixture

type manifest struct {
	Schema        string           `json:"$schema"`
	SchemaVersion string           `json:"schemaVersion"`
	Contracts     []string         `json:"contracts"`
	Cases         []caseDefinition `json:"cases"`
}

type caseDefinition struct {
	ID         string `json:"id"`
	Files      []file `json:"files"`
	CompareTo  string `json:"compareTo,omitempty"`
	Relation   string `json:"relation"`
	ExpectHash string `json:"expectHash,omitempty"`
}

type file struct {
	Path       string `json:"path"`
	ContentHex string `json:"contentHex"`
}

type Failure struct {
	CaseID string
	Detail string
}

type Report struct {
	CasesRun int
	Failures []Failure
}
