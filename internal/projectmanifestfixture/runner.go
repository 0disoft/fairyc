package projectmanifestfixture

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/0disoft/fairyc/internal/projectmanifest"
)

const (
	schemaVersion    = "fairyc.project-manifest-fixtures/v1"
	maxManifestBytes = 1 << 20
)

// Run strictly loads and executes every project-manifest fixture.
func Run(path string) (Report, error) {
	definition, err := loadManifest(path)
	if err != nil {
		return Report{}, err
	}
	if definition.Schema == "" || definition.SchemaVersion != schemaVersion || len(definition.Contracts) == 0 || len(definition.Cases) == 0 {
		return Report{}, fmt.Errorf("invalid project manifest fixture contract header")
	}
	report := Report{CasesRun: len(definition.Cases)}
	seen := make(map[string]struct{}, len(definition.Cases))
	root := filepath.Dir(path)
	for _, fixture := range definition.Cases {
		if fixture.ID == "" || fixture.File == "" {
			return Report{}, fmt.Errorf("project manifest fixture has an empty id or file")
		}
		if _, duplicate := seen[fixture.ID]; duplicate {
			return Report{}, fmt.Errorf("duplicate project manifest fixture %q", fixture.ID)
		}
		seen[fixture.ID] = struct{}{}
		if filepath.IsAbs(fixture.File) || filepath.VolumeName(fixture.File) != "" || strings.Contains(fixture.File, "\\") || filepath.ToSlash(filepath.Clean(fixture.File)) != fixture.File || strings.HasPrefix(fixture.File, "../") {
			return Report{}, fmt.Errorf("project manifest fixture %q has unsafe file %q", fixture.ID, fixture.File)
		}
		contents, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(fixture.File)))
		if err != nil {
			report.Failures = append(report.Failures, Failure{CaseID: fixture.ID, Detail: fmt.Sprintf("read fixture: %v", err)})
			continue
		}
		actual, parseErr := projectmanifest.Parse(contents)
		switch fixture.Expect.Status {
		case "pass":
			if parseErr != nil {
				report.Failures = append(report.Failures, Failure{CaseID: fixture.ID, Detail: fmt.Sprintf("unexpected rejection: %v", parseErr)})
				continue
			}
			bindings := make([]string, len(actual.Dependencies))
			for index, dependency := range actual.Dependencies {
				bindings[index] = dependency.Binding
			}
			if fixture.Expect.Reason != "" || fixture.Expect.Field != "" || actual.Package.UID != fixture.Expect.PackageUID || !slices.Equal(actual.Workspace.Members, fixture.Expect.Members) || !slices.Equal(bindings, fixture.Expect.Dependencies) {
				report.Failures = append(report.Failures, Failure{CaseID: fixture.ID, Detail: fmt.Sprintf("manifest=%+v bindings=%v expectation=%+v", actual, bindings, fixture.Expect)})
			}
		case "reject":
			var validation *projectmanifest.ValidationError
			if !errors.As(parseErr, &validation) || validation.Reason != fixture.Expect.Reason || validation.Field != fixture.Expect.Field {
				report.Failures = append(report.Failures, Failure{CaseID: fixture.ID, Detail: fmt.Sprintf("rejection=%#v expectation=%+v", parseErr, fixture.Expect)})
			}
		default:
			return Report{}, fmt.Errorf("project manifest fixture %q has unknown status %q", fixture.ID, fixture.Expect.Status)
		}
	}
	return report, nil
}

func loadManifest(path string) (manifest, error) {
	file, err := os.Open(path)
	if err != nil {
		return manifest{}, fmt.Errorf("open project manifest fixture: %w", err)
	}
	defer file.Close()
	contents, err := io.ReadAll(io.LimitReader(file, maxManifestBytes+1))
	if err != nil {
		return manifest{}, fmt.Errorf("read project manifest fixture: %w", err)
	}
	if len(contents) > maxManifestBytes {
		return manifest{}, fmt.Errorf("project manifest fixture exceeds byte limit")
	}
	decoder := json.NewDecoder(bytes.NewReader(contents))
	decoder.DisallowUnknownFields()
	var definition manifest
	if err := decoder.Decode(&definition); err != nil {
		return manifest{}, fmt.Errorf("decode project manifest fixture: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return manifest{}, fmt.Errorf("decode project manifest fixture: multiple JSON values")
		}
		return manifest{}, fmt.Errorf("decode project manifest fixture: %w", err)
	}
	return definition, nil
}
