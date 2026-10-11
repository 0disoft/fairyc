//go:build windows

package cdriver

import (
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

func TestToolCancellationHelper(t *testing.T) {
	mode := os.Getenv("FAIRYC_TOOL_CANCEL_TEST")
	if mode == "" {
		return
	}
	if mode == "child" {
		if err := os.WriteFile(os.Getenv("FAIRYC_TOOL_CANCEL_PID"), []byte(strconv.Itoa(os.Getpid())), 0600); err != nil {
			os.Exit(2)
		}
		time.Sleep(30 * time.Second)
		os.Exit(0)
	}
	if mode == "success" {
		fmt.Print("finished")
		os.Exit(0)
	}
	child := exec.Command(os.Args[0], "-test.run=^TestToolCancellationHelper$")
	child.Env = replaceWindowsEnvironment(os.Environ(), "FAIRYC_TOOL_CANCEL_TEST", "child")
	child.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := child.Run(); err != nil {
		os.Exit(2)
	}
	os.Exit(0)
}

func TestToolCompileDeadlineTerminatesChildrenAndUsesExplicitTimeout(t *testing.T) {
	pidFile := filepath.Join(t.TempDir(), "child.pid")
	environment := replaceWindowsEnvironment(os.Environ(), "FAIRYC_TOOL_CANCEL_TEST", "parent")
	environment = replaceWindowsEnvironment(environment, "FAIRYC_TOOL_CANCEL_PID", pidFile)
	_, err := runWindowsHostCommandWithLimitAndTimeout(os.Args[0], environment, "", 1024, time.Second, "-test.run=^TestToolCancellationHelper$")
	if err == nil || !strings.Contains(err.Error(), "exceeded 1s") {
		t.Fatalf("deadline result: %v", err)
	}
	data, err := os.ReadFile(pidFile)
	if err != nil {
		t.Fatalf("child did not report its PID: %v", err)
	}
	pid, err := strconv.Atoi(string(data))
	if err != nil {
		t.Fatal(err)
	}
	process, err := syscall.OpenProcess(0x1000, false, uint32(pid))
	if err == nil {
		defer syscall.CloseHandle(process)
		var code uint32
		if err := syscall.GetExitCodeProcess(process, &code); err != nil {
			t.Fatal(err)
		}
		if code == 259 {
			// Ensure a failed cancellation regression cannot leave the test child.
			cleanup, openErr := syscall.OpenProcess(1, false, uint32(pid))
			if openErr == nil {
				discard := syscall.TerminateProcess(cleanup, 2)
				_ = discard
				syscall.CloseHandle(cleanup)
			}
			t.Fatal("compiler child survived its parent's timeout")
		}
	} else if err != syscall.Errno(87) {
		t.Fatalf("cannot inspect timed-out child: %v", err)
	}
	environment = replaceWindowsEnvironment(environment, "FAIRYC_TOOL_CANCEL_TEST", "success")
	out, err := runWindowsHostCommandWithLimitAndTimeout(os.Args[0], environment, "", 1024, 2*time.Second, "-test.run=^TestToolCancellationHelper$")
	if err != nil || out != "finished" {
		t.Fatalf("explicit deadline successful command: %q, %v", out, err)
	}
	for _, timeout := range []time.Duration{0, -time.Second, 10*time.Minute + time.Nanosecond} {
		if _, err := runWindowsHostCommandWithLimitAndTimeout("must-not-run", nil, "", 1024, timeout); err == nil {
			t.Fatalf("accepted invalid deadline %s", timeout)
		}
	}
}
