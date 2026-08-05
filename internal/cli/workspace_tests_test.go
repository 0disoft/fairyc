package cli

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/cruntime"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/lockfile"
	"github.com/0disoft/fairyc/internal/mir"
)

func TestStableWorkspaceTestRunsIsolatedVoidTestsInCanonicalOrder(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeStableTestWorkspace(t, root)
	writeWorkspaceFile(t, root, "tests/zeta.fy", `module tests::zeta;
import library as library;
@test
fn void second() { assert(library::answer() == 42); }
`)
	writeWorkspaceFile(t, root, "tests/alpha.fy", `module tests::alpha;
import library as library;
fn bool helper() { return library::answer() == 42; }
@test
fn void first() {
    let i32 value = library::answer();
    assert(value == 42);
}
`)

	code, stdout, stderr := runForTest([]string{"test"}, root)
	want := "test tests::alpha::first ... ok\n" +
		"test tests::zeta::second ... ok\n" +
		"test result: ok. 2 passed; 0 failed; lock updated\n"
	if code != ExitSuccess || stdout != want || stderr != "" {
		t.Fatalf("test = (%d, %q, %q), want stdout %q", code, stdout, stderr, want)
	}
	if matches, err := filepath.Glob(filepath.Join(root, "build", "*")); err != nil || len(matches) != 0 {
		t.Fatalf("test build artifacts = %v, %v", matches, err)
	}

	lockPath := filepath.Join(root, lockfile.FileName)
	lockBytes, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatal(err)
	}
	lockTimestamp := time.Unix(1_700_000_000, 0)
	if err := os.Chtimes(lockPath, lockTimestamp, lockTimestamp); err != nil {
		t.Fatal(err)
	}
	code, stdout, stderr = runForTest([]string{"test", "--locked"}, root)
	want = strings.Replace(want, "lock updated", "lock verified", 1)
	if code != ExitSuccess || stdout != want || stderr != "" {
		t.Fatalf("locked test = (%d, %q, %q), want stdout %q", code, stdout, stderr, want)
	}
	lockedBytes, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatal(err)
	}
	lockedInfo, err := os.Stat(lockPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(lockedBytes, lockBytes) || !lockedInfo.ModTime().Equal(lockTimestamp) {
		t.Fatalf("locked test modified fairy.lock: bytesEqual=%t modTime=%s", bytes.Equal(lockedBytes, lockBytes), lockedInfo.ModTime())
	}
}

func TestStableWorkspaceTestRunsRemainingTestsAfterATrap(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeStableTestWorkspace(t, root)
	writeWorkspaceFile(t, root, "tests/outcomes.fy", `module tests::outcomes;
@test
fn void first_fails() { assert(false); }
@test
fn void second_passes() { assert(true); }
`)

	code, stdout, stderr := runForTest([]string{"test"}, root)
	want := "test tests::outcomes::first_fails ... FAILED (assertion-failed)\n" +
		"test tests::outcomes::second_passes ... ok\n" +
		"test result: FAILED. 1 passed; 1 failed; lock updated\n"
	if code != ExitProgramTrap || stdout != want || stderr != "" {
		t.Fatalf("test = (%d, %q, %q), want stdout %q", code, stdout, stderr, want)
	}
}

func TestStableWorkspaceTestValidatesHarnessOwnedDeclarations(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		source string
	}{
		{name: "attribute arguments", source: "@test(\"extra\")\nfn void invalid() {}"},
		{name: "duplicate attribute", source: "@test\n@test\nfn void invalid() {}"},
		{name: "public", source: "@test\npub fn void invalid() {}"},
		{name: "unsafe", source: "@test\nunsafe fn void invalid() {}"},
		{name: "parameter", source: "@test\nfn void invalid(i32 value) {}"},
		{name: "result", source: "@test\nfn i32 invalid() { return 0; }"},
		{name: "non-function", source: "@test\nstruct invalid { i32 value; }"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			writeStableTestWorkspace(t, root)
			writeWorkspaceFile(t, root, "tests/invalid.fy", "module tests::invalid;\n"+test.source+"\n")
			code, stdout, stderr := runForTest([]string{"test"}, root)
			if code != ExitSourceDiagnostics || stdout != "" || !strings.Contains(stderr, "tests/invalid.fy") || !strings.Contains(stderr, "E9001") || strings.Contains(stderr, "experimental") {
				t.Fatalf("test = (%d, %q, %q)", code, stdout, stderr)
			}
		})
	}
}

