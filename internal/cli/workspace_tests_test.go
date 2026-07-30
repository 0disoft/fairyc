package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/lockfile"
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
	for _, arguments := range [][]string{{"test", "--release"}, {"test", "--locked", "--locked"}} {
		code, stdout, stderr = runForTest(arguments, root)
		if code != ExitInvalidInvocation || stdout != "" || !strings.Contains(stderr, `command "test"`) {
			t.Fatalf("invalid invocation %v = (%d, %q, %q)", arguments, code, stdout, stderr)
		}
	}
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
language = "fairyc-language/6"
source = "src"
`)
	writeWorkspaceFile(t, root, "src/library.fy", "module library;\npub fn i32 answer() { return 42; }\n")
}
