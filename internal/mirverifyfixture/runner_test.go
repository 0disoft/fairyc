package mirverifyfixture

import (
	"path/filepath"
	"testing"
)

func TestMIRVerifierManifestV0(t *testing.T) {
	t.Parallel()
	manifestPath := filepath.Join("..", "..", "spec", "fixtures", "mir-verifier", "manifest-v0.json")
	report, err := Run(manifestPath)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if report.SeedsRun != 10 {
		t.Fatalf("Run() seeds = %d, want 10", report.SeedsRun)
	}
	if report.CasesRun != 20 {
		t.Fatalf("Run() cases = %d, want 20", report.CasesRun)
	}
	for _, failure := range report.Failures {
		t.Errorf("%s: %s", failure.CaseID, failure.Detail)
	}
}
