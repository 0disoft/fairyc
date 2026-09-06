package workspacecompiler

import (
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/genericplan"
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
	hasGeneric := false
	for _, file := range input.Files {
		for _, function := range file.Module.Functions {
			hasGeneric = hasGeneric || len(function.TypeParameters) != 0
		}
	}
	if !hasGeneric {
		// Preserve ordinary diagnostics and avoid generic preparation for workspaces
		// that do not contain a generic declaration.
		return compiler.CheckPackages(input)
	}
	coordinates, err := workspace.PackageCoordinates()
	if err != nil {
		return compiler.Result{}, err
	}
	result, _, err := compiler.CheckGenericPackages(input, coordinates, genericplan.DefaultPlanLimits())
	return result, err
}
