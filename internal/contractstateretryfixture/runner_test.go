package contractstateretryfixture

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestContractStateRetryManifestV0(t *testing.T) {
	t.Parallel()
	manifestPath := filepath.Join("..", "..", "spec", "fixtures", "contract-state-retry", "manifest-v0.json")
	report, err := Run(manifestPath)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if report.CasesRun != 2 {
		t.Fatalf("Run() cases = %d, want 2", report.CasesRun)
	}
	for _, failure := range report.Failures {
		t.Errorf("%s: %s", failure.CaseID, failure.Detail)
	}
}

func TestContractStateRetryManifestRejectsUnknownFields(t *testing.T) {
	t.Parallel()
	manifestPath := filepath.Join("..", "..", "spec", "fixtures", "contract-state-retry", "manifest-v0.json")
	contents, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	contents = []byte(strings.Replace(string(contents), `"schemaVersion":`, `"unknown": true, "schemaVersion":`, 1))
	path := filepath.Join(t.TempDir(), "manifest.json")
	if err := os.WriteFile(path, contents, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Run(path); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("Run() error = %v, want unknown-field rejection", err)
	}
}
