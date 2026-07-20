package resultfixture

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

const resultFixtureSchema = "fairyc.result-fixtures/v0"

// Run loads and executes every case in one Result manifest.
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
	if definition.Schema == "" || definition.SchemaVersion != resultFixtureSchema || len(definition.Contracts) == 0 {
		return manifest{}, fmt.Errorf("invalid Result fixture contract header")
	}
	return definition, nil
}

func runCase(manifestDirectory string, fixture caseDefinition) []Failure {
	if !sortedUnique(fixture.Sources) {
		return []Failure{{CaseID: fixture.ID, Detail: "source list is empty, unsorted, or duplicated"}}
	}
	sourceRoot, err := resolveWithin(manifestDirectory, manifestDirectory, fixture.SourceRoot)
	if err != nil {
		return []Failure{{CaseID: fixture.ID, Detail: err.Error()}}
	}
	files := make([]semantic.File, 0, len(fixture.Sources))
	for _, relativePath := range fixture.Sources {
		path, err := resolveWithin(sourceRoot, sourceRoot, relativePath)
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
	if fixture.Expect.Status == "pass" {
		return comparePass(fixture, actual)
	}
	return compareFailure(fixture, actual)
}

func comparePass(fixture caseDefinition, actual semantic.Result) []Failure {
	var failures []Failure
	if len(actual.Diagnostics) != 0 || actual.CompletedThrough != semantic.PhaseFlow {
		failures = append(failures, Failure{CaseID: fixture.ID, Detail: fmt.Sprintf("unexpected Flow result: phase=%q diagnostics=%+v", actual.CompletedThrough, actual.Diagnostics)})
	}
	for _, expected := range fixture.Expect.Handling {
		if !hasHandling(actual.ResultHandling, expected) {
			failures = append(failures, Failure{CaseID: fixture.ID, Detail: fmt.Sprintf("handling fact missing: %+v (actual %+v)", expected, actual.ResultHandling)})
		}
	}
	return failures
}

func compareFailure(fixture caseDefinition, actual semantic.Result) []Failure {
	expectedPhase := semantic.Phase(fixture.Expect.Phase)
	if actual.FailedPhase != expectedPhase {
		return []Failure{{CaseID: fixture.ID, Detail: fmt.Sprintf("failed phase: want %q, got %q", expectedPhase, actual.FailedPhase)}}
	}
	if len(actual.Diagnostics) != len(fixture.Expect.Diagnostics) {
		return []Failure{{CaseID: fixture.ID, Detail: fmt.Sprintf("diagnostic count: want %d, got %d (%+v)", len(fixture.Expect.Diagnostics), len(actual.Diagnostics), actual.Diagnostics)}}
	}
	var failures []Failure
	for index, expected := range fixture.Expect.Diagnostics {
		got := actual.Diagnostics[index]
		if got.Code != expected.Code || !spanEqual(got.Primary, expected.Primary) || !relatedEqual(got.Related, expected.Related) {
			failures = append(failures, Failure{CaseID: fixture.ID, Detail: fmt.Sprintf("diagnostic %d: want %+v, got %+v", index, expected, got)})
		}
	}
	return failures
}

func handlingEqual(actual semantic.ResultHandlingFact, expected handlingAssertion) bool {
	return spanEqual(actual.Expression, expected.Expression) && actual.Mode == expected.Mode && actual.ResultType == expected.ResultType && actual.SuccessType == expected.SuccessType && actual.ErrorType == expected.ErrorType && actual.EvaluationCount == expected.EvaluationCount
}

func hasHandling(actual []semantic.ResultHandlingFact, expected handlingAssertion) bool {
	for _, value := range actual {
		if handlingEqual(value, expected) {
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
	return actual.Package == hir.DefaultPackage && actual.Source == expected.Source && actual.StartByte == expected.StartByte && actual.EndByte == expected.EndByte
}

func sortedUnique(values []string) bool {
	if len(values) == 0 || !slices.IsSorted(values) {
		return false
	}
	for index := 1; index < len(values); index++ {
		if values[index-1] == values[index] {
			return false
		}
	}
	return true
}

func resolveWithin(boundary, base, relative string) (string, error) {
	if relative == "" || filepath.IsAbs(relative) || filepath.VolumeName(relative) != "" || strings.Contains(relative, "\\") {
		return "", fmt.Errorf("path %q is not a portable relative path", relative)
	}
	absoluteBoundary, err := filepath.Abs(boundary)
	if err != nil {
		return "", err
	}
	candidate, err := filepath.Abs(filepath.Join(base, filepath.FromSlash(relative)))
	if err != nil {
		return "", err
	}
	resolvedBoundary, err := filepath.EvalSymlinks(absoluteBoundary)
	if err != nil {
		return "", fmt.Errorf("resolve fixture boundary: %w", err)
	}
	resolvedCandidate, err := filepath.EvalSymlinks(candidate)
	if err != nil {
		return "", fmt.Errorf("resolve fixture path %q: %w", relative, err)
	}
	relativeToBoundary, err := filepath.Rel(resolvedBoundary, resolvedCandidate)
	if err != nil || relativeToBoundary == ".." || filepath.IsAbs(relativeToBoundary) || strings.HasPrefix(relativeToBoundary, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path %q escapes fixture boundary", relative)
	}
	return resolvedCandidate, nil
}
