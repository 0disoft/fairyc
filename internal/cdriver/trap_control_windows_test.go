//go:build windows

package cdriver

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"testing"
	"time"
)

func TestBoundedTrapControlDrainsAfterOverflow(t *testing.T) {
	t.Parallel()
	control := &boundedTrapControl{}
	input := make([]byte, maxTrapControlBytes+2)
	for index := range input {
		input[index] = byte(index % 251)
	}
	written, err := control.Write(input)
	if err != nil || written != len(input) || !bytes.Equal(control.bytes, input[:maxTrapControlBytes]) || !control.overflow {
		t.Fatalf("first write = (%d, %v), bytes %v, overflow %t", written, err, control.bytes, control.overflow)
	}
	written, err = control.Write([]byte{11, 12})
	if err != nil || written != 2 || !bytes.Equal(control.bytes, input[:maxTrapControlBytes]) || !control.overflow {
		t.Fatalf("drain write = (%d, %v), bytes %v, overflow %t", written, err, control.bytes, control.overflow)
	}
}

func duplicateTrapWriter(command *exec.Cmd) (*os.File, error) {
	handles := command.SysProcAttr.AdditionalInheritedHandles
	var handle syscall.Handle
	process, err := syscall.GetCurrentProcess()
	if err != nil {
		return nil, err
	}
	err = syscall.DuplicateHandle(process, handles[len(handles)-1], process, &handle, 0, false, syscall.DUPLICATE_SAME_ACCESS)
	if err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(handle), "retained-trap-writer"), nil
}

func TestWindowsTrapControlRetainedWriter(t *testing.T) {
	for _, name := range []string{"deadline", "cancellation", "default-deadline"} {
		cancel := name == "cancellation"
		t.Run(name, func(t *testing.T) {
			ctx, abort := context.WithCancel(context.Background())
			defer abort()
			var retained *os.File
			waited := make(chan struct{})
			done := make(chan error, 1)
			waitFailure := errors.New("process wait failed")
			options := windowsTrapControlOptions{
				Context: ctx,
				Timer: func(duration time.Duration) (<-chan time.Time, func()) {
					if duration != time.Second {
						t.Errorf("drain timeout=%s, want 1s", duration)
					}
					expired := make(chan time.Time, 1)
					if !cancel {
						expired <- time.Unix(0, 0)
					}
					return expired, func() {}
				},
				Start: func(command *exec.Cmd) (err error) { retained, err = duplicateTrapWriter(command); return err },
				Wait:  func(*exec.Cmd) error { close(waited); return waitFailure },
			}
			if name == "default-deadline" {
				options.Timer = nil
			}
			go func() {
				out, overflow, err := runWindowsProcessWithTrapControl(&exec.Cmd{}, "FAIRYC_TRAP_TEST", options)
				if out != "" || overflow {
					err = errors.Join(err, errors.New("unexpected receipt accepted"))
				}
				done <- err
			}()
			select {
			case <-waited:
			case err := <-done:
				t.Fatalf("query failed before wait: %v", err)
			case <-time.After(5 * time.Second):
				t.Fatal("query did not reach wait")
			}
			defer retained.Close()
			if cancel {
				abort()
			}
			select {
			case err := <-done:
				want := os.ErrDeadlineExceeded
				if cancel {
					want = context.Canceled
				}
				if !errors.Is(err, want) {
					t.Fatalf("error=%v, want %v", err, want)
				}
				if !errors.Is(err, waitFailure) {
					t.Fatalf("lost process wait error: %v", err)
				}
			case <-time.After(3 * time.Second):
				// Release exactly our own duplicated handle, then join the reader.
				retained.Close()
				<-done
				t.Fatal("retained writer blocked result after process completion")
			}
		})
	}
}

func TestWindowsTrapControlByteBoundary(t *testing.T) {
	for _, size := range []int{maxTrapControlBytes - 1, maxTrapControlBytes, maxTrapControlBytes + 1} {
		t.Run(strconv.Itoa(size), func(t *testing.T) {
			payload := bytes.Repeat([]byte{'x'}, size)
			waited, stopped := false, false
			out, overflow, err := runWindowsProcessWithTrapControl(&exec.Cmd{}, "FAIRYC_TRAP_TEST", windowsTrapControlOptions{
				Start: func(command *exec.Cmd) error {
					writer, err := duplicateTrapWriter(command)
					if err != nil {
						return err
					}
					_, writeErr := writer.Write(payload)
					return errors.Join(writeErr, writer.Close())
				},
				Wait: func(*exec.Cmd) error { waited = true; return nil },
				Timer: func(duration time.Duration) (<-chan time.Time, func()) {
					if !waited || duration != time.Second {
						t.Error("drain budget must start after process completion")
					}
					return make(chan time.Time), func() { stopped = true }
				},
			})
			if err != nil || overflow != (size > maxTrapControlBytes) || out != string(payload[:min(size, maxTrapControlBytes)]) || !stopped {
				t.Fatalf("bytes=%d overflow=%v stopped=%v err=%v", len(out), overflow, stopped, err)
			}
		})
	}
}

func TestWindowsTrapControlIOFailures(t *testing.T) {
	for _, stage := range []string{"pipe", "start", "wait"} {
		t.Run(stage, func(t *testing.T) {
			failure := errors.New(stage + " failed")
			options := windowsTrapControlOptions{
				Start: func(*exec.Cmd) error {
					if stage == "start" {
						return failure
					}
					return nil
				},
				Wait: func(*exec.Cmd) error {
					if stage == "wait" {
						return failure
					}
					t.Error("wait called after failed start")
					return nil
				},
			}
			if stage == "pipe" {
				options.Pipe = func() (*os.File, *os.File, error) { return nil, nil, failure }
			}
			out, overflow, err := runWindowsProcessWithTrapControl(&exec.Cmd{}, "FAIRYC_TRAP_TEST", options)
			if out != "" || overflow || !errors.Is(err, failure) {
				t.Fatalf("out=%q overflow=%v err=%v", out, overflow, err)
			}
		})
	}
}

func TestWindowsTrapControlHelper(t *testing.T) {
	if os.Getenv("FAIRYC_TRAP_HELPER") != "1" {
		return
	}
	handle, err := strconv.ParseUint(os.Getenv("FAIRYC_TRAP_TEST"), 10, 64)
	if err != nil {
		os.Exit(2)
	}
	writer := os.NewFile(uintptr(handle), "|trap-child")
	_, err = writer.Write([]byte("receipt\x00\xff"))
	if errors.Join(err, writer.Close()) != nil {
		os.Exit(3)
	}
	os.Exit(0)
}

func TestWindowsTrapControlSubprocess(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestWindowsTrapControlHelper$")
	command.Env = append(os.Environ(), "FAIRYC_TRAP_HELPER=1")
	out, overflow, err := runWindowsProcessWithTrapControl(command, "FAIRYC_TRAP_TEST")
	if out != "receipt\x00\xff" || overflow || err != nil {
		t.Fatalf("out=%q overflow=%v err=%v", out, overflow, err)
	}
}

func TestWindowsTrapControlCanceledBeforeIO(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	out, overflow, err := runWindowsProcessWithTrapControl(&exec.Cmd{}, "FAIRYC_TRAP_TEST", windowsTrapControlOptions{
		Context: ctx,
		Pipe: func() (*os.File, *os.File, error) {
			t.Error("canceled operation performed pipe I/O")
			return nil, nil, os.ErrInvalid
		},
	})
	if out != "" || overflow || !errors.Is(err, context.Canceled) {
		t.Fatalf("out=%q overflow=%v err=%v", out, overflow, err)
	}
}
