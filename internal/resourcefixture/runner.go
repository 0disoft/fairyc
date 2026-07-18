package resourcefixture

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
)

const resourceFixtureSchema = "fairyc.resource-fixtures/v0"

// Run loads and executes every case in one resource Flow manifest.
func Run(manifestPath string) (Report, error) {
	absoluteManifest, err := filepath.Abs(manifestPath)
	if err != nil {
		return Report{}, fmt.Errorf("resolve manifest: %w", err)
	}
	definition, err := loadManifest(absoluteManifest)
	if err != nil {
		return Report{}, err
	}
	report := Report{CasesRun: len(definition.Cases)}
	directory := filepath.Dir(absoluteManifest)
	seen := make(map[string]struct{}, len(definition.Cases))
	for _, fixture := range definition.Cases {
		if _, duplicate := seen[fixture.ID]; duplicate {
			report.Failures = append(report.Failures, Failure{CaseID: fixture.ID, Detail: "duplicate case id"})
			continue
		}
		seen[fixture.ID] = struct{}{}
		report.Failures = append(report.Failures, runCase(directory, fixture)...)
	}
	return report, nil
}

func loadManifest(path string) (manifest, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return manifest{}, fmt.Errorf("read manifest: %w", err)
	}
	var definition manifest
	decoder := json.NewDecoder(bytes.NewReader(contents))
	decoder.DisallowUnknownFields()
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
	if definition.Schema == "" {
		return manifest{}, fmt.Errorf("resource fixture schema reference is empty")
	}
	if definition.SchemaVersion != resourceFixtureSchema {
		return manifest{}, fmt.Errorf("unsupported resource fixture schema %q", definition.SchemaVersion)
	}
	if len(definition.Contracts) == 0 {
		return manifest{}, fmt.Errorf("resource fixture contract list is empty")
	}
	return definition, nil
}

func runCase(manifestDirectory string, fixture caseDefinition) []Failure {
	if len(fixture.Sources) == 0 || !slices.IsSorted(fixture.Sources) {
		return []Failure{{CaseID: fixture.ID, Detail: "source list is empty or unsorted"}}
	}
	sourceRoot, err := containedPath(manifestDirectory, fixture.SourceRoot)
	if err != nil {
		return []Failure{{CaseID: fixture.ID, Detail: err.Error()}}
	}
	files := make([]semantic.File, 0, len(fixture.Sources))
	for _, relativePath := range fixture.Sources {
		path, err := containedPath(sourceRoot, relativePath)
		if err != nil {
			return []Failure{{CaseID: fixture.ID, Detail: err.Error()}}
		}
		contents, err := os.ReadFile(path)
		if err != nil {
			return []Failure{{CaseID: fixture.ID, Detail: fmt.Sprintf("read source %q: %v", relativePath, err)}}
		}
		result, err := frontend.AnalyzeSyntax(source.NewFile(relativePath, contents), source.DefaultLimits())
		if err != nil {
			return []Failure{{CaseID: fixture.ID, Detail: fmt.Sprintf("analyze syntax %q: %v", relativePath, err)}}
		}
		if len(result.Tree.Diagnostics) != 0 || result.AST == nil {
			return []Failure{{CaseID: fixture.ID, Detail: fmt.Sprintf("source %q is not parser-valid: %+v", relativePath, result.Tree.Diagnostics)}}
		}
		files = append(files, semantic.File{Path: relativePath, Module: result.AST})
	}
	actual := semantic.Analyze(files)
	if len(actual.Unsupported) != 0 {
		return []Failure{{CaseID: fixture.ID, Detail: fmt.Sprintf("fixture setup uses unsupported semantics: %+v", actual.Unsupported)}}
	}
	if actual.FailedPhase != semantic.PhaseNone && actual.FailedPhase != semantic.PhaseFlow {
		return []Failure{{CaseID: fixture.ID, Detail: fmt.Sprintf("fixture setup failed during %q: %+v", actual.FailedPhase, actual.Diagnostics)}}
	}
	var failures []Failure
	for _, expected := range fixture.Expect.States {
		if !hasState(actual.ResourceStates, expected) {
			failures = append(failures, Failure{CaseID: fixture.ID, Detail: fmt.Sprintf("resource state missing: %+v", expected)})
		}
	}
	if fixture.Expect.Status == "pass" {
		if len(actual.Diagnostics) != 0 || actual.CompletedThrough != semantic.PhaseFlow {
			failures = append(failures, Failure{CaseID: fixture.ID, Detail: fmt.Sprintf("unexpected Flow result: phase=%q diagnostics=%+v", actual.CompletedThrough, actual.Diagnostics)})
		}
		return failures
	}
	if len(actual.Diagnostics) != len(fixture.Expect.Diagnostics) {
		failures = append(failures, Failure{CaseID: fixture.ID, Detail: fmt.Sprintf("diagnostic count: want %d, got %d (%+v)", len(fixture.Expect.Diagnostics), len(actual.Diagnostics), actual.Diagnostics)})
		return failures
	}
	for index, expected := range fixture.Expect.Diagnostics {
		got := actual.Diagnostics[index]
		if got.Code != expected.Code || !spanEqual(got.Primary, expected.Primary) || !relatedEqual(got.Related, expected.Related) {
			failures = append(failures, Failure{CaseID: fixture.ID, Detail: fmt.Sprintf("diagnostic %d: want %+v, got %+v", index, expected, got)})
		}
	}
	return failures
}

func hasState(actual []semantic.ResourceStateFact, expected stateAssertion) bool {
	for _, value := range actual {
		if spanEqual(value.Binding, expected.Binding) && spanEqual(value.At, expected.At) &&
			value.Observation == expected.Observation && string(value.State) == expected.State &&
			obligationsEqual(value.CleanupObligations, expected.CleanupObligations) {
			return true
		}
	}
	return false
}

func relatedEqual(actual []semantic.RelatedSpan, expected []relatedSpanAssertion) bool {
	if len(actual) != len(expected) {
		return false
	}
	for index := range actual {
		if actual[index].Role != expected[index].Role || !spanEqual(actual[index].Span, expected[index].Span) {
			return false
		}
	}
	return true
}

func spanEqual(actual hir.Span, expected spanAssertion) bool {
	return actual.Source == expected.Source && actual.StartByte == expected.StartByte && actual.EndByte == expected.EndByte
}

func obligationsEqual(actual, expected *int) bool {
	if actual == nil || expected == nil {
		return actual == nil && expected == nil
	}
	return *actual == *expected
}

func containedPath(root, relative string) (string, error) {
	if relative == "" || filepath.IsAbs(relative) || filepath.VolumeName(relative) != "" || strings.Contains(relative, "\\") {
		return "", fmt.Errorf("path %q is not a portable relative path", relative)
	}
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	candidate, err := filepath.Abs(filepath.Join(absoluteRoot, filepath.FromSlash(relative)))
	if err != nil {
		return "", err
	}
	relativeToRoot, err := filepath.Rel(absoluteRoot, candidate)
	if err != nil {
		return "", err
	}
	if relativeToRoot == ".." || filepath.IsAbs(relativeToRoot) || strings.HasPrefix(relativeToRoot, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path %q escapes %q", relative, root)
	}
	return candidate, nil
}
