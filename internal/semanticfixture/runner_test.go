package semanticfixture

import (
	"path/filepath"
	"testing"
)

func TestSemanticManifestV0(t *testing.T) {
	t.Parallel()
	manifestPath := filepath.Join("..", "..", "spec", "fixtures", "semantic", "manifest-v0.json")
	report, err := Run(manifestPath)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if report.CasesParsed != 23 || report.CasesRun != 23 || len(report.Deferred) != 0 {
		t.Fatalf("Run() report = %+v, want parsed=23 run=23 deferred=0", report)
	}
	if len(report.Failures) != 0 {
		for _, failure := range report.Failures {
			t.Errorf("%s: %s", failure.CaseID, failure.Detail)
		}
	}
}

func TestContainedPathRejectsEscape(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if _, err := containedPath(root, "../escape.fy"); err == nil {
		t.Fatal("containedPath() accepted a parent traversal")
	}
}
