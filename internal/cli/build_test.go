package cli

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
)

func TestParseBuildOptionsRequiresExplicitClosedChoices(t *testing.T) {
	t.Parallel()
	options, err := parseBuildOptions([]string{
		"main.fy",
		"--source", "helper.fy",
		"--output", "build/hello",
		"--profile", "optimized",
		"--cc", "clang",
	})
	if err != nil {
		t.Fatal(err)
	}
	if options.source != "main.fy" || strings.Join(options.additionalSources, ",") != "helper.fy" || options.compilerName != "clang" || options.profile != cdriver.OptimizedProfile || options.output != "build/hello" {
		t.Fatalf("options = %+v", options)
	}
}

func TestProductSupportSourceLinksOnlyCalledStandardLibraryClosure(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		symbols   []string
		prefix    string
		required  []string
		forbidden []string
	}{
		{
			name:      "scalar runtime only",
			required:  []string{"fy_runtime_trap"},
			forbidden: []string{"fy_std_text_utf8_valid_v0", "fy_std_system_heap_allocate_v0"},
		},
		{
			name:      "text view",
			symbols:   []string{"fy_std_text_utf8_valid_v0"},
			required:  []string{"fy_std_text_utf8_valid_v0", "fy_runtime_trap"},
			forbidden: []string{"fy_std_system_heap_allocate_v0", "fy_std_string_copy_v0"},
		},
		{
			name:    "owned text dependencies",
			symbols: []string{"fy_std_cstring_from_text_v0"},
			required: []string{
				"fy_std_system_heap_allocate_v0", "fy_std_buffer_initialize_v0",
				"fy_std_text_utf8_valid_v0", "fy_std_cstring_from_text_v0", "fy_runtime_trap",
			},
			forbidden: []string{"fy_std_arena_initialize_v0", "fy_std_linux_file_open_v0"},
		},
		{
			name:     "Linux feature macro stays first",
			symbols:  []string{"fy_std_linux_file_close_v0", "fy_std_buffer_drop_v0"},
			prefix:   "#define _POSIX_C_SOURCE 200809L\n",
			required: []string{"fy_std_linux_file_close_v0", "fy_std_buffer_drop_v0", "fy_runtime_trap"},
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			source, err := productSupportSource(test.symbols)
			if err != nil {
				t.Fatal(err)
			}
			if test.prefix != "" && !strings.HasPrefix(source, test.prefix) {
				t.Errorf("support source does not start with %q", test.prefix)
			}
			for _, required := range test.required {
				if !strings.Contains(source, required) {
					t.Errorf("support source is missing %q", required)
				}
			}
			for _, forbidden := range test.forbidden {
				if strings.Contains(source, forbidden) {
					t.Errorf("support source unexpectedly contains %q", forbidden)
				}
			}
		})
	}
}

func TestProductSupportSourceRejectsUnknownStandardLibrarySymbol(t *testing.T) {
	t.Parallel()
	if _, err := productSupportSource([]string{"fy_std_missing_v0"}); err == nil || !strings.Contains(err.Error(), "unrecognized Stage 0 standard-library symbol") {
		t.Fatalf("productSupportSource() error = %v", err)
	}
}

func TestParseBuildOptionsRejectsIncompleteOrOpenEndedArguments(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		arguments []string
		contains  string
	}{
		{name: "empty", contains: "expects one entry source file"},
		{name: "flag first", arguments: []string{"--cc", "clang"}, contains: "source file before"},
		{name: "missing compiler", arguments: []string{"main.fy", "--profile", "low", "--output", "build/main"}, contains: "requires --cc"},
		{name: "missing profile", arguments: []string{"main.fy", "--cc", "clang", "--output", "build/main"}, contains: "requires --profile"},
		{name: "missing output", arguments: []string{"main.fy", "--cc", "clang", "--profile", "low"}, contains: "requires --output"},
		{name: "unknown compiler", arguments: []string{"main.fy", "--cc", "cc", "--profile", "low", "--output", "build/main"}, contains: "clang or --cc gcc"},
		{name: "unknown profile", arguments: []string{"main.fy", "--cc", "gcc", "--profile", "fast", "--output", "build/main"}, contains: "low or --profile optimized"},
		{name: "sanitizer profile stays internal", arguments: []string{"main.fy", "--cc", "gcc", "--profile", cdriver.AddressUndefinedSanitizerMode, "--output", "build/main"}, contains: "low or --profile optimized"},
		{name: "unknown flag", arguments: []string{"main.fy", "--cc", "gcc", "--profile", "low", "--output", "build/main", "--force"}, contains: "does not accept flag"},
		{name: "duplicate", arguments: []string{"main.fy", "--cc", "gcc", "--cc", "clang", "--profile", "low", "--output", "build/main"}, contains: "more than once"},
		{name: "missing value", arguments: []string{"main.fy", "--cc", "--profile", "low", "--output", "build/main"}, contains: "requires a value for --cc"},
		{name: "missing source value", arguments: []string{"main.fy", "--source", "--profile", "low"}, contains: "requires a value for --source"},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, err := parseBuildOptions(test.arguments)
			if err == nil || !strings.Contains(err.Error(), test.contains) {
				t.Fatalf("parseBuildOptions() error = %v; want containing %q", err, test.contains)
			}
		})
	}
}

