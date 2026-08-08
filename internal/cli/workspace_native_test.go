package cli

import (
	"io"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/target"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

func TestParseWorkspaceTargetOptions(t *testing.T) {
	t.Parallel()
	build, err := parseWorkspaceTargetOptions("build-workspace", []string{
		"--profile", "optimized", "--locked", "--output", "build/app.exe", "--target", string(target.X8664WindowsMSVC),
	}, true)
	if err != nil {
		t.Fatal(err)
	}
	if build.targetID != target.X8664WindowsMSVC || build.profile != cdriver.OptimizedProfile || build.output != "build/app.exe" || !build.locked {
		t.Fatalf("build options = %+v", build)
	}
	run, err := parseWorkspaceTargetOptions("run-workspace", []string{
		"--target", string(target.X8664WindowsMSVC), "--profile", "low",
	}, false)
	if err != nil {
		t.Fatal(err)
	}
	if run.targetID != target.X8664WindowsMSVC || run.profile != cdriver.LowProfile || run.output != "" || run.locked {
		t.Fatalf("run options = %+v", run)
	}

	for _, test := range []struct {
		command       string
		arguments     []string
		requireOutput bool
		contains      string
	}{
		{command: "build-workspace", requireOutput: true, contains: "requires --target"},
		{command: "build-workspace", arguments: []string{"main.fy"}, requireOutput: true, contains: "does not accept"},
		{command: "build-workspace", arguments: []string{"--target", string(target.X8664WindowsMSVC), "--profile", "low"}, requireOutput: true, contains: "requires --output"},
		{command: "run-workspace", arguments: []string{"--target", string(target.X8664WindowsMSVC), "--profile", "low", "--output", "build/app.exe"}, contains: "does not accept"},
		{command: "run-workspace", arguments: []string{"--target", string(target.X8664LinuxGNU), "--profile", "low"}, contains: "currently requires --target"},
		{command: "run-workspace", arguments: []string{"--target", string(target.X8664WindowsMSVC), "--profile", "fast"}, contains: "profile low"},
		{command: "run-workspace", arguments: []string{"--target", string(target.X8664WindowsMSVC), "--profile"}, contains: "requires a value"},
		{command: "run-workspace", arguments: []string{"--target", string(target.X8664WindowsMSVC), "--profile", "low", "--locked", "--locked"}, contains: "more than once"},
	} {
		if _, err := parseWorkspaceTargetOptions(test.command, test.arguments, test.requireOutput); err == nil || !strings.Contains(err.Error(), test.contains) {
			t.Errorf("parseWorkspaceTargetOptions(%q, %q) error = %v; want containing %q", test.command, test.arguments, err, test.contains)
		}
	}
}

func TestParseStableNativeOptions(t *testing.T) {
	t.Parallel()
	for _, command := range []string{"build", "run"} {
		development, err := parseStableNativeOptions(command, nil)
		if err != nil || development.locked || development.release {
			t.Fatalf("%s development options = %+v, %v", command, development, err)
		}
		release, err := parseStableNativeOptions(command, []string{"--locked", "--release"})
		if err != nil || !release.locked || !release.release {
			t.Fatalf("%s release options = %+v, %v", command, release, err)
		}
		for _, arguments := range [][]string{
			{"main.fy"},
			{"--profile", "optimized"},
			{"--release", "--release"},
			{"--locked", "--locked"},
		} {
			if _, err := parseStableNativeOptions(command, arguments); err == nil {
				t.Errorf("parseStableNativeOptions(%q, %q) succeeded", command, arguments)
			}
		}
	}
}

func TestSplitStableRunArgumentsSeparatesProgramArguments(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name              string
		arguments         []string
		compilerArguments []string
		programArguments  []string
	}{
		{name: "no delimiter", arguments: []string{"--release", "--locked"}, compilerArguments: []string{"--release", "--locked"}},
		{name: "bare delimiter", arguments: []string{"--"}, programArguments: []string{}},
		{name: "option then delimiter", arguments: []string{"--release", "--", "x"}, compilerArguments: []string{"--release"}, programArguments: []string{"x"}},
		{name: "flag-looking program argument", arguments: []string{"--", "--locked"}, programArguments: []string{"--locked"}},
		{name: "empty program argument", arguments: []string{"--", ""}, programArguments: []string{""}},
		{name: "multiple program arguments", arguments: []string{"--", "a", "b", "c"}, programArguments: []string{"a", "b", "c"}},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			compilerArguments, programArguments, err := splitStableRunArguments(test.arguments)
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(compilerArguments, test.compilerArguments) {
				t.Errorf("compiler arguments = %v, want %v", compilerArguments, test.compilerArguments)
			}
			if !slices.Equal(programArguments, test.programArguments) {
				t.Errorf("program arguments = %v, want %v", programArguments, test.programArguments)
			}
		})
	}
}