func TestStableWorkspaceTestKeepsProductionAndTestSourceFailuresSeparate(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeStableTestWorkspace(t, root)
	writeWorkspaceFile(t, root, "tests/broken.fy", "module tests::broken;\n@test\nfn void broken( {\n")

	code, stdout, stderr := runForTest([]string{"check"}, root)
	if code != ExitSuccess || stdout != "check ok: 1 instance(s), 1 source file(s), 1 MIR function(s), lock updated\n" || stderr != "" {
		t.Fatalf("stable check read tests/ = (%d, %q, %q)", code, stdout, stderr)
	}
	code, stdout, stderr = runForTest([]string{"test", "--locked"}, root)
	if code != ExitSourceDiagnostics || stdout != "" || !strings.Contains(stderr, "tests/broken.fy") {
		t.Fatalf("stable test syntax failure = (%d, %q, %q)", code, stdout, stderr)
	}
}

func TestStableWorkspaceTestHandlesZeroTestsInvocationAndOutputFailure(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeStableTestWorkspace(t, root)

	code, stdout, stderr := runForTest([]string{"test"}, root)
	if code != ExitSuccess || stdout != "test result: ok. 0 passed; 0 failed; lock updated\n" || stderr != "" {
		t.Fatalf("zero tests = (%d, %q, %q)", code, stdout, stderr)
	}
	var outputError bytes.Buffer
	code = Run([]string{"test", "--locked"}, root, failingWriter{}, &outputError)
	if code != ExitInternalFailure || !strings.Contains(outputError.String(), "write output failed") {
		t.Fatalf("output failure = (%d, %q)", code, outputError.String())
	}
	for _, arguments := range [][]string{{"test", "--release"}, {"test", "--locked", "--locked"}, {"test", "--native", "--native"}} {
		code, stdout, stderr = runForTest(arguments, root)
		if code != ExitInvalidInvocation || stdout != "" || !strings.Contains(stderr, `command "test"`) {
			t.Fatalf("invalid invocation %v = (%d, %q, %q)", arguments, code, stdout, stderr)
		}
	}
}

func TestStableWorkspaceTestNativeOptionsRequireExplicitNativeMode(t *testing.T) {
	t.Parallel()
	options, err := parseStableTestOptions([]string{"--release", "--locked", "--native"})
	if err != nil || !options.native || !options.release || !options.locked {
		t.Fatalf("native options = %+v, %v", options, err)
	}
	if _, err := parseStableTestOptions([]string{"--release"}); err == nil || !strings.Contains(err.Error(), "requires --native") {
		t.Fatalf("release-only options error = %v", err)
	}
}

func TestNativeWorkspaceTestProgramAddsVerifiedI32Harness(t *testing.T) {
	t.Parallel()
	program := nativeTestFixtureProgram()
	native, entry, err := nativeWorkspaceTestProgram(program, "tests::native::passes")
	if err != nil {
		t.Fatal(err)
	}
	if entry != nativeWorkspaceTestEntry || len(native.Functions) != len(program.Functions)+1 {
		t.Fatalf("native test program entry=%q functions=%d", entry, len(native.Functions))
	}
	if failures := mir.Verify(native); len(failures) != 0 {
		t.Fatalf("native test MIR failures = %v", failures)
	}
	unit, err := cbackend.Emit(native, entry, cbackend.Stage0WindowsMSVCProfile())
	if err != nil {
		t.Fatal(err)
	}
	if unit.Entry != nativeWorkspaceTestEntry || !strings.Contains(unit.Source, "int main(void)") ||
		!strings.Contains(unit.Source, "fy_fn_0();") {
		t.Fatalf("native test C unit = entry %q, functions %v", unit.Entry, unit.Functions)
	}
}

