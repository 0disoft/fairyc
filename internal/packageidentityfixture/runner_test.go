package packageidentityfixture

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunPackageIdentityManifest(t *testing.T) {
	t.Parallel()
	report, err := Run(filepath.Join("..", "..", "spec", "fixtures", "package-identity", "manifest-v1.json"))
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if report.CasesRun != 7 || len(report.Failures) != 0 {
		t.Fatalf("Run() report = %+v", report)
	}
}

func TestManifestDecodeRejectsUnknownAndTrailingData(t *testing.T) {
	t.Parallel()
	directory := t.TempDir()
	tests := []struct {
		name string
		body string
	}{
		{name: "unknown", body: `{"$schema":"schema","schemaVersion":"fairyc.package-identity-fixtures/v1","contracts":["contract"],"cases":[],"unknown":true}`},
		{name: "trailing", body: `{"$schema":"schema","schemaVersion":"fairyc.package-identity-fixtures/v1","contracts":["contract"],"cases":[]} {}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(directory, test.name+".json")
			if err := os.WriteFile(path, []byte(test.body), 0o600); err != nil {
				t.Fatalf("write fixture: %v", err)
			}
			_, err := loadManifest(path)
			if err == nil || !strings.Contains(err.Error(), "decode package identity manifest") {
				t.Fatalf("loadManifest() error = %v", err)
			}
		})
	}
}