func TestSplitStableRunArgumentsRejectsEmbeddedNUL(t *testing.T) {
	t.Parallel()
	for _, arguments := range [][]string{
		{"--", "a\x00b"},
		{"--locked", "--", "ok", "\x00"},
	} {
		_, _, err := splitStableRunArguments(arguments)
		if err == nil || !strings.Contains(err.Error(), "NUL byte") {
			t.Errorf("splitStableRunArguments(%q) error = %v; want NUL rejection", arguments, err)
		}
	}
}

func TestStableRunRejectsEmbeddedNULBeforeCompilation(t *testing.T) {
	t.Parallel()
	code, stdout, stderr := runForTest([]string{"run", "--", "a\x00b"}, t.TempDir())
	if code != ExitInvalidInvocation || stdout != "" || !strings.Contains(stderr, "NUL byte") {
		t.Fatalf("Run() = code %d, stdout %q, stderr %q", code, stdout, stderr)
	}
}

func TestStableRunRejectsUnknownFlagBeforeDelimiter(t *testing.T) {
	t.Parallel()
	code, stdout, stderr := runForTest([]string{"run", "--bogus", "--", "x"}, t.TempDir())
	if code != ExitInvalidInvocation || stdout != "" || !strings.Contains(stderr, "accepts only optional --release and --locked") {
		t.Fatalf("Run() = code %d, stdout %q, stderr %q", code, stdout, stderr)
	}
}

func TestStableBuildRejectsProgramDelimiter(t *testing.T) {
	t.Parallel()
	code, stdout, stderr := runForTest([]string{"build", "--", "x"}, t.TempDir())
	if code != ExitInvalidInvocation || stdout != "" || !strings.Contains(stderr, "accepts only optional --release and --locked") {
		t.Fatalf("Run() = code %d, stdout %q, stderr %q", code, stdout, stderr)
	}
}

func TestClassifyStableRunOutcomeSeparatesProgramStreamsAndReservedResultFromTrap(t *testing.T) {
	t.Parallel()
	normalResult := int32(productTrapExitBase + 3)
	programStderr, category, err := classifyStableRunOutcome(cdriver.WindowsHostOutcome{
		ExitCode: normalResult,
		Stdout:   "program stdout\n",
		Stderr:   "program stderr\n",
	})
	if err != nil || programStderr != "program stderr\n" || category != "" {
		t.Fatalf("normal reserved result = (%q, %q, %v)", programStderr, category, err)
	}
	programStderr, category, err = classifyStableRunOutcome(cdriver.WindowsHostOutcome{
		ExitCode:    normalResult,
		Stdout:      "program stdout\n",
		Stderr:      "before trap\nfairyc-run-trap-v1:3",
		TrapControl: string([]byte{3}),
	})
	if err != nil || programStderr != "before trap\nfairyc-run-trap-v1:3" || category != "division-by-zero" {
		t.Fatalf("receipt trap = (%q, %q, %v)", programStderr, category, err)
	}
	for _, test := range []struct {
		outcome cdriver.WindowsHostOutcome
		stderr  string
	}{
		{outcome: cdriver.WindowsHostOutcome{ExitCode: normalResult, Stderr: "fairyc-run-trap-v1:3"}, stderr: "fairyc-run-trap-v1:3"},
		{outcome: cdriver.WindowsHostOutcome{ExitCode: normalResult, Stderr: "ordinary\n"}, stderr: "ordinary\n"},
	} {
		programStderr, category, err := classifyStableRunOutcome(test.outcome)
		if err != nil || programStderr != test.stderr || category != "" {
			t.Errorf("classifyStableRunOutcome(%+v) = (%q, %q, %v)", test.outcome, programStderr, category, err)
		}
	}
	for _, outcome := range []cdriver.WindowsHostOutcome{
		{ExitCode: 42, TrapControl: string([]byte{3})},
		{ExitCode: normalResult, TrapControl: string([]byte{2})},
		{ExitCode: normalResult, TrapControl: string([]byte{3, 3})},
	} {
		if _, _, err := classifyStableRunOutcome(outcome); err == nil {
			t.Errorf("classifyStableRunOutcome(%+v) accepted invalid control data", outcome)
		}
	}
}

