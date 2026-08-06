package workspaceloader

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/packagecontent"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/source"
)

const (
	// TestSourceDirectory is the one root-package-relative source tree admitted only by test
	// workspace loading. Ordinary workspace loading never reads it.
	TestSourceDirectory  = "tests"
	ReasonTestSourceRoot = "invalid-test-source-root"
)

// WithRootTests returns an ephemeral workspace whose root package additionally owns the contained
// tests/ source tree. It preserves the original root-only API while WithPackageTests owns the
// package-selectable implementation.
func WithRootTests(workspace Workspace) (Workspace, error) {
	return WithPackageTests(workspace, ".")
}

// WithWorkspaceTests returns an ephemeral workspace containing the tests/ trees of every
// unambiguous root-reachable local package in the production graph. Catalog-only unreachable
// members remain excluded, and the production workspace and lock inputs are not mutated.
func WithWorkspaceTests(workspace Workspace) (Workspace, error) {
	packagePaths, err := workspace.RootReachableLocalPackagePaths()
	if err != nil {
		return Workspace{}, &LoadError{Reason: ReasonTestSourceRoot, Detail: err.Error()}
	}
	withTests := workspace
	for _, packagePath := range packagePaths {
		withTests, err = WithPackageTests(withTests, packagePath)
		if err != nil {
			return Workspace{}, err
		}
	}
	return withTests, nil
}

// WithPackageTests returns an ephemeral workspace whose selected local package additionally owns
// its contained tests/ source tree. The production workspace is not mutated, and its canonical
// lock remains based only on manifest-owned production sources.
func WithPackageTests(workspace Workspace, packagePath string) (Workspace, error) {
	selectedPackage, found := workspace.Package(packagePath)
	if !found || selectedPackage.SourceKind != SourceKindLocal {
		return Workspace{}, &LoadError{Reason: ReasonTestSourceRoot, Path: packagePath, Detail: "selected local package is missing"}
	}
	if portableDirectoriesOverlap(selectedPackage.Manifest.Package.Source, TestSourceDirectory) {
		return Workspace{}, &LoadError{
			Reason: ReasonTestSourceRoot,
			Path:   TestSourceDirectory,
			Detail: "test sources must not overlap the production source root",
		}
	}
	if packagePath == "." {
		for _, member := range selectedPackage.Manifest.Workspace.Members {
			if portableDirectoriesOverlap(member, TestSourceDirectory) {
				return Workspace{}, &LoadError{
					Reason: ReasonTestSourceRoot,
					Path:   TestSourceDirectory,
					Detail: fmt.Sprintf("test sources must not overlap workspace member %q", member),
				}
			}
		}
	}

	testRoot, present, err := optionalExactDirectory(selectedPackage.Directory, TestSourceDirectory)
	if err != nil {
		return Workspace{}, err
	}
	if !present {
		return workspace, nil
	}

	budget := limits{}
	for _, loaded := range workspace.Packages {
		if loaded.SourceKind != SourceKindLocal {
			continue
		}
		budget.files += len(loaded.Sources)
		for _, file := range loaded.Sources {
			budget.bytes += int64(len(file.Bytes))
		}
	}
	testSources, testModules, err := loadTestSources(testRoot, &budget)
	if err != nil {
		return Workspace{}, err
	}
	if len(testSources) == 0 {
		return workspace, nil
	}

	packages := make([]Package, 0, len(workspace.Packages))
	pathIndexes := make(map[string]int)
	for _, loaded := range workspace.Packages {
		if loaded.SourceKind != SourceKindLocal {
			continue
		}
		loaded.Sources = slices.Clone(loaded.Sources)
		loaded.Modules = slices.Clone(loaded.Modules)
		if loaded.Path == packagePath {
			if err := appendTestSources(&loaded, testSources, testModules); err != nil {
				return Workspace{}, err
			}
		}
		pathIndexes[loaded.Path] = len(packages)
		packages = append(packages, loaded)
	}
	if err := validateDependencyDeclarations(packages, pathIndexes); err != nil {
		return Workspace{}, err
	}
	graph, instances := buildGraph(packages, pathIndexes)
	packages, graph, instances, err = injectStandardLibrary(packages, graph, instances)
	if err != nil {
		return Workspace{}, err
	}
	resolution, rejection, err := packagegraph.Resolve(graph)
	if err != nil {
		return Workspace{}, &LoadError{Reason: ReasonPackageGraph, Detail: err.Error()}
	}
	if rejection != nil {
		return Workspace{}, &LoadError{Reason: ReasonPackageGraph, Detail: fmt.Sprintf("%s: %v", rejection.Reason, rejection.Subjects)}
	}
	return Workspace{
		Root: workspace.Root, Packages: packages, Instances: instances, Graph: graph, Resolution: resolution,
	}, nil
}

