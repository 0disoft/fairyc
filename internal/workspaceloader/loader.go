// Package workspaceloader turns one local-exact Fairy C workspace into a validated package graph.
package workspaceloader

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/packagecontent"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/projectmanifest"
	"github.com/0disoft/fairyc/internal/sdkstdlib"
	"github.com/0disoft/fairyc/internal/source"
)

const (
	MaxPackages    = 1024
	MaxSourceFiles = 4096
	MaxSourceBytes = 64 << 20

	ReasonRoot                = "invalid-workspace-root"
	ReasonPath                = "invalid-workspace-path"
	ReasonPathCase            = "noncanonical-path-case"
	ReasonSymlink             = "symlink-not-supported"
	ReasonManifest            = "invalid-project-manifest"
	ReasonNestedWorkspace     = "nested-workspace"
	ReasonOverlappingMember   = "overlapping-workspace-member"
	ReasonDuplicatePackageUID = "duplicate-package-uid"
	ReasonUnknownDependency   = "unknown-path-dependency"
	ReasonDependencyVersion   = "dependency-version-mismatch"
	ReasonLanguageMismatch    = "language-revision-mismatch"
	ReasonSourceRoot          = "invalid-source-root"
	ReasonSourceCaseCollision = "source-path-case-collision"
	ReasonSourceLimit         = "source-limit-exceeded"
	ReasonSourceSyntax        = "invalid-source-syntax"
	ReasonDuplicateModule     = "duplicate-module-path"
	ReasonPackageGraph        = "invalid-package-graph"
	ReasonStandardLibrary     = "invalid-standard-library"
)

const (
	SourceKindLocal = "local"
	SourceKindSDK   = "sdk"
)

// LoadError provides stable internal failure categories without freezing public diagnostics.
type LoadError struct {
	Reason string
	Path   string
	Detail string
}

func (e *LoadError) Error() string {
	if e.Path == "" {
		return fmt.Sprintf("workspace %s: %s", e.Reason, e.Detail)
	}
	return fmt.Sprintf("workspace %s %s: %s", e.Path, e.Reason, e.Detail)
}

// SourceFile preserves the exact source bytes and parsed module needed by later compiler stages.
type SourceFile struct {
	Path   string
	Bytes  []byte
	Module *ast.Module
}

// Package is one source catalog package, either an explicitly declared local member or an exact
// compiler-bundled SDK package.
type Package struct {
	Path        string
	Directory   string
	SourceRoot  string
	Manifest    projectmanifest.Manifest
	Sources     []SourceFile
	Modules     []packagegraph.Module
	Digest      packagecontent.Digest
	SourceKind  string
	SDKIdentity string
	RuntimeABI  string
}

// Workspace is the validated catalog and root-reachable, feature-distinct resolved graph.
type Workspace struct {
	Root       string
	Packages   []Package
	Instances  []Instance
	Graph      packagegraph.Graph
	Resolution packagegraph.Result
}

// Instance maps one internal graph node to its canonical local package path and feature set.
type Instance struct {
	NodeID      string
	PackagePath string
	Features    []string
}

// Package returns one catalog package by its canonical workspace-relative path.
func (workspace Workspace) Package(path string) (Package, bool) {
	for _, loaded := range workspace.Packages {
		if loaded.Path == path {
			return loaded, true
		}
	}
	return Package{}, false
}

type limits struct {
	files int
	bytes int64
}

