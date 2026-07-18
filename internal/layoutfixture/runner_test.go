package layoutfixture

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/target"
)

func TestRepositoryLayoutManifestPasses(t *testing.T) {
	t.Parallel()
	report, err := Run(filepath.Join("..", "..", "spec", "fixtures", "layout", "manifest-v0.json"))
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if report.Target != target.X8664LinuxGNU || report.CasesRun != 14 || report.Passed != 14 || len(report.Failures) != 0 {
		t.Fatalf("Run() report = %+v", report)
	}
}

func TestManifestRejectsUnknownFieldsAndTrailingData(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	for _, test := range []struct {
		name string
		body string
	}{
		{name: "unknown", body: `{"schemaVersion":"fairyc.layout-fixtures/v0","target":"x86_64-linux-gnu","contracts":["contract"],"deferred":["deferred"],"cases":[],"unknown":true}`},
		{name: "trailing", body: `{"schemaVersion":"fairyc.layout-fixtures/v0","target":"x86_64-linux-gnu","contracts":["contract"],"deferred":["deferred"],"cases":[]} {}`},
	} {
		test := test
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(root, test.name+".json")
			if err := os.WriteFile(path, []byte(test.body), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := Run(path); err == nil || !strings.Contains(err.Error(), "decode layout manifest") {
				t.Fatalf("Run() error = %v", err)
			}
		})
	}
}
