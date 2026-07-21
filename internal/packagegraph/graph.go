// Package packagegraph validates and resolves an already-resolved Fairy C package graph.
// It deliberately starts after version solving and source acquisition.
package packagegraph

import (
	"fmt"
	"slices"
	"strings"

	"github.com/0disoft/fairyc/internal/source"
)

const (
	RoleRoot            = "root"
	RoleDependency      = "dependency"
	RoleStandardLibrary = "standard-library"

	ReasonAmbiguousVisibleModule = "ambiguous-visible-module"
	ReasonDuplicateBinding       = "duplicate-dependency-binding"
	ReasonMissingVisibleModule   = "missing-visible-module"
)

// Identity is the exact Stage 0 resolved-instance equality tuple. PackageInstanceId v1 hashes its
// canonical encoding; source resolution still owns the meaning of each opaque component.
type Identity struct {
	PackageUID      string
	SourceIdentity  string
	VersionIdentity string
	Features        []string
}

// Module is one package-owned module and its source-order import paths.
type Module struct {
	Path    string
	Imports []string
}

// Node is one fixture-local handle for a resolved package instance.
type Node struct {
	ID       string
	Role     string
	Identity Identity
	Modules  []Module
}

// Dependency is one already-resolved direct edge. Binding identifies the edge but does not
// rewrite source module paths.
type Dependency struct {
	Owner   string
	Binding string
	Target  string
}

// Graph is an already-resolved package graph in deterministic input order.
type Graph struct {
	Nodes        []Node
	Dependencies []Dependency
}

// InstanceClass lists all node handles that denote one exact resolved instance.
type InstanceClass struct {
	ID      InstanceID
	Members []string
}

// Lookup records one successfully resolved source import.
type Lookup struct {
	From            string
	ImportPath      string
	ProviderMembers []string
	ModulePath      string
}

// Rejection is a package-fixture rule failure, not a released user diagnostic.
type Rejection struct {
	Reason     string
	Subjects   []string
	ImportPath string
}

// Result contains deterministic identity classes and every successful lookup before a rejection.
type Result struct {
	InstanceClasses []InstanceClass
	Lookups         []Lookup
}

type resolvedClass struct {
	id      InstanceID
	members []string
	nodes   []int
}

// Resolve validates graph structure, deduplicates exact instances, and resolves imports through
// the current instance plus direct dependency instances only.
func Resolve(graph Graph) (Result, *Rejection, error) {
	if len(graph.Nodes) == 0 {
		return Result{}, nil, fmt.Errorf("package graph has no nodes")
	}
	nodeIndexes := make(map[string]int, len(graph.Nodes))
	classes := make([]resolvedClass, 0, len(graph.Nodes))
	classByID := make(map[InstanceID]int, len(graph.Nodes))
	nodeClasses := make([]int, len(graph.Nodes))
	for index, node := range graph.Nodes {
		if err := validateNode(node); err != nil {
			return Result{}, nil, fmt.Errorf("node %q: %w", node.ID, err)
		}
		if _, duplicate := nodeIndexes[node.ID]; duplicate {
			return Result{}, nil, fmt.Errorf("duplicate package node %q", node.ID)
		}
		nodeIndexes[node.ID] = index
		instanceID, err := ComputeInstanceID(node.Identity)
		if err != nil {
			return Result{}, nil, fmt.Errorf("node %q: %w", node.ID, err)
		}
		classIndex, foundClass := classByID[instanceID]
		if foundClass {
			representative := graph.Nodes[classes[classIndex].nodes[0]]
			if !identitiesEqual(node.Identity, representative.Identity) {
				return Result{}, nil, fmt.Errorf("package instance id collision between nodes %q and %q", representative.ID, node.ID)
			}
			if !modulesEqual(node.Modules, representative.Modules) {
				return Result{}, nil, fmt.Errorf("equal package instance nodes %q and %q expose different module sets", representative.ID, node.ID)
			}
		} else {
			classIndex = len(classes)
			classByID[instanceID] = classIndex
			classes = append(classes, resolvedClass{id: instanceID})
		}
		classes[classIndex].members = append(classes[classIndex].members, node.ID)
		classes[classIndex].nodes = append(classes[classIndex].nodes, index)
		nodeClasses[index] = classIndex
	}
	result := Result{InstanceClasses: make([]InstanceClass, len(classes))}
	for index, class := range classes {
		result.InstanceClasses[index] = InstanceClass{ID: class.id, Members: slices.Clone(class.members)}
	}

	type resolvedDependency struct {
		ownerClass  int
		binding     string
		targetClass int
	}
	resolvedDependencies := make([]resolvedDependency, 0, len(graph.Dependencies))
	bindingGroups := make(map[string][]resolvedDependency, len(graph.Dependencies))
	for _, dependency := range graph.Dependencies {
		ownerIndex, ownerFound := nodeIndexes[dependency.Owner]
		targetIndex, targetFound := nodeIndexes[dependency.Target]
		if !ownerFound {
			return Result{}, nil, fmt.Errorf("dependency owner %q is not a declared node", dependency.Owner)
		}
		if !targetFound {
			return Result{}, nil, fmt.Errorf("dependency target %q is not a declared node", dependency.Target)
		}
		if dependency.Binding == "" {
			return Result{}, nil, fmt.Errorf("dependency from %q has an empty binding", dependency.Owner)
		}
		resolved := resolvedDependency{
			ownerClass:  nodeClasses[ownerIndex],
			binding:     dependency.Binding,
			targetClass: nodeClasses[targetIndex],
		}
		resolvedDependencies = append(resolvedDependencies, resolved)
		key := fmt.Sprintf("%d:%s", resolved.ownerClass, resolved.binding)
		bindingGroups[key] = append(bindingGroups[key], resolved)
	}
	for _, dependency := range resolvedDependencies {
		key := fmt.Sprintf("%d:%s", dependency.ownerClass, dependency.binding)
		group := bindingGroups[key]
		if len(group) < 2 {
			continue
		}
		subjects := slices.Clone(classes[dependency.ownerClass].members)
		seenTargets := make(map[int]struct{}, len(group))
		for _, candidate := range group {
			if _, seen := seenTargets[candidate.targetClass]; seen {
				continue
			}
			seenTargets[candidate.targetClass] = struct{}{}
			subjects = appendUnique(subjects, classes[candidate.targetClass].members...)
		}
		return result, &Rejection{Reason: ReasonDuplicateBinding, Subjects: subjects}, nil
	}

	outgoing := make(map[int][]int, len(classes))
	for _, dependency := range resolvedDependencies {
		if !slices.Contains(outgoing[dependency.ownerClass], dependency.targetClass) {
			outgoing[dependency.ownerClass] = append(outgoing[dependency.ownerClass], dependency.targetClass)
		}
	}
	for classIndex, class := range classes {
		representative := graph.Nodes[class.nodes[0]]
		visible := []int{classIndex}
		for _, targetClass := range outgoing[classIndex] {
			if !slices.Contains(visible, targetClass) {
				visible = append(visible, targetClass)
			}
		}
		for _, module := range representative.Modules {
			for _, importPath := range module.Imports {
				providers := providerClasses(graph, classes, visible, importPath)
				switch len(providers) {
				case 0:
					subjects := slices.Clone(class.members)
					for candidate := range classes {
						if slices.Contains(visible, candidate) || !classProvides(graph, classes[candidate], importPath) {
							continue
						}
						subjects = appendUnique(subjects, classes[candidate].members...)
					}
					return result, &Rejection{Reason: ReasonMissingVisibleModule, Subjects: subjects, ImportPath: importPath}, nil
				case 1:
					provider := classes[providers[0]]
					result.Lookups = append(result.Lookups, Lookup{
						From:            representative.ID,
						ImportPath:      importPath,
						ProviderMembers: slices.Clone(provider.members),
						ModulePath:      importPath,
					})
				default:
					var subjects []string
					for _, provider := range providers {
						subjects = appendUnique(subjects, classes[provider].members...)
					}
					return result, &Rejection{Reason: ReasonAmbiguousVisibleModule, Subjects: subjects, ImportPath: importPath}, nil
				}
			}
		}
	}
	return result, nil, nil
}

