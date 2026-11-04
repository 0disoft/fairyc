//go:build linux

package cdriver

import (
	"errors"
	"fmt"
	"os/exec"

	"golang.org/x/sys/unix"
)

func waitOwnedLinuxProcess(command *exec.Cmd) (error, error) {
	if command.Process == nil || command.Process.Pid <= 1 {
		return nil, fmt.Errorf("invalid owned Linux process")
	}
	var observation error
	return completeOwnedProcessGroup(func() error {
		var info unix.Siginfo
		for {
			observation = unix.Waitid(unix.P_PID, command.Process.Pid, &info, unix.WEXITED|unix.WNOWAIT, nil)
			if !errors.Is(observation, unix.EINTR) {
				return observation
			}
		}
	}, func() error {
		if errors.Is(observation, unix.ECHILD) {
			return nil // Ownership was lost; never signal a potentially reused group.
		}
		if observation != nil {
			return command.Process.Kill() // Abort the leader before its final wait.
		}
		return command.Cancel()
	}, command.Wait)
}
