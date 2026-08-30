package lockfile

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/sdkstdlib"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

func TestWorkspaceLockRoundTripAndAtomicReplacement(t *testing.T) {
	t.Parallel()
	workspace, err := workspaceloader.Load(filepath.Join("..", "..", "spec", "fixtures", "workspace-loader", "cases", "pass-feature-distinct"))
	if err != nil {
		t.Fatal(err)
	}
	document, err := FromWorkspace(workspace)
	if err != nil {
		t.Fatal(err)
	}
	contents, err := Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := Parse(contents)
	if err != nil {
		t.Fatal(err)
	}
	if err := MatchesWorkspace(parsed, workspace); err != nil {
		t.Fatal(err)
	}
	if !bytes.HasSuffix(contents, []byte("\n")) || !bytes.Contains(contents, []byte(`"schema": "fairyc.lock/v1"`)) {
		t.Fatalf("noncanonical lock bytes: %q", contents)
	}
	if bytes.Contains(contents, []byte(`"features": null`)) || bytes.Contains(contents, []byte(`"dependencies": null`)) {
		t.Fatalf("canonical collections must be arrays: %q", contents)
	}
	path := filepath.Join(t.TempDir(), FileName)
	if err := WriteAtomic(path, contents); err != nil {
		t.Fatal(err)
	}
	if err := WriteAtomic(path, contents); err != nil {
		t.Fatalf("replace existing lockfile: %v", err)
	}
	actual, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(actual, contents) {
		t.Fatalf("written lock = %q, error = %v", actual, err)
	}
}

func TestLockRejectsContradictionsAndStaleness(t *testing.T) {
	t.Parallel()
	workspace, err := workspaceloader.Load(filepath.Join("..", "..", "spec", "fixtures", "workspace-loader", "cases", "pass-feature-distinct"))
	if err != nil {
		t.Fatal(err)
	}
	document, err := FromWorkspace(workspace)
	if err != nil {
		t.Fatal(err)
	}
	t.Run("identity", func(t *testing.T) {
		mutated := document
		mutated.Packages = append([]PackageRecord(nil), document.Packages...)
		mutated.Packages[0].Version = "9.0.0"
		if err := Validate(mutated); err == nil || !strings.Contains(err.Error(), "contradictory identity") {
			t.Fatalf("Validate() error = %v", err)
		}
	})
	t.Run("source", func(t *testing.T) {
		mutated := document
		mutated.Packages = append([]PackageRecord(nil), document.Packages...)
		mutated.Packages[0].SourceDigest = document.Packages[1].SourceDigest
		if err := Validate(mutated); err == nil || !strings.Contains(err.Error(), "source identity") {
			t.Fatalf("Validate() error = %v", err)
		}
	})
	t.Run("stale", func(t *testing.T) {
		mutated := document
		mutated.Language = "fairyc-language/119"
		if err := MatchesWorkspace(mutated, workspace); err == nil || !strings.Contains(err.Error(), "stale") {
			t.Fatalf("MatchesWorkspace() error = %v", err)
		}
	})
	t.Run("duplicate json", func(t *testing.T) {
		contents, err := Marshal(document)
		if err != nil {
			t.Fatal(err)
		}
		duplicate := bytes.Replace(contents, []byte(`"schema": "fairyc.lock/v1"`), []byte(`"schema": "fairyc.lock/v1", "schema": "fairyc.lock/v1"`), 1)
		if _, err := Parse(duplicate); err == nil || !strings.Contains(err.Error(), "duplicate JSON key") {
			t.Fatalf("Parse() error = %v", err)
		}
	})
	t.Run("unknown json", func(t *testing.T) {
		contents, err := Marshal(document)
		if err != nil {
			t.Fatal(err)
		}
		unknown := bytes.Replace(contents, []byte("{\n"), []byte("{\n  \"unknown\": true,\n"), 1)
		if _, err := Parse(unknown); err == nil || !strings.Contains(err.Error(), "unknown field") {
			t.Fatalf("Parse() error = %v", err)
		}
	})
}

func TestLockRecordsAndValidatesExactSDKOrigin(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeSDKWorkspace(t, root)
	workspace, err := workspaceloader.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	document, err := FromWorkspace(workspace)
	if err != nil {
		t.Fatal(err)
	}
	var sdkIndex = -1
	for index, record := range document.Packages {
		if record.SourceKind == workspaceloader.SourceKindSDK {
			sdkIndex = index
		}
	}
	if sdkIndex < 0 {
		t.Fatalf("lock packages = %+v", document.Packages)
	}
	record := document.Packages[sdkIndex]
	if record.Role != packagegraph.RoleStandardLibrary || record.PackageUID != sdkstdlib.PackageUID ||
		record.LocalPath != "" || record.SDKIdentity != sdkstdlib.SDKIdentity || record.RuntimeABI != sdkstdlib.RuntimeABI ||
		record.Version != sdkstdlib.Version || len(record.Features) != 0 {
		t.Fatalf("SDK lock record = %+v", record)
	}
	contents, err := Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := Parse(contents)
	if err != nil {
		t.Fatal(err)
	}
	if err := MatchesWorkspace(parsed, workspace); err != nil {
		t.Fatal(err)
	}

	for _, mutation := range []struct {
		name     string
		contains string
		change   func(*PackageRecord)
	}{
		{name: "sdk identity", contains: "source origin", change: func(record *PackageRecord) { record.SDKIdentity = "fairyc-sdk-stdlib/99" }},
		{name: "runtime abi", contains: "source origin", change: func(record *PackageRecord) { record.RuntimeABI = "fairyc-runtime/99" }},
		{name: "local path", contains: "source origin", change: func(record *PackageRecord) { record.LocalPath = "vendor/stdlib" }},
		{name: "source kind", contains: "invalid source kind", change: func(record *PackageRecord) { record.SourceKind = "registry" }},
	} {
		mutation := mutation
		t.Run(mutation.name, func(t *testing.T) {
			mutated := document
			mutated.Packages = append([]PackageRecord(nil), document.Packages...)
			mutation.change(&mutated.Packages[sdkIndex])
			if err := Validate(mutated); err == nil || !strings.Contains(err.Error(), mutation.contains) {
				t.Fatalf("Validate() error = %v", err)
			}
		})
	}
}

func writeSDKWorkspace(t *testing.T, root string) {
	t.Helper()
	manifest := `schema = "fairyc.manifest/v1"
[package]
uid = "example.app"
name = "app"
version = "1.0.0"
language = "fairyc-language/12"
source = "src"
`
	source := `module main;

import std::alloc as alloc;

fn i32 inspect(ref mut alloc::Arena arena) {
    return 1;
}
`
	if err := os.MkdirAll(filepath.Join(root, "src"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "fairy.toml"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "src", "main.fy"), []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
}
