package workspacecompiler

import (
	"github.com/0disoft/fairyc/internal/admission"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

// Check compiles the current resolved workspace without reading or writing an
// artifact cache. Source input and generic ownership come from the same loaded
// workspace; callers cannot supply a disconnected coordinate map. Stable workspace
// commands share this entry point for ordinary and explicit scalar generic source.
func Check(workspace workspaceloader.Workspace) (compiler.Result, error) {
	input, err := workspace.SemanticInput()
	if err != nil {
		return compiler.Result{}, err
	}
	owners, err := workspace.SDKStorageOwners()
	if err != nil {
		return compiler.Result{}, err
	}
	if len(owners) == 0 && (hasGenericAggregates(input) || !hasGenericDeclarations(input)) {
		// Preserve ordinary analysis and the unsupported-aggregate boundary.
		return compiler.CheckPackages(input)
	}
	policy := admission.Policy[hir.PackageID]{}
	if len(owners) != 0 {
		policy = admission.Policy[hir.PackageID]{Capabilities: sdkStorageCapabilities(), StorageOwners: owners}
	}
	coordinates, err := workspace.PackageCoordinates()
	if err != nil {
		return compiler.Result{}, err
	}
	result, _, err := compiler.CheckWithPolicy(input, coordinates, genericplan.DefaultPlanLimits(), policy)
	return result, err
}

// Stable source admission stays separate from compiler-private aggregate
// interfaces until cache and Atlas transport have their own promotion evidence.
func hasGenericAggregates(input semantic.PackageInput) bool {
	for _, file := range input.Files {
		for _, declaration := range file.Module.Structs {
			if len(declaration.TypeParameters) != 0 {
				return true
			}
		}
		for _, declaration := range file.Module.Resources {
			if len(declaration.TypeParameters) != 0 {
				return true
			}
		}
	}
	return false
}

func hasGenericDeclarations(input semantic.PackageInput) bool {
	for _, file := range input.Files {
		for _, function := range file.Module.Functions {
			if len(function.TypeParameters) != 0 {
				return true
			}
		}
	}
	return false
}
