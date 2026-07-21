// Package sdkstdlib owns the exact compiler-bundled Stage 0 Fairy standard-library package.
package sdkstdlib

import (
	"embed"
	"fmt"
	"slices"
	"strings"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/packagecontent"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/source"
)

const (
	SDKIdentity      = "fairyc-sdk-stdlib/0"
	PackageUID       = "fairyc.std"
	PackageName      = "fairyc-std"
	Version          = "0.0.0-stage0.1"
	LanguageRevision = "fairyc-language/0"
	RuntimeABI       = "fairyc-runtime/0"
	PackagePath      = "@sdk/stdlib"
)

//go:embed sources/std/alloc.fy
var sourceFS embed.FS

type Source struct {
	Path   string
	Bytes  []byte
	Module *ast.Module
}

type Definition struct {
	Sources        []Source
	Modules        []packagegraph.Module
	Digest         packagecontent.Digest
	SourceIdentity string
}

func Load() (Definition, error) {
	const relative = "std/alloc.fy"
	contents, err := sourceFS.ReadFile("sources/" + relative)
	if err != nil {
		return Definition{}, err
	}
	analysis, err := frontend.AnalyzeSyntax(source.NewFile(relative, contents), source.DefaultLimits())
	if err != nil || analysis.AST == nil || len(analysis.Tree.Diagnostics) != 0 {
		return Definition{}, fmt.Errorf("embedded standard library source is invalid")
	}
	imports := make([]string, len(analysis.AST.Imports))
	for index, imported := range analysis.AST.Imports {
		imports[index] = imported.Path
	}
	digest, err := packagecontent.Compute([]packagecontent.File{{Path: relative, Bytes: contents}})
	if err != nil {
		return Definition{}, err
	}
	return Definition{
		Sources: []Source{{Path: relative, Bytes: slices.Clone(contents), Module: analysis.AST}},
		Modules: []packagegraph.Module{{Path: analysis.AST.Path, Imports: imports}},
		Digest:  digest, SourceIdentity: SourceIdentity(SDKIdentity, RuntimeABI, digest),
	}, nil
}

func SourceIdentity(sdkIdentity, runtimeABI string, digest packagecontent.Digest) string {
	return strings.Join([]string{sdkIdentity, runtimeABI, digest.String()}, ";")
}
