package desktopapp

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"
)

// Reuse the test binary instead of requiring Python or launching any GUI.
func TestDesktopHelperProcess(t *testing.T) {
	switch os.Getenv("FAIRYC_DESKTOP_HELPER_MODE") {
	case "child":
		time.Sleep(6 * time.Second) // Finite even if the parent test fails.
		os.Exit(0)
	case "orphan":
		child := exec.Command(os.Args[0], "-test.run=^TestDesktopHelperProcess$", "--")
		child.Env = append(os.Environ(), "FAIRYC_DESKTOP_HELPER_MODE=child")
		child.Stdout, child.Stderr = os.Stdout, os.Stderr
		if err := child.Start(); err != nil {
			os.Exit(2)
		}
		if err := os.WriteFile(os.Getenv("FAIRYC_DESKTOP_HELPER_PID"), []byte(strconv.Itoa(child.Process.Pid)), 0600); err != nil {
			_ = child.Process.Kill()
			os.Exit(3)
		}
		fmt.Print("incomplete receipt")
		os.Exit(0)
	case "success":
		fmt.Print("complete receipt")
		os.Exit(0)
	case "block":
		fmt.Print("ready")
		time.Sleep(6 * time.Second)
		os.Exit(0)
	}
}

func helperFixture(t *testing.T) (Python, string) {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "helper.py"), []byte("# process test placeholder\n"), 0600); err != nil {
		t.Fatal(err)
	}
	return Python{path: os.Args[0], prefix: []string{"-test.run=^TestDesktopHelperProcess$", "--"}}, root
}

func TestRunHelperRejectsInheritedPipes(t *testing.T) {
	python, root := helperFixture(t)
	pidFile := filepath.Join(root, "child.pid")
	t.Setenv("FAIRYC_DESKTOP_HELPER_MODE", "orphan")
	t.Setenv("FAIRYC_DESKTOP_HELPER_PID", pidFile)
	t.Cleanup(func() {
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
			_ = process.Kill() // Only this test's own descendant.
		}
	})
	data, err := RunHelper(python, root, "helper.py")
	if !errors.Is(err, exec.ErrWaitDelay) || data != nil {
		t.Fatalf("inherited pipes must fail closed: data=%q error=%v", data, err)
	}
}

func TestRunHelperOutputBoundaries(t *testing.T) {
	python, root := helperFixture(t)
	for _, stream := range []string{"stdout", "stderr"} {
		for _, size := range []int{helperOutputBytes - 1, helperOutputBytes, helperOutputBytes + 1} {
			t.Run(fmt.Sprintf("%s/%d", stream, size), func(t *testing.T) {
				payload := bytes.Repeat([]byte{'x'}, size)
				data, err := RunHelperWithOptions(python, root, "helper.py", nil, HelperRunOptions{Run: func(command *exec.Cmd) error {
					writer := command.Stdout
					if stream == "stderr" {
						writer = command.Stderr
					}
					// Split writes across the cap, including a write to a full buffer.
					for offset := 0; offset < len(payload); {
						end := min(offset+1024, len(payload))
						if n, err := writer.Write(payload[offset:end]); err != nil || n != end-offset {
							t.Fatalf("write: n=%d error=%v", n, err)
						}
						offset = end
					}
					return nil
				}})
				if size > helperOutputBytes {
					if err == nil || !strings.Contains(err.Error(), "64 KiB") || data != nil {
						t.Fatalf("oversize accepted: bytes=%d error=%v", len(data), err)
					}
				} else if err != nil || stream == "stdout" && !bytes.Equal(data, payload) || stream == "stderr" && len(data) != 0 {
					t.Fatalf("boundary rejected: bytes=%d error=%v", len(data), err)
				}
			})
		}
	}
}

