// Package regionfixture executes the versioned Region acceptance manifest.
package regionfixture

type manifest struct {
	Schema        string           `json:"$schema"`
	SchemaVersion string           `json:"schemaVersion"`
	Deferred      []string         `json:"deferred"`
	Contracts     []string         `json:"contracts"`
	Cases         []caseDefinition `json:"cases"`
}

type caseDefinition struct {
	ID           string                 `json:"id"`
	SourceRoot   string                 `json:"sourceRoot"`
	Sources      []string               `json:"sources"`
	PackageGraph *packageGraphReference `json:"packageGraph,omitempty"`
	Expect       expectation            `json:"expect"`
}

type packageGraphReference struct {
	Manifest string `json:"manifest"`
	CaseID   string `json:"caseId"`
	RootNode string `json:"rootNode"`
}

type expectation struct {
	Status           string                `json:"status"`
	CompletedThrough string                `json:"completedThrough,omitempty"`
	Phase            string                `json:"phase,omitempty"`
	Origins          []originAssertion     `json:"origins,omitempty"`
	Diagnostics      []diagnosticAssertion `json:"diagnostics,omitempty"`
}

type spanAssertion struct {
	Source    string `json:"source"`
	StartByte int    `json:"startByte"`
	EndByte   int    `json:"endByte"`
}

type originAssertion struct {
	Operation  string         `json:"operation"`
	Allocation spanAssertion  `json:"allocation"`
	Origin     string         `json:"origin"`
	LastUse    *spanAssertion `json:"lastUse,omitempty"`
	Escapes    bool           `json:"escapes"`
}

type diagnosticAssertion struct {
	Code    string                 `json:"code"`
	Primary spanAssertion          `json:"primary"`
	Related []relatedSpanAssertion `json:"related"`
}

type relatedSpanAssertion struct {
	Role string        `json:"role"`
	Span spanAssertion `json:"span"`
}

type packageManifest struct {
	SchemaVersion string        `json:"schemaVersion"`
	Cases         []packageCase `json:"cases"`
}

type packageCase struct {
	ID           string              `json:"id"`
	CaseRoot     string              `json:"caseRoot"`
	Packages     []packageNode       `json:"packages"`
	Dependencies []packageDependency `json:"dependencies"`
	Expect       packageExpectation  `json:"expect"`
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

type packageExpectation struct {
	Status          string                 `json:"status"`
	InstanceClasses []packageInstanceClass `json:"instanceClasses"`
	Lookups         []packageLookup        `json:"lookups"`
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

// Failure identifies one manifest assertion mismatch.
type Failure struct {
	CaseID string
	Detail string
}

// Report is deterministic Region fixture execution evidence.
type Report struct {
	CasesRun int
	Failures []Failure
}