func TestValidateBuildOutputRejectsUnsafeOrNonPortablePaths(t *testing.T) {
	t.Parallel()
	valid := []string{
		"build/main",
		"build/debug/main",
		"build/fairy-c.exe",
		"build/한글",
	}
	for _, value := range valid {
		if got, err := validateBuildOutput(value); err != nil || got != value {
			t.Errorf("validateBuildOutput(%q) = %q, %v", value, got, err)
		}
	}
	tooLongComponent := strings.Repeat("a", maxBuildComponentSize+1)
	tooLongPath := "build/" + strings.Repeat("a/", maxBuildPathBytes/2) + "x"
	invalid := []string{
		"",
		"main",
		"out/main",
		"build",
		"build/../main",
		"build/./main",
		"build//main",
		"/tmp/main",
		`C:\tmp\main.exe`,
		"C:tmp/main.exe",
		`\\server\share\main.exe`,
		"build/CON",
		"build/com1.exe",
		"build/name.",
		"build/name ",
		"build/a:b",
		"build/" + tooLongComponent,
		tooLongPath,
		"build/\x00name",
	}
	for _, value := range invalid {
		if got, err := validateBuildOutput(value); err == nil {
			t.Errorf("validateBuildOutput(%q) = %q, nil", value, got)
		}
	}
}

func TestWriteBuildArtifactPromotesWithoutOverwrite(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	contents := []byte("native-image")
	if err := writeBuildArtifact(root, "build/bin/main", contents); err != nil {
		t.Fatal(err)
	}
	outputPath := filepath.Join(root, "build", "bin", "main")
	got, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(contents) {
		t.Fatalf("artifact = %q", got)
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(outputPath)
		if err != nil || info.Mode().Perm() != 0o700 {
			t.Fatalf("artifact mode = %v, %v", info, err)
		}
	}
	if err := writeBuildArtifact(root, "build/bin/main", []byte("replacement")); err == nil || !strings.Contains(err.Error(), "refusing to overwrite") {
		t.Fatalf("second write error = %v", err)
	}
	got, err = os.ReadFile(outputPath)
	if err != nil || string(got) != string(contents) {
		t.Fatalf("artifact after rejected overwrite = %q, %v", got, err)
	}
	staged, err := filepath.Glob(filepath.Join(root, "build", "bin", ".fairyc-*.tmp"))
	if err != nil || len(staged) != 0 {
		t.Fatalf("staging files = %v, %v", staged, err)
	}
}

func TestWriteBuildArtifactRejectsSymlinkEscape(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "root")
	outside := filepath.Join(parent, "outside")
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "build")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	err := writeBuildArtifact(root, "build/main", []byte("native-image"))
	if err == nil {
		t.Fatal("writeBuildArtifact() followed a build-directory symlink outside the source root")
	}
	if _, statErr := os.Stat(filepath.Join(outside, "main")); !os.IsNotExist(statErr) {
		t.Fatalf("outside artifact stat error = %v", statErr)
	}
}

func TestExperimentalBuildFailsClosedOnIneligibleHost(t *testing.T) {
	if runtime.GOOS == "linux" && runtime.GOARCH == "amd64" {
		t.Skip("eligible-host execution belongs to the configured native C command")
	}
	root := t.TempDir()
	writeTestSource(t, root, "main.fy", "module main;\nfn main() -> i32 { 42 }\n")
	code, stdout, stderr := runForTest([]string{
		"experimental", "build", "main.fy",
		"--cc", "clang",
		"--profile", "optimized",
		"--output", "build/main",
	}, root)
	if code != ExitBuildFailure || stdout != "" || !strings.Contains(stderr, "C toolchain requires linux/amd64") {
		t.Fatalf("Run() = code %d, stdout %q, stderr %q", code, stdout, stderr)
	}
	if _, err := os.Stat(filepath.Join(root, "build", "main")); !os.IsNotExist(err) {
		t.Fatalf("build output stat error = %v", err)
	}
}

func TestExperimentalBuildRejectsUnsafeOutputBeforeLoadingSource(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	code, stdout, stderr := runForTest([]string{
		"experimental", "build", "missing.fy",
		"--cc", "clang",
		"--profile", "low",
		"--output", "build/../outside",
	}, root)
	if code != ExitInvalidInvocation || stdout != "" || !strings.Contains(stderr, "canonical relative path") || strings.Contains(stderr, "cannot load") {
		t.Fatalf("Run() = code %d, stdout %q, stderr %q", code, stdout, stderr)
	}
}