func TestNativeWorkspaceTestsContinueAfterTrapsAndPreserveOrder(t *testing.T) {
	t.Parallel()
	program := nativeTestFixtureProgram()
	tests := []workspaceTest{
		{display: "tests::native::fails", entry: "tests::native::fails"},
		{display: "tests::native::passes", entry: "tests::native::passes"},
	}
	compiled := 0
	compile := func(program mir.Program, entry string, profile cdriver.Profile, stderr io.Writer) (cdriver.WindowsHostArtifact, int) {
		if entry != nativeWorkspaceTestEntry || profile != cdriver.OptimizedProfile || len(mir.Verify(program)) != 0 {
			t.Fatalf("compile native test entry=%q profile=%+v", entry, profile)
		}
		compiled++
		return cdriver.WindowsHostArtifact{Bytes: []byte{byte(compiled)}}, ExitSuccess
	}
	execute := func(artifact cdriver.WindowsHostArtifact) (cdriver.WindowsHostOutcome, error) {
		if artifact.Bytes[0] == 1 {
			code, found := cbackend.TrapCode("assertion-failed")
			if !found {
				t.Fatal("assertion-failed trap code is missing")
			}
			return cdriver.WindowsHostOutcome{
				ExitCode: int32(productTrapExitBase + code),
				Stderr:   cruntime.RunTrapReceiptPrefix + strconv.Itoa(int(code)),
			}, nil
		}
		return cdriver.WindowsHostOutcome{}, nil
	}
	var stderr bytes.Buffer
	outcomes, code := runNativeWorkspaceTestsWith(
		program, tests, cdriver.OptimizedProfile, &stderr, compile, execute,
	)
	if code != ExitSuccess || stderr.String() != "" || compiled != 2 || len(outcomes) != 2 ||
		outcomes[0].display != tests[0].display || outcomes[0].trapCategory != "assertion-failed" ||
		outcomes[1].display != tests[1].display || outcomes[1].trapCategory != "" {
		t.Fatalf("native outcomes=%+v code=%d stderr=%q compiled=%d", outcomes, code, stderr.String(), compiled)
	}
}

func TestNativeWorkspaceTestsRejectProgramOutputAndUnreceiptedExit(t *testing.T) {
	t.Parallel()
	program := nativeTestFixtureProgram()
	tests := []workspaceTest{{display: "tests::native::passes", entry: "tests::native::passes"}}
	compile := func(program mir.Program, entry string, profile cdriver.Profile, stderr io.Writer) (cdriver.WindowsHostArtifact, int) {
		return cdriver.WindowsHostArtifact{Bytes: []byte{1}}, ExitSuccess
	}
	for _, test := range []struct {
		name     string
		outcome  cdriver.WindowsHostOutcome
		wantCode int
		contains string
	}{
		{name: "stdout", outcome: cdriver.WindowsHostOutcome{Stdout: "unexpected"}, wantCode: ExitInvalidInvocation, contains: "produced program output"},
		{name: "unreceipted exit", outcome: cdriver.WindowsHostOutcome{ExitCode: 42}, wantCode: ExitBuildFailure, contains: "without a valid trap receipt"},
	} {
		t.Run(test.name, func(t *testing.T) {
			var stderr bytes.Buffer
			outcomes, code := runNativeWorkspaceTestsWith(
				program,
				tests,
				cdriver.LowProfile,
				&stderr,
				compile,
				func(cdriver.WindowsHostArtifact) (cdriver.WindowsHostOutcome, error) { return test.outcome, nil },
			)
			if outcomes != nil || code != test.wantCode || !strings.Contains(stderr.String(), test.contains) {
				t.Fatalf("native failure outcomes=%+v code=%d stderr=%q", outcomes, code, stderr.String())
			}
		})
	}
}

func nativeTestFixtureProgram() mir.Program {
	makeTest := func(symbol hir.SymbolID, name string) mir.Function {
		return mir.Function{
			Symbol: symbol, Name: name, ResultType: hir.TypeVoid, Entry: 0,
			Blocks: []mir.Block{{
				ID: 0,
				Terminator: mir.Terminator{
					Kind: mir.TerminatorReturn, UnwindSuccessor: mir.NoBlock, DestinationLocal: mir.NoLocal,
				},
			}},
		}
	}
	return mir.Program{Functions: []mir.Function{
		makeTest(1, "tests::native::passes"),
		makeTest(2, "tests::native::fails"),
	}}
}

func TestStableWorkspaceTestLockedModeRequiresCanonicalProductionLock(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeStableTestWorkspace(t, root)
	writeWorkspaceFile(t, root, "tests/pass.fy", "module tests::pass;\n@test\nfn void passes() { assert(true); }\n")

	code, stdout, stderr := runForTest([]string{"test", "--locked"}, root)
	if code != ExitPackageFailure || stdout != "" || !strings.Contains(stderr, "requires a valid fairy.lock") {
		t.Fatalf("locked test = (%d, %q, %q)", code, stdout, stderr)
	}
	if _, err := os.Stat(filepath.Join(root, lockfile.FileName)); !os.IsNotExist(err) {
		t.Fatalf("locked test created fairy.lock: %v", err)
	}
}

func writeStableTestWorkspace(t *testing.T, root string) {
	t.Helper()
	writeWorkspaceFile(t, root, "fairy.toml", `schema = "fairyc.manifest/v1"
[package]
uid = "example.stable-test.app"
name = "stable-test-app"
version = "1.0.0"
language = "fairyc-language/8"
source = "src"
`)
	writeWorkspaceFile(t, root, "src/library.fy", "module library;\npub fn i32 answer() { return 42; }\n")
}
