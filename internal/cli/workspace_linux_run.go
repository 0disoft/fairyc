package cli

import (
	"fmt"
	"io"

	"github.com/0disoft/fairyc/internal/cdriver"
)

func runLinuxWorkspaceTarget(sourceRoot string, options workspaceTargetOptions, stdout, stderr io.Writer) int {
	artifact, lockState, code := compileLinuxWorkspace(sourceRoot, options, true, stderr)
	if code != ExitSuccess {
		return code
	}
	outcome, err := cdriver.ExecuteLinuxTarget(artifact)
	if err != nil {
		return buildFailure(stderr, err)
	}
	if outcome.Stdout != "" || outcome.Stderr != "" {
		return buildFailure(stderr, fmt.Errorf("Linux workspace executable produced unexpected output"))
	}
	if outcome.TrapCategory != 0 {
		category, found := windowsHostTrapCategory(int32(productTrapExitBase) + int32(outcome.TrapCategory))
		if !found {
			return buildFailure(stderr, fmt.Errorf("Linux workspace executable produced unknown trap category"))
		}
		if _, err := fmt.Fprintf(stderr, "trap: %s\n", category); err != nil {
			return ExitInternalFailure
		}
		return ExitProgramTrap
	}
	if _, err := fmt.Fprintf(stdout, "native workspace status: %d (%s, %s %s, %s, lock %s)\n",
		outcome.ExitCode, options.targetID, artifact.Toolchain.Name, artifact.Toolchain.Version,
		artifact.Profile.Name, lockState); err != nil {
		return outputError(stderr)
	}
	return ExitSuccess
}
