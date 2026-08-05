package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/lockfile"
	"github.com/0disoft/fairyc/internal/target"
)

func TestParseStableEmitOptions(t *testing.T) {
	t.Parallel()
	for _, id := range []target.ID{target.X8664LinuxGNU, target.X8664WindowsMSVC} {
		options, err := parseStableEmitOptions([]string{"c", "--locked", "--target", string(id)})
		if err != nil || options.targetID != id || !options.locked {
			t.Fatalf("parseStableEmitOptions(%q) = %+v, %v", id, options, err)
		}
	}
	for _, test := range []struct {
		arguments []string
		contains  string
	}{
		{contains: "expects form c"},
		{arguments: []string{"mir", "--target", string(target.X8664LinuxGNU)}, contains: "only form c"},
		{arguments: []string{"c"}, contains: "requires --target"},
		{arguments: []string{"c", "--target"}, contains: "requires a value"},
		{arguments: []string{"c", "--target", "aarch64-linux-gnu"}, contains: "does not know target"},
		{arguments: []string{"c", "--target", string(target.X8664LinuxGNU), "--target", string(target.X8664WindowsMSVC)}, contains: "more than once"},
		{arguments: []string{"c", "--locked", "--locked", "--target", string(target.X8664LinuxGNU)}, contains: "more than once"},
		{arguments: []string{"c", "--target", string(target.X8664LinuxGNU), "--release"}, contains: "does not accept"},
	} {
		if _, err := parseStableEmitOptions(test.arguments); err == nil || !strings.Contains(err.Error(), test.contains) {
			t.Errorf("parseStableEmitOptions(%q) error = %v; want containing %q", test.arguments, err, test.contains)
		}
	}
}

