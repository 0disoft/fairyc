//go:build linux && amd64

package cdriver

import (
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

func TestExecuteLinuxTargetCompletionTerminatesOwnedDescendants(t *testing.T) {
	support, err := cruntime.LinuxRunSource(128)
	if err != nil {
		t.Fatal(err)
	}
	for _, compiler := range []string{"gcc", "clang"} {
		t.Run(compiler, func(t *testing.T) {
			toolchain, err := Resolve(target.X8664LinuxGNU, compiler)
			if err != nil {
				t.Fatal(err)
			}
			artifact, err := Build(toolchain, LowProfile, `
#include <stdio.h>
#include <stdint.h>
#include <sys/types.h>
#include <unistd.h>
_Noreturn void fy_runtime_trap(uint32_t);
int main(int argc, char **argv) {
    int ready[2];
    if (argc != 3 || pipe(ready) != 0) return 2;
    pid_t child = fork();
    if (child < 0) return 3;
    if (child == 0) {
        close(ready[0]);
        FILE *output = fopen(argv[1], "w");
        if (output == NULL) _exit(4);
        if (fprintf(output, "%ld", (long)getpid()) < 0 || fclose(output) != 0) _exit(5);
        if (argv[2][0] == '1') { for (int fd = 0; fd <= 3; ++fd) close(fd); }
        if (write(ready[1], "r", 1) != 1) _exit(6);
        close(ready[1]);
        for (;;) pause();
    }
    close(ready[1]);
    char marker;
    if (read(ready[0], &marker, 1) != 1) return 7;
    close(ready[0]);
    if (argv[2][0] == '2') fy_runtime_trap(3);
    return 23;
}
`, support)
			if err != nil {
				t.Fatal(err)
			}
			for _, mode := range []string{"0", "1", "2"} {
				t.Run("mode="+mode, func(t *testing.T) {
					pidPath := filepath.Join(t.TempDir(), "child.pid")
					stopped := false
					t.Cleanup(func() {
						if stopped {
							return
						}
						if data, err := os.ReadFile(pidPath); err == nil {
							if pid, err := strconv.Atoi(string(data)); err == nil && pid > 1 {
								if process, err := os.FindProcess(pid); err == nil {
									_ = process.Kill()
									_ = process.Release()
								}
							}
						}
					})
					outcome, err := ExecuteLinuxTargetWithOptions(artifact, LinuxExecutionOptions{Input: strings.NewReader(""), Arguments: []string{pidPath, mode}, Timeout: 5 * time.Second})
					wantExit, wantTrap := int32(23), uint8(0)
					if mode == "2" {
						wantExit, wantTrap = 131, 3
					}
					if err != nil || outcome.ExitCode != wantExit || outcome.TrapCategory != wantTrap || outcome.Stdout != "" || outcome.Stderr != "" {
						t.Fatalf("owned completion: %+v, %v", outcome, err)
					}
					data, err := os.ReadFile(pidPath)
					if err != nil {
						t.Fatal(err)
					}
					pid, err := strconv.Atoi(string(data))
					if err != nil || pid <= 1 {
						t.Fatalf("invalid child PID: %q", data)
					}
					deadline := time.Now().Add(2 * time.Second)
					for time.Now().Before(deadline) {
						status, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
						if os.IsNotExist(err) {
							stopped = true
							return
						}
						if err != nil {
							t.Fatal(err)
						}
						end := strings.LastIndexByte(string(status), ')')
						if end >= 0 && len(status) > end+2 && status[end+2] == 'Z' {
							stopped = true
							return
						}
						time.Sleep(10 * time.Millisecond)
					}
					t.Fatal("owned descendant survived normal/trap completion")
				})
			}
		})
	}
}
