package packagesource

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestGitProcessFixture(t *testing.T) {
	switch os.Getenv("FAIRYC_GIT_PROCESS_TEST") {
	case "child":
		time.Sleep(5 * time.Second) // Finite even if the test's cleanup fails.
		os.Exit(0)
	case "parent":
		child := exec.Command(os.Args[0], "-test.run=^TestGitProcessFixture$")
		child.Env = append(os.Environ(), "FAIRYC_GIT_PROCESS_TEST=child")
		child.Stdout = os.Stdout
		if err := child.Start(); err != nil {
			os.Exit(2)
		}
		if err := os.WriteFile(os.Getenv("FAIRYC_GIT_PROCESS_PID"), []byte(strconv.Itoa(child.Process.Pid)), 0600); err != nil {
			_ = child.Process.Kill()
			os.Exit(3)
		}
		fmt.Print("ready")
		os.Exit(0)
	}
}

type gitOutputStub struct {
	io.Reader
	readHook  func([]byte) (int, error)
	closeHook func() error
	closes    atomic.Int32
}

func (output *gitOutputStub) Read(buffer []byte) (int, error) {
	if output.closes.Load() != 0 {
		return 0, os.ErrClosed
	}
	if output.readHook != nil {
		return output.readHook(buffer)
	}
	return output.Reader.Read(buffer)
}

func (output *gitOutputStub) Close() error {
	output.closes.Add(1)
	if output.closeHook != nil {
		return output.closeHook()
	}
	return nil
}

func TestGitProcessIOFailuresAndOwnership(t *testing.T) {
	for _, phase := range []string{"success", "open", "open-with-resource", "missing-output", "start", "read", "consume", "close", "wait", "combined"} {
		t.Run(phase, func(t *testing.T) {
			readFailure, closeFailure, waitFailure := errors.New("read failed"), errors.New("close failed"), errors.New("wait failed")
			output := &gitOutputStub{Reader: strings.NewReader("payload")}
			if phase == "read" {
				output.readHook = func([]byte) (int, error) { return 0, readFailure }
			}
			if phase == "close" || phase == "combined" {
				output.closeHook = func() error { return closeFailure }
			}
			opened, started, waited, consumed, cancelled := 0, 0, 0, 0, 0
			policy := gitProcessOptions{
				OpenOutput: func(*exec.Cmd) (io.ReadCloser, error) {
					opened++
					if phase == "open" {
						return nil, readFailure
					}
					if phase == "open-with-resource" {
						return output, readFailure
					}
					if phase == "missing-output" {
						return nil, nil
					}
					return output, nil
				},
				Start: func(*exec.Cmd) error {
					started++
					if phase == "start" {
						return readFailure
					}
					return nil
				},
				Wait: func(*exec.Cmd) error {
					waited++
					if phase == "wait" || phase == "combined" {
						return waitFailure
					}
					return nil
				},
			}
			err := consumeGitProcess(&exec.Cmd{}, func() { cancelled++ }, func(reader io.Reader) error {
				consumed++
				if phase == "consume" || phase == "combined" {
					return readFailure
				}
				_, err := io.Copy(io.Discard, reader)
				return err
			}, policy)
			if phase == "success" {
				if err != nil {
					t.Fatal(err)
				}
			} else if err == nil {
				t.Fatal("failed process was accepted")
			}
			if phase == "open" || phase == "open-with-resource" || phase == "start" || phase == "read" || phase == "consume" || phase == "combined" {
				if !errors.Is(err, readFailure) {
					t.Fatalf("lost primary failure: %v", err)
				}
			}
			if (phase == "close" || phase == "combined") && !errors.Is(err, closeFailure) {
				t.Fatalf("lost close failure: %v", err)
			}
			if (phase == "wait" || phase == "combined") && !errors.Is(err, waitFailure) {
				t.Fatalf("lost wait failure: %v", err)
			}
			if errors.Is(err, context.Canceled) {
				t.Fatalf("consumer refusal was mislabeled as caller cancellation: %v", err)
			}
			wantStart, wantWait, wantClose := 1, 1, int32(1)
			if phase == "open" || phase == "open-with-resource" || phase == "missing-output" {
				wantStart, wantWait = 0, 0
			}
			if phase == "start" {
				wantWait = 0
			}
			if phase == "open" || phase == "missing-output" {
				wantClose = 0
			}
			if opened != 1 || started != wantStart || waited != wantWait || consumed != wantWait || output.closes.Load() != wantClose {
				t.Fatalf("resource counts open/start/wait/consume/close=%d/%d/%d/%d/%d", opened, started, waited, consumed, output.closes.Load())
			}
			if (phase == "read" || phase == "consume" || phase == "combined") != (cancelled == 1) {
				t.Fatalf("producer cancellation count=%d", cancelled)
			}
		})
	}
}

