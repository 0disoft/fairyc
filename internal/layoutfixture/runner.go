// Package layoutfixture executes the versioned target-layout fixture contract.
package layoutfixture

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"

	"github.com/0disoft/fairyc/internal/layout"
	"github.com/0disoft/fairyc/internal/target"
)

const (
	fixtureSchema    = "fairyc.layout-fixtures/v0"
	maxManifestBytes = 1024 * 1024
)

// Failure is one case-local contract mismatch.
type Failure struct {
	CaseID string
	Detail string
}

// Report summarizes one complete manifest execution.
type Report struct {
	Target   target.ID
	CasesRun int
	Passed   int
	Failures []Failure
}

type manifest struct {
	Schema        string        `json:"$schema"`
	SchemaVersion string        `json:"schemaVersion"`
	Target        string        `json:"target"`
	Contracts     []string      `json:"contracts"`
	Deferred      []string      `json:"deferred"`
	Cases         []fixtureCase `json:"cases"`
}

type fixtureCase struct {
	ID     string      `json:"id"`
	Type   layout.Type `json:"type"`
	Expect expectation `json:"expect"`
}

type expectation struct {
	Status    string           `json:"status"`
	Layout    *layout.Result   `json:"layout,omitempty"`
	ErrorCode layout.ErrorCode `json:"errorCode,omitempty"`
}

// Run decodes, semantically validates, and executes every layout case in manifestPath.
func Run(manifestPath string) (Report, error) {
	definition, err := loadManifest(manifestPath)
	if err != nil {
		return Report{}, err
	}
	targetID := target.ID(definition.Target)
	spec, found := target.Lookup(targetID)
	if !found {
		return Report{}, fmt.Errorf("layout fixture target %q is unknown", definition.Target)
	}
	engine, err := layout.New(spec)
	if err != nil {
		return Report{}, err
	}
	report := Report{Target: targetID, CasesRun: len(definition.Cases), Failures: []Failure{}}
	seen := make(map[string]struct{}, len(definition.Cases))
	previousID := ""
	for _, fixture := range definition.Cases {
		if fixture.ID == "" {
			report.Failures = append(report.Failures, Failure{Detail: "case id is empty"})
			continue
		}
		if _, duplicate := seen[fixture.ID]; duplicate {
			report.Failures = append(report.Failures, Failure{CaseID: fixture.ID, Detail: "duplicate case id"})
			continue
		}
		seen[fixture.ID] = struct{}{}
		if previousID != "" && previousID >= fixture.ID {
			report.Failures = append(report.Failures, Failure{CaseID: fixture.ID, Detail: "case ids are not strictly sorted"})
			continue
		}
		previousID = fixture.ID
		if failure := executeCase(engine, fixture); failure != nil {
			report.Failures = append(report.Failures, *failure)
			continue
		}
		report.Passed++
	}
	return report, nil
}

func executeCase(engine *layout.Engine, fixture fixtureCase) *Failure {
	if !strings.HasPrefix(fixture.ID, fixture.Expect.Status+".") {
		return &Failure{CaseID: fixture.ID, Detail: "case id prefix does not match expectation status"}
	}
	actual, err := engine.Of(fixture.Type)
	switch fixture.Expect.Status {
	case "pass":
		if fixture.Expect.Layout == nil || fixture.Expect.ErrorCode != "" {
			return &Failure{CaseID: fixture.ID, Detail: "pass expectation must contain only layout"}
		}
		if err != nil {
			return &Failure{CaseID: fixture.ID, Detail: fmt.Sprintf("layout failed: %v", err)}
		}
		if !equalResult(actual, *fixture.Expect.Layout) {
			return &Failure{CaseID: fixture.ID, Detail: fmt.Sprintf("layout mismatch: got %+v, want %+v", actual, *fixture.Expect.Layout)}
		}
	case "fail":
		if fixture.Expect.Layout != nil || fixture.Expect.ErrorCode == "" {
			return &Failure{CaseID: fixture.ID, Detail: "fail expectation must contain only errorCode"}
		}
		if err == nil {
			return &Failure{CaseID: fixture.ID, Detail: fmt.Sprintf("layout succeeded unexpectedly: %+v", actual)}
		}
		code, ok := layout.Code(err)
		if !ok || code != fixture.Expect.ErrorCode {
			return &Failure{CaseID: fixture.ID, Detail: fmt.Sprintf("error code mismatch: got %q from %v, want %q", code, err, fixture.Expect.ErrorCode)}
		}
	default:
		return &Failure{CaseID: fixture.ID, Detail: fmt.Sprintf("unknown expectation status %q", fixture.Expect.Status)}
	}
	return nil
}

func equalResult(left, right layout.Result) bool {
	return left.SizeBytes == right.SizeBytes &&
		left.AlignmentBytes == right.AlignmentBytes &&
		slices.Equal(left.Fields, right.Fields) &&
		slices.Equal(left.Padding, right.Padding)
}

func loadManifest(path string) (manifest, error) {
	file, err := os.Open(path)
	if err != nil {
		return manifest{}, fmt.Errorf("open layout manifest: %w", err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return manifest{}, fmt.Errorf("inspect layout manifest: %w", err)
	}
	if !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > maxManifestBytes {
		return manifest{}, fmt.Errorf("layout manifest has invalid size %d", info.Size())
	}
	contents, err := io.ReadAll(io.LimitReader(file, maxManifestBytes+1))
	if err != nil {
		return manifest{}, fmt.Errorf("read layout manifest: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(contents))
	decoder.DisallowUnknownFields()
	var value manifest
	if err := decoder.Decode(&value); err != nil {
		return manifest{}, fmt.Errorf("decode layout manifest: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return manifest{}, fmt.Errorf("decode layout manifest trailing data")
	}
	if value.Schema != "./schema-v0.json" {
		return manifest{}, fmt.Errorf("layout manifest schema reference is %q, want %q", value.Schema, "./schema-v0.json")
	}
	if value.SchemaVersion != fixtureSchema {
		return manifest{}, fmt.Errorf("layout manifest schema is %q, want %q", value.SchemaVersion, fixtureSchema)
	}
	if strings.TrimSpace(value.Target) == "" || value.Target != strings.TrimSpace(value.Target) || len(value.Contracts) == 0 || len(value.Deferred) == 0 || len(value.Cases) == 0 {
		return manifest{}, fmt.Errorf("layout manifest is missing target, contracts, deferred coverage, or cases")
	}
	if err := validateStringList("contract", value.Contracts); err != nil {
		return manifest{}, err
	}
	if err := validateStringList("deferred coverage", value.Deferred); err != nil {
		return manifest{}, err
	}
	return value, nil
}

func validateStringList(name string, values []string) error {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) == "" || value != strings.TrimSpace(value) {
			return fmt.Errorf("layout manifest %s contains an empty or padded value", name)
		}
		if _, duplicate := seen[value]; duplicate {
			return fmt.Errorf("layout manifest %s %q is repeated", name, value)
		}
		seen[value] = struct{}{}
	}
	return nil
}
