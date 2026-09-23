package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/lockfile"
	"github.com/0disoft/fairyc/internal/sdkstdlib"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

func TestExperimentalWorkspaceCheckUpdatesAndVerifiesLock(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeWorkspaceFile(t, root, "fairy.toml", `schema = "fairyc.manifest/v1"
[package]
uid = "example.workspace.app"
name = "app"
version = "1.0.0"
language = "fairyc-language/17"
source = "src"
[workspace]
members = ["libs/net"]
[dependencies.net]
path = "libs/net"
version = "1.0.0"
features = []
`)
	writeWorkspaceFile(t, root, "src/main.fy", "module main;\nimport net as n;\nfn i32 answer() { return n::value(); }\n")
	writeWorkspaceFile(t, root, "libs/net/fairy.toml", `schema = "fairyc.manifest/v1"
[package]
uid = "example.workspace.net"
name = "net"
version = "1.0.0"
language = "fairyc-language/17"
source = "src"
`)
	writeWorkspaceFile(t, root, "libs/net/src/net.fy", "module net;\npub fn i32 value() { return 1; }\n")

	code, stdout, stderr := runForTest([]string{"experimental", "check-workspace"}, root)
	if code != ExitSuccess || stdout != "workspace check ok: 2 instance(s), 2 source file(s), 2 MIR function(s), lock updated\n" || stderr != "" {
		t.Fatalf("ordinary check = (%d, %q, %q)", code, stdout, stderr)
	}
	if _, err := os.Stat(filepath.Join(root, "fairy.lock")); err != nil {
		t.Fatalf("lockfile was not written: %v", err)
	}
	code, stdout, stderr = runForTest([]string{"experimental", "check-workspace", "--locked"}, root)
	if code != ExitSuccess || !strings.HasSuffix(stdout, "lock verified\n") || stderr != "" {
		t.Fatalf("locked check = (%d, %q, %q)", code, stdout, stderr)
	}

	writeWorkspaceFile(t, root, "libs/net/src/net.fy", "module net;\npub fn i32 value() { return 2; }\n")
	code, stdout, stderr = runForTest([]string{"experimental", "check-workspace", "--locked"}, root)
	if code != ExitPackageFailure || stdout != "" || !strings.Contains(stderr, "stale") {
		t.Fatalf("stale locked check = (%d, %q, %q)", code, stdout, stderr)
	}
	code, stdout, stderr = runForTest([]string{"experimental", "check-workspace"}, root)
	if code != ExitSuccess || !strings.HasSuffix(stdout, "lock updated\n") || stderr != "" {
		t.Fatalf("refresh check = (%d, %q, %q)", code, stdout, stderr)
	}
}

