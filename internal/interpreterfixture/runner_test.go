package interpreterfixture

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestInterpreterSchemaV0Header(t *testing.T) {
	t.Parallel()
	path := filepath.Join("..", "..", "spec", "fixtures", "interpreter", "schema-v0.json")
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var schema struct {
		Dialect string `json:"$schema"`
		ID      string `json:"$id"`
	}
	if err := json.Unmarshal(contents, &schema); err != nil {
		t.Fatalf("schema is not JSON: %v", err)
	}
	if schema.Dialect != "https://json-schema.org/draft/2020-12/schema" || schema.ID != "urn:fairyc:interpreter-fixtures:v0" {
		t.Fatalf("schema header = %+v", schema)
	}
}

func TestInterpreterManifestV0(t *testing.T) {
	t.Parallel()
	manifestPath := filepath.Join("..", "..", "spec", "fixtures", "interpreter", "manifest-v0.json")
	report, err := Run(manifestPath)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if report.CasesRun != 87 {
		t.Fatalf("Run() cases = %d, want 87", report.CasesRun)
	}
	for _, failure := range report.Failures {
		t.Errorf("%s: %s", failure.CaseID, failure.Detail)
	}
}

func TestResolveWithinRejectsEscape(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if _, err := resolveWithin(root, root, "../escape.fy"); err == nil {
		t.Fatal("resolveWithin() accepted a parent traversal")
	}
}

func TestResolveWithinRejectsSymlinkEscape(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "link")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if _, err := resolveWithin(root, root, "link"); err == nil {
		t.Fatal("resolveWithin() accepted a symlink outside the fixture boundary")
	}
}
