package cdriver

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

func TestBoundedCommandHelper(t *testing.T) {
	mode := os.Getenv("FAIRYC_BOUNDED_COMMAND_TEST")
	if mode == "" {
		return
	}
	if mode == "child" {
		if err := os.WriteFile(os.Getenv("FAIRYC_BOUNDED_COMMAND_PID"), []byte(strconv.Itoa(os.Getpid())), 0o600); err != nil {
			os.Exit(2)
		}
		time.Sleep(30 * time.Second)
		os.Exit(0)
	}
	if mode == "success" {
		fmt.Print("finished")
		os.Exit(0)
	}
	child := exec.CommandContext(context.Background(), os.Args[0], "-test.run=^TestBoundedCommandHelper$")
	child.Env = append(os.Environ(), "FAIRYC_BOUNDED_COMMAND_TEST=child")
	configureToolCancellation(child)
	child.Stdout, child.Stderr = os.Stdout, os.Stderr
	if err := child.Start(); err != nil {
		os.Exit(2)
	}
	fmt.Print("incomplete compiler output")
	os.Exit(0)
}

func TestRunBoundedRejectsInheritedOutputPipes(t *testing.T) {
	pidFile := filepath.Join(t.TempDir(), "child.pid")
	t.Setenv("FAIRYC_BOUNDED_COMMAND_TEST", "orphan")
	t.Setenv("FAIRYC_BOUNDED_COMMAND_PID", pidFile)
	type result struct {
		output string
		err    error
	}
	finished := make(chan result, 1)
	started := time.Now()
	go func() {
		output, err := runBounded(os.Args[0], "-test.run=^TestBoundedCommandHelper$")
		finished <- result{output, err}
	}()
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
		t.Fatalf("child did not report its PID: %q, %v", pidData, err)
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		t.Fatal(err)
	}
	defer process.Release()
	// The helper stays alive for 30s, beyond every test wait below. Windows
	// also retains its exact process handle until cleanup has finished.
	defer process.Kill()
	select {
	case got := <-finished:
		if !errors.Is(got.err, exec.ErrWaitDelay) {
			t.Fatalf("inherited pipes must fail: output=%q, error=%v", got.output, got.err)
		}
		t.Logf("returned after %s with bounded inherited pipes", time.Since(started))
	case <-time.After(8 * time.Second):
		_ = process.Kill()
		select {
		case <-finished:
		case <-time.After(3 * time.Second):
			t.Fatal("command did not finish after child cleanup")
		}
		t.Fatal("command remained blocked by an orphan's output pipes")
	}
}

func TestRunBoundedSuccessfulCommand(t *testing.T) {
	t.Setenv("FAIRYC_BOUNDED_COMMAND_TEST", "success")
	out, err := runBounded(os.Args[0], "-test.run=^TestBoundedCommandHelper$")
	if err != nil || out != "finished" {
		t.Fatalf("successful command: output=%q, error=%v", out, err)
	}
}