// Load reads only the named root manifest, its explicit members, and their explicit source roots.
// It performs no parent search, recursive manifest discovery, network access, or lockfile writes.
func Load(root string) (Workspace, error) {
	workspaceRoot, err := resolveRoot(root)
	if err != nil {
		return Workspace{}, err
	}
	rootManifest, err := readManifest(workspaceRoot, ".")
	if err != nil {
		return Workspace{}, err
	}
	if len(rootManifest.Workspace.Members)+1 > MaxPackages {
		return Workspace{}, &LoadError{Reason: ReasonPath, Detail: fmt.Sprintf("workspace exceeds %d packages", MaxPackages)}
	}
	if err := rejectOverlappingMembers(rootManifest.Workspace.Members); err != nil {
		return Workspace{}, err
	}

	packagePaths := make([]string, 0, len(rootManifest.Workspace.Members)+1)
	packagePaths = append(packagePaths, ".")
	packagePaths = append(packagePaths, rootManifest.Workspace.Members...)
	packages := make([]Package, 0, len(packagePaths))
	uidPaths := make(map[string]string, len(packagePaths))
	pathIndexes := make(map[string]int, len(packagePaths))
	budget := limits{}
	for _, packagePath := range packagePaths {
		directory := workspaceRoot
		manifest := rootManifest
		if packagePath != "." {
			directory, err = resolveDirectory(workspaceRoot, packagePath)
			if err != nil {
				return Workspace{}, err
			}
			manifest, err = readManifest(directory, packagePath)
			if err != nil {
				return Workspace{}, err
			}
			if manifest.Workspace.Present {
				return Workspace{}, &LoadError{Reason: ReasonNestedWorkspace, Path: packagePath, Detail: "member manifests cannot declare [workspace]"}
			}
		}
		if previous, duplicate := uidPaths[manifest.Package.UID]; duplicate {
			return Workspace{}, &LoadError{Reason: ReasonDuplicatePackageUID, Path: packagePath, Detail: fmt.Sprintf("package UID %q is also declared by %q", manifest.Package.UID, previous)}
		}
		uidPaths[manifest.Package.UID] = packagePath
		sourceRoot, err := resolveDirectory(directory, manifest.Package.Source)
		if err != nil {
			return Workspace{}, &LoadError{Reason: ReasonSourceRoot, Path: packagePath + "/" + manifest.Package.Source, Detail: err.Error()}
		}
		loaded, err := loadPackage(packagePath, directory, sourceRoot, manifest, &budget)
		if err != nil {
			return Workspace{}, err
		}
		pathIndexes[packagePath] = len(packages)
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
	return Workspace{Root: workspaceRoot, Packages: packages, Instances: instances, Graph: graph, Resolution: resolution}, nil
}

func resolveRoot(root string) (string, error) {
	absolute, err := filepath.Abs(root)
	if err != nil {
		return "", &LoadError{Reason: ReasonRoot, Detail: err.Error()}
	}
	info, err := os.Lstat(absolute)
	if err != nil {
		return "", &LoadError{Reason: ReasonRoot, Path: absolute, Detail: err.Error()}
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return "", &LoadError{Reason: ReasonSymlink, Path: absolute, Detail: "workspace root cannot be a symlink"}
	}
	if !info.IsDir() {
		return "", &LoadError{Reason: ReasonRoot, Path: absolute, Detail: "workspace root is not a directory"}
	}
	return filepath.Clean(absolute), nil
}

func readManifest(directory, packagePath string) (projectmanifest.Manifest, error) {
	path := filepath.Join(directory, projectmanifest.FileName)
	contents, err := readBoundedRegularFile(path, projectmanifest.MaxBytes)
	if err != nil {
		return projectmanifest.Manifest{}, &LoadError{Reason: ReasonManifest, Path: packagePath + "/" + projectmanifest.FileName, Detail: err.Error()}
	}
	manifest, err := projectmanifest.Parse(contents)
	if err != nil {
		return projectmanifest.Manifest{}, &LoadError{Reason: ReasonManifest, Path: packagePath + "/" + projectmanifest.FileName, Detail: err.Error()}
	}
	return manifest, nil
}

func resolveDirectory(base, relative string) (string, error) {
	current := base
	for _, segment := range strings.Split(relative, "/") {
		entries, err := os.ReadDir(current)
		if err != nil {
			return "", &LoadError{Reason: ReasonPath, Path: relative, Detail: err.Error()}
		}
		folded := make([]string, 0, 1)
		for _, entry := range entries {
			if strings.EqualFold(entry.Name(), segment) {
				folded = append(folded, entry.Name())
			}
		}
		if len(folded) != 1 || folded[0] != segment {
			return "", &LoadError{Reason: ReasonPathCase, Path: relative, Detail: fmt.Sprintf("segment %q does not have one exact on-disk spelling", segment)}
		}
		current = filepath.Join(current, segment)
		info, err := os.Lstat(current)
		if err != nil {
			return "", &LoadError{Reason: ReasonPath, Path: relative, Detail: err.Error()}
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return "", &LoadError{Reason: ReasonSymlink, Path: relative, Detail: "directory path contains a symlink"}
		}
		if !info.IsDir() {
			return "", &LoadError{Reason: ReasonPath, Path: relative, Detail: "path is not a directory"}
		}
	}
	return current, nil
}

func rejectOverlappingMembers(members []string) error {
	for index, member := range members {
		folded := strings.ToLower(member)
		for previous := 0; previous < index; previous++ {
			other := strings.ToLower(members[previous])
			if folded == other {
				return &LoadError{Reason: ReasonPathCase, Path: member, Detail: fmt.Sprintf("collides with member %q", members[previous])}
			}
			if strings.HasPrefix(folded, other+"/") || strings.HasPrefix(other, folded+"/") {
				return &LoadError{Reason: ReasonOverlappingMember, Path: member, Detail: fmt.Sprintf("overlaps member %q", members[previous])}
			}
		}
	}
	return nil
}

func loadPackage(packagePath, directory, sourceRoot string, manifest projectmanifest.Manifest, budget *limits) (Package, error) {
	var relativePaths []string
	foldedPaths := make(map[string]string)
	err := filepath.WalkDir(sourceRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return &LoadError{Reason: ReasonSymlink, Path: portablePath(packagePath, sourceRoot, path), Detail: "source tree contains a symlink"}
		}
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".fy" {
			return nil
		}
		info, err := entry.Info()
		if err != nil || !info.Mode().IsRegular() {
			return &LoadError{Reason: ReasonSourceRoot, Path: portablePath(packagePath, sourceRoot, path), Detail: "source is not a regular file"}
		}
		relative, err := filepath.Rel(sourceRoot, path)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		folded := strings.ToLower(relative)
		if previous, collision := foldedPaths[folded]; collision {
			return &LoadError{Reason: ReasonSourceCaseCollision, Path: relative, Detail: fmt.Sprintf("collides with %q", previous)}
		}
		foldedPaths[folded] = relative
		relativePaths = append(relativePaths, relative)
		budget.files++
		if budget.files > MaxSourceFiles {
			return &LoadError{Reason: ReasonSourceLimit, Detail: fmt.Sprintf("workspace exceeds %d source files", MaxSourceFiles)}
		}
		return nil
	})
	if err != nil {
		return Package{}, normalizeLoadError(ReasonSourceRoot, packagePath, err)
	}
	if len(relativePaths) == 0 {
		return Package{}, &LoadError{Reason: ReasonSourceRoot, Path: packagePath, Detail: "source root contains no .fy files"}
	}
	slices.Sort(relativePaths)
	sources := make([]SourceFile, 0, len(relativePaths))
	digestFiles := make([]packagecontent.File, 0, len(relativePaths))
	modules := make([]packagegraph.Module, 0, len(relativePaths))
	modulePaths := make(map[string]string, len(relativePaths))
	for _, relative := range relativePaths {
		absolute := filepath.Join(sourceRoot, filepath.FromSlash(relative))
		contents, err := readBoundedRegularFile(absolute, source.DefaultLimits().MaxBytes)
		if err != nil {
			return Package{}, &LoadError{Reason: ReasonSourceRoot, Path: portablePath(packagePath, sourceRoot, absolute), Detail: err.Error()}
		}
		budget.bytes += int64(len(contents))
		if budget.bytes > MaxSourceBytes {
			return Package{}, &LoadError{Reason: ReasonSourceLimit, Detail: fmt.Sprintf("workspace exceeds %d source bytes", MaxSourceBytes)}
		}
		analysis, err := frontend.AnalyzeSyntax(source.NewFile(relative, contents), source.DefaultLimits())
		if err != nil {
			return Package{}, &LoadError{Reason: ReasonSourceSyntax, Path: portablePath(packagePath, sourceRoot, absolute), Detail: err.Error()}
		}
		if analysis.AST == nil || len(analysis.Tree.Diagnostics) != 0 {
			return Package{}, &LoadError{Reason: ReasonSourceSyntax, Path: portablePath(packagePath, sourceRoot, absolute), Detail: fmt.Sprintf("parser diagnostics: %+v", analysis.Tree.Diagnostics)}
		}
		if previous, duplicate := modulePaths[analysis.AST.Path]; duplicate {
			return Package{}, &LoadError{Reason: ReasonDuplicateModule, Path: analysis.AST.Path, Detail: fmt.Sprintf("declared by %q and %q", previous, relative)}
		}
		modulePaths[analysis.AST.Path] = relative
		imports := make([]string, len(analysis.AST.Imports))
		for index, imported := range analysis.AST.Imports {
			imports[index] = imported.Path
		}
		modules = append(modules, packagegraph.Module{Path: analysis.AST.Path, Imports: imports})
		sources = append(sources, SourceFile{Path: relative, Bytes: slices.Clone(contents), Module: analysis.AST})
		digestFiles = append(digestFiles, packagecontent.File{Path: relative, Bytes: contents})
	}
	slices.SortFunc(modules, func(left, right packagegraph.Module) int { return strings.Compare(left.Path, right.Path) })
	digest, err := packagecontent.Compute(digestFiles)
	if err != nil {
		return Package{}, &LoadError{Reason: ReasonSourceRoot, Path: packagePath, Detail: err.Error()}
	}
	return Package{Path: packagePath, Directory: directory, SourceRoot: sourceRoot, Manifest: manifest, Sources: sources, Modules: modules, Digest: digest, SourceKind: SourceKindLocal}, nil
}

