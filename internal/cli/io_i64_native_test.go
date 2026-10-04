package cli

import (
	"bytes"
	"os"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/target"
	"github.com/0disoft/fairyc/internal/workspacecompiler"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

func TestI64ConsoleNative(t *testing.T) {
	if runtime.GOARCH != "amd64" || runtime.GOOS != "windows" && runtime.GOOS != "linux" {
		t.Skip("requires Windows MSVC or Linux GCC/Clang amd64")
	}
	source, err := os.ReadFile("../../examples/io_i64_quickstart/src/main.fy")
	if err != nil {
		t.Fatal(err)
	}
	var expected strings.Builder
	for _, value := range []int64{0, 1, -1, 9, -9, 10, -10, 2147483647, -2147483648, 2147483648, -2147483649, 4000000000, -4000000000, 9223372036854775807, -9223372036854775808} {
		expected.WriteString(strconv.FormatInt(value, 10))
		expected.WriteByte('\n')
	}
	root := t.TempDir()
	writeVecI64ViewWorkspace(t, root, string(source))
	workspace, err := workspaceloader.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	checked, err := workspacecompiler.Check(workspace)
	if err != nil || !checked.Verified {
		t.Fatalf("check: %v %+v", err, checked.Semantic)
	}
	storage, err := workspaceSDKStorage(workspace)
	if err != nil {
		t.Fatal(err)
	}
	for _, release := range []bool{false, true} {
		name := "dev"
		if release {
			name = "release"
		}
		t.Run(name, func(t *testing.T) {
			var code int
			var out, diag string
			if runtime.GOOS == "windows" {
				profile := cdriver.LowProfile
				if release {
					profile = cdriver.OptimizedProfile
				}
				var diagnostics bytes.Buffer
				artifact, buildCode := compileWindowsProgramPolicy(checked.MIR, tokenScannerEntry(t, checked), target.X8664WindowsMSVC, profile, true, storage, &diagnostics)
				if buildCode != ExitSuccess {
					t.Fatal(diagnostics.String())
				}
				outcome, err := cdriver.ExecuteWindowsMSVCTarget(artifact)
				if err != nil {
					t.Fatal(err)
				}
				code, out, diag = int(outcome.ExitCode), outcome.Stdout, outcome.Stderr
			} else {
				args := []string{"run"}
				if release {
					args = append(args, "--release", "--locked")
				}
				code, out, diag = runForTest(args, root)
			}
			if code != 0 || out != expected.String() || diag != expected.String() {
				t.Fatalf("exit=%d stdout=%q stderr=%q; want %q on both streams", code, out, diag, expected.String())
			}
		})
	}
}
