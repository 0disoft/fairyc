package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/0disoft/fairyc/internal/buildmeasure"
	"github.com/0disoft/fairyc/internal/cli"
	"github.com/0disoft/fairyc/internal/packagecontent"
	"github.com/0disoft/fairyc/internal/pecoff"
)

func runStandaloneCompiler(
	repositoryRoot, sourceRoot string,
	clockResolution int64,
	atlasCallProcessMeasurement *buildmeasure.AtlasCallProcessReport,
) (int64, error) {
	temporaryRoot, err := os.MkdirTemp("", "fairyc-standalone-compiler-smoke-")
	if err != nil {
		return 0, fmt.Errorf("create standalone compiler directory: %w", err)
	}
	defer os.RemoveAll(temporaryRoot)

	executablePath := filepath.Join(temporaryRoot, "fairyc.exe")
	buildContext, cancelBuild := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancelBuild()
	build := exec.CommandContext(buildContext, "go", "build", "-mod=readonly", "-trimpath", "-o", executablePath, "./cmd/fairyc")
	build.Dir = repositoryRoot
	build.Stdin = nil
	var buildStdout boundedBuffer
	var buildStderr boundedBuffer
	buildStdout.limit = subprocessOutputLimit
	buildStderr.limit = subprocessOutputLimit
	build.Stdout = &buildStdout
	build.Stderr = &buildStderr
	if err := build.Run(); err != nil {
		if errors.Is(buildContext.Err(), context.DeadlineExceeded) {
			return 0, fmt.Errorf("standalone compiler build exceeded timeout")
		}
		return 0, fmt.Errorf("build standalone compiler: %w; stdout %q; stderr %q", err, buildStdout.String(), buildStderr.String())
	}
	if buildStdout.Len() != 0 || buildStderr.Len() != 0 {
		return 0, fmt.Errorf("standalone compiler build produced unexpected output: stdout %q; stderr %q", buildStdout.String(), buildStderr.String())
	}
	info, err := os.Stat(executablePath)
	if err != nil {
		return 0, fmt.Errorf("stat standalone compiler: %w", err)
	}
	if !info.Mode().IsRegular() || info.Size() == 0 {
		return 0, fmt.Errorf("standalone compiler is not a nonempty regular file")
	}

	helpCode, helpStdout, helpStderr, err := runStandaloneCommand(sourceRoot, executablePath, "--help")
	if err != nil {
		return 0, fmt.Errorf("execute standalone compiler help: %w", err)
	}
	if helpCode != cli.ExitSuccess || helpStderr != "" ||
		!strings.Contains(helpStdout, "fairyc new <package-name> --uid <package-uid>") ||
		!strings.Contains(helpStdout, "fairyc check [--locked]") ||
		!strings.Contains(helpStdout, "fairyc build [--release] [--locked]") ||
		!strings.Contains(helpStdout, "fairyc run [--release] [--locked]") ||
		!strings.Contains(helpStdout, "fairyc test [--json] [--locations] [--native [--release] [--stdin | --stdin-file <portable-workspace-file>] [--timeout-seconds <1..60>] [--capture-output] [-- <program-args>...]] [--workspace | --package <package-uid>] [--dir <portable-test-directory>]... [--file <portable-test-path>]... [--exact <test-identity>]... [--locked]") ||
		!strings.Contains(helpStdout, "fairyc test --list [--json] [--locations] [--workspace | --package <package-uid>] [--dir <portable-test-directory>]... [--file <portable-test-path>]... [--locked]") ||
		!strings.Contains(helpStdout, "every\nunambiguous root-reachable local package with --workspace") ||
		!strings.Contains(helpStdout, "native tests receive any process arguments") ||
		!strings.Contains(helpStdout, "Explicit --stdin attaches caller input only when exactly one") ||
		!strings.Contains(helpStdout, "Explicit --stdin-file replays one bounded") ||
		!strings.Contains(helpStdout, "Explicit --timeout-seconds bounds each native test process") ||
		!strings.Contains(helpStdout, "Explicit --capture-output requires") ||
		!strings.Contains(helpStdout, "fairyc experimental emit-workspace-c --target <target-id> [--locked]") ||
		strings.Contains(helpStdout, "\n  fairyc emit c --target") ||
		!strings.Contains(helpStdout, "fairyc experimental run-target") ||
		!strings.Contains(helpStdout, "fairyc experimental exec-target") ||
		!strings.Contains(helpStdout, "fairyc experimental check-workspace-session") ||
		!strings.Contains(helpStdout, "fairyc experimental check-workspace-cache") ||
		!strings.Contains(helpStdout, "fairyc experimental atlas-calls list") ||
		!strings.Contains(helpStdout, "fairyc experimental build-workspace") ||
		!strings.Contains(helpStdout, "fairyc experimental run-workspace") {
		return 0, fmt.Errorf("standalone compiler help returned code %d, stdout %q, stderr %q", helpCode, helpStdout, helpStderr)
	}

	const starterName = "starter-smoke"
	newCode, newStdout, newStderr, err := runStandaloneCommand(
		temporaryRoot,
		executablePath,
		"new", starterName, "--uid", "smoke.starter",
	)
	if err != nil {
		return 0, fmt.Errorf("execute standalone compiler starter creation: %w", err)
	}
	if newCode != cli.ExitSuccess || newStderr != "" ||
		newStdout != "created Fairy C package \"starter-smoke\" at starter-smoke\n" {
		return 0, fmt.Errorf("standalone compiler new returned code %d, stdout %q, stderr %q", newCode, newStdout, newStderr)
	}
	duplicateCode, duplicateStdout, duplicateStderr, err := runStandaloneCommand(
		temporaryRoot,
		executablePath,
		"new", starterName, "--uid", "smoke.starter",
	)
	if err != nil {
		return 0, fmt.Errorf("execute standalone compiler duplicate starter rejection: %w", err)
	}
	if duplicateCode != cli.ExitPackageFailure || duplicateStdout != "" ||
		duplicateStderr != "fairyc: project creation failed: project path \"starter-smoke\" already exists\n" {
		return 0, fmt.Errorf("standalone compiler duplicate new returned code %d, stdout %q, stderr %q", duplicateCode, duplicateStdout, duplicateStderr)
	}
	starterRoot := filepath.Join(temporaryRoot, starterName)
	starterReadme, err := os.ReadFile(filepath.Join(starterRoot, "README.md"))
	if err != nil {
		return 0, fmt.Errorf("read standalone compiler generated starter guide: %w", err)
	}
	const starterExecutableCommand = `.\build\starter-smoke.exe`
	if bytes.Count(starterReadme, []byte(starterExecutableCommand)) != 1 ||
		bytes.Count(starterReadme, []byte("fairyc test --native --locked")) != 1 ||
		!bytes.Contains(starterReadme, []byte("The default test command uses the fast semantic interpreter.")) ||
		!bytes.Contains(starterReadme, []byte("```fairyc\nmodule main;\n\nimport std::io as io;\n")) ||
		!bytes.Contains(starterReadme, []byte("`fairyc run` forwards program output and returns `main`'s `i32` as the process status.")) {
		return 0, fmt.Errorf("standalone compiler generated an invalid starter guide: %q", starterReadme)
	}
	starterCheckCode, starterCheckStdout, starterCheckStderr, err := runStandaloneCommand(
		starterRoot,
		executablePath,
		"check",
	)
	if err != nil {
		return 0, fmt.Errorf("execute standalone compiler generated starter check: %w", err)
	}
	if starterCheckCode != cli.ExitSuccess || starterCheckStderr != "" ||
		starterCheckStdout != "check ok: 1 instance(s), 1 source file(s), 2 MIR function(s), lock updated\n" {
		return 0, fmt.Errorf("standalone compiler starter check returned code %d, stdout %q, stderr %q", starterCheckCode, starterCheckStdout, starterCheckStderr)
	}
	starterTestCode, starterTestStdout, starterTestStderr, err := runStandaloneCommand(
		starterRoot,
		executablePath,
		"test", "--locked",
	)
	if err != nil {
		return 0, fmt.Errorf("execute standalone compiler generated starter test: %w", err)
	}
	if starterTestCode != cli.ExitSuccess || starterTestStderr != "" ||
		starterTestStdout != "test tests::main::answer_is_42 ... ok\ntest result: ok. 1 passed; 0 failed; lock verified\n" {
		return 0, fmt.Errorf("standalone compiler starter test returned code %d, stdout %q, stderr %q", starterTestCode, starterTestStdout, starterTestStderr)
	}
	starterNativeTestSource := `module tests::main;

@test
fn void first_fails() { assert(false); }

@test
fn void second_passes() { assert(true); }
`
	if err := os.WriteFile(filepath.Join(starterRoot, "tests", "main.fy"), []byte(starterNativeTestSource), 0o600); err != nil {
		return 0, fmt.Errorf("write standalone compiler native trap-isolation test: %w", err)
	}
	starterListCode, starterListStdout, starterListStderr, err := runStandaloneCommand(
		starterRoot,
		executablePath,
		"test", "--list", "--locked",
	)
	if err != nil {
		return 0, fmt.Errorf("execute standalone compiler generated starter test list: %w", err)
	}
	wantStarterList := "test tests::main::first_fails\n" +
		"test tests::main::second_passes\n" +
		"test list: 2 discovered; lock verified\n"
	if starterListCode != cli.ExitSuccess || starterListStderr != "" || starterListStdout != wantStarterList {
		return 0, fmt.Errorf("standalone compiler starter test list returned code %d, stdout %q, stderr %q", starterListCode, starterListStdout, starterListStderr)
	}
	starterJSONListCode, starterJSONListStdout, starterJSONListStderr, err := runStandaloneCommand(
		starterRoot,
		executablePath,
		"test", "--list", "--json", "--locked",
	)
	if err != nil {
		return 0, fmt.Errorf("execute standalone compiler generated starter JSON test list: %w", err)
	}
	wantStarterJSONList := `{"schemaVersion":"fairyc.test-list/v1","tests":["tests::main::first_fails","tests::main::second_passes"],"count":2,"lock":"verified"}` + "\n"
	if starterJSONListCode != cli.ExitSuccess || starterJSONListStderr != "" || starterJSONListStdout != wantStarterJSONList {
		return 0, fmt.Errorf("standalone compiler starter JSON test list returned code %d, stdout %q, stderr %q", starterJSONListCode, starterJSONListStdout, starterJSONListStderr)
	}
	starterJSONTestCode, starterJSONTestStdout, starterJSONTestStderr, err := runStandaloneCommand(
		starterRoot,
		executablePath,
		"test", "--json",
		"--exact", "tests::main::second_passes",
		"--exact", "tests::main::first_fails",
		"--locked",
	)
	if err != nil {
		return 0, fmt.Errorf("execute standalone compiler generated starter JSON semantic test: %w", err)
	}
	wantStarterJSONTest := `{"schemaVersion":"fairyc.test-report/v1","engine":"interpreter","target":null,"profile":null,"tests":[{"identity":"tests::main::first_fails","outcome":"failed","trapCategory":"assertion-failed"},{"identity":"tests::main::second_passes","outcome":"passed","trapCategory":null}],"count":2,"passed":1,"failed":1,"status":"failed","lock":"verified"}` + "\n"
	if starterJSONTestCode != cli.ExitProgramTrap || starterJSONTestStderr != "" || starterJSONTestStdout != wantStarterJSONTest {
		return 0, fmt.Errorf("standalone compiler starter JSON semantic test returned code %d, stdout %q, stderr %q", starterJSONTestCode, starterJSONTestStdout, starterJSONTestStderr)
	}
	starterNativeRuntimeTestSource := `module tests::main;

import std::process as process;

@test
fn void first_fails() { assert(process::arg_count() == 0); }

@test
fn void second_passes() { assert(process::arg_count() >= 1); }
`
	if err := os.WriteFile(filepath.Join(starterRoot, "tests", "main.fy"), []byte(starterNativeRuntimeTestSource), 0o600); err != nil {
		return 0, fmt.Errorf("write standalone compiler runtime-dependent native trap-isolation test: %w", err)
	}
	const starterExcludedNativeTestSource = `module tests::excluded;

@test
fn void excluded_fails() { assert(false); }
`
	if err := os.WriteFile(filepath.Join(starterRoot, "tests", "excluded.fy"), []byte(starterExcludedNativeTestSource), 0o600); err != nil {
		return 0, fmt.Errorf("write standalone compiler excluded native test file: %w", err)
	}
	const starterDirectoryNativeTestSource = `module tests::focused::directory;

import std::process as process;

@test
fn void directory_passes() { assert(process::arg_count() >= 1); }
`
	focusedDirectory := filepath.Join(starterRoot, "tests", "focused")
	if err := os.MkdirAll(focusedDirectory, 0o700); err != nil {
		return 0, fmt.Errorf("create standalone compiler focused native test directory: %w", err)
	}
	if err := os.WriteFile(filepath.Join(focusedDirectory, "directory.fy"), []byte(starterDirectoryNativeTestSource), 0o600); err != nil {
		return 0, fmt.Errorf("write standalone compiler focused native directory test: %w", err)
	}
	const starterExcludedDirectoryNativeTestSource = `module tests::other::directory;

@test
fn void directory_excluded_fails() { assert(false); }
`
	otherDirectory := filepath.Join(starterRoot, "tests", "other")
	if err := os.MkdirAll(otherDirectory, 0o700); err != nil {
		return 0, fmt.Errorf("create standalone compiler excluded native test directory: %w", err)
	}
	if err := os.WriteFile(filepath.Join(otherDirectory, "directory.fy"), []byte(starterExcludedDirectoryNativeTestSource), 0o600); err != nil {
		return 0, fmt.Errorf("write standalone compiler excluded native directory test: %w", err)
	}
	starterNativeCode, starterNativeStdout, starterNativeStderr, err := runStandaloneCommand(
		starterRoot,
		executablePath,
		"test", "--native", "--locations",
		"--file", "tests/main.fy",
		"--locked",
	)
	if err != nil {
		return 0, fmt.Errorf("execute standalone compiler native trap-isolation test: %w", err)
	}
	wantStarterNative := "test tests::main::first_fails at tests/main.fy:6:9 ... FAILED (assertion-failed)\n" +
		"test tests::main::second_passes at tests/main.fy:9:9 ... ok\n" +
		"test result: FAILED. 1 passed; 1 failed; lock verified\n"
	if starterNativeCode != cli.ExitProgramTrap || starterNativeStderr != "" || starterNativeStdout != wantStarterNative {
		return 0, fmt.Errorf("standalone compiler native trap-isolation test returned code %d, stdout %q, stderr %q", starterNativeCode, starterNativeStdout, starterNativeStderr)
	}
	starterNativeJSONCode, starterNativeJSONStdout, starterNativeJSONStderr, err := runStandaloneCommand(
		starterRoot,
		executablePath,
		"test", "--json", "--locations", "--native", "--release",
		"--file", "tests/main.fy",
		"--locked",
	)
	if err != nil {
		return 0, fmt.Errorf("execute standalone compiler native JSON test: %w", err)
	}
	starterNativeFileIdentity, starterNativeLineMapIdentity := packagecontent.ComputeFileIdentities([]byte(starterNativeRuntimeTestSource))
	firstStart := strings.Index(starterNativeRuntimeTestSource, "first_fails")
	secondStart := strings.Index(starterNativeRuntimeTestSource, "second_passes")
	wantStarterNativeJSON := fmt.Sprintf(
		`{"schemaVersion":"fairyc.test-report/v3","engine":"native","target":"x86_64-windows-msvc","profile":"release","tests":[{"identity":"tests::main::first_fails","outcome":"failed","trapCategory":"assertion-failed","source":{"packageUid":"smoke.starter","path":"tests/main.fy","line":6,"byteColumn":9,"startByte":%d,"endByte":%d,"fileIdentity":"%s","lineMapIdentity":"%s"}},{"identity":"tests::main::second_passes","outcome":"passed","trapCategory":null,"source":{"packageUid":"smoke.starter","path":"tests/main.fy","line":9,"byteColumn":9,"startByte":%d,"endByte":%d,"fileIdentity":"%s","lineMapIdentity":"%s"}}],"count":2,"passed":1,"failed":1,"status":"failed","lock":"verified"}`+"\n",
		firstStart, firstStart+len("first_fails"), starterNativeFileIdentity, starterNativeLineMapIdentity,
		secondStart, secondStart+len("second_passes"), starterNativeFileIdentity, starterNativeLineMapIdentity,
	)
	if starterNativeJSONCode != cli.ExitProgramTrap || starterNativeJSONStderr != "" || starterNativeJSONStdout != wantStarterNativeJSON {
		return 0, fmt.Errorf("standalone compiler native JSON test returned code %d, stdout %q, stderr %q", starterNativeJSONCode, starterNativeJSONStdout, starterNativeJSONStderr)
	}
	starterDirectoryCode, starterDirectoryStdout, starterDirectoryStderr, err := runStandaloneCommand(
		starterRoot,
		executablePath,
		"test", "--native", "--locations",
		"--dir", "tests/focused",
		"--locked",
	)
	if err != nil {
		return 0, fmt.Errorf("execute standalone compiler native directory-selected test: %w", err)
	}
	wantStarterDirectory := "test tests::focused::directory::directory_passes at tests/focused/directory.fy:6:9 ... ok\n" +
		"test result: ok. 1 passed; 0 failed; lock verified\n"
	if starterDirectoryCode != cli.ExitSuccess || starterDirectoryStderr != "" || starterDirectoryStdout != wantStarterDirectory {
		return 0, fmt.Errorf("standalone compiler native directory-selected test returned code %d, stdout %q, stderr %q", starterDirectoryCode, starterDirectoryStdout, starterDirectoryStderr)
	}
	starterDirectoryJSONCode, starterDirectoryJSONStdout, starterDirectoryJSONStderr, err := runStandaloneCommand(
		starterRoot,
		executablePath,
		"test", "--json", "--locations", "--native", "--release",
		"--dir", "tests/focused",
		"--locked",
	)
	if err != nil {
		return 0, fmt.Errorf("execute standalone compiler optimized native directory-selected test: %w", err)
	}
	directoryFileIdentity, directoryLineMapIdentity := packagecontent.ComputeFileIdentities([]byte(starterDirectoryNativeTestSource))
	directoryStart := strings.Index(starterDirectoryNativeTestSource, "directory_passes")
	wantStarterDirectoryJSON := fmt.Sprintf(
		`{"schemaVersion":"fairyc.test-report/v3","engine":"native","target":"x86_64-windows-msvc","profile":"release","tests":[{"identity":"tests::focused::directory::directory_passes","outcome":"passed","trapCategory":null,"source":{"packageUid":"smoke.starter","path":"tests/focused/directory.fy","line":6,"byteColumn":9,"startByte":%d,"endByte":%d,"fileIdentity":"%s","lineMapIdentity":"%s"}}],"count":1,"passed":1,"failed":0,"status":"ok","lock":"verified"}`+"\n",
		directoryStart, directoryStart+len("directory_passes"), directoryFileIdentity, directoryLineMapIdentity,
	)
	if starterDirectoryJSONCode != cli.ExitSuccess || starterDirectoryJSONStderr != "" || starterDirectoryJSONStdout != wantStarterDirectoryJSON {
		return 0, fmt.Errorf("standalone compiler optimized native directory-selected test returned code %d, stdout %q, stderr %q", starterDirectoryJSONCode, starterDirectoryJSONStdout, starterDirectoryJSONStderr)
	}
	starterRunCode, starterRunStdout, starterRunStderr, err := runStandaloneCommand(
		starterRoot,
		executablePath,
		"run", "--locked",
	)
	if err != nil {
		return 0, fmt.Errorf("execute standalone compiler generated starter run: %w", err)
	}
	if starterRunCode != cli.ExitSuccess || starterRunStderr != "" || starterRunStdout != "" {
		return 0, fmt.Errorf("standalone compiler starter run returned code %d, stdout %q, stderr %q", starterRunCode, starterRunStdout, starterRunStderr)
	}

	checkCode, checkStdout, checkStderr, err := runStandaloneCommand(
		sourceRoot,
		executablePath,
		"check", "--locked",
	)
	if err != nil {
		return 0, fmt.Errorf("execute standalone compiler stable workspace check: %w", err)
	}
	if checkCode != cli.ExitSuccess || checkStderr != "" ||
		checkStdout != "check ok: 2 instance(s), 2 source file(s), 2 MIR function(s), lock verified\n" {
		return 0, fmt.Errorf("standalone compiler stable check returned code %d, stdout %q, stderr %q", checkCode, checkStdout, checkStderr)
	}
	if atlasCallProcessMeasurement != nil {
		measurement, err := measureStandaloneAtlasCallProcesses(sourceRoot, executablePath, clockResolution)
		if err != nil {
			return 0, fmt.Errorf("measure standalone Atlas calls: %w", err)
		}
		*atlasCallProcessMeasurement = measurement
	}
	stableBuildCode, stableBuildStdout, stableBuildStderr, err := runStandaloneCommand(
		sourceRoot,
		executablePath,
		"build", "--release", "--locked",
	)
	if err != nil {
		return 0, fmt.Errorf("execute standalone compiler stable workspace build: %w", err)
	}
	if stableBuildCode != cli.ExitSuccess || stableBuildStderr != "" ||
		stableBuildStdout != "built: "+stableWorkspaceOutputPath+" (x86_64-windows-msvc, release, lock verified)\n" {
		return 0, fmt.Errorf("standalone compiler stable build returned code %d, stdout %q, stderr %q", stableBuildCode, stableBuildStdout, stableBuildStderr)
	}
	stableArtifactPath := filepath.Join(sourceRoot, filepath.FromSlash(stableWorkspaceOutputPath))
	stableArtifact, err := os.ReadFile(stableArtifactPath)
	if err != nil {
		return 0, fmt.Errorf("read standalone compiler stable build artifact: %w", err)
	}
	if _, err := pecoff.InspectExecutable(stableArtifact); err != nil {
		return 0, fmt.Errorf("inspect standalone compiler stable build PE image: %w", err)
	}
	if _, err := pecoff.InspectAMD64Unwind(stableArtifact); err != nil {
		return 0, fmt.Errorf("inspect standalone compiler stable build unwind data: %w", err)
	}
	if err := executeExpectResult(stableArtifactPath, 42); err != nil {
		return 0, fmt.Errorf("execute standalone compiler stable build artifact: %w", err)
	}
	stableRunCode, stableRunStdout, stableRunStderr, err := runStandaloneCommand(
		sourceRoot,
		executablePath,
		"run", "--release", "--locked",
	)
	if err != nil {
		return 0, fmt.Errorf("execute standalone compiler stable workspace run: %w", err)
	}
	if stableRunCode != 42 || stableRunStderr != "" || stableRunStdout != "" {
		return 0, fmt.Errorf("standalone compiler stable run returned code %d, stdout %q, stderr %q", stableRunCode, stableRunStdout, stableRunStderr)
	}
	stableArtifactAfterRun, err := os.ReadFile(stableArtifactPath)
	if err != nil {
		return 0, fmt.Errorf("read stable build artifact after standalone run: %w", err)
	}
	if !bytes.Equal(stableArtifactAfterRun, stableArtifact) {
		return 0, fmt.Errorf("standalone stable run modified the retained build artifact")
	}
	testCode, testStdout, testStderr, err := runStandaloneCommand(
		sourceRoot,
		executablePath,
		"test", "--locked",
	)
	if err != nil {
		return 0, fmt.Errorf("execute standalone compiler stable workspace tests: %w", err)
	}
	if testCode != cli.ExitSuccess || testStderr != "" ||
		testStdout != "test tests::answer::dependency_answer_is_42 ... ok\ntest result: ok. 1 passed; 0 failed; lock verified\n" {
		return 0, fmt.Errorf("standalone compiler stable test returned code %d, stdout %q, stderr %q", testCode, testStdout, testStderr)
	}
	for _, nativeArguments := range [][]string{
		{"test", "--native", "--exact", "tests::answer::dependency_answer_is_42", "--locked"},
		{"test", "--native", "--release", "--exact", "tests::answer::dependency_answer_is_42", "--locked"},
	} {
		nativeTestCode, nativeTestStdout, nativeTestStderr, err := runStandaloneCommand(
			sourceRoot,
			executablePath,
			nativeArguments...,
		)
		if err != nil {
			return 0, fmt.Errorf("execute standalone compiler native workspace tests %v: %w", nativeArguments, err)
		}
		if nativeTestCode != cli.ExitSuccess || nativeTestStderr != "" ||
			nativeTestStdout != "test tests::answer::dependency_answer_is_42 ... ok\ntest result: ok. 1 passed; 0 failed; lock verified\n" {
			return 0, fmt.Errorf("standalone compiler native test %v returned code %d, stdout %q, stderr %q", nativeArguments, nativeTestCode, nativeTestStdout, nativeTestStderr)
		}
	}
	memberJSONCode, memberJSONStdout, memberJSONStderr, err := runStandaloneCommand(
		sourceRoot,
		executablePath,
		"test", "--package", "example.workspace-quickstart.answer", "--json", "--locked",
	)
	if err != nil {
		return 0, fmt.Errorf("execute standalone compiler semantic workspace-member test: %w", err)
	}
	wantMemberJSON := `{"schemaVersion":"fairyc.test-report/v1","engine":"interpreter","target":null,"profile":null,"tests":[{"identity":"tests::answer::member_answer_is_42","outcome":"passed","trapCategory":null}],"count":1,"passed":1,"failed":0,"status":"ok","lock":"verified"}` + "\n"
	if memberJSONCode != cli.ExitSuccess || memberJSONStderr != "" || memberJSONStdout != wantMemberJSON {
		return 0, fmt.Errorf("standalone compiler semantic workspace-member test returned code %d, stdout %q, stderr %q", memberJSONCode, memberJSONStdout, memberJSONStderr)
	}
	for _, nativeArguments := range [][]string{
		{"test", "--package", "example.workspace-quickstart.answer", "--native", "--locked"},
		{"test", "--package", "example.workspace-quickstart.answer", "--native", "--release", "--locked"},
	} {
		memberNativeCode, memberNativeStdout, memberNativeStderr, err := runStandaloneCommand(
			sourceRoot,
			executablePath,
			nativeArguments...,
		)
		if err != nil {
			return 0, fmt.Errorf("execute standalone compiler native workspace-member test %v: %w", nativeArguments, err)
		}
		if memberNativeCode != cli.ExitSuccess || memberNativeStderr != "" ||
			memberNativeStdout != "test tests::answer::member_answer_is_42 ... ok\ntest result: ok. 1 passed; 0 failed; lock verified\n" {
			return 0, fmt.Errorf("standalone compiler native workspace-member test %v returned code %d, stdout %q, stderr %q", nativeArguments, memberNativeCode, memberNativeStdout, memberNativeStderr)
		}
	}
	workspaceJSONCode, workspaceJSONStdout, workspaceJSONStderr, err := runStandaloneCommand(
		sourceRoot,
		executablePath,
		"test", "--workspace", "--json", "--locked",
	)
	if err != nil {
		return 0, fmt.Errorf("execute standalone compiler semantic workspace aggregate test: %w", err)
	}
	wantWorkspaceJSON := `{"schemaVersion":"fairyc.test-report/v2","engine":"interpreter","target":null,"profile":null,"tests":[{"identity":"example.workspace-quickstart.answer::tests::answer::member_answer_is_42","outcome":"passed","trapCategory":null},{"identity":"example.workspace-quickstart.app::tests::answer::dependency_answer_is_42","outcome":"passed","trapCategory":null}],"count":2,"passed":2,"failed":0,"status":"ok","lock":"verified"}` + "\n"
	if workspaceJSONCode != cli.ExitSuccess || workspaceJSONStderr != "" || workspaceJSONStdout != wantWorkspaceJSON {
		return 0, fmt.Errorf("standalone compiler semantic workspace aggregate test returned code %d, stdout %q, stderr %q", workspaceJSONCode, workspaceJSONStdout, workspaceJSONStderr)
	}
	workspaceListCode, workspaceListStdout, workspaceListStderr, err := runStandaloneCommand(
		sourceRoot,
		executablePath,
		"test", "--workspace", "--list", "--json", "--locked",
	)
	if err != nil {
		return 0, fmt.Errorf("execute standalone compiler workspace aggregate test list: %w", err)
	}
	wantWorkspaceList := `{"schemaVersion":"fairyc.test-list/v2","tests":["example.workspace-quickstart.answer::tests::answer::member_answer_is_42","example.workspace-quickstart.app::tests::answer::dependency_answer_is_42"],"count":2,"lock":"verified"}` + "\n"
	if workspaceListCode != cli.ExitSuccess || workspaceListStderr != "" || workspaceListStdout != wantWorkspaceList {
		return 0, fmt.Errorf("standalone compiler workspace aggregate test list returned code %d, stdout %q, stderr %q", workspaceListCode, workspaceListStdout, workspaceListStderr)
	}
	for _, nativeArguments := range [][]string{
		{"test", "--workspace", "--native", "--locked"},
		{"test", "--workspace", "--native", "--release", "--locked"},
	} {
		workspaceNativeCode, workspaceNativeStdout, workspaceNativeStderr, err := runStandaloneCommand(
			sourceRoot,
			executablePath,
			nativeArguments...,
		)
		if err != nil {
			return 0, fmt.Errorf("execute standalone compiler native workspace aggregate test %v: %w", nativeArguments, err)
		}
		wantWorkspaceNative := "test example.workspace-quickstart.answer::tests::answer::member_answer_is_42 ... ok\n" +
			"test example.workspace-quickstart.app::tests::answer::dependency_answer_is_42 ... ok\n" +
			"test result: ok. 2 passed; 0 failed; lock verified\n"
		if workspaceNativeCode != cli.ExitSuccess || workspaceNativeStderr != "" || workspaceNativeStdout != wantWorkspaceNative {
			return 0, fmt.Errorf("standalone compiler native workspace aggregate test %v returned code %d, stdout %q, stderr %q", nativeArguments, workspaceNativeCode, workspaceNativeStdout, workspaceNativeStderr)
		}
	}
	stableArtifactAfterTest, err := os.ReadFile(stableArtifactPath)
	if err != nil {
		return 0, fmt.Errorf("read stable build artifact after standalone test: %w", err)
	}
	if !bytes.Equal(stableArtifactAfterTest, stableArtifact) {
		return 0, fmt.Errorf("standalone stable test modified the retained build artifact")
	}
	for _, targetID := range []string{"x86_64-windows-msvc", "x86_64-linux-gnu"} {
		emitCode, emitStdout, emitStderr, err := runStandaloneCommand(
			sourceRoot,
			executablePath,
			"experimental", "emit-workspace-c", "--target", targetID, "--locked",
		)
		if err != nil {
			return 0, fmt.Errorf("execute standalone compiler experimental %s C transport: %w", targetID, err)
		}
		if emitCode != cli.ExitSuccess || emitStderr != "" ||
			!strings.Contains(emitStdout, "int main(void)") ||
			!strings.HasSuffix(emitStdout, "\n") || strings.Contains(emitStdout, "emitted:") {
			return 0, fmt.Errorf("standalone compiler experimental %s C transport returned code %d, stdout %q, stderr %q", targetID, emitCode, emitStdout, emitStderr)
		}
	}
	stableArtifactAfterEmit, err := os.ReadFile(stableArtifactPath)
	if err != nil {
		return 0, fmt.Errorf("read stable build artifact after standalone experimental C transport: %w", err)
	}
	if !bytes.Equal(stableArtifactAfterEmit, stableArtifact) {
		return 0, fmt.Errorf("standalone experimental C transport modified the retained build artifact")
	}

	runCode, runStdout, runStderr, err := runStandaloneCommand(
		sourceRoot,
		executablePath,
		"experimental", "run-target", checkedInWindowsQuickstartPath,
		"--target", "x86_64-windows-msvc", "--profile", "optimized",
	)
	if err != nil {
		return 0, fmt.Errorf("execute standalone compiler quickstart: %w", err)
	}
	if runCode != cli.ExitSuccess || runStderr != "" || !strings.HasPrefix(runStdout, "native result: 42 (x86_64-windows-msvc, msvc ") {
		return 0, fmt.Errorf("standalone compiler run-target returned code %d, stdout %q, stderr %q", runCode, runStdout, runStderr)
	}

	execCode, execStdout, execStderr, err := runStandaloneCommand(
		sourceRoot,
		executablePath,
		"experimental", "exec-target", checkedInWindowsOutputPath,
		"--source", checkedInWindowsOutputSupportPath,
		"--target", "x86_64-windows-msvc", "--profile", "optimized",
	)
	if err != nil {
		return 0, fmt.Errorf("execute standalone compiler output program: %w", err)
	}
	if execCode != 42 || execStdout != "Fairy C says hi\n" || execStderr != "" {
		return 0, fmt.Errorf("standalone compiler exec-target returned code %d, stdout %q, stderr %q", execCode, execStdout, execStderr)
	}
	stdIORoot := filepath.Join(sourceRoot, "examples", "std_io_quickstart")
	stdIORunCode, stdIORunStdout, stdIORunStderr, err := runStandaloneCommandWithInput(
		stdIORoot,
		executablePath,
		"stdin!\n",
		"run", "--release", "--locked",
	)
	if err != nil {
		return 0, fmt.Errorf("execute standalone compiler std::io workspace run: %w", err)
	}
	if stdIORunCode != cli.ExitSuccess || stdIORunStdout != "Fairy C says hi\nbytes on stdout\n-2147483648\nstdin!\n" ||
		stdIORunStderr != "Fairy C warns too\nbytes on stderr\n" {
		return 0, fmt.Errorf("standalone compiler std::io run returned code %d, stdout %q, stderr %q", stdIORunCode, stdIORunStdout, stdIORunStderr)
	}
	stdIOBuildCode, stdIOBuildStdout, stdIOBuildStderr, err := runStandaloneCommand(
		stdIORoot,
		executablePath,
		"build", "--release", "--locked",
	)
	if err != nil {
		return 0, fmt.Errorf("execute standalone compiler std::io workspace build: %w", err)
	}
	if stdIOBuildCode != cli.ExitSuccess || stdIOBuildStderr != "" ||
		stdIOBuildStdout != "built: build/std-io-quickstart.exe (x86_64-windows-msvc, release, lock verified)\n" {
		return 0, fmt.Errorf("standalone compiler std::io build returned code %d, stdout %q, stderr %q", stdIOBuildCode, stdIOBuildStdout, stdIOBuildStderr)
	}
	stdIOArtifactPath := filepath.Join(stdIORoot, "build", "std-io-quickstart.exe")
	if err := executeExpectOutput(
		stdIOArtifactPath,
		0,
		"Fairy C says hi\nbytes on stdout\n-2147483648\n",
		"Fairy C warns too\nbytes on stderr\n",
	); err != nil {
		return 0, fmt.Errorf("execute standalone compiler std::io artifact: %w", err)
	}
	if err := executeExpectOutputWithInput(
		stdIOArtifactPath,
		"stdin!\n",
		0,
		"Fairy C says hi\nbytes on stdout\n-2147483648\nstdin!\n",
		"Fairy C warns too\nbytes on stderr\n",
	); err != nil {
		return 0, fmt.Errorf("execute standalone compiler std::io artifact with stdin: %w", err)
	}

	workspaceCode, workspaceStdout, workspaceStderr, err := runStandaloneCommand(
		sourceRoot,
		executablePath,
		"experimental", "run-workspace", "--locked",
		"--target", "x86_64-windows-msvc", "--profile", "optimized",
	)
	if err != nil {
		return 0, fmt.Errorf("execute standalone compiler workspace: %w", err)
	}
	if workspaceCode != cli.ExitSuccess || workspaceStderr != "" || !strings.HasPrefix(workspaceStdout, "native workspace result: 42 (x86_64-windows-msvc, msvc ") {
		return 0, fmt.Errorf("standalone compiler run-workspace returned code %d, stdout %q, stderr %q", workspaceCode, workspaceStdout, workspaceStderr)
	}
	if atlasCallProcessMeasurement != nil {
		persistentCacheMeasurement, err := measureStandaloneWorkspaceCache(executablePath, sourceRoot, clockResolution)
		if err != nil {
			return 0, err
		}
		persistentCacheMeasurementJSON, err := buildmeasure.MarshalPersistentCacheProcess(persistentCacheMeasurement)
		if err != nil {
			return 0, fmt.Errorf("encode standalone persistent-cache process measurement: %w", err)
		}
		fmt.Printf("Fairy C persistent-cache process measurement: %s\n", persistentCacheMeasurementJSON)
		persistentCacheGrowth, err := measureStandaloneWorkspaceCacheGrowth(executablePath, sourceRoot)
		if err != nil {
			return 0, err
		}
		persistentCacheGrowthJSON, err := buildmeasure.MarshalPersistentCacheGrowth(persistentCacheGrowth)
		if err != nil {
			return 0, fmt.Errorf("encode standalone persistent-cache growth measurement: %w", err)
		}
		fmt.Printf("Fairy C persistent-cache growth measurement: %s\n", persistentCacheGrowthJSON)

		sessionMeasurement, err := measureStandaloneWorkspaceSession(executablePath, sourceRoot)
		if err != nil {
			return 0, err
		}
		sessionMeasurementJSON, err := buildmeasure.MarshalSessionProcess(sessionMeasurement)
		if err != nil {
			return 0, fmt.Errorf("encode standalone workspace-session process measurement: %w", err)
		}
		fmt.Printf("Fairy C workspace-session process measurement: %s\n", sessionMeasurementJSON)
	}
	return info.Size(), nil
}

