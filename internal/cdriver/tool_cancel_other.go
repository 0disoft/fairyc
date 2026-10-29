//go:build !windows && !linux

package cdriver

import "os/exec"

func configureToolCancellation(command *exec.Cmd) {}
