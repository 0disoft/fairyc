package cli

import (
	"runtime"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
)

func TestParseHostRunOptionsRequiresClosedProfile(t *testing.T) {
	t.Parallel()
	options, err := parseHostRunOptions([]string{"main.fy", "--source", "helper.fy", "--profile", "optimized"})
	if err != nil {
		t.Fatal(err)
	}
	if options.source != "main.fy" || strings.Join(options.additionalSources, ",") != "helper.fy" || options.profile != cdriver.OptimizedProfile {
		t.Fatalf("options = %+v", options)
	}
	for _, test := range []struct {
		arguments []string
		contains  string
	}{
		{nil, "expects one entry source file"},
		{[]string{"main.fy"}, "requires --profile"},
		{[]string{"--profile", "low", "main.fy"}, "entry source file before"},
		{[]string{"main.fy", "--cc", "msvc"}, "does not accept flag"},
		{[]string{"main.fy", "--profile", "fast"}, "profile low"},
		{[]string{"main.fy", "--profile", "low", "extra"}, "does not accept flag"},
		{[]string{"main.fy", "--profile", "low", "--profile", "optimized"}, "more than once"},
		{[]string{"main.fy", "--source", "--profile", "low"}, "requires a value for --source"},
	} {
		if _, err := parseHostRunOptions(test.arguments); err == nil || !strings.Contains(err.Error(), test.contains) {
			t.Errorf("parseHostRunOptions(%q) error = %v; want containing %q", test.arguments, err, test.contains)
		}
	}
}

func TestValidateWindowsHostUnitRejectsTargetSensitiveFeatures(t *testing.T) {
	t.Parallel()
	if err := validateWindowsHostUnit(cbackend.Unit{Features: []string{"checked-arithmetic", "static-utf8-string"}}); err != nil {
		t.Fatalf("target-neutral unit error = %v", err)
	}
	for _, unit := range []cbackend.Unit{
		{ForeignSymbols: []string{"puts"}},
		{Features: []string{"repr-c-struct"}},
		{Features: []string{"c-long"}},
		{Features: []string{"foreign-c-call"}},
	} {
		if err := validateWindowsHostUnit(unit); err == nil {
			t.Errorf("validateWindowsHostUnit(%+v) succeeded", unit)
		}
	}
}

func TestWindowsHostTrapCategoryUsesProductTrapBase(t *testing.T) {
	t.Parallel()
	if category, found := windowsHostTrapCategory(productTrapExitBase + 3); !found || category != "division-by-zero" {
		t.Fatalf("trap = %q, %t", category, found)
	}
	if category, found := windowsHostTrapCategory(42); found || category != "" {
		t.Fatalf("ordinary result became trap = %q, %t", category, found)
	}
}

func TestExperimentalRunHostFailsClosedOutsideWindowsAMD64(t *testing.T) {
	if runtime.GOOS == "windows" && runtime.GOARCH == "amd64" {
		t.Skip("live MSVC execution belongs to the configured Windows host-smoke intent")
	}
	root := t.TempDir()
	writeTestSource(t, root, "main.fy", "module main;\nfn main() -> i32 { 42 }\n")
	code, stdout, stderr := runForTest([]string{"experimental", "run-host", "main.fy", "--profile", "low"}, root)
	if code != ExitBuildFailure || stdout != "" || !strings.Contains(stderr, "requires windows/amd64") {
		t.Fatalf("Run() = code %d, stdout %q, stderr %q", code, stdout, stderr)
	}
}