func injectStandardLibrary(packages []Package, graph packagegraph.Graph, instances []Instance) ([]Package, packagegraph.Graph, []Instance, error) {
	owners := make([]string, 0)
	for _, node := range graph.Nodes {
		needsStandardLibrary := false
		for _, module := range node.Modules {
			for _, imported := range module.Imports {
				if imported == "std" || strings.HasPrefix(imported, "std::") {
					needsStandardLibrary = true
				}
			}
		}
		if needsStandardLibrary {
			owners = append(owners, node.ID)
		}
	}
	if len(owners) == 0 {
		return packages, graph, instances, nil
	}
	if packages[0].Manifest.Package.Language != sdkstdlib.LanguageRevision {
		return nil, packagegraph.Graph{}, nil, &LoadError{Reason: ReasonStandardLibrary, Detail: fmt.Sprintf("SDK requires %s", sdkstdlib.LanguageRevision)}
	}
	definition, err := sdkstdlib.Load()
	if err != nil {
		return nil, packagegraph.Graph{}, nil, &LoadError{Reason: ReasonStandardLibrary, Detail: err.Error()}
	}
	manifest := projectmanifest.Manifest{Schema: projectmanifest.SchemaVersion, Package: projectmanifest.Package{UID: sdkstdlib.PackageUID, Name: sdkstdlib.PackageName, Version: sdkstdlib.Version, Language: sdkstdlib.LanguageRevision, Source: "src"}}
	sources := make([]SourceFile, len(definition.Sources))
	for index, file := range definition.Sources {
		sources[index] = SourceFile{Path: file.Path, Bytes: slices.Clone(file.Bytes), Module: file.Module}
	}
	packages = append(packages, Package{
		Path: sdkstdlib.PackagePath, Manifest: manifest, Sources: sources,
		Modules: slices.Clone(definition.Modules), Digest: definition.Digest,
		SourceKind: SourceKindSDK, SDKIdentity: sdkstdlib.SDKIdentity, RuntimeABI: sdkstdlib.RuntimeABI,
	})
	const standardLibraryNode = "sdk:stdlib[]"
	graph.Nodes = append(graph.Nodes, packagegraph.Node{
		ID: standardLibraryNode, Role: packagegraph.RoleStandardLibrary,
		Identity: packagegraph.Identity{PackageUID: sdkstdlib.PackageUID, SourceIdentity: definition.SourceIdentity, VersionIdentity: sdkstdlib.Version, Features: []string{}},
		Modules:  slices.Clone(definition.Modules),
	})
	instances = append(instances, Instance{NodeID: standardLibraryNode, PackagePath: sdkstdlib.PackagePath, Features: []string{}})
	for _, owner := range owners {
		graph.Dependencies = append(graph.Dependencies, packagegraph.Dependency{Owner: owner, Binding: "stdlib", Target: standardLibraryNode})
	}
	return packages, graph, instances, nil
}