func TestStableNativeCommandsPreserveWorkspaceFailureCategories(t *testing.T) {
	t.Parallel()
	for _, command := range []string{"build", "run"} {
		code, stdout, stderr := runForTest([]string{command}, t.TempDir())
		if code != ExitPackageFailure || stdout != "" || !strings.Contains(stderr, "fairy.toml") {
			t.Fatalf("%s missing stable workspace = (%d, %q, %q)", command, code, stdout, stderr)
		}
	}

	root := t.TempDir()
	writeWorkspaceFile(t, root, "fairy.toml", `schema = "fairyc.manifest/v1"
[package]
uid = "example.stable-build.invalid"
name = "invalid"
version = "1.0.0"
language = "fairyc-language/10"
source = "src"
`)
	writeWorkspaceFile(t, root, "src/main.fy", "module main;\nfn i32 broken( { return 0; }\n")
	for _, command := range []string{"build", "run"} {
		code, stdout, stderr := runForTest([]string{command}, root)
		if code != ExitSourceDiagnostics || stdout != "" || !strings.Contains(stderr, "E0007") {
			t.Fatalf("stable %s source diagnostics = (%d, %q, %q)", command, code, stdout, stderr)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "build")); !os.IsNotExist(err) {
		t.Fatalf("build path after source failure = %v", err)
	}

	writeWorkspaceFile(t, root, "src/main.fy", "module main;\n@mystery\nfn i32 main() { return 42; }\n")
	for _, command := range []string{"build", "run"} {
		code, stdout, stderr := runForTest([]string{command}, root)
		if code != ExitSourceDiagnostics || stdout != "" ||
			!strings.Contains(stderr, "semantic subset does not support function attribute @mystery") ||
			strings.Contains(stderr, "experimental semantic subset") {
			t.Fatalf("stable %s unsupported semantics = (%d, %q, %q)", command, code, stdout, stderr)
		}
	}
}

func TestPackageWorkspaceReachesQualifiedWindowsCEmission(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeNativeWorkspace(t, root, true)
	workspace, err := workspaceloader.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	input, err := workspace.SemanticInput()
	if err != nil {
		t.Fatal(err)
	}
	checked, err := compiler.CheckPackages(input)
	if err != nil {
		t.Fatal(err)
	}
	if !checked.Verified {
		t.Fatalf("workspace did not reach verified MIR: %+v", checked.Semantic)
	}
	entry, err := workspaceEntryName(workspace)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(entry, "$p") || !strings.HasSuffix(entry, "::main::main") {
		t.Fatalf("workspace entry = %q", entry)
	}
	unit, err := cbackend.Emit(checked.MIR, entry, cbackend.Stage0WindowsMSVCProfile())
	if err != nil {
		t.Fatal(err)
	}
	if unit.Entry != entry || len(unit.Functions) != 2 || !strings.Contains(unit.Source, "int main(void)") {
		t.Fatalf("C unit = entry %q, functions %v", unit.Entry, unit.Functions)
	}
}

func TestStdTextQuickstartReachesQualifiedWindowsCEmission(t *testing.T) {
	t.Parallel()
	repositoryRoot := filepath.Join("..", "..")
	manifest, err := os.ReadFile(filepath.Join(repositoryRoot, "examples", "std_text_quickstart", "fairy.toml"))
	if err != nil {
		t.Fatal(err)
	}
	entrySource, err := os.ReadFile(filepath.Join(repositoryRoot, "examples", "std_text_quickstart", "src", "main.fy"))
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	writeWorkspaceFile(t, root, "fairy.toml", string(manifest))
	writeWorkspaceFile(t, root, "src/main.fy", string(entrySource))
	workspace, err := workspaceloader.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	input, err := workspace.SemanticInput()
	if err != nil {
		t.Fatal(err)
	}
	checked, err := compiler.CheckPackages(input)
	if err != nil {
		t.Fatal(err)
	}
	if !checked.Verified {
		t.Fatalf("std::text quickstart did not reach verified MIR: %+v", checked.Semantic)
	}
	entry, err := workspaceEntryName(workspace)
	if err != nil {
		t.Fatal(err)
	}
	unit, err := cbackend.Emit(checked.MIR, entry, cbackend.Stage0WindowsMSVCProfile())
	if err != nil {
		t.Fatal(err)
	}
	if unit.Entry != entry || !strings.Contains(unit.Source, "fy_std_string_copy_storage_v0") ||
		!strings.Contains(unit.Source, "fy_std_cstring_from_text_storage_v0") ||
		!strings.Contains(unit.Source, "fy_std_text_utf8_valid_v0") ||
		!strings.Contains(unit.Source, "int main(void)") {
		t.Fatalf("std::text C unit = entry %q, foreign symbols %v", unit.Entry, unit.ForeignSymbols)
	}
}

