//go:build !windows

package cdriver

import "os/exec"

func configureToolCancellation(command *exec.Cmd) {}
