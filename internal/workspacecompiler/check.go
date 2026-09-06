package workspacecompiler

import (
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

// Check compiles the current resolved workspace without reading or writing an
// artifact cache. Source input and generic ownership come from the same loaded
// workspace; callers cannot supply a disconnected coordinate map. This internal
// entry point prepares shared CLI admission without changing command routing.
func Check(workspace workspaceloader.Workspace) (compiler.Result, error) {
	input, err := workspace.SemanticInput()
	if err != nil {
		return compiler.Result{}, err
	}
	coordinates, err := workspace.PackageCoordinates()
	if err != nil {
		return compiler.Result{}, err
	}
	result, _, err := compiler.CheckGenericPackages(input, coordinates, genericplan.DefaultPlanLimits())
	return result, err
}
