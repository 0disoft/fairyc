package contractstatefixture

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
)

func TestContractStateManifestV0(t *testing.T) {
	t.Parallel()
	manifestPath := filepath.Join("..", "..", "spec", "fixtures", "contract-state", "manifest-v0.json")
	report, err := Run(manifestPath)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if report.CasesRun != 5 {
		t.Fatalf("Run() cases = %d, want 5", report.CasesRun)
	}
	for _, failure := range report.Failures {
		t.Errorf("%s: %s", failure.CaseID, failure.Detail)
	}
}

func TestContractStateManifestV1(t *testing.T) {
	t.Parallel()
	manifestPath := filepath.Join("..", "..", "spec", "fixtures", "contract-state", "manifest-v1.json")
	report, err := Run(manifestPath)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if report.CasesRun != 8 {
		t.Fatalf("Run() cases = %d, want 8", report.CasesRun)
	}
	for _, failure := range report.Failures {
		t.Errorf("%s: %s", failure.CaseID, failure.Detail)
	}
}

func TestContractStateManifestV2(t *testing.T) {
	t.Parallel()
	manifestPath := filepath.Join("..", "..", "spec", "fixtures", "contract-state", "manifest-v2.json")
	report, err := Run(manifestPath)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if report.CasesRun != 6 {
		t.Fatalf("Run() cases = %d, want 6", report.CasesRun)
	}
	for _, failure := range report.Failures {
		t.Errorf("%s: %s", failure.CaseID, failure.Detail)
	}
}

func TestContractStateManifestV3(t *testing.T) {
	t.Parallel()
	manifestPath := filepath.Join("..", "..", "spec", "fixtures", "contract-state", "manifest-v3.json")
	report, err := Run(manifestPath)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if report.CasesRun != 4 {
		t.Fatalf("Run() cases = %d, want 4", report.CasesRun)
	}
	for _, failure := range report.Failures {
		t.Errorf("%s: %s", failure.CaseID, failure.Detail)
	}
}

func TestContractStateManifestV4(t *testing.T) {
	t.Parallel()
	manifestPath := filepath.Join("..", "..", "spec", "fixtures", "contract-state", "manifest-v4.json")
	report, err := Run(manifestPath)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if report.CasesRun != 5 {
		t.Fatalf("Run() cases = %d, want 5", report.CasesRun)
	}
	for _, failure := range report.Failures {
		t.Errorf("%s: %s", failure.CaseID, failure.Detail)
	}
}

func TestContractStateManifestRejectsUnknownFields(t *testing.T) {
	t.Parallel()
	manifestPath := filepath.Join("..", "..", "spec", "fixtures", "contract-state", "manifest-v0.json")
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

func TestContractStateManifestRejectsDuplicateIdentities(t *testing.T) {
	t.Parallel()
	definition := manifest{
		Schema: fixtureSchemaV1, SchemaVersion: fixtureSchemaV1, Contracts: []string{"contract"},
		Layout: layoutDefinition{ContractUID: "fixture", Fields: []fieldDefinition{
			{Name: "value", Kind: "cell", ValueTypeID: "u64/be/v1"},
			{Name: "value", Kind: "cell", ValueTypeID: "u64/be/v1"},
		}},
		Limits: limitsDefinition{MaxWrites: 1, MaxWriteBytes: 1, MaxValueBytes: 1, MaxMapKeyBytes: 1},
	}
	if err := validateManifest(definition); err == nil || !strings.Contains(err.Error(), "duplicated") {
		t.Fatalf("validateManifest() error = %v, want duplicate rejection", err)
	}
}

func TestContractStateManifestKeepsHelpersVersionGated(t *testing.T) {
	t.Parallel()
	definition := manifest{
		Schema: fixtureSchemaV2, SchemaVersion: fixtureSchemaV2, Contracts: []string{"contract"},
		Layout: layoutDefinition{ContractUID: "fixture", Fields: []fieldDefinition{{Name: "value", Kind: "cell", ValueTypeID: "u64/be/v1"}}},
		Limits: limitsDefinition{MaxWrites: 1, MaxWriteBytes: 1, MaxValueBytes: 1, MaxMapKeyBytes: 1},
		Cases:  []caseDefinition{{ID: "legacy", Operations: []operation{{Kind: "call", Callee: "helper", Cost: 1}}, Exit: "commit", Helpers: []helperDefinition{{Name: "helper", Operations: []operation{}, Exit: "return"}}, Expect: expectation{Status: "commit"}}},
	}
	if err := validateManifest(definition); err == nil || !strings.Contains(err.Error(), "before fixture schema v3") {
		t.Fatalf("validateManifest() error = %v, want helper version rejection", err)
	}
}

func TestCompilerPublicationRejectsFixtureDriftAtomically(t *testing.T) {
	t.Parallel()
	manifestPath := filepath.Join("..", "..", "spec", "fixtures", "contract-state", "manifest-v1.json")
	definition, err := loadManifest(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	input, err := buildHIRPackage(definition.Layout, definition.Cases[0])
	if err != nil {
		t.Fatal(err)
	}
	input.StateTransactions[0].Accesses[0].Value.Encoded = []byte{1}
	result, err := publishCompilerResult(input)
	var pipelineError *compiler.Error
	if !errors.As(err, &pipelineError) || pipelineError.Stage != compiler.StageStateMIRLower {
		t.Fatalf("publishCompilerResult() error = %v, want state MIR lower failure", err)
	}
	if result.Verified || len(result.MIR.Functions) != 0 || len(result.StateTransactions) != 0 {
		t.Fatalf("failed compiler result published executable IR: %+v", result)
	}
}
