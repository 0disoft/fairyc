package workspacecompiler

import (
	"context"
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
	return CheckContext(context.Background(), workspace)
}

func CheckContext(ctx context.Context, workspace workspaceloader.Workspace) (result compiler.Result, err error) {
	if err := ctx.Err(); err != nil {
		return compiler.Result{}, &compiler.Error{Stage: compiler.StageSemantic, Cause: err}
	}
	defer func() {
		if canceled := ctx.Err(); canceled != nil {
			result, err = compiler.Result{}, &compiler.Error{Stage: compiler.StageSemantic, Cause: canceled}
		}
	}()
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
		return compiler.CheckPackagesContext(ctx, input)
	}
	policy := admission.Policy[hir.PackageID]{}
	if len(owners) != 0 {
		policy = admission.Policy[hir.PackageID]{Capabilities: sdkStorageCapabilities(), StorageOwners: owners}
	}
	coordinates, err := workspace.PackageCoordinates()
	if err != nil {
		return compiler.Result{}, err
	}
	compiled, _, err := compiler.CheckWithPolicyContext(ctx, input, coordinates, genericplan.DefaultPlanLimits(), policy)
	return compiled, err
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
