//go:build windows

package cdriver

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"time"

	"golang.org/x/sys/windows"
)

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

type windowsTrapControlOptions struct {
	Context context.Context
	Timer   func(time.Duration) (<-chan time.Time, func())
	Pipe    func() (*os.File, *os.File, error)
	Start   func(*exec.Cmd) error
	Wait    func(*exec.Cmd) error
}

// The parent needs overlapped reads so closing its handle can cancel a pending
// receipt read. The inherited child writer remains an ordinary synchronous pipe.
// os.Pipe creates synchronous handles on Windows and cannot provide this bound.
func newWindowsTrapPipe() (*os.File, *os.File, error) {
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return nil, nil, err
	}
	name, err := windows.UTF16PtrFromString(`\\.\pipe\fairyc-trap-` + hex.EncodeToString(nonce[:]))
	if err != nil {
		return nil, nil, err
	}
	reader, err := windows.CreateNamedPipe(name,
		windows.PIPE_ACCESS_INBOUND|windows.FILE_FLAG_OVERLAPPED|windows.FILE_FLAG_FIRST_PIPE_INSTANCE,
		windows.PIPE_TYPE_BYTE|windows.PIPE_WAIT|windows.PIPE_REJECT_REMOTE_CLIENTS, 1, 4096, 4096, 0, nil)
	if err != nil {
		return nil, nil, err
	}
	writer, err := windows.CreateFile(name, windows.GENERIC_WRITE, 0, nil, windows.OPEN_EXISTING, 0, 0)
	if err != nil {
		windows.CloseHandle(reader)
		return nil, nil, err
	}
	// CreateFile already connected our sole client. No pending connection I/O
	// needs to survive this function or own an additional OVERLAPPED structure.
	return os.NewFile(uintptr(reader), "|trap-read"), os.NewFile(uintptr(writer), "|trap-write"), nil
}

// The command retains ownership of process cancellation. These optional ports
// control receipt I/O and its cancellation without replacing the caller's Cmd.
func runWindowsProcessWithTrapControl(command *exec.Cmd, environmentName string, options ...windowsTrapControlOptions) (string, bool, error) {
	if len(options) > 1 {
		return "", false, fmt.Errorf("private trap control accepts at most one options value")
	}
	settings := windowsTrapControlOptions{Context: context.Background(), Pipe: newWindowsTrapPipe, Start: (*exec.Cmd).Start, Wait: (*exec.Cmd).Wait,
		Timer: func(duration time.Duration) (<-chan time.Time, func()) {
			timer := time.NewTimer(duration)
			return timer.C, func() { timer.Stop() }
		},
	}
	if len(options) != 0 {
		if options[0].Context != nil {
			settings.Context = options[0].Context
		}
		if options[0].Timer != nil {
			settings.Timer = options[0].Timer
		}
		if options[0].Pipe != nil {
			settings.Pipe = options[0].Pipe
		}
		if options[0].Start != nil {
			settings.Start = options[0].Start
		}
		if options[0].Wait != nil {
			settings.Wait = options[0].Wait
		}
	}
	if err := settings.Context.Err(); err != nil {
		return "", false, err
	}
	reader, writer, err := settings.Pipe()
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
	stop := context.AfterFunc(settings.Context, func() { reader.Close() })
	defer stop()
	if err := settings.Context.Err(); err != nil {
		writer.Close()
		reader.Close()
		<-readDone
		return "", false, err
	}
	startErr := settings.Start(command)
	closeErr := writer.Close()
	if startErr != nil {
		reader.Close()
		<-readDone
		return "", false, errors.Join(startErr, settings.Context.Err())
	}
	if closeErr != nil {
		reader.Close()
	}
	waitErr := settings.Wait(command)
	// A descendant may retain the inherited writer after the command exits.
	// Bound only the final drain, not the running program's receipt production.
	// Close our overlapped read handle on expiry and join the copy goroutine
	// before inspecting its buffer. A synchronous pipe cannot safely do this.
	expired, stopTimer := settings.Timer(time.Second)
	defer stopTimer()
	var readErr error
	select {
	case readErr = <-readDone:
	case <-settings.Context.Done():
		reader.Close()
		<-readDone
		return "", false, errors.Join(waitErr, settings.Context.Err())
	case <-expired:
		reader.Close()
		<-readDone
		return "", false, errors.Join(waitErr, fmt.Errorf("private trap control drain exceeded 1s: %w", os.ErrDeadlineExceeded))
	}
	if err := settings.Context.Err(); err != nil {
		return "", false, errors.Join(waitErr, err)
	}
	if closeErr != nil {
		return "", false, errors.Join(waitErr, fmt.Errorf("close private trap control writer: %w", closeErr))
	}
	if readErr != nil {
		return "", false, errors.Join(waitErr, fmt.Errorf("read private trap control pipe: %w", readErr))
	}
	return string(control.bytes), control.overflow, waitErr
}
