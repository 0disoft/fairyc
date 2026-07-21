package lockfile

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

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
		mutated.Language = "fairyc-language/99"
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
