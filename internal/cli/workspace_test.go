package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

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
language = "fairyc-language/0"
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
language = "fairyc-language/0"
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
language = "fairyc-language/0"
source = "src"
`)
	writeWorkspaceFile(t, root, "src/main.fy", "module main;\n\nimport std::alloc as alloc;\n\nfn i32 inspect(ref mut alloc::Arena arena) {\n    return 1;\n}\n")

	code, stdout, stderr := runForTest([]string{"experimental", "check-workspace"}, root)
	if code != ExitSuccess || !strings.Contains(stdout, "2 instance(s), 2 source file(s)") || !strings.HasSuffix(stdout, "lock updated\n") || stderr != "" {
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