func TestRunHelperPreservesArgumentsAndDefaults(t *testing.T) {
	python, root := helperFixture(t)
	beforePrefix := slices.Clone(python.prefix)
	arguments := []string{"--output", "path with spaces;$(literal)"}
	beforeArguments := slices.Clone(arguments)
	data, err := RunHelperWithOptions(python, root, "helper.py", arguments, HelperRunOptions{Run: func(command *exec.Cmd) error {
		want := append([]string{python.path}, beforePrefix...)
		want = append(want, "-E", "-s", "-B", filepath.Join(root, "helper.py"))
		want = append(want, beforeArguments...)
		if !slices.Equal(command.Args, want) || command.Dir != root || command.WaitDelay != 2*time.Second || command.Cancel == nil {
			t.Fatalf("invalid command configuration: %+v", command)
		}
		_, err := io.WriteString(command.Stdout, "receipt")
		return err
	}})
	if err != nil || string(data) != "receipt" || !slices.Equal(arguments, beforeArguments) || !slices.Equal(python.prefix, beforePrefix) {
		t.Fatalf("arguments or successful output changed: %q %v", data, err)
	}
	t.Setenv("FAIRYC_DESKTOP_HELPER_MODE", "success")
	data, err = RunHelperWithOptions(python, root, "helper.py", nil, HelperRunOptions{})
	if err != nil || string(data) != "complete receipt" {
		t.Fatalf("default process runner: %q %v", data, err)
	}
}

func TestRunHelperFailuresAndCancellation(t *testing.T) {
	python, root := helperFixture(t)
	failure := errors.New("process I/O failed")
	cause := errors.New("caller stopped packaging")
	for _, tc := range []struct {
		name   string
		cancel bool
		runErr error
	}{
		{"process error", false, failure},
		{"cancel at final output", true, nil},
		{"cancel with process error", true, failure},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			data, err := RunHelperWithOptions(python, root, "helper.py", nil, HelperRunOptions{Context: ctx, Run: func(command *exec.Cmd) error {
				_, _ = io.WriteString(command.Stdout, "partial receipt")
				if tc.cancel {
					cancel(cause)
				}
				return tc.runErr
			}})
			if err == nil || data != nil || tc.runErr != nil && !errors.Is(err, tc.runErr) || tc.cancel && (!errors.Is(err, context.Canceled) || !errors.Is(err, cause)) {
				t.Fatalf("lost process/cancellation cause: data=%q error=%v", data, err)
			}
		})
	}
	for _, expired := range []bool{false, true} {
		ctx, cancel := context.WithCancel(context.Background())
		want := context.Canceled
		if expired {
			cancel()
			ctx, cancel = context.WithDeadline(context.Background(), time.Unix(0, 0))
			want = context.DeadlineExceeded
		} else {
			cancel()
		}
		data, err := RunHelperWithOptions(python, filepath.Join(root, "missing"), "helper.py", nil, HelperRunOptions{Context: ctx, Run: func(*exec.Cmd) error {
			t.Fatal("pre-cancelled helper started a process")
			return nil
		}})
		cancel()
		if !errors.Is(err, want) || data != nil {
			t.Fatalf("pre-cancel/deadline lost: data=%q error=%v", data, err)
		}
	}
	if _, err := RunHelperWithOptions(python, root, "helper.py", nil, HelperRunOptions{}, HelperRunOptions{}); err == nil {
		t.Fatal("multiple option values accepted")
	}
	started := false
	if _, err := RunHelperWithOptions(python, root, "missing.py", nil, HelperRunOptions{Run: func(*exec.Cmd) error { started = true; return nil }}); err == nil || started {
		t.Fatal("missing helper reached process I/O")
	}
}

type cancelHelperWriter struct {
	io.Writer
	cancel context.CancelFunc
}

func (writer cancelHelperWriter) Write(data []byte) (int, error) {
	n, err := writer.Writer.Write(data)
	writer.cancel()
	return n, err
}

func TestRunHelperCancelsRunningProcess(t *testing.T) {
	python, root := helperFixture(t)
	t.Setenv("FAIRYC_DESKTOP_HELPER_MODE", "block")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	data, err := RunHelperWithOptions(python, root, "helper.py", nil, HelperRunOptions{Context: ctx, Run: func(command *exec.Cmd) error {
		command.Stdout = cancelHelperWriter{Writer: command.Stdout, cancel: cancel}
		return command.Run()
	}})
	if !errors.Is(err, context.Canceled) || data != nil {
		t.Fatalf("running process cancellation: data=%q error=%v", data, err)
	}
}