func TestStableEmitWritesDeterministicTargetSelectedCAndMaintainsLock(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeEmitWorkspace(t, root, true)

	windowsArgs := []string{"emit", "c", "--target", string(target.X8664WindowsMSVC)}
	code, windowsC, stderr := runForTest(windowsArgs, root)
	if code != ExitSuccess || stderr != "" {
		t.Fatalf("Windows emit = (%d, %q)", code, stderr)
	}
	if !strings.Contains(windowsC, "_Static_assert(sizeof(long) == UINT64_C(4)") ||
		!strings.Contains(windowsC, "extern long inspect_c_long(void);") ||
		!strings.Contains(windowsC, "int main(void)") || strings.Contains(windowsC, "emitted:") ||
		!strings.HasSuffix(windowsC, "\n") {
		t.Fatalf("Windows C output does not match the raw LLP64 C11 contract:\n%s", windowsC)
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

	lockedWindowsArgs := append(windowsArgs, "--locked")
	code, repeatedWindowsC, stderr := runForTest(lockedWindowsArgs, root)
	if code != ExitSuccess || repeatedWindowsC != windowsC || stderr != "" {
		t.Fatalf("locked repeated Windows emit = (%d, equal=%t, %q)", code, repeatedWindowsC == windowsC, stderr)
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
		t.Fatalf("locked emit modified fairy.lock: bytesEqual=%t modTime=%s", bytes.Equal(lockedBytes, lockBytes), lockedInfo.ModTime())
	}

	code, linuxC, stderr := runForTest([]string{"emit", "c", "--target", string(target.X8664LinuxGNU), "--locked"}, root)
	if code != ExitSuccess || stderr != "" || linuxC == windowsC ||
		!strings.Contains(linuxC, "_Static_assert(sizeof(long) == UINT64_C(8)") ||
		!strings.Contains(linuxC, "int main(void)") {
		t.Fatalf("Linux emit = (%d, differs=%t, %q)\n%s", code, linuxC != windowsC, stderr, linuxC)
	}
	if _, err := os.Stat(filepath.Join(root, "build")); !os.IsNotExist(err) {
		t.Fatalf("emit created a build directory: %v", err)
	}
}

func TestStableEmitPreservesFailureCategoriesAndWritesNoPartialArtifact(t *testing.T) {
	t.Parallel()
	code, stdout, stderr := runForTest([]string{"emit", "c", "--target", string(target.X8664WindowsMSVC)}, t.TempDir())
	if code != ExitPackageFailure || stdout != "" || !strings.Contains(stderr, "fairy.toml") {
		t.Fatalf("missing workspace = (%d, %q, %q)", code, stdout, stderr)
	}

	syntaxRoot := t.TempDir()
	writeEmitWorkspace(t, syntaxRoot, true)
	writeWorkspaceFile(t, syntaxRoot, "src/main.fy", "module main;\nfn i32 broken( { return 0; }\n")
	code, stdout, stderr = runForTest([]string{"emit", "c", "--target", string(target.X8664WindowsMSVC)}, syntaxRoot)
	if code != ExitSourceDiagnostics || stdout != "" || !strings.Contains(stderr, "E0007") {
		t.Fatalf("source diagnostics = (%d, %q, %q)", code, stdout, stderr)
	}

	unsupportedRoot := t.TempDir()
	writeEmitWorkspace(t, unsupportedRoot, true)
	writeWorkspaceFile(t, unsupportedRoot, "src/main.fy", "module main;\n@mystery\nfn i32 main() { return 42; }\n")
	code, stdout, stderr = runForTest([]string{"emit", "c", "--target", string(target.X8664WindowsMSVC)}, unsupportedRoot)
	if code != ExitSourceDiagnostics || stdout != "" || !strings.Contains(stderr, "semantic subset does not support function attribute @mystery") || strings.Contains(stderr, "experimental semantic subset") {
		t.Fatalf("unsupported semantics = (%d, %q, %q)", code, stdout, stderr)
	}

	missingEntryRoot := t.TempDir()
	writeEmitWorkspace(t, missingEntryRoot, false)
	code, stdout, stderr = runForTest([]string{"emit", "c", "--target", string(target.X8664WindowsMSVC)}, missingEntryRoot)
	if code != ExitInvalidInvocation || stdout != "" || !strings.Contains(stderr, "root entry") || strings.Contains(stderr, "experimental") {
		t.Fatalf("missing entry = (%d, %q, %q)", code, stdout, stderr)
	}
}

func TestStableEmitHandlesHelpAndOutputFailure(t *testing.T) {
	t.Parallel()
	for _, flag := range []string{"-h", "--help"} {
		code, stdout, stderr := runForTest([]string{"emit", flag}, t.TempDir())
		if code != ExitSuccess || !strings.Contains(stdout, "fairyc emit c --target <target-id> [--locked]") || stderr != "" {
			t.Fatalf("emit %s = (%d, %q, %q)", flag, code, stdout, stderr)
		}
	}
	root := t.TempDir()
	writeEmitWorkspace(t, root, true)
	var failure bytes.Buffer
	code := Run(
		[]string{"emit", "c", "--target", string(target.X8664WindowsMSVC)},
		root,
		failingWriter{},
		&failure,
	)
	if code != ExitInternalFailure || !strings.Contains(failure.String(), "write output failed") {
		t.Fatalf("emit output failure = (%d, %q)", code, failure.String())
	}
	partial := &prefixFailingWriter{limit: 32}
	failure.Reset()
	code = Run(
		[]string{"emit", "c", "--target", string(target.X8664WindowsMSVC), "--locked"},
		root,
		partial,
		&failure,
	)
	if code != ExitInternalFailure || partial.Len() != partial.limit ||
		!strings.Contains(failure.String(), "write output failed") {
		t.Fatalf("emit partial output failure = (%d, bytes=%d, %q)", code, partial.Len(), failure.String())
	}
	if _, err := os.Stat(filepath.Join(root, "build")); !os.IsNotExist(err) {
		t.Fatalf("output failure created a build directory: %v", err)
	}
}

type prefixFailingWriter struct {
	buffer bytes.Buffer
	limit  int
}

func (w *prefixFailingWriter) Write(value []byte) (int, error) {
	if len(value) <= w.limit {
		return w.buffer.Write(value)
	}
	written, _ := w.buffer.Write(value[:w.limit])
	return written, errors.New("intentional partial write failure")
}

func (w *prefixFailingWriter) Len() int {
	return w.buffer.Len()
}

func writeEmitWorkspace(t *testing.T, root string, withEntry bool) {
	t.Helper()
	writeWorkspaceFile(t, root, "fairy.toml", `schema = "fairyc.manifest/v1"
[package]
uid = "example.emit.app"
name = "emit-app"
version = "1.0.0"
language = "fairyc-language/9"
source = "src"
`)
	entry := "fn i32 not_main() { return 42; }"
	if withEntry {
		entry = "fn i32 main() { let c::long inspected = unsafe { inspect_c_long() }; return 42; }"
	}
	writeWorkspaceFile(
		t,
		root,
		"src/main.fy",
		"module main;\n\nunsafe extern C fn c::long inspect_c_long() ;\n\n"+entry+"\n",
	)
}