func optionalExactDirectory(parent, name string) (string, bool, error) {
	entries, err := os.ReadDir(parent)
	if err != nil {
		return "", false, &LoadError{Reason: ReasonTestSourceRoot, Path: name, Detail: err.Error()}
	}
	var matches []string
	for _, entry := range entries {
		if strings.EqualFold(entry.Name(), name) {
			matches = append(matches, entry.Name())
		}
	}
	if len(matches) == 0 {
		return "", false, nil
	}
	if len(matches) != 1 || matches[0] != name {
		return "", false, &LoadError{Reason: ReasonPathCase, Path: name, Detail: "test source directory does not have one exact on-disk spelling"}
	}
	absolute := filepath.Join(parent, name)
	info, err := os.Lstat(absolute)
	if err != nil {
		return "", false, &LoadError{Reason: ReasonTestSourceRoot, Path: name, Detail: err.Error()}
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return "", false, &LoadError{Reason: ReasonSymlink, Path: name, Detail: "test source root cannot be a symlink"}
	}
	if !info.IsDir() {
		return "", false, &LoadError{Reason: ReasonTestSourceRoot, Path: name, Detail: "test source root is not a directory"}
	}
	return absolute, true, nil
}

func loadTestSources(testRoot string, budget *limits) ([]SourceFile, []packagegraph.Module, error) {
	var relativePaths []string
	foldedPaths := make(map[string]string)
	err := filepath.WalkDir(testRoot, func(absolute string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, relErr := filepath.Rel(testRoot, absolute)
		if relErr != nil {
			return relErr
		}
		virtualPath := path.Join(TestSourceDirectory, filepath.ToSlash(relative))
		if entry.Type()&os.ModeSymlink != 0 {
			return &LoadError{Reason: ReasonSymlink, Path: virtualPath, Detail: "test source tree contains a symlink"}
		}
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".fy" {
			return nil
		}
		info, infoErr := entry.Info()
		if infoErr != nil || !info.Mode().IsRegular() {
			return &LoadError{Reason: ReasonTestSourceRoot, Path: virtualPath, Detail: "test source is not a regular file"}
		}
		folded := strings.ToLower(virtualPath)
		if previous, collision := foldedPaths[folded]; collision {
			return &LoadError{Reason: ReasonSourceCaseCollision, Path: virtualPath, Detail: fmt.Sprintf("collides with %q", previous)}
		}
		foldedPaths[folded] = virtualPath
		relativePaths = append(relativePaths, virtualPath)
		budget.files++
		if budget.files > MaxSourceFiles {
			return &LoadError{Reason: ReasonSourceLimit, Detail: fmt.Sprintf("workspace exceeds %d source files", MaxSourceFiles)}
		}
		return nil
	})
	if err != nil {
		return nil, nil, normalizeLoadError(ReasonTestSourceRoot, TestSourceDirectory, err)
	}
	slices.Sort(relativePaths)
	sources := make([]SourceFile, 0, len(relativePaths))
	modules := make([]packagegraph.Module, 0, len(relativePaths))
	modulePaths := make(map[string]string, len(relativePaths))
	for _, virtualPath := range relativePaths {
		relative := strings.TrimPrefix(virtualPath, TestSourceDirectory+"/")
		absolute := filepath.Join(testRoot, filepath.FromSlash(relative))
		contents, err := readBoundedRegularFile(absolute, source.DefaultLimits().MaxBytes)
		if err != nil {
			return nil, nil, &LoadError{Reason: ReasonTestSourceRoot, Path: virtualPath, Detail: err.Error()}
		}
		budget.bytes += int64(len(contents))
		if budget.bytes > MaxSourceBytes {
			return nil, nil, &LoadError{Reason: ReasonSourceLimit, Detail: fmt.Sprintf("workspace exceeds %d source bytes", MaxSourceBytes)}
		}
		analysis, err := frontend.AnalyzeSyntax(source.NewFile(virtualPath, contents), source.DefaultLimits())
		if err != nil {
			return nil, nil, &LoadError{Reason: ReasonSourceSyntax, Path: virtualPath, Detail: err.Error()}
		}
		if analysis.AST == nil || len(analysis.Tree.Diagnostics) != 0 {
			return nil, nil, &LoadError{
				Reason: ReasonSourceSyntax, Path: virtualPath,
				Detail: fmt.Sprintf("parser diagnostics: %+v", analysis.Tree.Diagnostics),
				Source: slices.Clone(contents), Diagnostics: slices.Clone(analysis.Tree.Diagnostics),
			}
		}
		if previous, duplicate := modulePaths[analysis.AST.Path]; duplicate {
			return nil, nil, &LoadError{Reason: ReasonDuplicateModule, Path: analysis.AST.Path, Detail: fmt.Sprintf("declared by %q and %q", previous, virtualPath)}
		}
		modulePaths[analysis.AST.Path] = virtualPath
		imports := make([]string, len(analysis.AST.Imports))
		for index, imported := range analysis.AST.Imports {
			imports[index] = imported.Path
		}
		sources = append(sources, SourceFile{Path: virtualPath, Bytes: slices.Clone(contents), Module: analysis.AST, Test: true})
		modules = append(modules, packagegraph.Module{Path: analysis.AST.Path, Imports: imports})
	}
	return sources, modules, nil
}

