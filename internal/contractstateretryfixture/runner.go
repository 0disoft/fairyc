package contractstateretryfixture

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/0disoft/fairyc/internal/contractstate"
)

const fixtureSchemaV0 = "fairyc.contract-state-retry-fixtures/v0"

const (
	statusCommit    = "commit"
	statusExhausted = "exhausted"
)

// Run executes caller-owned bounded loops. The contract-state kernel itself
// still performs one body evaluation and one commit attempt per Execute call.
func Run(manifestPath string) (Report, error) {
	absoluteManifest, err := filepath.Abs(manifestPath)
	if err != nil {
		return Report{}, fmt.Errorf("resolve manifest: %w", err)
	}
	definition, err := loadManifest(absoluteManifest)
	if err != nil {
		return Report{}, err
	}
	if err := validateManifest(definition); err != nil {
		return Report{}, err
	}

	report := Report{CasesRun: len(definition.Cases)}
	for _, fixture := range definition.Cases {
		if failure := runCase(definition, fixture); failure != nil {
			report.Failures = append(report.Failures, *failure)
		}
	}
	return report, nil
}

func loadManifest(path string) (manifest, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return manifest{}, fmt.Errorf("read manifest: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(contents))
	decoder.DisallowUnknownFields()
	var definition manifest
	if err := decoder.Decode(&definition); err != nil {
		return manifest{}, fmt.Errorf("decode manifest: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return manifest{}, fmt.Errorf("decode manifest: multiple JSON values")
		}
		return manifest{}, fmt.Errorf("decode manifest: %w", err)
	}
	return definition, nil
}

func validateManifest(definition manifest) error {
	if definition.Schema != "./schema-v0.json" || definition.SchemaVersion != fixtureSchemaV0 || len(definition.Contracts) == 0 {
		return fmt.Errorf("invalid contract-state retry fixture header")
	}
	if definition.Limits.MaxWrites <= 0 || definition.Limits.MaxWriteBytes == 0 || definition.Limits.MaxValueBytes <= 0 || definition.Limits.MaxMapKeyBytes <= 0 {
		return fmt.Errorf("contract-state retry fixture limits must be positive")
	}
	seen := make(map[string]struct{}, len(definition.Cases))
	for _, fixture := range definition.Cases {
		if fixture.ID == "" || fixture.MaxAttempts <= 0 || fixture.InjectedConflicts < 0 {
			return fmt.Errorf("contract-state retry fixture has an invalid case")
		}
		if _, duplicate := seen[fixture.ID]; duplicate {
			return fmt.Errorf("contract-state retry fixture case %q is duplicated", fixture.ID)
		}
		seen[fixture.ID] = struct{}{}
		if fixture.Expect.Status != statusCommit && fixture.Expect.Status != statusExhausted {
			return fmt.Errorf("contract-state retry fixture case %q has invalid status %q", fixture.ID, fixture.Expect.Status)
		}
		if fixture.Expect.Attempts <= 0 || fixture.Expect.Attempts > fixture.MaxAttempts {
			return fmt.Errorf("contract-state retry fixture case %q has invalid expected attempts", fixture.ID)
		}
	}
	if len(definition.Cases) == 0 {
		return fmt.Errorf("contract-state retry fixture has no cases")
	}
	return nil
}

func runCase(definition manifest, fixture caseDefinition) *Failure {
	builder, err := contractstate.NewLayoutBuilder("example.explicit-retry." + fixture.ID)
	if err != nil {
		return failure(fixture.ID, "build layout: %v", err)
	}
	counter, err := contractstate.DeclareCell(builder, "counter", contractstate.U64)
	if err != nil {
		return failure(fixture.ID, "declare counter: %v", err)
	}
	marker, err := contractstate.DeclareCell(builder, "marker", contractstate.U64)
	if err != nil {
		return failure(fixture.ID, "declare marker: %v", err)
	}
	layout, err := builder.Build()
	if err != nil {
		return failure(fixture.ID, "seal layout: %v", err)
	}
	store, err := contractstate.NewMemoryStore(layout)
	if err != nil {
		return failure(fixture.ID, "construct store: %v", err)
	}
	limits := contractstate.Limits{
		MaxWrites:      definition.Limits.MaxWrites,
		MaxWriteBytes:  definition.Limits.MaxWriteBytes,
		MaxValueBytes:  definition.Limits.MaxValueBytes,
		MaxMapKeyBytes: definition.Limits.MaxMapKeyBytes,
	}

	attempts := 0
	status := statusExhausted
	for attempts < fixture.MaxAttempts {
		attempts++
		attemptNumber := attempts
		_, err := contractstate.Execute(store, layout, limits, func(transaction *contractstate.Transaction) error {
			value, found, err := counter.Get(transaction)
			if err != nil {
				return err
			}
			if !found {
				value = 0
			}
			if err := counter.Set(transaction, value+1); err != nil {
				return err
			}
			if err := marker.Set(transaction, value+1); err != nil {
				return err
			}
			if attemptNumber > fixture.InjectedConflicts {
				return nil
			}

			competing, err := store.Begin(layout, limits)
			if err != nil {
				return err
			}
			competingValue := uint64(attemptNumber * 10)
			if err := counter.Set(competing, competingValue); err != nil {
				return err
			}
			if err := marker.Set(competing, competingValue); err != nil {
				return err
			}
			_, err = competing.Commit()
			return err
		})
		if err == nil {
			status = statusCommit
			break
		}
		if !errors.Is(err, contractstate.ErrConflict) {
			return failure(fixture.ID, "attempt %d error = %v, want ErrConflict or success", attempts, err)
		}
	}

	if status != fixture.Expect.Status || attempts != fixture.Expect.Attempts {
		return failure(
			fixture.ID,
			"result = %s after %d attempts, want %s after %d",
			status,
			attempts,
			fixture.Expect.Status,
			fixture.Expect.Attempts,
		)
	}
	if store.Version() != fixture.Expect.Version {
		return failure(fixture.ID, "version = %d, want %d", store.Version(), fixture.Expect.Version)
	}
	if err := contractstate.View(store, layout, limits, func(transaction *contractstate.Transaction) error {
		counterValue, counterFound, err := counter.Get(transaction)
		if err != nil {
			return err
		}
		markerValue, markerFound, err := marker.Get(transaction)
		if err != nil {
			return err
		}
		if !counterFound || !markerFound {
			return fmt.Errorf("final overlay is incomplete: counter=%v marker=%v", counterFound, markerFound)
		}
		if counterValue != fixture.Expect.Value || markerValue != fixture.Expect.Value {
			return fmt.Errorf(
				"final overlay is mixed: counter=%d marker=%d want=%d",
				counterValue,
				markerValue,
				fixture.Expect.Value,
			)
		}
		return nil
	}); err != nil {
		return failure(fixture.ID, "fresh view: %v", err)
	}
	return nil
}

func failure(caseID, format string, arguments ...any) *Failure {
	return &Failure{CaseID: caseID, Detail: fmt.Sprintf(format, arguments...)}
}
