// Package packagecachefixture executes the versioned whole-package cache-key manifest.
package packagecachefixture

type manifest struct {
	Schema        string           `json:"$schema"`
	SchemaVersion string           `json:"schemaVersion"`
	Contracts     []string         `json:"contracts"`
	Base          inputDefinition  `json:"base"`
	Cases         []caseDefinition `json:"cases"`
}

type inputDefinition struct {
	CompilerBuild        string                 `json:"compilerBuild"`
	LanguageRevision     string                 `json:"languageRevision"`
	MIRVersion           string                 `json:"mirVersion"`
	TargetSchema         string                 `json:"targetSchema"`
	ABIProfile           string                 `json:"abiProfile"`
	ToolchainFingerprint string                 `json:"toolchainFingerprint"`
	Backend              string                 `json:"backend"`
	ProfileFlags         []string               `json:"profileFlags"`
	RuntimeABI           string                 `json:"runtimeAbi"`
	StandardLibrary      string                 `json:"standardLibrary"`
	Package              string                 `json:"package"`
	Source               string                 `json:"source"`
	Dependencies         []dependencyDefinition `json:"dependencies"`
	Features             []string               `json:"features"`
}

type dependencyDefinition struct {
	Coordinate string `json:"coordinate"`
	Interface  string `json:"interface"`
}

type inputOverride struct {
	CompilerBuild        *string                 `json:"compilerBuild,omitempty"`
	LanguageRevision     *string                 `json:"languageRevision,omitempty"`
	MIRVersion           *string                 `json:"mirVersion,omitempty"`
	TargetSchema         *string                 `json:"targetSchema,omitempty"`
	ABIProfile           *string                 `json:"abiProfile,omitempty"`
	ToolchainFingerprint *string                 `json:"toolchainFingerprint,omitempty"`
	Backend              *string                 `json:"backend,omitempty"`
	ProfileFlags         *[]string               `json:"profileFlags,omitempty"`
	RuntimeABI           *string                 `json:"runtimeAbi,omitempty"`
	StandardLibrary      *string                 `json:"standardLibrary,omitempty"`
	Package              *string                 `json:"package,omitempty"`
	Source               *string                 `json:"source,omitempty"`
	Dependencies         *[]dependencyDefinition `json:"dependencies,omitempty"`
	Features             *[]string               `json:"features,omitempty"`
}

type caseDefinition struct {
	ID        string        `json:"id"`
	Override  inputOverride `json:"override"`
	CompareTo string        `json:"compareTo,omitempty"`
	Relation  string        `json:"relation"`
	ExpectKey string        `json:"expectKey,omitempty"`
}

type Failure struct {
	CaseID string
	Detail string
}

type Report struct {
	CasesRun int
	Failures []Failure
}
