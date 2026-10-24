//go:build windows

package cdriver

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
	"testing"
	"time"
)

func TestRuntimePreprocessCancellationClosesInheritedPipes(t *testing.T) {
	testRuntimePreprocessInheritedPipes(t, "parent", true)
}

func TestRuntimePreprocessWaitBoundsOrphanedPipes(t *testing.T) {
	testRuntimePreprocessInheritedPipes(t, "orphan", false)
}

func TestRuntimePreprocessSuccessfulCommand(t *testing.T) {
	environment := replaceWindowsEnvironment(os.Environ(), "FAIRYC_TOOL_CANCEL_TEST", "success")
	out, err := preprocessRuntime(os.Args[0], environment, "", []string{"-test.run=^TestToolCancellationHelper$"})
	if err != nil || out != "finished" {
		t.Fatalf("preprocessor output: %q, %v", out, err)
	}
}

func testRuntimePreprocessInheritedPipes(t *testing.T, mode string, cancelOnReady bool) {
	t.Helper()
	pidFile := filepath.Join(t.TempDir(), "child.pid")
	environment := replaceWindowsEnvironment(os.Environ(), "FAIRYC_TOOL_CANCEL_TEST", mode)
	environment = replaceWindowsEnvironment(environment, "FAIRYC_TOOL_CANCEL_PID", pidFile)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	finished := make(chan error, 1)
	go func() {
		out, err := preprocessRuntimeContext(ctx, os.Args[0], environment, "", []string{"-test.run=^TestToolCancellationHelper$"})
		if out != "" {
			t.Error("incomplete preprocessor output was accepted")
		}
		finished <- err
	}()

	// Cancel only after the descendant owns both output pipes, so startup speed
	// cannot turn the regression into cancellation of an unstarted process.
	var pidData []byte
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		pidData, _ = os.ReadFile(pidFile)
		if len(pidData) != 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	pid, err := strconv.Atoi(string(pidData))
	if err != nil {
		t.Fatalf("child did not report a PID: %q, %v", pidData, err)
	}
	process, err := syscall.OpenProcess(0x1000|1, false, uint32(pid))
	if err != nil {
		t.Fatal(err)
	}
	defer syscall.CloseHandle(process)
	// Keep the exact process handle for cleanup; never kill a reused PID.
	defer syscall.TerminateProcess(process, 2)
	started := time.Now()
	if cancelOnReady {
		cancel()
	}
	select {
	case err := <-finished:
		if err == nil {
			t.Fatal("canceled preprocessing succeeded")
		}
	case <-time.After(8 * time.Second):
		_ = syscall.TerminateProcess(process, 2)
		select {
		case <-finished:
		case <-time.After(3 * time.Second):
			t.Fatal("preprocessing did not finish after test cleanup")
		}
		t.Fatal("preprocessing remained blocked by inherited output pipes after cancellation")
	}
	var code uint32
	if err := syscall.GetExitCodeProcess(process, &code); err != nil {
		t.Fatal(err)
	}
	if code == 259 {
		if cancelOnReady {
			t.Fatal("preprocessor child survived cancellation")
		}
	}
	t.Logf("preprocessing returned in %s; descendant exit code %d", time.Since(started), code)
}
