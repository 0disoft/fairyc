package resourcefixture

import (
	"path/filepath"
	"testing"
)

func TestResourceManifestV0(t *testing.T) {
	t.Parallel()
	manifestPath := filepath.Join("..", "..", "spec", "fixtures", "resource", "manifest-v0.json")
	report, err := Run(manifestPath)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if report.CasesRun != 10 {
		t.Fatalf("Run() cases = %d, want 10", report.CasesRun)
	}
	for _, failure := range report.Failures {
		t.Errorf("%s: %s", failure.CaseID, failure.Detail)
	}
}

func TestContainedPathRejectsEscape(t *testing.T) {
	t.Parallel()
	if _, err := containedPath(t.TempDir(), "../escape.fy"); err == nil {
		t.Fatal("containedPath() accepted a parent traversal")
	}
}
