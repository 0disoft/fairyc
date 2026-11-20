package cli

import (
	"io"

	"github.com/0disoft/fairyc/internal/scipindex"
	"github.com/0disoft/fairyc/internal/workspacecompiler"
)

func runSCIP(args []string, sourceRoot string, stdout, stderr io.Writer) int {
	if len(args) != 0 {
		return invocationError(stderr, "experimental scip accepts no arguments")
	}
	workspace, _, err := loadWorkspaceWithLock(sourceRoot, true)
	if err != nil {
		return packageFailure(stderr, err)
	}
	result, err := workspacecompiler.Check(workspace)
	if len(result.Semantic.Diagnostics) != 0 {
		if err := writeWorkspaceDiagnostics(stderr, workspace, result.Semantic.Diagnostics); err != nil {
			return ExitInternalFailure
		}
		return ExitSourceDiagnostics
	}
	if len(result.Semantic.Unsupported) != 0 {
		if err := writeStableUnsupportedSemantics(stderr, result.Semantic.Unsupported); err != nil {
			return ExitInternalFailure
		}
		return ExitSourceDiagnostics
	}
	if err != nil || !result.Verified {
		return internalError(stderr)
	}
	data, err := scipindex.Marshal(workspace, result)
	if err != nil {
		return packageFailure(stderr, err)
	}
	if written, err := stdout.Write(data); err != nil || written != len(data) {
		return outputError(stderr)
	}
	return ExitSuccess
}
