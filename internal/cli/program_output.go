package cli

import "io"

func forwardProgramOutput(programStdout, programStderr string, stdout, stderr io.Writer) int {
	if _, err := io.WriteString(stdout, programStdout); err != nil {
		return outputError(stderr)
	}
	if _, err := io.WriteString(stderr, programStderr); err != nil {
		return ExitInternalFailure
	}
	return ExitSuccess
}
