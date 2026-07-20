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
	artifact, code := compileWindowsTarget(options.source, options.targetID, options.profile, sourceRoot, stderr)
	if code != ExitSuccess {
		return code
	}
	outcome, err := cdriver.ExecuteWindowsMSVCTarget(artifact)
	if err != nil {
		return buildFailure(stderr, err)
	}
	if _, err := io.WriteString(stdout, outcome.Stdout); err != nil {
		return outputError(stderr)
	}
	if _, err := io.WriteString(stderr, outcome.Stderr); err != nil {
		return ExitInternalFailure
	}
	return int(uint32(outcome.ExitCode))
}
