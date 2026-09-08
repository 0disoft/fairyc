package sdkstdlib

import (
	_ "embed"
	"fmt"
	"slices"

	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/packagecontent"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/source"
)

//go:embed sources/optional/std/vec.fy
var vectorSource []byte

const VectorCandidateIdentity = "fairyc-sdk-vector-candidate/0"

// LoadVectorCandidate retains an explicit, separately identified test lane.
func LoadVectorCandidate() (Definition, error) {
	definition, err := LoadVector()
	if err != nil {
		return Definition{}, err
	}
	definition.SourceIdentity = SourceIdentity(VectorCandidateIdentity, RuntimeABI, definition.Digest)
	return definition, nil
}

// LoadVector adds the on-demand std::vec module to the core SDK. The loader
// selects this exact bundle only when a resolved source imports std::vec.
func LoadVector() (Definition, error) {
	definition, err := Load()
	if err != nil {
		return Definition{}, err
	}
	const path = "std/vec.fy"
	contents := slices.Clone(vectorSource)
	analysis, err := frontend.AnalyzeSyntax(source.NewFile(path, contents), source.DefaultLimits())
	if err != nil || analysis.AST == nil || len(analysis.Tree.Diagnostics) != 0 || analysis.AST.Path != "std::vec" {
		return Definition{}, fmt.Errorf("embedded vector module is invalid")
	}
	imports := make([]string, len(analysis.AST.Imports))
	for i, imported := range analysis.AST.Imports {
		imports[i] = imported.Path
	}
	definition.Sources = append(definition.Sources, Source{Path: path, Bytes: contents, Module: analysis.AST})
	definition.Modules = append(definition.Modules, packagegraph.Module{Path: analysis.AST.Path, Imports: imports})
	files := make([]packagecontent.File, len(definition.Sources))
	for i, file := range definition.Sources {
		files[i] = packagecontent.File{Path: file.Path, Bytes: file.Bytes}
	}
	definition.Digest, err = packagecontent.Compute(files)
	if err != nil {
		return Definition{}, err
	}
	definition.SourceIdentity = SourceIdentity(SDKIdentity, RuntimeABI, definition.Digest)
	return definition, nil
}
