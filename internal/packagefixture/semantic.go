package packagefixture

import (
	"fmt"
	"slices"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/semantic"
)

// SemanticInput converts one passing resolved fixture graph into the general build-local semantic
// package input. Fixture node labels and numeric ids remain process-local and are never serialized.
func (c ResolvedCase) SemanticInput() (semantic.PackageInput, error) {
	if c.ExpectedStatus != "pass" || c.Rejection != nil {
		return semantic.PackageInput{}, fmt.Errorf("package case %q is not a passing semantic input", c.ID)
	}
	memberPackages := make(map[string]hir.PackageID)
	input := semantic.PackageInput{}
	for index, class := range c.Resolution.InstanceClasses {
		if len(class.Members) == 0 {
			return semantic.PackageInput{}, fmt.Errorf("package case %q has an empty instance class", c.ID)
		}
		packageID := hir.PackageID(index + 1)
		for _, member := range class.Members {
			if _, duplicate := memberPackages[member]; duplicate {
				return semantic.PackageInput{}, fmt.Errorf("package node %q belongs to multiple instance classes", member)
			}
			memberPackages[member] = packageID
		}
		representative, found := c.Node(class.Members[0])
		if !found {
			return semantic.PackageInput{}, fmt.Errorf("package node %q is missing", class.Members[0])
		}
		for _, file := range representative.Files {
			file.Package = packageID
			input.Files = append(input.Files, file)
		}
	}
	type importKey struct {
		from hir.PackageID
		path string
	}
	bindings := make(map[importKey]hir.PackageID)
	for _, lookup := range c.Resolution.Lookups {
		from, found := memberPackages[lookup.From]
		if !found {
			return semantic.PackageInput{}, fmt.Errorf("lookup source node %q has no instance class", lookup.From)
		}
		if len(lookup.ProviderMembers) == 0 {
			return semantic.PackageInput{}, fmt.Errorf("lookup %q has no provider instance", lookup.ImportPath)
		}
		to, found := memberPackages[lookup.ProviderMembers[0]]
		if !found {
			return semantic.PackageInput{}, fmt.Errorf("lookup provider node %q has no instance class", lookup.ProviderMembers[0])
		}
		key := importKey{from: from, path: lookup.ImportPath}
		if previous, duplicate := bindings[key]; duplicate {
			if previous != to {
				return semantic.PackageInput{}, fmt.Errorf("package %d import %q resolves to multiple instances", from, lookup.ImportPath)
			}
			continue
		}
		bindings[key] = to
		input.Imports = append(input.Imports, semantic.PackageImport{
			FromPackage: from, ImportPath: lookup.ImportPath, ToPackage: to,
		})
	}
	slices.SortFunc(input.Imports, func(left, right semantic.PackageImport) int {
		if left.FromPackage < right.FromPackage {
			return -1
		}
		if left.FromPackage > right.FromPackage {
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