func TestStdAllocQuickstartReachesQualifiedWindowsCEmission(t *testing.T) {
	t.Parallel()
	repositoryRoot := filepath.Join("..", "..")
	manifest, err := os.ReadFile(filepath.Join(repositoryRoot, "examples", "std_alloc_quickstart", "fairy.toml"))
	if err != nil {
		t.Fatal(err)
	}
	entrySource, err := os.ReadFile(filepath.Join(repositoryRoot, "examples", "std_alloc_quickstart", "src", "main.fy"))
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	writeWorkspaceFile(t, root, "fairy.toml", string(manifest))
	writeWorkspaceFile(t, root, "src/main.fy", string(entrySource))
	workspace, err := workspaceloader.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	input, err := workspace.SemanticInput()
	if err != nil {
		t.Fatal(err)
	}
	checked, err := compiler.CheckPackages(input)
	if err != nil {
		t.Fatal(err)
	}
	if !checked.Verified {
		t.Fatalf("std::alloc quickstart did not reach verified MIR: %+v", checked.Semantic)
	}
	entry, err := workspaceEntryName(workspace)
	if err != nil {
		t.Fatal(err)
	}
	unit, err := cbackend.Emit(checked.MIR, entry, cbackend.Stage0WindowsMSVCProfile())
	if err != nil {
		t.Fatal(err)
	}
	if unit.Entry != entry || !strings.Contains(unit.Source, "fy_std_arena_initialize_v0") ||
		!strings.Contains(unit.Source, "fy_std_arena_drop_v0") || !strings.Contains(unit.Source, "int main(void)") {
		t.Fatalf("std::alloc C unit = entry %q, foreign symbols %v", unit.Entry, unit.ForeignSymbols)
	}
}

func TestRegionQuickstartReachesQualifiedWindowsCEmission(t *testing.T) {
	t.Parallel()
	repositoryRoot := filepath.Join("..", "..")
	manifest, err := os.ReadFile(filepath.Join(repositoryRoot, "examples", "regions", "fairy.toml"))
	if err != nil {
		t.Fatal(err)
	}
	entrySource, err := os.ReadFile(filepath.Join(repositoryRoot, "examples", "regions", "src", "main.fy"))
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	writeWorkspaceFile(t, root, "fairy.toml", string(manifest))
	writeWorkspaceFile(t, root, "src/main.fy", string(entrySource))
	workspace, err := workspaceloader.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	input, err := workspace.SemanticInput()
	if err != nil {
		t.Fatal(err)
	}
	checked, err := compiler.CheckPackages(input)
	if err != nil {
		t.Fatal(err)
	}
	if !checked.Verified {
		t.Fatalf("Region quickstart did not reach verified MIR: %+v", checked.Semantic)
	}
	entry, err := workspaceEntryName(workspace)
	if err != nil {
		t.Fatal(err)
	}
	unit, err := cbackend.Emit(checked.MIR, entry, cbackend.Stage0WindowsMSVCProfile())
	if err != nil {
		t.Fatal(err)
	}
	wantSymbols := []string{
		"fy_std_arena_drop_v0",
		"fy_std_arena_initialize_v0",
		"fy_std_region_allocate_zeroed_v0",
		"fy_std_region_begin_v0",
		"fy_std_region_reset_v0",
	}
	for _, symbol := range wantSymbols {
		if !slices.Contains(unit.ForeignSymbols, symbol) {
			t.Fatalf("Region C unit is missing foreign symbol %q: %v", symbol, unit.ForeignSymbols)
		}
	}
	if unit.Entry != entry ||
		!strings.Contains(unit.Source, "fy_std_region_begin_v0") ||
		!strings.Contains(unit.Source, "fy_std_region_allocate_zeroed_v0") ||
		!strings.Contains(unit.Source, "fy_std_region_reset_v0") ||
		!strings.Contains(unit.Source, ".tag = FY_RESULT_OK") ||
		!strings.Contains(unit.Source, ".tag = FY_RESULT_ERR") ||
		!strings.Contains(unit.Source, "sizeof(int32_t), _Alignof(int32_t)") ||
		!strings.Contains(unit.Source, "sizeof(fy_array_") ||
		!strings.Contains(unit.Source, "sizeof(fy_struct_") {
		t.Fatalf("Region C unit = entry %q, foreign symbols %v", unit.Entry, unit.ForeignSymbols)
	}
}