func validateDependencyDeclarations(packages []Package, pathIndexes map[string]int) error {
	rootLanguage := packages[0].Manifest.Package.Language
	for _, loaded := range packages {
		if loaded.Manifest.Package.UID == sdkstdlib.PackageUID {
			return &LoadError{Reason: ReasonStandardLibrary, Path: loaded.Path, Detail: fmt.Sprintf("package UID %q is reserved by the SDK", sdkstdlib.PackageUID)}
		}
		for _, module := range loaded.Modules {
			if module.Path == "std" || strings.HasPrefix(module.Path, "std::") {
				return &LoadError{Reason: ReasonStandardLibrary, Path: loaded.Path + ":" + module.Path, Detail: "module namespace is reserved by the SDK"}
			}
		}
		if loaded.Manifest.Package.Language != rootLanguage {
			return &LoadError{Reason: ReasonLanguageMismatch, Path: loaded.Path, Detail: fmt.Sprintf("got %q, root requires %q", loaded.Manifest.Package.Language, rootLanguage)}
		}
		for _, dependency := range loaded.Manifest.Dependencies {
			if dependency.Binding == "stdlib" {
				return &LoadError{Reason: ReasonStandardLibrary, Path: loaded.Path + ":" + dependency.Binding, Detail: "dependency binding is reserved for SDK injection"}
			}
			targetIndex, found := pathIndexes[dependency.Path]
			if !found || dependency.Path == "." {
				return &LoadError{Reason: ReasonUnknownDependency, Path: loaded.Path + ":" + dependency.Binding, Detail: fmt.Sprintf("path %q is not an explicit workspace member", dependency.Path)}
			}
			target := packages[targetIndex]
			if dependency.Version != target.Manifest.Package.Version {
				return &LoadError{Reason: ReasonDependencyVersion, Path: loaded.Path + ":" + dependency.Binding, Detail: fmt.Sprintf("requires %q, member %q declares %q", dependency.Version, dependency.Path, target.Manifest.Package.Version)}
			}
		}
	}
	return nil
}

