package cabifixture

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cabi"
	"github.com/0disoft/fairyc/internal/target"
)

func TestRepositoryCABIManifestPasses(t *testing.T) {
	t.Parallel()
	manifest := filepath.Join("..", "..", "spec", "fixtures", "c-abi", "manifest-v0.json")
	report, err := Run(manifest)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if report.Target != target.X8664LinuxGNU || report.CasesRun != 17 || report.Passed != 17 || len(report.Failures) != 0 {
		t.Fatalf("Run() report = %+v", report)
	}
	targetID, cases, err := NativeProbeCases(manifest)
	if err != nil {
		t.Fatalf("NativeProbeCases() error = %v", err)
	}
	if targetID != target.X8664LinuxGNU || len(cases) != 14 || cases[0].ID != "pass.aggregate-rollback" || cases[len(cases)-1].ID != "pass.void-return" {
		t.Fatalf("NativeProbeCases() = %q, %+v", targetID, cases)
	}
	if cases[0].Plan.Arguments[5].Kind != cabi.LocationStack || cases[0].Plan.Arguments[6].Registers[0] != "r9" {
		t.Fatalf("aggregate rollback probe = %+v", cases[0])
	}
	cases[0].Signature.Parameters[0].Name = "mutated"
	cases[0].Plan.Arguments[0].Classification.Classes[0] = cabi.ClassMemory
	_, repeated, err := NativeProbeCases(manifest)
	if err != nil || repeated[0].Signature.Parameters[0].Name != "i64" || repeated[0].Plan.Arguments[0].Classification.Classes[0] != cabi.ClassInteger {
		t.Fatalf("NativeProbeCases() leaked caller mutation: %v, %+v", err, repeated[0])
	}
}

func TestManifestRejectsUnknownFieldsAndTrailingData(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	for _, test := range []struct {
		name string
		body string
	}{
		{name: "unknown", body: `{"schemaVersion":"fairyc.c-abi-fixtures/v0","target":"x86_64-linux-gnu","contracts":["contract"],"deferred":["deferred"],"cases":[],"unknown":true}`},
		{name: "trailing", body: `{"schemaVersion":"fairyc.c-abi-fixtures/v0","target":"x86_64-linux-gnu","contracts":["contract"],"deferred":["deferred"],"cases":[]} {}`},
	} {
		test := test
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(root, test.name+".json")
			if err := os.WriteFile(path, []byte(test.body), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := Run(path); err == nil || !strings.Contains(err.Error(), "decode C ABI manifest") {
				t.Fatalf("Run() error = %v", err)
			}
		})
	}
}
