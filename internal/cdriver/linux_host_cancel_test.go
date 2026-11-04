//go:build linux && amd64

package cdriver

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/cruntime"
	"github.com/0disoft/fairyc/internal/target"
)

func TestExecuteLinuxTargetTimeoutTerminatesOwnedDescendants(t *testing.T) {
	toolchain, err := Resolve(target.X8664LinuxGNU, "gcc")
	if err != nil {
		t.Fatal(err)
	}
	support, err := cruntime.LinuxRunSource(128)
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := Build(toolchain, LowProfile, `
#include <stdio.h>
#include <sys/types.h>
#include <unistd.h>

int main(int argc, char **argv) {
    if (argc != 2) return 2;
    pid_t child = fork();
    if (child < 0) return 3;
    if (child == 0) {
        FILE *output = fopen(argv[1], "w");
        if (output == NULL) _exit(4);
        if (fprintf(output, "%ld", (long)getpid()) < 0 || fclose(output) != 0) _exit(5);
        for (;;) pause();
    }
    for (;;) pause();
}
`, support)
	if err != nil {
		t.Fatal(err)
	}

	pidPath := filepath.Join(t.TempDir(), "descendant.pid")
	t.Cleanup(func() {
		data, err := os.ReadFile(pidPath)
		if err != nil {
			return
		}
		pid, err := strconv.Atoi(string(data))
		if err != nil || pid <= 0 {
			return
		}
		process, err := os.FindProcess(pid)
		if err == nil {
			_ = process.Kill()
			_ = process.Release()
		}
	})
	timeout := 2 * time.Second
	_, err = ExecuteLinuxTargetWithOptions(artifact, LinuxExecutionOptions{
		Input: strings.NewReader(""), Timeout: timeout, Arguments: []string{pidPath},
	})
	if !errors.Is(err, context.DeadlineExceeded) || !strings.HasPrefix(err.Error(), "Linux executable exceeded 2s: ") {
		t.Fatalf("timeout error = %v", err)
	}

	data, err := os.ReadFile(pidPath)
	if err != nil {
		t.Fatalf("read descendant PID: %v", err)
	}
	pid, err := strconv.Atoi(string(data))
	if err != nil {
		t.Fatalf("parse descendant PID %q: %v", data, err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		status, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
		if os.IsNotExist(err) {
			return
		}
		if err != nil {
			t.Fatal(err)
		}
		close := strings.LastIndexByte(string(status), ')')
		if close >= 0 && len(status) > close+2 && status[close+2] == 'Z' {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("Linux target descendant survived its process-group timeout")
}

func TestExecuteLinuxTargetPreservesBinaryInputAndTrapReceipt(t *testing.T) {
	toolchain, err := Resolve(target.X8664LinuxGNU, "gcc")
	if err != nil {
		t.Fatal(err)
	}
	support, err := cruntime.LinuxRunSource(128)
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := Build(toolchain, LowProfile, `
#include <stdint.h>
#include <unistd.h>

_Noreturn void fy_runtime_trap(uint32_t);

int main(void) {
    unsigned char input[4];
    size_t offset = 0;
    while (offset < sizeof input) {
        ssize_t count = read(0, input + offset, sizeof input - offset);
        if (count <= 0) return 4;
        offset += (size_t)count;
    }
    if (input[0] != 0 || input[1] != 255 || input[2] != 128 || input[3] != 10) return 5;
    fy_runtime_trap(3);
}
`, support)
	if err != nil {
		t.Fatal(err)
	}

	outcome, err := ExecuteLinuxTargetWithOptions(artifact, LinuxExecutionOptions{
		Input: bytes.NewReader([]byte{0, 255, 128, 10}), Timeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	if outcome.ExitCode != 131 || outcome.TrapCategory != 3 || outcome.TrapControl != "\x03" ||
		outcome.TrapLocation != nil || outcome.Stdout != "" || outcome.Stderr != "" {
		t.Fatalf("outcome = %+v", outcome)
	}
}
