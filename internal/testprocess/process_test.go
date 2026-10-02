package testprocess

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

func TestProcessFixture(t *testing.T) {
	args := os.Args
	for i, arg := range args {
		if arg != "--test-child" {
			continue
		}
		mode := args[i+1]
		if mode == "timeout" {
			time.Sleep(10 * time.Second)
			os.Exit(0)
		}
		if mode == "invalid" {
			os.Stdout.Write([]byte{255})
			os.Exit(0)
		}
		data, _ := json.Marshal(args[i+2:])
		fmt.Print(string(data) + strings.Repeat("x", 70000))
		fmt.Fprint(os.Stderr, "한글\n"+strings.Repeat("y", 70000))
		os.Exit(7)
	}
}

func TestRun(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	args := []string{"", "a b", "\"quoted\"", "한글", "a;$()&|", "e\u0301"}
	data, _ := json.Marshal(args)
	got, err := Run(exe, t.TempDir(), append([]string{"-test.run=^TestProcessFixture$", "--", "--test-child", "outputs"}, args...), 10*time.Second)
	want := Observation{7, string(data) + strings.Repeat("x", 70000), "한글\n" + strings.Repeat("y", 70000)}
	if err != nil || got != want {
		t.Fatalf("capture mismatch: err=%v code=%d", err, got.ExitCode)
	}
	_, err = Run(exe, t.TempDir(), []string{"-test.run=^TestProcessFixture$", "--", "--test-child", "timeout"}, 250*time.Millisecond)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("timeout: %v", err)
	}
	_, err = Run(exe, t.TempDir(), []string{"-test.run=^TestProcessFixture$", "--", "--test-child", "invalid"}, 10*time.Second)
	if err == nil || !strings.Contains(err.Error(), "UTF-8") {
		t.Fatalf("invalid encoding: %v", err)
	}
}
