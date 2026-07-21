package packageidentityfixture

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/0disoft/fairyc/internal/packagegraph"
)

const schemaVersion = "fairyc.package-identity-fixtures/v1"

// Run loads and executes every PackageInstanceId fixture case.
func Run(path string) (Report, error) {
	definition, err := loadManifest(path)
	if err != nil {
		return Report{}, err
	}
	if definition.Schema == "" || definition.SchemaVersion != schemaVersion || len(definition.Contracts) == 0 || len(definition.Cases) == 0 {
		return Report{}, fmt.Errorf("invalid package identity fixture contract header")
	}

	report := Report{CasesRun: len(definition.Cases)}
	seen := make(map[string]struct{}, len(definition.Cases))
	for _, fixture := range definition.Cases {
		if fixture.ID == "" {
			return Report{}, fmt.Errorf("package identity fixture has an empty case id")
		}
		if _, duplicate := seen[fixture.ID]; duplicate {
			return Report{}, fmt.Errorf("duplicate package identity fixture %q", fixture.ID)
		}
		seen[fixture.ID] = struct{}{}
		report.Failures = append(report.Failures, runCase(fixture)...)
	}
	return report, nil
}

func loadManifest(path string) (manifest, error) {
	file, err := os.Open(path)
	if err != nil {
		return manifest{}, fmt.Errorf("open package identity manifest: %w", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	var definition manifest
	if err := decoder.Decode(&definition); err != nil {
		return manifest{}, fmt.Errorf("decode package identity manifest: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return manifest{}, fmt.Errorf("decode package identity manifest: multiple JSON values")
		}
		return manifest{}, fmt.Errorf("decode package identity manifest: %w", err)
	}
	return definition, nil
}

func runCase(fixture caseDefinition) []Failure {
	value := packagegraph.Identity{
		PackageUID:      fixture.Identity.PackageUID,
		SourceIdentity:  fixture.Identity.SourceIdentity,
		VersionIdentity: fixture.Identity.VersionIdentity,
		Features:        fixture.Identity.Features,
	}
	encoded, encodeErr := packagegraph.EncodeIdentity(value)
	id, idErr := packagegraph.ComputeInstanceID(value)

	switch fixture.Expect.Status {
	case "pass":
		if encodeErr != nil || idErr != nil {
			return []Failure{{CaseID: fixture.ID, Detail: fmt.Sprintf("pass case failed: encode=%v id=%v", encodeErr, idErr)}}
		}
		actualEncoding := hex.EncodeToString(encoded)
		actualID := id.String()
		var failures []Failure
		if actualEncoding != fixture.Expect.EncodingHex {
			failures = append(failures, Failure{CaseID: fixture.ID, Detail: fmt.Sprintf("encodingHex=%q want=%q", actualEncoding, fixture.Expect.EncodingHex)})
		}
		if actualID != fixture.Expect.InstanceID {
			failures = append(failures, Failure{CaseID: fixture.ID, Detail: fmt.Sprintf("instanceId=%q want=%q", actualID, fixture.Expect.InstanceID)})
		}
		return failures
	case "reject":
		if encodeErr == nil || idErr == nil {
			return []Failure{{CaseID: fixture.ID, Detail: "reject case produced an encoding or instance id"}}
		}
		var validationError *packagegraph.IdentityValidationError
		if !errors.As(encodeErr, &validationError) {
			return []Failure{{CaseID: fixture.ID, Detail: fmt.Sprintf("reject case returned non-validation error %v", encodeErr)}}
		}
		if validationError.Reason != fixture.Expect.Reason {
			return []Failure{{CaseID: fixture.ID, Detail: fmt.Sprintf("reason=%q want=%q", validationError.Reason, fixture.Expect.Reason)}}
		}
		return nil
	default:
		return []Failure{{CaseID: fixture.ID, Detail: fmt.Sprintf("unknown expectation status %q", fixture.Expect.Status)}}
	}
}