func buildGraph(packages []Package, pathIndexes map[string]int) (packagegraph.Graph, []Instance) {
	type instance struct {
		path     string
		features []string
	}
	root := instance{path: "."}
	queue := []instance{root}
	seen := map[string]string{instanceKey(root): nodeID(root)}
	graph := packagegraph.Graph{}
	instances := make([]Instance, 0)
	for next := 0; next < len(queue); next++ {
		current := queue[next]
		loaded := packages[pathIndexes[current.path]]
		role := packagegraph.RoleDependency
		if current.path == "." {
			role = packagegraph.RoleRoot
		}
		graph.Nodes = append(graph.Nodes, packagegraph.Node{
			ID: nodeID(current), Role: role,
			Identity: packagegraph.Identity{
				PackageUID: loaded.Manifest.Package.UID, SourceIdentity: loaded.Digest.String(),
				VersionIdentity: loaded.Manifest.Package.Version, Features: slices.Clone(current.features),
			},
			Modules: slices.Clone(loaded.Modules),
		})
		instances = append(instances, Instance{NodeID: nodeID(current), PackagePath: current.path, Features: slices.Clone(current.features)})
		for _, dependency := range loaded.Manifest.Dependencies {
			target := instance{path: dependency.Path, features: slices.Clone(dependency.Features)}
			key := instanceKey(target)
			targetID, found := seen[key]
			if !found {
				targetID = nodeID(target)
				seen[key] = targetID
				queue = append(queue, target)
			}
			graph.Dependencies = append(graph.Dependencies, packagegraph.Dependency{Owner: nodeID(current), Binding: dependency.Binding, Target: targetID})
		}
	}
	return graph, instances
}

func instanceKey(value struct {
	path     string
	features []string
}) string {
	return value.path + "\x00" + strings.Join(value.features, "\x00")
}

func nodeID(value struct {
	path     string
	features []string
}) string {
	path := value.path
	if path == "." {
		path = "root"
	}
	return "pkg:" + path + "[" + strings.Join(value.features, ",") + "]"
}

func readBoundedRegularFile(path string, maximum int) ([]byte, error) {
	linkInfo, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if linkInfo.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("symlinks are not supported")
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	currentLinkInfo, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if currentLinkInfo.Mode()&os.ModeSymlink != 0 || !os.SameFile(linkInfo, currentLinkInfo) || !os.SameFile(linkInfo, info) {
		return nil, fmt.Errorf("file changed or became a symlink while opening")
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("not a regular file")
	}
	if info.Size() > int64(maximum) {
		return nil, fmt.Errorf("file exceeds %d-byte implementation guard", maximum)
	}
	contents, err := io.ReadAll(io.LimitReader(file, int64(maximum)+1))
	if err != nil {
		return nil, err
	}
	if len(contents) > maximum {
		return nil, fmt.Errorf("file exceeds %d-byte implementation guard", maximum)
	}
	return contents, nil
}

func portablePath(packagePath, sourceRoot, absolute string) string {
	relative, err := filepath.Rel(sourceRoot, absolute)
	if err != nil {
		return packagePath
	}
	if packagePath == "." {
		return filepath.ToSlash(relative)
	}
	return packagePath + "/" + filepath.ToSlash(relative)
}

func normalizeLoadError(reason, path string, err error) error {
	if _, ok := err.(*LoadError); ok {
		return err
	}
	return &LoadError{Reason: reason, Path: path, Detail: err.Error()}
}
