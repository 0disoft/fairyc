package cdriver

import "os/exec"

// ConfigureCommandCancellation applies the host tool's process-tree deadline
// handling to a CommandContext invocation, including hidden Windows launches.
func ConfigureCommandCancellation(command *exec.Cmd) {
	configureToolCancellation(command)
}
