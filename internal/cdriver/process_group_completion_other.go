//go:build !linux

package cdriver

import (
	"fmt"
	"os/exec"
)

func waitOwnedLinuxProcess(command *exec.Cmd) (error, error) {
	return nil, fmt.Errorf("owned Linux process completion requires Linux")
}
