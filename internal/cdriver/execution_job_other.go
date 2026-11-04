//go:build !windows

package cdriver

import (
	"os"
	"os/exec"
)

func runOwnedWindowsProcess(command *exec.Cmd, trapEnvironment string) (string, bool, error) {
	if trapEnvironment != "" {
		return runWindowsProcessWithTrapControl(command, trapEnvironment)
	}
	return "", false, command.Run()
}

func removeOwnedWindowsDirectory(path string) error { return os.RemoveAll(path) }