func TestStableCheckOwnsTheCurrentWorkspaceAndLockContract(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeWorkspaceFile(t, root, "fairy.toml", `schema = "fairyc.manifest/v1"
[package]
uid = "example.stable-check.app"
name = "app"
version = "1.0.0"
language = "fairyc-language/17"
source = "src"
`)
	writeWorkspaceFile(t, root, "src/main.fy", "module main;\nfn i32 answer() { return 42; }\n")

	code, stdout, stderr := runForTest([]string{"check"}, root)
	if code != ExitSuccess || stdout != "check ok: 1 instance(s), 1 source file(s), 1 MIR function(s), lock updated\n" || stderr != "" {
		t.Fatalf("stable check = (%d, %q, %q)", code, stdout, stderr)
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
	code, stdout, stderr = runForTest([]string{"check", "--locked"}, root)
	if code != ExitSuccess || stdout != "check ok: 1 instance(s), 1 source file(s), 1 MIR function(s), lock verified\n" || stderr != "" {
		t.Fatalf("stable locked check = (%d, %q, %q)", code, stdout, stderr)
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
		t.Fatalf("locked check modified fairy.lock: bytesEqual=%t modTime=%s", bytes.Equal(lockedBytes, lockBytes), lockedInfo.ModTime())
	}
	var outputError bytes.Buffer
	code = Run([]string{"check", "--locked"}, root, failingWriter{}, &outputError)
	if code != ExitInternalFailure || !strings.Contains(outputError.String(), "write output failed") {
		t.Fatalf("stable check output failure = (%d, %q)", code, outputError.String())
	}
	code, stdout, stderr = runForTest([]string{"check", "extra"}, root)
	if code != ExitInvalidInvocation || stdout != "" || !strings.Contains(stderr, `command "check" accepts only optional --locked`) {
		t.Fatalf("invalid stable check = (%d, %q, %q)", code, stdout, stderr)
	}
}

func TestStableCheckPreservesPackageAndSourceFailureCategories(t *testing.T) {
	t.Parallel()
	code, stdout, stderr := runForTest([]string{"check"}, t.TempDir())
	if code != ExitPackageFailure || stdout != "" || !strings.Contains(stderr, "fairy.toml") {
		t.Fatalf("missing stable workspace = (%d, %q, %q)", code, stdout, stderr)
	}

	root := t.TempDir()
	writeWorkspaceFile(t, root, "fairy.toml", `schema = "fairyc.manifest/v1"
[package]
uid = "example.stable-check.invalid"
name = "invalid"
version = "1.0.0"
language = "fairyc-language/17"
source = "src"
`)
	writeWorkspaceFile(t, root, "src/main.fy", "module main;\nfn i32 broken( { return 0; }\n")
	code, stdout, stderr = runForTest([]string{"check"}, root)
	if code != ExitSourceDiagnostics || stdout != "" || !strings.Contains(stderr, "E0007") {
		t.Fatalf("stable source diagnostics = (%d, %q, %q)", code, stdout, stderr)
	}

	writeWorkspaceFile(t, root, "src/main.fy", "module main;\n@mystery\nfn i32 main() { return 42; }\n")
	code, stdout, stderr = runForTest([]string{"check"}, root)
	if code != ExitSourceDiagnostics || stdout != "" ||
		!strings.Contains(stderr, "semantic subset does not support function attribute @mystery") ||
		strings.Contains(stderr, "experimental semantic subset") {
		t.Fatalf("stable unsupported semantics = (%d, %q, %q)", code, stdout, stderr)
	}
}

func TestExperimentalWorkspaceCheckRejectsInvocationAndMissingManifest(t *testing.T) {
	t.Parallel()
	code, stdout, stderr := runForTest([]string{"experimental", "check-workspace", "extra"}, t.TempDir())
	if code != ExitInvalidInvocation || stdout != "" || !strings.Contains(stderr, "optional --locked") {
		t.Fatalf("invalid invocation = (%d, %q, %q)", code, stdout, stderr)
	}
	code, stdout, stderr = runForTest([]string{"experimental", "check-workspace"}, t.TempDir())
	if code != ExitPackageFailure || stdout != "" || !strings.Contains(stderr, "fairy.toml") {
		t.Fatalf("missing manifest = (%d, %q, %q)", code, stdout, stderr)
	}
}

func TestExperimentalWorkspaceCheckLocksBundledStandardLibrary(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeWorkspaceFile(t, root, "fairy.toml", `schema = "fairyc.manifest/v1"
[package]
uid = "example.workspace.app"
name = "app"
version = "1.0.0"
language = "fairyc-language/17"
source = "src"
`)
	writeWorkspaceFile(t, root, "src/main.fy", "module main;\n\nimport std::alloc as alloc;\n\nfn i32 inspect(ref mut alloc::Arena arena) {\n    return 1;\n}\n")

	code, stdout, stderr := runForTest([]string{"experimental", "check-workspace"}, root)
	if code != ExitSuccess || !strings.Contains(stdout, "2 instance(s), 7 source file(s)") || !strings.HasSuffix(stdout, "lock updated\n") || stderr != "" {
		t.Fatalf("workspace check = (%d, %q, %q)", code, stdout, stderr)
	}
	document, _, err := lockfile.Read(filepath.Join(root, lockfile.FileName))
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, record := range document.Packages {
		if record.SourceKind != workspaceloader.SourceKindSDK {
			continue
		}
		found = record.PackageUID == sdkstdlib.PackageUID && record.SDKIdentity == sdkstdlib.SDKIdentity && record.RuntimeABI == sdkstdlib.RuntimeABI && record.LocalPath == ""
	}
	if !found {
		t.Fatalf("SDK lock record missing: %+v", document.Packages)
	}
}

func writeWorkspaceFile(t *testing.T, root, relative, contents string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
}
