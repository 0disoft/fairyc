package cli

import (
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/target"
)

func TestStableNativeRuntimeCache(t *testing.T) {
	if runtime.GOARCH != "amd64" || (runtime.GOOS != "windows" && runtime.GOOS != "linux") {
		t.Skip("requires Windows or Linux amd64 native toolchain")
	}
	if runtime.GOOS == "windows" {
		t.Setenv("LocalAppData", t.TempDir())
	} else {
		t.Setenv("XDG_CACHE_HOME", t.TempDir())
	}
	root := t.TempDir()
	writeNativeWorkspace(t, root, true)
	writeTestSource(t, root, "src/main.fy", linuxStableStreams)
	if code, _, diag := runForTest([]string{"check"}, root); code != 0 {
		t.Fatalf("check: %d %s", code, diag)
	}
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		for _, want := range []cdriver.RuntimeObjectCacheStatus{cdriver.RuntimeCacheMiss, cdriver.RuntimeCacheHit} {
			var diag strings.Builder
			options := workspaceTargetOptions{profile: profile, locked: true}
			policy := workspaceCompilePolicy{stableDiagnostics: true, runTrapReceipt: true}
			var status cdriver.RuntimeObjectCacheStatus
			var exit int32
			var stdout, stderr string
			if runtime.GOOS == "windows" {
				options.targetID = target.X8664WindowsMSVC
				compiled, code := compileWindowsWorkspace(root, options, policy, &diag)
				if code != 0 {
					t.Fatalf("compile: %d %s", code, diag.String())
				}
				status = compiled.artifact.RuntimeCache
				outcome, err := cdriver.ExecuteWindowsMSVCTargetWithInput(compiled.artifact, strings.NewReader("A"))
				if err != nil {
					t.Fatal(err)
				}
				exit, stdout, stderr = outcome.ExitCode, outcome.Stdout, outcome.Stderr
			} else {
				options.targetID = target.X8664LinuxGNU
				compiled, code := compileLinuxWorkspacePolicy(root, options, policy, &diag)
				if code != 0 {
					t.Fatalf("compile: %d %s", code, diag.String())
				}
				status = compiled.artifact.RuntimeCache
				outcome, err := cdriver.ExecuteLinuxTargetWithOptions(compiled.artifact, cdriver.LinuxExecutionOptions{Input: strings.NewReader("A"), Timeout: time.Minute})
				if err != nil {
					t.Fatal(err)
				}
				exit, stdout, stderr = outcome.ExitCode, outcome.Stdout, outcome.Stderr
			}
			if status != want || exit != 131 || stdout != "한글\n" || stderr != "diagnostic\n" {
				t.Fatalf("%s %s: cache=%s exit=%d stdout=%q stderr=%q", profile.Name, want, status, exit, stdout, stderr)
			}
		}
		args := []string{"run", "--locked"}
		if profile == cdriver.OptimizedProfile {
			args = append(args, "--release")
		}
		var out, diag strings.Builder
		code := RunWithInput(args, root, strings.NewReader("B"), &out, &diag)
		if code != ExitProgramTrap || out.String() != "" || diag.String() != "trap: assertion-failed\n" {
			t.Fatalf("warm CLI trap: %d %q %q", code, out.String(), diag.String())
		}
	}
}
