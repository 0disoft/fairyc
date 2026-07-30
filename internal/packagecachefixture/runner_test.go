package packagecachefixture

import (
	"path/filepath"
	"testing"
)

func TestRunPackageCacheManifest(t *testing.T) {
	t.Parallel()
	report, err := Run(filepath.Join("..", "..", "spec", "fixtures", "package-cache", "manifest-v4.json"))
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if report.CasesRun != 8 || len(report.Failures) != 0 {
		t.Fatalf("Run() report = %+v", report)
	}
}
