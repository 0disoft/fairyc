//go:build !windows

package cdriver

import "os/exec"

func runWindowsProcessWithTrapControl(command *exec.Cmd, _ string) (string, bool, error) {
	return "", false, command.Run()
}
