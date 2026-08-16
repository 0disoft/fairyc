// Package sdkstdlib owns the exact compiler-bundled Stage 0 Fairy standard-library package.
package sdkstdlib

import (
	"embed"
	"fmt"
	"slices"
	"sort"
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
	Version          = "0.0.0-stage0.50"
	LanguageRevision = "fairyc-language/10"
	RuntimeABI       = "fairyc-runtime/0"
	PackagePath      = "@sdk/stdlib"
)

//go:embed sources/std/*.fy
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
	entries, err := sourceFS.ReadDir("sources/std")
	if err != nil {
		return Definition{}, err
	}
	sources := make([]Source, 0, len(entries))
	modules := make([]packagegraph.Module, 0, len(entries))
	digestFiles := make([]packagecontent.File, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".fy") {
			continue
		}
		relative := "std/" + entry.Name()
		contents, readErr := sourceFS.ReadFile("sources/" + relative)
		if readErr != nil {
			return Definition{}, readErr
		}
		analysis, analyzeErr := frontend.AnalyzeSyntax(source.NewFile(relative, contents), source.DefaultLimits())
		if analyzeErr != nil || analysis.AST == nil || len(analysis.Tree.Diagnostics) != 0 {
			return Definition{}, fmt.Errorf("embedded standard library source %q is invalid", relative)
		}
		imports := make([]string, len(analysis.AST.Imports))
		for index, imported := range analysis.AST.Imports {
			imports[index] = imported.Path
		}
		sources = append(sources, Source{Path: relative, Bytes: slices.Clone(contents), Module: analysis.AST})
		modules = append(modules, packagegraph.Module{Path: analysis.AST.Path, Imports: imports})
		digestFiles = append(digestFiles, packagecontent.File{Path: relative, Bytes: contents})
	}
	if len(sources) == 0 {
		return Definition{}, fmt.Errorf("embedded standard library has no Fairy source files")
	}
	sort.Slice(sources, func(left, right int) bool { return sources[left].Path < sources[right].Path })
	sort.Slice(modules, func(left, right int) bool { return modules[left].Path < modules[right].Path })
	sort.Slice(digestFiles, func(left, right int) bool { return digestFiles[left].Path < digestFiles[right].Path })
	digest, err := packagecontent.Compute(digestFiles)
	if err != nil {
		return Definition{}, err
	}
	return Definition{
		Sources: sources, Modules: modules,
		Digest: digest, SourceIdentity: SourceIdentity(SDKIdentity, RuntimeABI, digest),
	}, nil
}

func SourceIdentity(sdkIdentity, runtimeABI string, digest packagecontent.Digest) string {
	return strings.Join([]string{sdkIdentity, runtimeABI, digest.String()}, ";")
}
