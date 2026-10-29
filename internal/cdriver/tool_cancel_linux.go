//go:build linux

package cdriver

import (
	"errors"
	"os"
	"os/exec"
	"syscall"
)

// GCC and Clang delegate to child tools. Give this standalone invocation its
// own process group so a deadline cannot leave a compiler consuming CPU or
// retaining inherited pipes. The captured Linux companion uses its separate
// group-inheriting project runner instead of this helper.
func configureToolCancellation(command *exec.Cmd) {
	if command.SysProcAttr == nil {
		command.SysProcAttr = &syscall.SysProcAttr{}
	}
	command.SysProcAttr.Setpgid = true
	command.SysProcAttr.Pgid = 0
	command.Cancel = func() error {
		if command.Process == nil {
			return os.ErrProcessDone
		}
		err := syscall.Kill(-command.Process.Pid, syscall.SIGKILL)
		if errors.Is(err, syscall.ESRCH) {
			return os.ErrProcessDone
		}
		if err != nil {
			return errors.Join(err, command.Process.Kill())
		}
		return nil
	}
}
