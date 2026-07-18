package cbackendfixture

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestCEmissionSchemaV0Header(t *testing.T) {
	t.Parallel()
	path := filepath.Join("..", "..", "spec", "fixtures", "c-emission", "schema-v0.json")
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
	if schema.Dialect != "https://json-schema.org/draft/2020-12/schema" || schema.ID != "urn:fairyc:c-emission-fixtures:v0" {
		t.Fatalf("schema header = %+v", schema)
	}
}

func TestCEmissionManifestV0(t *testing.T) {
	t.Parallel()
	manifestPath := filepath.Join("..", "..", "spec", "fixtures", "c-emission", "manifest-v0.json")
	report, err := Run(manifestPath)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if report.CasesRun != 4 {
		t.Fatalf("Run() cases = %d, want 4", report.CasesRun)
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
