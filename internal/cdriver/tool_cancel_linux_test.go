//go:build linux

package cdriver

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestLinuxToolCancellationHelper(t *testing.T) {
	mode := os.Getenv("FAIRYC_LINUX_TOOL_CANCEL_TEST")
	if mode == "" {
		return
	}
	if mode == "child" {
		if err := os.WriteFile(os.Getenv("FAIRYC_LINUX_TOOL_CANCEL_PID"), []byte(strconv.Itoa(os.Getpid())), 0600); err != nil {
			os.Exit(2)
		}
		time.Sleep(30 * time.Second)
		os.Exit(0)
	}
	child := exec.Command(os.Args[0], "-test.run=^TestLinuxToolCancellationHelper$")
	child.Env = append(os.Environ(), "FAIRYC_LINUX_TOOL_CANCEL_TEST=child")
	child.Stdout, child.Stderr = os.Stdout, os.Stderr
	if err := child.Run(); err != nil {
		os.Exit(2)
	}
	os.Exit(0)
}

func TestLinuxToolCancellationTerminatesChildren(t *testing.T) {
	pidFile := filepath.Join(t.TempDir(), "child.pid")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestLinuxToolCancellationHelper$")
	command.Env = append(os.Environ(), "FAIRYC_LINUX_TOOL_CANCEL_TEST=parent", "FAIRYC_LINUX_TOOL_CANCEL_PID="+pidFile)
	configureToolCancellation(command)
	command.WaitDelay = 200 * time.Millisecond
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	// Cancel only once the compiler's descendant is actually running.
	var data []byte
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		data, _ = os.ReadFile(pidFile)
		if len(data) != 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	pid, err := strconv.Atoi(string(data))
	if err != nil {
		cancel()
		_ = command.Wait()
		t.Fatalf("child did not report PID: %q: %v", data, err)
	}
	child, err := os.FindProcess(pid)
	if err != nil {
		cancel()
		_ = command.Wait()
		t.Fatal(err)
	}
	defer child.Release()
	defer child.Kill() // A regression must not leave its own helper behind.
	cancel()
	if err := command.Wait(); err == nil {
		t.Fatal("cancelled compiler reported success")
	}
	deadline = time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		status, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
		if os.IsNotExist(err) {
			return
		}
		if err != nil {
			t.Fatal(err)
		}
		// PID 1 in a container may retain a dead child as a zombie.
		close := strings.LastIndexByte(string(status), ')')
		if close >= 0 && len(status) > close+2 && status[close+2] == 'Z' {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("compiler child survived its parent's cancellation")
}
