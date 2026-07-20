package cli

import (
	"runtime"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/target"
)

func TestParseExecTargetOptionsUsesClosedWindowsProfile(t *testing.T) {
	t.Parallel()
	options, err := parseTargetExecutionOptions("exec-target", []string{
		"main.fy", "--target", string(target.X8664WindowsMSVC), "--profile", "optimized",
	})
	if err != nil {
		t.Fatal(err)
	}
	if options.source != "main.fy" || options.targetID != target.X8664WindowsMSVC || options.profile.Name != "optimized" {
		t.Fatalf("options = %+v", options)
	}
	if _, err := parseTargetExecutionOptions("exec-target", []string{"main.fy", "--profile", "low"}); err == nil || !strings.Contains(err.Error(), "exec-target") {
		t.Fatalf("missing target error = %v", err)
	}
}

func TestExperimentalExecTargetFailsClosedOutsideWindowsAMD64(t *testing.T) {
	if runtime.GOOS == "windows" && runtime.GOARCH == "amd64" {
		t.Skip("live MSVC target execution belongs to the configured Windows host-smoke environment")
	}
	root := t.TempDir()
	writeTestSource(t, root, "main.fy", "module main;\nfn main() -> i32 { 42 }\n")
	code, stdout, stderr := runForTest([]string{
		"experimental", "exec-target", "main.fy",
		"--target", string(target.X8664WindowsMSVC), "--profile", "low",
	}, root)
	if code != ExitBuildFailure || stdout != "" || !strings.Contains(stderr, "requires windows/amd64") {
		t.Fatalf("Run() = code %d, stdout %q, stderr %q", code, stdout, stderr)
	}
}
