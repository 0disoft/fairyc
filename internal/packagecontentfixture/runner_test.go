package packagecontentfixture

import (
	"path/filepath"
	"testing"
)

func TestRunPackageSourceManifest(t *testing.T) {
	t.Parallel()
	report, err := Run(filepath.Join("..", "..", "spec", "fixtures", "package-source", "manifest-v1.json"))
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if report.CasesRun != 5 || len(report.Failures) != 0 {
		t.Fatalf("Run() report = %+v", report)
	}
}
