package workspaceloader

import (
	"fmt"
	"slices"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/semantic"
)

// SemanticInput converts the validated, root-reachable instance graph into build-local package
// numbers and exact import-provider bindings. Numeric ids never enter lockfiles or diagnostics.
func (workspace Workspace) SemanticInput() (semantic.PackageInput, error) {
	nodePackages := make(map[string]hir.PackageID, len(workspace.Graph.Nodes))
	input := semantic.PackageInput{}
	for index, class := range workspace.Resolution.InstanceClasses {
		if len(class.Members) == 0 {
			return semantic.PackageInput{}, fmt.Errorf("workspace has an empty package instance class")
		}
		packageID := hir.PackageID(index + 1)
		for _, member := range class.Members {
			if _, duplicate := nodePackages[member]; duplicate {
				return semantic.PackageInput{}, fmt.Errorf("workspace node %q belongs to multiple instance classes", member)
			}
			nodePackages[member] = packageID
		}
		mapping, found := workspace.instance(class.Members[0])
		if !found {
			return semantic.PackageInput{}, fmt.Errorf("workspace node %q has no package mapping", class.Members[0])
		}
		loaded, found := workspace.Package(mapping.PackagePath)
		if !found {
			return semantic.PackageInput{}, fmt.Errorf("workspace package %q is missing", mapping.PackagePath)
		}
		for _, file := range loaded.Sources {
			input.Files = append(input.Files, semantic.File{Package: packageID, Path: file.Path, Module: file.Module})
		}
	}

	type importKey struct {
		from hir.PackageID
		path string
	}
	bindings := make(map[importKey]hir.PackageID, len(workspace.Resolution.Lookups))
	for _, lookup := range workspace.Resolution.Lookups {
		from, found := nodePackages[lookup.From]
		if !found || len(lookup.ProviderMembers) == 0 {
			return semantic.PackageInput{}, fmt.Errorf("workspace lookup %q has incomplete instance mapping", lookup.ImportPath)
		}
		to, found := nodePackages[lookup.ProviderMembers[0]]
		if !found {
			return semantic.PackageInput{}, fmt.Errorf("workspace lookup provider %q has no package id", lookup.ProviderMembers[0])
		}
		key := importKey{from: from, path: lookup.ImportPath}
		if previous, duplicate := bindings[key]; duplicate {
			if previous != to {
				return semantic.PackageInput{}, fmt.Errorf("package %d import %q resolves to multiple instances", from, lookup.ImportPath)
			}
			continue
		}
		bindings[key] = to
		input.Imports = append(input.Imports, semantic.PackageImport{FromPackage: from, ImportPath: lookup.ImportPath, ToPackage: to})
	}
	slices.SortFunc(input.Imports, func(left, right semantic.PackageImport) int {
		if left.FromPackage != right.FromPackage {
			if left.FromPackage < right.FromPackage {
				return -1
			}
			return 1
		}
		if left.ImportPath < right.ImportPath {
			return -1
		}
		if left.ImportPath > right.ImportPath {
			return 1
		}
		return 0
	})
	return input, nil
}

func (workspace Workspace) instance(nodeID string) (Instance, bool) {
	for _, instance := range workspace.Instances {
		if instance.NodeID == nodeID {
			return instance, true
		}
	}
	return Instance{}, false
}

// RootPackageID returns the disposable semantic package number assigned to the one root graph
// instance. It is valid only for this Workspace value and must not enter a lockfile or ABI name.
func (workspace Workspace) RootPackageID() (hir.PackageID, error) {
	rootNode := ""
	for _, node := range workspace.Graph.Nodes {
		if node.Role != packagegraph.RoleRoot {
			continue
		}
		if rootNode != "" {
			return hir.DefaultPackage, fmt.Errorf("workspace has multiple root graph nodes")
		}
		rootNode = node.ID
	}
	if rootNode == "" {
		return hir.DefaultPackage, fmt.Errorf("workspace has no root graph node")
	}
	for index, class := range workspace.Resolution.InstanceClasses {
		if slices.Contains(class.Members, rootNode) {
			return hir.PackageID(index + 1), nil
		}
	}
	return hir.DefaultPackage, fmt.Errorf("workspace root graph node has no instance class")
}

// DiagnosticSource resolves one build-local semantic span back to a host-path-free workspace path
// and immutable source bytes.
func (workspace Workspace) DiagnosticSource(packageID hir.PackageID, sourcePath string) (string, []byte, bool) {
	index := int(packageID) - 1
	if index < 0 || index >= len(workspace.Resolution.InstanceClasses) {
		return "", nil, false
	}
	class := workspace.Resolution.InstanceClasses[index]
	if len(class.Members) == 0 {
		return "", nil, false
	}
	mapping, found := workspace.instance(class.Members[0])
	if !found {
		return "", nil, false
	}
	loaded, found := workspace.Package(mapping.PackagePath)
	if !found {
		return "", nil, false
	}
	for _, file := range loaded.Sources {
		if file.Path != sourcePath {
			continue
		}
		display := file.Path
		if mapping.PackagePath != "." {
			display = mapping.PackagePath + "/" + file.Path
		}
		return display, slices.Clone(file.Bytes), true
	}
	return "", nil, false
}
