package workspaceloaderfixture

import (
	"path/filepath"
	"testing"
)

func TestWorkspaceLoaderFixtures(t *testing.T) {
	t.Parallel()
	report, err := Run(filepath.Join("..", "..", "spec", "fixtures", "workspace-loader", "manifest-v1.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Failures) != 0 {
		t.Fatalf("fixture failures = %+v", report.Failures)
	}
}
