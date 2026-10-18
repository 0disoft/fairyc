package cli

import (
	"io"

	"github.com/0disoft/fairyc/internal/cdriver"
)

func classifyStableRunOutcome(outcome cdriver.WindowsHostOutcome) (string, string, error) {
	classified, err := cdriver.ClassifyWindowsNativeOutcome(outcome)
	return classified.Stderr, classified.TrapCategory, err
}

type compiledWorkspace struct {
	artifact        cdriver.WindowsHostArtifact
	lockState       string
	rootPackageName string
}

func compileWindowsWorkspace(sourceRoot string, options workspaceTargetOptions, policy workspaceCompilePolicy, stderr io.Writer) (compiledWorkspace, int) {
	lowered, code := lowerWorkspaceExecutable(sourceRoot, options.locked, policy, "Windows workspace C-emission", stderr)
	if code != ExitSuccess {
		return compiledWorkspace{}, code
	}
	artifact, code := compileWindowsProgramPolicy(lowered.program, lowered.entryName, options.targetID, options.profile, policy.runTrapReceipt, lowered.sdkStorage, stderr)
	if code != ExitSuccess {
		return compiledWorkspace{}, code
	}
	return compiledWorkspace{
		artifact: artifact, lockState: lowered.lockState, rootPackageName: lowered.rootPackageName,
	}, ExitSuccess
}