func TestGitProcessCancellationAtIOBoundaries(t *testing.T) {
	for _, phase := range []string{"open", "start", "read", "eof", "close", "wait"} {
		t.Run(phase, func(t *testing.T) {
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause := errors.New("caller cancelled at " + phase)
			var cancelled atomic.Int32
			started, waited, consumed := 0, 0, 0
			output := &gitOutputStub{Reader: strings.NewReader("payload")}
			output.readHook = func(buffer []byte) (int, error) {
				n, err := output.Reader.Read(buffer)
				if phase == "read" || phase == "eof" && err == io.EOF {
					cancel(cause)
				}
				return n, err
			}
			output.closeHook = func() error {
				if phase == "close" {
					cancel(cause)
				}
				return nil
			}
			err := consumeGitProcess(&exec.Cmd{}, func() { cancelled.Add(1) }, func(reader io.Reader) error {
				consumed++
				_, err := io.Copy(io.Discard, reader)
				return err
			}, gitProcessOptions{
				Context: ctx,
				OpenOutput: func(*exec.Cmd) (io.ReadCloser, error) {
					if phase == "open" {
						cancel(cause)
					}
					return output, nil
				},
				Start: func(*exec.Cmd) error {
					started++
					if phase == "start" {
						cancel(cause)
					}
					return nil
				},
				Wait: func(*exec.Cmd) error {
					waited++
					if phase == "wait" {
						cancel(cause)
					}
					return nil
				},
			})
			if !errors.Is(err, context.Canceled) || !errors.Is(err, cause) {
				t.Fatalf("lost caller cancellation: %v", err)
			}
			wantStart, wantConsume := 1, 1
			if phase == "open" {
				wantStart, wantConsume = 0, 0
			}
			if phase == "start" {
				wantConsume = 0
			}
			if started != wantStart || waited != wantStart || consumed != wantConsume || output.closes.Load() != 1 {
				t.Fatalf("boundary counts start/wait/consume/close=%d/%d/%d/%d", started, waited, consumed, output.closes.Load())
			}
			if wantStart != 0 && cancelled.Load() == 0 {
				t.Fatal("producer was not cancelled")
			}
		})
	}
}

func TestGitProcessRejectsBeforeIO(t *testing.T) {
	for _, mode := range []string{"cancelled", "expired", "options", "command", "cancel-function", "consumer"} {
		t.Run(mode, func(t *testing.T) {
			ctx := context.Background()
			want := error(nil)
			if mode == "cancelled" {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
				want = context.Canceled
			}
			if mode == "expired" {
				var cancel context.CancelFunc
				ctx, cancel = context.WithDeadline(ctx, time.Unix(0, 0))
				defer cancel()
				want = context.DeadlineExceeded
			}
			command, cancel, consumer := &exec.Cmd{}, context.CancelFunc(func() {}), func(io.Reader) error { return nil }
			if mode == "command" {
				command = nil
			}
			if mode == "cancel-function" {
				cancel = nil
			}
			if mode == "consumer" {
				consumer = nil
			}
			options := []gitProcessOptions{{Context: ctx, OpenOutput: func(*exec.Cmd) (io.ReadCloser, error) { t.Fatal("unexpected process I/O"); return nil, nil }}}
			if mode == "options" {
				options = append(options, gitProcessOptions{})
			}
			if err := consumeGitProcess(command, cancel, consumer, options...); err == nil || want != nil && !errors.Is(err, want) {
				t.Fatalf("invalid state accepted: %v", err)
			}
		})
	}
}

func TestGitProcessCancellationClosesInheritedOutput(t *testing.T) {
	pidFile := filepath.Join(t.TempDir(), "child.pid")
	ctx, cancel := context.WithCancelCause(t.Context())
	defer cancel(nil)
	command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestGitProcessFixture$")
	command.Env = append(os.Environ(), "FAIRYC_GIT_PROCESS_TEST=parent", "FAIRYC_GIT_PROCESS_PID="+pidFile)
	command.WaitDelay = 100 * time.Millisecond
	var cleanupOnce sync.Once
	cleanup := func() {
		cleanupOnce.Do(func() {
			data, err := os.ReadFile(pidFile)
			if err != nil {
				t.Error(err)
				return
			}
			pid, err := strconv.Atoi(string(data))
			if err != nil {
				t.Error(err)
				return
			}
			process, err := os.FindProcess(pid)
			if err == nil {
				defer process.Release()
				_ = process.Kill()
			} // Only this test's descendant.
		})
	}
	t.Cleanup(cleanup)
	ready, done := make(chan struct{}), make(chan error, 1)
	cause := errors.New("stop while descendant holds stdout")
	go func() {
		done <- consumeGitProcess(command, func() { cancel(nil) }, func(reader io.Reader) error {
			buffer := make([]byte, 5)
			if _, err := io.ReadFull(reader, buffer); err != nil {
				return err
			}
			cancel(cause)
			close(ready)
			_, err := io.Copy(io.Discard, reader)
			return err
		}, gitProcessOptions{Context: ctx})
	}()
	select {
	case <-ready:
	case err := <-done:
		t.Fatalf("producer failed before fixture became ready: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("fixture did not start")
	}
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) || !errors.Is(err, cause) {
			t.Fatalf("lost cancellation cause: %v", err)
		}
		if command.ProcessState == nil {
			t.Fatal("producer was not reaped")
		}
	case <-time.After(time.Second):
		cleanup()
		<-done
		t.Fatal("cancelled Git read remained blocked on inherited stdout")
	}
}
