//go:build windows

package cdriver

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestWindowsExecutionDrainHelper(t *testing.T) {
	mode := os.Getenv("FAIRYC_EXECUTION_DRAIN_MODE")
	if mode == "" {
		return
	}
	if mode == "descendant" {
		if err := os.WriteFile(os.Getenv("FAIRYC_EXECUTION_DRAIN_PID"), []byte(strconv.Itoa(os.Getpid())), 0600); err != nil {
			os.Exit(2)
		}
		time.Sleep(30 * time.Second)
		os.Exit(0)
	}
	_, _ = os.Stdout.Write([]byte("out\x00\xff"))
	_, _ = os.Stderr.Write([]byte("err\x00\xff"))
	if mode == "success" {
		os.Exit(0)
	}
	if mode == "failure" {
		os.Exit(23)
	}
	// Use the original test image so the descendant cannot hold the owned
	// temporary program.exe open after its parent has exited.
	child := exec.Command(os.Getenv("FAIRYC_EXECUTION_DRAIN_IMAGE"), "-test.run=^TestWindowsExecutionDrainHelper$")
	child.Env = replaceWindowsEnvironment(os.Environ(), "FAIRYC_EXECUTION_DRAIN_MODE", "descendant")
	child.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	child.Stdout, child.Stderr = os.Stdout, os.Stderr
	if err := child.Start(); err != nil {
		os.Exit(2)
	}
	if mode == "timeout" {
		time.Sleep(30 * time.Second)
	}
	os.Exit(0)
}

func TestWindowsExecutionBoundsInheritedOutput(t *testing.T) {
	image, err := os.ReadFile(os.Args[0])
	if err != nil {
		t.Fatal(err)
	}
	for _, trap := range []bool{false, true} {
		for _, mode := range []string{"exited", "timeout"} {
			t.Run(fmt.Sprintf("trap=%t/%s", trap, mode), func(t *testing.T) {
				pidFile := filepath.Join(t.TempDir(), "descendant.pid")
				t.Setenv("FAIRYC_EXECUTION_DRAIN_MODE", mode)
				t.Setenv("FAIRYC_EXECUTION_DRAIN_PID", pidFile)
				t.Setenv("FAIRYC_EXECUTION_DRAIN_IMAGE", os.Args[0])
				timeout := 10 * time.Second
				if mode == "timeout" {
					timeout = time.Second
				}
				done := make(chan error, 1)
				go func() {
					_, err := executeWindowsMSVCArtifactWithInputArgsAndTimeout(WindowsHostArtifact{Bytes: image, TrapControl: trap},
						"drain helper", nil, []string{"-test.run=^TestWindowsExecutionDrainHelper$"}, timeout)
					done <- err
				}()
				// Always kill the exact descendant and join the execution, even if
				// the old unbounded implementation fails this regression.
				var process *os.Process
				joined := false
				defer func() {
					if process != nil {
						_ = process.Kill()
						defer process.Release()
					}
					if !joined {
						select {
						case <-done:
						case <-time.After(12 * time.Second):
							t.Error("execution did not finish after descendant cleanup")
						}
					}
				}()
				deadline := time.Now().Add(5 * time.Second)
				for time.Now().Before(deadline) {
					data, _ := os.ReadFile(pidFile)
					if pid, parseErr := strconv.Atoi(string(data)); parseErr == nil {
						process, err = os.FindProcess(pid)
						if err != nil {
							t.Fatal(err)
						}
						break
					}
					time.Sleep(10 * time.Millisecond)
				}
				if process == nil {
					t.Fatal("descendant did not report its PID")
				}
				select {
				case err := <-done:
					joined = true
					if mode == "timeout" {
						if err == nil || !strings.Contains(err.Error(), "executable exceeded 1s") {
							t.Fatalf("lost timeout diagnostic: %v", err)
						}
					} else if !errors.Is(err, exec.ErrWaitDelay) {
						t.Fatalf("incomplete output must fail with ErrWaitDelay: %v", err)
					}
				case <-time.After(4 * time.Second):
					t.Fatal("execution remained blocked by descendant output handles")
				}
			})
		}
	}
}

func TestWindowsExecutionDrainPreservesCompletedOutput(t *testing.T) {
	image, err := os.ReadFile(os.Args[0])
	if err != nil {
		t.Fatal(err)
	}
	for _, trap := range []bool{false, true} {
		for _, code := range []int32{0, 23} {
			t.Run(fmt.Sprintf("trap=%t/exit=%d", trap, code), func(t *testing.T) {
				mode := "success"
				if code != 0 {
					mode = "failure"
				}
				t.Setenv("FAIRYC_EXECUTION_DRAIN_MODE", mode)
				out, err := executeWindowsMSVCArtifactWithInputArgsAndTimeout(WindowsHostArtifact{Bytes: image, TrapControl: trap},
					"drain helper", nil, []string{"-test.run=^TestWindowsExecutionDrainHelper$"}, 5*time.Second)
				if err != nil || out.ExitCode != code || out.Stdout != "out\x00\xff" || out.Stderr != "err\x00\xff" || out.TrapControl != "" {
					t.Fatalf("completed output changed: %+v, %v", out, err)
				}
			})
		}
	}
}
