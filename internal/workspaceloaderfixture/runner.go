// Package workspaceloaderfixture executes versioned local workspace fixtures.
package workspaceloaderfixture

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/0disoft/fairyc/internal/workspaceloader"
)

const (
	schemaVersion    = "fairyc.workspace-loader-fixtures/v1"
	maxManifestBytes = 1 << 20
)

type manifest struct {
	Schema        string           `json:"$schema"`
	SchemaVersion string           `json:"schemaVersion"`
	Contracts     []string         `json:"contracts"`
	Cases         []caseDefinition `json:"cases"`
}

type caseDefinition struct {
	ID       string      `json:"id"`
	Root     string      `json:"root"`
	Expected expectation `json:"expect"`
}

type expectation struct {
	Status       string `json:"status"`
	Reason       string `json:"reason,omitempty"`
	PackageCount int    `json:"packageCount,omitempty"`
	NodeCount    int    `json:"nodeCount,omitempty"`
	EdgeCount    int    `json:"edgeCount,omitempty"`
}

type Failure struct {
	CaseID string
	Detail string
}

type Report struct {
	CasesRun int
	Failures []Failure
}

// Run strictly loads and executes every declared complete-workspace fixture.
func Run(path string) (Report, error) {
	definition, err := loadManifest(path)
	if err != nil {
		return Report{}, err
	}
	if definition.Schema != "./schema-v1.json" || definition.SchemaVersion != schemaVersion || len(definition.Contracts) == 0 || len(definition.Cases) == 0 {
		return Report{}, fmt.Errorf("invalid workspace loader fixture contract header")
	}
	root := filepath.Dir(path)
	seen := make(map[string]struct{}, len(definition.Cases))
	report := Report{CasesRun: len(definition.Cases)}
	for _, fixture := range definition.Cases {
		if fixture.ID == "" || !safeRelative(fixture.Root) {
			return Report{}, fmt.Errorf("workspace loader fixture has invalid id or root")
		}
		if _, duplicate := seen[fixture.ID]; duplicate {
			return Report{}, fmt.Errorf("duplicate workspace loader fixture %q", fixture.ID)
		}
		seen[fixture.ID] = struct{}{}
		actual, loadErr := workspaceloader.Load(filepath.Join(root, filepath.FromSlash(fixture.Root)))
		switch fixture.Expected.Status {
		case "pass":
			if loadErr != nil {
				report.Failures = append(report.Failures, Failure{CaseID: fixture.ID, Detail: fmt.Sprintf("unexpected rejection: %v", loadErr)})
				continue
			}
			if fixture.Expected.Reason != "" || len(actual.Packages) != fixture.Expected.PackageCount || len(actual.Graph.Nodes) != fixture.Expected.NodeCount || len(actual.Graph.Dependencies) != fixture.Expected.EdgeCount {
				report.Failures = append(report.Failures, Failure{CaseID: fixture.ID, Detail: fmt.Sprintf("counts packages=%d nodes=%d edges=%d expectation=%+v", len(actual.Packages), len(actual.Graph.Nodes), len(actual.Graph.Dependencies), fixture.Expected)})
			}
		case "reject":
			var validation *workspaceloader.LoadError
			if !errors.As(loadErr, &validation) || validation.Reason != fixture.Expected.Reason {
				report.Failures = append(report.Failures, Failure{CaseID: fixture.ID, Detail: fmt.Sprintf("rejection=%#v expectation=%+v", loadErr, fixture.Expected)})
			}
		default:
			return Report{}, fmt.Errorf("workspace loader fixture %q has unknown status %q", fixture.ID, fixture.Expected.Status)
		}
	}
	return report, nil
}

func loadManifest(path string) (manifest, error) {
	file, err := os.Open(path)
	if err != nil {
		return manifest{}, fmt.Errorf("open workspace loader fixture: %w", err)
	}
	defer file.Close()
	contents, err := io.ReadAll(io.LimitReader(file, maxManifestBytes+1))
	if err != nil {
		return manifest{}, fmt.Errorf("read workspace loader fixture: %w", err)
	}
	if len(contents) > maxManifestBytes {
		return manifest{}, fmt.Errorf("workspace loader fixture exceeds byte limit")
	}
	decoder := json.NewDecoder(bytes.NewReader(contents))
	decoder.DisallowUnknownFields()
	var definition manifest
	if err := decoder.Decode(&definition); err != nil {
		return manifest{}, fmt.Errorf("decode workspace loader fixture: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return manifest{}, fmt.Errorf("decode workspace loader fixture: multiple JSON values")
		}
		return manifest{}, fmt.Errorf("decode workspace loader fixture: %w", err)
	}
	return definition, nil
}

func safeRelative(value string) bool {
	return value != "" && !filepath.IsAbs(value) && filepath.VolumeName(value) == "" && !strings.Contains(value, "\\") && filepath.ToSlash(filepath.Clean(value)) == value && value != ".." && !strings.HasPrefix(value, "../")
}
