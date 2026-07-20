package cli

import (
	"fmt"
	"runtime"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/target"
)

func TestParseTargetRunOptionsRequiresClosedWindowsProfile(t *testing.T) {
	t.Parallel()
	options, err := parseTargetRunOptions([]string{
		"main.fy", "--source", "helper.fy", "--profile", "optimized", "--target", string(target.X8664WindowsMSVC),
	})
	if err != nil {
		t.Fatal(err)
	}
	if options.source != "main.fy" || strings.Join(options.additionalSources, ",") != "helper.fy" || options.targetID != target.X8664WindowsMSVC || options.profile != cdriver.OptimizedProfile {
		t.Fatalf("options = %+v", options)
	}
	for _, test := range []struct {
		arguments []string
		contains  string
	}{
		{nil, "expects one entry source file"},
		{[]string{"--target", string(target.X8664WindowsMSVC)}, "source file before"},
		{[]string{"main.fy", "--profile", "low"}, "requires --target"},
		{[]string{"main.fy", "--target", string(target.X8664WindowsMSVC)}, "requires --profile"},
		{[]string{"main.fy", "--target", string(target.X8664LinuxGNU), "--profile", "low"}, "currently requires --target"},
		{[]string{"main.fy", "--target", string(target.X8664WindowsMSVC), "--profile", "fast"}, "profile low"},
		{[]string{"main.fy", "--target", string(target.X8664WindowsMSVC), "--profile", "low", "--output", "build/main.exe"}, "does not accept"},
		{[]string{"main.fy", "--target", string(target.X8664WindowsMSVC), "--target", string(target.X8664WindowsMSVC), "--profile", "low"}, "more than once"},
		{[]string{"main.fy", "--source", "--profile", "low"}, "requires a value for --source"},
	} {
		if _, err := parseTargetRunOptions(test.arguments); err == nil || !strings.Contains(err.Error(), test.contains) {
			t.Errorf("parseTargetRunOptions(%q) error = %v; want containing %q", test.arguments, err, test.contains)
		}
	}
	overLimit := []string{"main.fy"}
	for index := 0; index < maxExplicitInputFiles; index++ {
		overLimit = append(overLimit, "--source", fmt.Sprintf("module-%d.fy", index))
	}
	overLimit = append(overLimit, "--target", string(target.X8664WindowsMSVC), "--profile", "low")
	if _, err := parseTargetRunOptions(overLimit); err == nil || !strings.Contains(err.Error(), "accepts at most 64 source files") {
		t.Fatalf("over-limit source error = %v", err)
	}
}

func TestExperimentalRunTargetFailsClosedOutsideWindowsAMD64(t *testing.T) {
	if runtime.GOOS == "windows" && runtime.GOARCH == "amd64" {
		t.Skip("live MSVC target execution belongs to the configured Windows host-smoke environment")
	}
	root := t.TempDir()
	writeTestSource(t, root, "main.fy", "module main;\nfn main() -> i32 { 42 }\n")
	code, stdout, stderr := runForTest([]string{
		"experimental", "run-target", "main.fy",
		"--target", string(target.X8664WindowsMSVC), "--profile", "low",
	}, root)
	if code != ExitBuildFailure || stdout != "" || !strings.Contains(stderr, "requires windows/amd64") {
		t.Fatalf("Run() = code %d, stdout %q, stderr %q", code, stdout, stderr)
	}
}