func appendTestSources(loaded *Package, testSources []SourceFile, testModules []packagegraph.Module) error {
	paths := make(map[string]string, len(loaded.Sources)+len(testSources))
	modules := make(map[string]string, len(loaded.Modules)+len(testModules))
	for _, file := range loaded.Sources {
		paths[strings.ToLower(file.Path)] = file.Path
		modules[file.Module.Path] = file.Path
	}
	for _, file := range testSources {
		if previous, duplicate := paths[strings.ToLower(file.Path)]; duplicate {
			return &LoadError{Reason: ReasonSourceCaseCollision, Path: file.Path, Detail: fmt.Sprintf("collides with production source %q", previous)}
		}
		if previous, duplicate := modules[file.Module.Path]; duplicate {
			return &LoadError{Reason: ReasonDuplicateModule, Path: file.Module.Path, Detail: fmt.Sprintf("declared by %q and %q", previous, file.Path)}
		}
		paths[strings.ToLower(file.Path)] = file.Path
		modules[file.Module.Path] = file.Path
	}
	loaded.Sources = append(loaded.Sources, testSources...)
	loaded.Modules = append(loaded.Modules, testModules...)
	slices.SortFunc(loaded.Sources, func(left, right SourceFile) int { return strings.Compare(left.Path, right.Path) })
	slices.SortFunc(loaded.Modules, func(left, right packagegraph.Module) int { return strings.Compare(left.Path, right.Path) })
	digestFiles := make([]packagecontent.File, 0, len(loaded.Sources))
	for _, file := range loaded.Sources {
		digestFiles = append(digestFiles, packagecontent.File{Path: file.Path, Bytes: file.Bytes})
	}
	digest, err := packagecontent.Compute(digestFiles)
	if err != nil {
		return &LoadError{Reason: ReasonTestSourceRoot, Path: TestSourceDirectory, Detail: err.Error()}
	}
	loaded.Digest = digest
	return nil
}

func portableDirectoriesOverlap(left, right string) bool {
	left = strings.ToLower(strings.Trim(left, "/"))
	right = strings.ToLower(strings.Trim(right, "/"))
	return left == right || strings.HasPrefix(left, right+"/") || strings.HasPrefix(right, left+"/")
}
