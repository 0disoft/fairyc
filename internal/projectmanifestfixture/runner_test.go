package projectmanifestfixture

import (
	"path/filepath"
	"testing"
)

func TestRunProjectManifestV1(t *testing.T) {
	t.Parallel()
	report, err := Run(filepath.Join("..", "..", "spec", "fixtures", "project-manifest", "manifest-v1.json"))
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if report.CasesRun != 8 || len(report.Failures) != 0 {
		t.Fatalf("Run() report = %+v", report)
	}
}
