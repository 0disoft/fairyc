// Package packagefixture executes the versioned resolved-package-graph acceptance manifest.
package packagefixture

import (
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/semantic"
)

type manifest struct {
	Schema        string           `json:"$schema"`
	SchemaVersion string           `json:"schemaVersion"`
	Includes      []string         `json:"includes,omitempty"`
	Deferred      []string         `json:"deferred"`
	Contracts     []string         `json:"contracts"`
	Cases         []caseDefinition `json:"cases"`
}

type caseDefinition struct {
	ID           string              `json:"id"`
	CaseRoot     string              `json:"caseRoot"`
	Packages     []packageNode       `json:"packages"`
	Dependencies []packageDependency `json:"dependencies"`
	Expect       expectation         `json:"expect"`
}

type packageNode struct {
	Node            string   `json:"node"`
	Role            string   `json:"role"`
	PackageUID      string   `json:"packageUid"`
	SourceIdentity  string   `json:"sourceIdentity"`
	VersionIdentity string   `json:"versionIdentity"`
	Features        []string `json:"features"`
	SourceRoot      string   `json:"sourceRoot"`
	Sources         []string `json:"sources"`
}

type packageDependency struct {
	Owner   string `json:"owner"`
	Binding string `json:"binding"`
	Target  string `json:"target"`
}

type expectation struct {
	Status          string                 `json:"status"`
	InstanceClasses []packageInstanceClass `json:"instanceClasses,omitempty"`
	Lookups         []packageLookup        `json:"lookups,omitempty"`
	Reason          string                 `json:"reason,omitempty"`
	Subjects        []string               `json:"subjects,omitempty"`
	ImportPath      string                 `json:"importPath,omitempty"`
}

type packageInstanceClass struct {
	Label   string   `json:"label"`
	Members []string `json:"members"`
}

type packageLookup struct {
	From          string `json:"from"`
	ImportPath    string `json:"importPath"`
	ProviderClass string `json:"providerClass"`
	ModulePath    string `json:"modulePath"`
}

// Failure identifies one manifest assertion or fixture-integrity mismatch.
type Failure struct {
	CaseID string
	Detail string
}

// Report is deterministic resolved-package-graph fixture execution evidence.
type Report struct {
	CasesRun int
	Failures []Failure
}

// ResolvedNode is one validated fixture node with its contained source root and parsed modules.
type ResolvedNode struct {
	ID         string
	Role       string
	SourceRoot string
	Sources    []string
	Modules    []packagegraph.Module
	Files      []semantic.File
}

// ResolvedCase exposes one fully validated fixture case to other compiler acceptance runners.
// Paths are local test-harness facts and are never a package format or compiler output.
type ResolvedCase struct {
	ID             string
	ExpectedStatus string
	Nodes          []ResolvedNode
	Dependencies   []packagegraph.Dependency
	Resolution     packagegraph.Result
	Rejection      *packagegraph.Rejection
}

// Node returns one fixture-local package node by id.
func (c ResolvedCase) Node(id string) (ResolvedNode, bool) {
	for _, node := range c.Nodes {
		if node.ID == id {
			return node, true
		}
	}
	return ResolvedNode{}, false
}