func validateNode(node Node) error {
	if node.ID == "" {
		return fmt.Errorf("node id is empty")
	}
	switch node.Role {
	case RoleRoot, RoleDependency, RoleStandardLibrary:
	default:
		return fmt.Errorf("unknown role %q", node.Role)
	}
	if err := ValidateIdentity(node.Identity); err != nil {
		return err
	}
	if len(node.Modules) == 0 {
		return fmt.Errorf("module set is empty")
	}
	previous := ""
	for index, module := range node.Modules {
		if !validModulePath(module.Path) {
			return fmt.Errorf("invalid module path %q", module.Path)
		}
		if index > 0 && previous >= module.Path {
			return fmt.Errorf("module set is not sorted and unique")
		}
		previous = module.Path
		for _, importPath := range module.Imports {
			if !validModulePath(importPath) {
				return fmt.Errorf("module %q has invalid import path %q", module.Path, importPath)
			}
		}
	}
	return nil
}

func identitiesEqual(left, right Identity) bool {
	return left.PackageUID == right.PackageUID &&
		left.SourceIdentity == right.SourceIdentity &&
		left.VersionIdentity == right.VersionIdentity &&
		slices.Equal(left.Features, right.Features)
}

func modulesEqual(left, right []Module) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index].Path != right[index].Path || !slices.Equal(left[index].Imports, right[index].Imports) {
			return false
		}
	}
	return true
}

func providerClasses(graph Graph, classes []resolvedClass, visible []int, modulePath string) []int {
	providers := make([]int, 0, len(visible))
	for _, classIndex := range visible {
		if classProvides(graph, classes[classIndex], modulePath) {
			providers = append(providers, classIndex)
		}
	}
	return providers
}

func classProvides(graph Graph, class resolvedClass, modulePath string) bool {
	representative := graph.Nodes[class.nodes[0]]
	for _, module := range representative.Modules {
		if module.Path == modulePath {
			return true
		}
	}
	return false
}

func validModulePath(value string) bool {
	segments := strings.Split(value, "::")
	if len(segments) == 0 {
		return false
	}
	for _, segment := range segments {
		if !source.ValidIdentifier(segment) {
			return false
		}
	}
	return true
}

func sortedUnique(values []string, allowEmpty bool) bool {
	if len(values) == 0 {
		return allowEmpty
	}
	if !slices.IsSorted(values) {
		return false
	}
	for index := 1; index < len(values); index++ {
		if values[index-1] == values[index] {
			return false
		}
	}
	return true
}

func appendUnique(values []string, candidates ...string) []string {
	for _, candidate := range candidates {
		if !slices.Contains(values, candidate) {
			values = append(values, candidate)
		}
	}
	return values
}
