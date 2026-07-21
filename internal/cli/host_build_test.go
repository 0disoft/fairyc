package cli

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
)

func TestParseHostBuildOptionsRequiresClosedProfileAndExeOutput(t *testing.T) {
	t.Parallel()
	options, err := parseHostBuildOptions([]string{
		"main.fy", "--source", "helper.fy", "--output", "build/fairy.exe", "--profile", "optimized",
	})
	if err != nil {
		t.Fatal(err)
	}
	if options.source != "main.fy" || strings.Join(options.additionalSources, ",") != "helper.fy" || options.profile != cdriver.OptimizedProfile || options.output != "build/fairy.exe" {
		t.Fatalf("options = %+v", options)
	}
	for _, test := range []struct {
		arguments []string
		contains  string
	}{
		{nil, "expects one entry source file"},
		{[]string{"--profile", "low"}, "source file before"},
		{[]string{"main.fy", "--output", "build/main.exe"}, "requires --profile"},
		{[]string{"main.fy", "--profile", "low"}, "requires --output"},
		{[]string{"main.fy", "--profile", "fast", "--output", "build/main.exe"}, "profile low"},
		{[]string{"main.fy", "--profile", "low", "--output", "build/main"}, "end with .exe"},
		{[]string{"main.fy", "--profile", "low", "--output", "build/../main.exe"}, "canonical relative"},
		{[]string{"main.fy", "--profile", "low", "--output", "build/main.exe", "--force", "yes"}, "does not accept"},
		{[]string{"main.fy", "--source", "--profile", "low"}, "requires a value for --source"},
	} {
		if _, err := parseHostBuildOptions(test.arguments); err == nil || !strings.Contains(err.Error(), test.contains) {
			t.Errorf("parseHostBuildOptions(%q) error = %v; want containing %q", test.arguments, err, test.contains)
		}
	}
}

func TestExperimentalBuildHostFailsClosedOutsideWindowsAMD64(t *testing.T) {
	if runtime.GOOS == "windows" && runtime.GOARCH == "amd64" {
		t.Skip("live MSVC build belongs to the configured Windows host-smoke environment")
	}
	root := t.TempDir()
	writeTestSource(t, root, "main.fy", "module main;\nfn i32 main() { return 42; }\n")
	code, stdout, stderr := runForTest([]string{
		"experimental", "build-host", "main.fy",
		"--profile", "low", "--output", "build/main.exe",
	}, root)
	if code != ExitBuildFailure || stdout != "" || !strings.Contains(stderr, "requires windows/amd64") {
		t.Fatalf("Run() = code %d, stdout %q, stderr %q", code, stdout, stderr)
	}
	matches, err := filepath.Glob(filepath.Join(root, "build", "*.exe"))
	if err != nil || len(matches) != 0 {
		t.Fatalf("build artifacts = %v, %v", matches, err)
	}
}

func TestExperimentalBuildHostRejectsUnsafeOutputBeforeLoadingSource(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	code, stdout, stderr := runForTest([]string{
		"experimental", "build-host", "missing.fy",
		"--profile", "low", "--output", "build/../outside.exe",
	}, root)
	if code != ExitInvalidInvocation || stdout != "" || !strings.Contains(stderr, "canonical relative") || strings.Contains(stderr, "cannot load") {
		t.Fatalf("Run() = code %d, stdout %q, stderr %q", code, stdout, stderr)
	}
}
