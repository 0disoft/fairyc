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

//go:embed candidates/std/vec.fy
var vectorCandidate []byte

// LoadVectorCandidate adds the SDK-owned container candidate to the complete
// stable bundle. Only the explicit private storage-transfer compiler lane may
// consume it; Load and the installed SDK identity remain unchanged.
func LoadVectorCandidate() (Definition, error) {
	definition, err := Load()
	if err != nil {
		return Definition{}, err
	}
	const path = "std/vec.fy"
	contents := slices.Clone(vectorCandidate)
	analysis, err := frontend.AnalyzeSyntax(source.NewFile(path, contents), source.DefaultLimits())
	if err != nil || analysis.AST == nil || len(analysis.Tree.Diagnostics) != 0 || analysis.AST.Path != "std::vec" {
		return Definition{}, fmt.Errorf("embedded vector candidate is invalid")
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
	definition.SourceIdentity = SourceIdentity("fairyc-sdk-vector-candidate/0", RuntimeABI, definition.Digest)
	return definition, nil
}