func TestExperimentalWorkspaceBuildRejectsMissingRootEntryBeforeToolchain(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeNativeWorkspace(t, root, false)
	code, stdout, stderr := runForTest([]string{
		"experimental", "build-workspace", "--target", string(target.X8664WindowsMSVC),
		"--profile", "low", "--output", "build/app.exe",
	}, root)
	if code != ExitInvalidInvocation || stdout != "" || !strings.Contains(stderr, "root entry") {
		t.Fatalf("Run() = code %d, stdout %q, stderr %q", code, stdout, stderr)
	}
}

func TestStableWorkspaceBuildRejectsMissingRootEntryBeforeToolchain(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeNativeWorkspace(t, root, false)
	code, stdout, stderr := runForTest([]string{"build"}, root)
	if code != ExitInvalidInvocation || stdout != "" || !strings.Contains(stderr, "root entry") {
		t.Fatalf("Run() = code %d, stdout %q, stderr %q", code, stdout, stderr)
	}
	if _, err := os.Stat(filepath.Join(root, "build", "app.exe")); !os.IsNotExist(err) {
		t.Fatalf("stable build artifact after missing entry = %v", err)
	}
}

func TestStableWorkspaceRunRejectsMissingRootEntryBeforeToolchain(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeNativeWorkspace(t, root, false)
	stdin := &countingReader{}
	var stdout strings.Builder
	var stderr strings.Builder
	code := RunWithInput([]string{"run"}, root, stdin, &stdout, &stderr)
	if code != ExitInvalidInvocation || stdout.String() != "" || !strings.Contains(stderr.String(), "root entry") {
		t.Fatalf("RunWithInput() = code %d, stdout %q, stderr %q", code, stdout.String(), stderr.String())
	}
	if stdin.reads != 0 {
		t.Fatalf("stable run consumed stdin %d time(s) before successful compilation", stdin.reads)
	}
	if matches, err := filepath.Glob(filepath.Join(root, "build", "*.exe")); err != nil || len(matches) != 0 {
		t.Fatalf("stable run artifacts after missing entry = %v, %v", matches, err)
	}
}

type countingReader struct {
	reads int
}

func (reader *countingReader) Read([]byte) (int, error) {
	reader.reads++
	return 0, io.EOF
}

func TestExperimentalWorkspaceBuildFailsClosedOutsideWindowsAMD64(t *testing.T) {
	if runtime.GOOS == "windows" && runtime.GOARCH == "amd64" {
		t.Skip("live package-aware MSVC build belongs to the configured Windows host smoke")
	}
	root := t.TempDir()
	writeNativeWorkspace(t, root, true)
	code, stdout, stderr := runForTest([]string{
		"experimental", "build-workspace", "--target", string(target.X8664WindowsMSVC),
		"--profile", "low", "--output", "build/app.exe",
	}, root)
	if code != ExitBuildFailure || stdout != "" || !strings.Contains(stderr, "requires windows/amd64") {
		t.Fatalf("Run() = code %d, stdout %q, stderr %q", code, stdout, stderr)
	}
	if matches, err := filepath.Glob(filepath.Join(root, "build", "*.exe")); err != nil || len(matches) != 0 {
		t.Fatalf("build artifacts = %v, %v", matches, err)
	}
}

func writeNativeWorkspace(t *testing.T, root string, withEntry bool) {
	t.Helper()
	writeWorkspaceFile(t, root, "fairy.toml", `schema = "fairyc.manifest/v1"
[package]
uid = "example.native.app"
name = "app"
version = "1.0.0"
language = "fairyc-language/10"
source = "src"
[workspace]
members = ["libs/answer"]
[dependencies.answer]
path = "libs/answer"
version = "1.0.0"
features = []
`)
	entry := "fn i32 not_main() { return answer::value(); }"
	if withEntry {
		entry = "fn i32 main() { return answer::value(); }"
	}
	writeWorkspaceFile(t, root, "src/main.fy", "module main;\n\nimport answer as answer;\n\n"+entry+"\n")
	writeWorkspaceFile(t, root, "libs/answer/fairy.toml", `schema = "fairyc.manifest/v1"
[package]
uid = "example.native.answer"
name = "answer"
version = "1.0.0"
language = "fairyc-language/10"
source = "src"
`)
	writeWorkspaceFile(t, root, "libs/answer/src/answer.fy", "module answer;\n\npub fn i32 value() { return 42; }\n")
}
