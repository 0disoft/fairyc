//go:build windows

package cdriver

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"syscall"
)

const maxTrapControlBytes = 8

type boundedTrapControl struct {
	bytes    []byte
	overflow bool
}

func (control *boundedTrapControl) Write(value []byte) (int, error) {
	originalLength := len(value)
	available := maxTrapControlBytes - len(control.bytes)
	if available > 0 {
		control.bytes = append(control.bytes, value[:min(available, len(value))]...)
	}
	if originalLength > max(available, 0) {
		control.overflow = true
	}
	return originalLength, nil
}

func runWindowsProcessWithTrapControl(command *exec.Cmd, environmentName string) (string, bool, error) {
	reader, writer, err := os.Pipe()
	if err != nil {
		return "", false, fmt.Errorf("create private trap control pipe: %w", err)
	}
	defer reader.Close()
	writerHandle := syscall.Handle(writer.Fd())
	if err := syscall.SetHandleInformation(writerHandle, syscall.HANDLE_FLAG_INHERIT, syscall.HANDLE_FLAG_INHERIT); err != nil {
		writer.Close()
		return "", false, fmt.Errorf("make private trap control handle inheritable: %w", err)
	}
	if command.SysProcAttr == nil {
		command.SysProcAttr = &syscall.SysProcAttr{}
	}
	command.SysProcAttr.AdditionalInheritedHandles = append(command.SysProcAttr.AdditionalInheritedHandles, writerHandle)
	environment := command.Env
	if environment == nil {
		environment = os.Environ()
	}
	command.Env = replaceWindowsEnvironment(environment, environmentName, strconv.FormatUint(uint64(writer.Fd()), 10))

	control := &boundedTrapControl{}
	readDone := make(chan error, 1)
	go func() {
		_, copyErr := io.Copy(control, reader)
		readDone <- copyErr
	}()
	startErr := command.Start()
	closeErr := writer.Close()
	if startErr != nil {
		reader.Close()
		<-readDone
		return "", false, startErr
	}
	if closeErr != nil {
		reader.Close()
	}
	waitErr := command.Wait()
	readErr := <-readDone
	if closeErr != nil {
		return "", false, fmt.Errorf("close private trap control writer: %w", closeErr)
	}
	if readErr != nil {
		return "", false, fmt.Errorf("read private trap control pipe: %w", readErr)
	}
	return string(control.bytes), control.overflow, waitErr
}
