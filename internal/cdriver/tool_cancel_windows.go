//go:build windows

package cdriver

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
	"time"
	"unsafe"
)

// CL delegates compilation to child tools. Cancel its process tree before
// returning the deadline failure so inherited output pipes and objects close.
func configureToolCancellation(command *exec.Cmd) {
	command.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	command.Cancel = func() error {
		if command.Process == nil {
			return nil
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		// Resolve the OS tool from the Windows system directory, not PATH.
		buffer := make([]uint16, 32768)
		getSystemDirectory := syscall.NewLazyDLL("kernel32.dll").NewProc("GetSystemDirectoryW")
		length, _, callErr := getSystemDirectory.Call(uintptr(unsafe.Pointer(&buffer[0])), uintptr(len(buffer)))
		if length == 0 || length >= uintptr(len(buffer)) {
			return errors.Join(fmt.Errorf("resolve Windows system directory: %w", callErr), command.Process.Kill())
		}
		tool := filepath.Join(syscall.UTF16ToString(buffer[:length]), "taskkill.exe")
		kill := exec.CommandContext(ctx, tool, "/PID", strconv.Itoa(command.Process.Pid), "/T", "/F")
		kill.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		kill.WaitDelay = time.Second
		if err := kill.Run(); err != nil {
			return errors.Join(fmt.Errorf("terminate compiler process tree: %w", err), command.Process.Kill())
		}
		return nil
	}
}