type standaloneWorkspaceSessionResponse struct {
	Sequence     int    `json:"sequence"`
	Status       string `json:"status"`
	ExitCode     int    `json:"exitCode"`
	MIRFunctions int    `json:"mirFunctions"`
	Cache        struct {
		Outcome         string `json:"outcome"`
		Hits            int    `json:"hits"`
		Misses          int    `json:"misses"`
		Lowered         bool   `json:"lowered"`
		RetainedEntries int    `json:"retainedEntries"`
		RetainedBytes   uint64 `json:"retainedBytes"`
		VerifiedHits    uint64 `json:"verifiedHits"`
	} `json:"cache"`
}

type standaloneWorkspaceCacheReport struct {
	SchemaVersion string `json:"schemaVersion"`
	CompilerBuild string `json:"compilerBuild"`
	MIRFunctions  int    `json:"mirFunctions"`
	Cache         struct {
		Storage            string `json:"storage"`
		Outcome            string `json:"outcome"`
		Packages           int    `json:"packages"`
		Hits               int    `json:"hits"`
		Misses             int    `json:"misses"`
		Rejected           int    `json:"rejected"`
		Stored             int    `json:"stored"`
		StoreSkipped       int    `json:"storeSkipped"`
		StoreFailed        int    `json:"storeFailed"`
		Lowered            bool   `json:"lowered"`
		ReassemblyFallback bool   `json:"reassemblyFallback"`
		RetainedEntries    int    `json:"retainedEntries"`
		RetainedBytes      uint64 `json:"retainedBytes"`
		RecordHits         uint64 `json:"recordHits"`
		RecordMisses       uint64 `json:"recordMisses"`
		RecordRejected     uint64 `json:"recordRejected"`
		Evictions          uint64 `json:"evictions"`
		VerifiedHits       uint64 `json:"verifiedHits"`
	} `json:"cache"`
}
