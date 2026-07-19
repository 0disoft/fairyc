package cli

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/target"
)

func TestParseTargetBuildOptionsRequiresClosedWindowsProfile(t *testing.T) {
	t.Parallel()
	options, err := parseTargetBuildOptions([]string{
		"main.fy", "--output", "build/fairy.exe", "--profile", "optimized", "--target", string(target.X8664WindowsMSVC),
	})
	if err != nil {
		t.Fatal(err)
	}
	if options.source != "main.fy" || options.targetID != target.X8664WindowsMSVC || options.profile != cdriver.OptimizedProfile || options.output != "build/fairy.exe" {
		t.Fatalf("options = %+v", options)
	}
	for _, test := range []struct {
		arguments []string
		contains  string
	}{
		{nil, "expects one source file"},
		{[]string{"--target", string(target.X8664WindowsMSVC)}, "source file before"},
		{[]string{"main.fy", "--profile", "low", "--output", "build/main.exe"}, "requires --target"},
		{[]string{"main.fy", "--target", string(target.X8664WindowsMSVC), "--output", "build/main.exe"}, "requires --profile"},
		{[]string{"main.fy", "--target", string(target.X8664WindowsMSVC), "--profile", "low"}, "requires --output"},
		{[]string{"main.fy", "--target", string(target.X8664LinuxGNU), "--profile", "low", "--output", "build/main.exe"}, "currently requires --target"},
		{[]string{"main.fy", "--target", string(target.X8664WindowsMSVC), "--profile", "fast", "--output", "build/main.exe"}, "profile low"},
		{[]string{"main.fy", "--target", string(target.X8664WindowsMSVC), "--profile", "low", "--output", "build/main"}, "end with .exe"},
		{[]string{"main.fy", "--target", string(target.X8664WindowsMSVC), "--profile", "low", "--output", "build/../main.exe"}, "canonical relative"},
		{[]string{"main.fy", "--target", string(target.X8664WindowsMSVC), "--profile", "low", "--output", "build/main.exe", "--force", "yes"}, "does not accept"},
	} {
		if _, err := parseTargetBuildOptions(test.arguments); err == nil || !strings.Contains(err.Error(), test.contains) {
			t.Errorf("parseTargetBuildOptions(%q) error = %v; want containing %q", test.arguments, err, test.contains)
		}
	}
}

func TestExperimentalBuildTargetFailsClosedOutsideWindowsAMD64(t *testing.T) {
	if runtime.GOOS == "windows" && runtime.GOARCH == "amd64" {
		t.Skip("live MSVC target build belongs to the configured Windows host-smoke environment")
	}
	root := t.TempDir()
	writeTestSource(t, root, "main.fy", "module main;\nfn main() -> i32 { 42 }\n")
	code, stdout, stderr := runForTest([]string{
		"experimental", "build-target", "main.fy",
		"--target", string(target.X8664WindowsMSVC),
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

func TestWindowsTargetSupportSourceAdmitsOnlyProvenStandardLibraryCall(t *testing.T) {
	t.Parallel()
	source, err := windowsTargetSupportSource([]string{
		"fy_std_text_utf8_valid_v0",
		"fy_std_buffer_initialize_v0",
		"fy_std_buffer_resize_zeroed_v0",
		"fy_std_buffer_drop_v0",
		"fy_std_arena_initialize_v0",
		"fy_std_region_begin_v0",
		"fy_std_region_allocate_zeroed_v0",
		"fy_std_region_reset_v0",
		"fy_std_arena_drop_v0",
		"fy_std_system_heap_allocate_v0",
		"fy_std_system_heap_deallocate_v0",
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{
		"fy_std_system_heap_allocate_v0", "fy_std_system_heap_deallocate_v0",
		"fy_std_buffer_initialize_v0", "fy_std_buffer_resize_zeroed_v0", "fy_std_buffer_drop_v0",
		"fy_std_arena_initialize_v0", "fy_std_region_begin_v0",
		"fy_std_region_allocate_zeroed_v0", "fy_std_region_reset_v0", "fy_std_arena_drop_v0",
		"fy_std_text_utf8_valid_v0", "fy_runtime_trap",
	} {
		if !strings.Contains(source, required) {
			t.Errorf("support source is missing %q", required)
		}
	}
	if strings.Contains(source, "fy_std_string_copy_v0") || strings.Contains(source, "_POSIX_C_SOURCE") {
		t.Fatal("support source widened beyond the proven Windows SystemHeap, Buffer, Arena, and text-validation components")
	}
}

func TestExperimentalBuildTargetRejectsUncontractedForeignSymbolBeforeToolchainResolution(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeTestSource(t, root, "main.fy", `module main;
@link_name("fairyc_missing")
unsafe extern C fn missing(value: i32) -> i32;
fn main() -> i32 { unsafe { missing(0) } }
`)
	code, stdout, stderr := runForTest([]string{
		"experimental", "build-target", "main.fy",
		"--target", string(target.X8664WindowsMSVC),
		"--profile", "low", "--output", "build/main.exe",
	}, root)
	if code != ExitInvalidInvocation || stdout != "" || !strings.Contains(stderr, "foreign symbol \"fairyc_missing\" has no closed x86_64-windows-msvc companion contract") {
		t.Fatalf("Run() = code %d, stdout %q, stderr %q", code, stdout, stderr)
	}
}

func TestExperimentalBuildTargetRejectsUnprovenWindowsStandardLibrarySymbolBeforeToolchainResolution(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeTestSource(t, root, "main.fy", `module main;
@link_name("fy_std_string_copy_v0")
unsafe extern C fn copy(input: *void, output: *void) -> u32;
fn main() -> i32 { if unsafe { copy(null, null) } == 0 { 42 } else { 1 } }
`)
	code, stdout, stderr := runForTest([]string{
		"experimental", "build-target", "main.fy",
		"--target", string(target.X8664WindowsMSVC),
		"--profile", "low", "--output", "build/main.exe",
	}, root)
	if code != ExitInvalidInvocation || stdout != "" || !strings.Contains(stderr, "has no x86_64-windows-msvc source contract") {
		t.Fatalf("Run() = code %d, stdout %q, stderr %q", code, stdout, stderr)
	}
}
