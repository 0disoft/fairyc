package cli

import (
	"io"

	"github.com/0disoft/fairyc/internal/cdriver"
)

func execTarget(arguments []string, sourceRoot string, stdout, stderr io.Writer) int {
	options, err := parseTargetExecutionOptions("exec-target", arguments)
	if err != nil {
		return invocationError(stderr, err.Error())
	}
	artifact, code := compileWindowsTarget(programSourceSet(options.source, options.additionalSources), options.targetID, options.profile, sourceRoot, stderr)
	if code != ExitSuccess {
		return code
	}
	outcome, err := cdriver.ExecuteWindowsMSVCTarget(artifact)
	if err != nil {
		return buildFailure(stderr, err)
	}
	if code := forwardProgramOutput(outcome.Stdout, outcome.Stderr, stdout, stderr); code != ExitSuccess {
		return code
	}
	return int(uint32(outcome.ExitCode))
}
