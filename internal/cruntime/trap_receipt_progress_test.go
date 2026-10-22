package cruntime_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/cruntime"
	"github.com/0disoft/fairyc/internal/target"
)

func TestLinuxTrapReceiptProgressNative(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("requires Linux GNU amd64")
	}
	support, err := cruntime.LinuxRunSource(192)
	if err != nil {
		t.Fatal(err)
	}
	const call = "write(3, fy_trap_receipt + fy_trap_offset, fy_trap_receipt_length - fy_trap_offset)"
	if strings.Count(support, call) != 1 {
		t.Fatal("expected exactly one production receipt write call")
	}
	// Replace only the OS call; execute the production encoding and progress loop.
	support = strings.Replace(support, call, "fy_trap_probe_"+call, 1)
	declaration := "#include <unistd.h>\nstatic ssize_t fy_trap_probe_write(int, const void *, size_t);\n"
	// Independent wire oracle: category, location version, LE path length/start/end, path.
	expected := []byte{3, 1, 7, 0, 2, 0, 0, 0, 0, 0, 0, 0, 5, 0, 0, 0, 0, 0, 0, 0}
	expected = append(expected, []byte("main.fy")...)
	var literals []string
	for _, value := range expected {
		literals = append(literals, fmt.Sprint(value))
	}
	source := declaration + support + "\nstatic const unsigned char fy_expected[] = {" + strings.Join(literals, ",") + "};\n" + trapReceiptProgressProbe
	cases := []struct {
		name, argument, trace string
		exit                  int
	}{
		{"short-writes", "0", "1:2\n2:7\n3:27\n", 195},
		{"eintr-and-progress", "1", "1:0\n2:2\n3:2\n4:7\n5:27\n", 195},
		{"zero-progress", "2", "1:0\n", 254},
		{"failed-write", "3", "1:0\n", 254},
		{"zero-after-progress", "4", "1:2\n2:2\n", 254},
		{"failure-after-progress", "5", "1:2\n2:2\n", 254},
	}
	for _, cc := range []string{"gcc", "clang"} {
		for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
			t.Run(cc+"/"+profile.Name, func(t *testing.T) {
				tc, err := cdriver.Resolve(target.X8664LinuxGNU, cc)
				if err != nil {
					t.Fatal(err)
				}
				artifact, err := cdriver.Build(tc, profile, source, "typedef int fy_trap_probe_runtime;\n")
				if err != nil {
					t.Fatal(err)
				}
				executable := filepath.Join(t.TempDir(), "probe")
				if err := os.WriteFile(executable, artifact.Bytes, 0o700); err != nil {
					t.Fatal(err)
				}
				// Keep one CI leaf per compiler/profile; each runs every scenario.
				for _, test := range cases {
					ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					cmd := exec.CommandContext(ctx, executable, test.argument)
					output, runErr := cmd.CombinedOutput()
					deadlineErr := ctx.Err()
					cancel()
					var exitErr *exec.ExitError
					if deadlineErr != nil || !errors.As(runErr, &exitErr) || exitErr.ExitCode() != test.exit || string(output) != test.trace {
						t.Fatalf("%s: error=%v deadline=%v output=%q; want exit=%d trace=%q", test.name, runErr, deadlineErr, output, test.exit, test.trace)
					}
					t.Logf("%s: exit=%d, exact receipt suffix and progress verified", test.name, test.exit)
				}
			})
		}
	}
}

const trapReceiptProgressProbe = `
#include <stdio.h>
#include <string.h>

static const ssize_t fy_steps[][5] = {
    {2, 5, 20}, {-1, 2, -1, 5, 20}, {0}, {-1}, {2, 0}, {2, -1}
};
static const size_t fy_step_counts[] = {3, 5, 1, 1, 2, 2};
static size_t fy_scenario, fy_calls, fy_offset;
static const unsigned char *fy_first;

static ssize_t fy_trap_probe_write(int fd, const void *data, size_t length) {
    if (fy_calls == 0) fy_first = data;
    if (fd != 3 || fy_calls >= fy_step_counts[fy_scenario] ||
        fy_offset >= sizeof fy_expected || length != sizeof fy_expected - fy_offset ||
        data != fy_first + fy_offset || memcmp(data, fy_expected + fy_offset, length) != 0) {
        _Exit(253);
    }
    ssize_t result = fy_steps[fy_scenario][fy_calls++];
    if (result > 0) {
        if ((size_t)result > length) _Exit(253);
        fy_offset += (size_t)result;
    }
    if (printf("%zu:%zu\n", fy_calls, fy_offset) < 0 || fflush(stdout) != 0) _Exit(253);
    errno = fy_scenario == 1 ? EINTR : EIO;
    return result;
}

int main(int argc, char **argv) {
    if (argc != 2 || strlen(argv[1]) != 1 || argv[1][0] < '0' || argv[1][0] > '5') return 252;
    fy_scenario = (size_t)(argv[1][0] - '0');
    fy_runtime_trap_at_v0(3, "main.fy", 7, 2, 5);
}
`
