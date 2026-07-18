package interpreterfixture

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
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
)

const interpreterFixtureSchema = "fairyc.interpreter-fixtures/v0"

// Run loads and executes every case in one source-to-interpreter manifest.
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
	if definition.Schema == "" || definition.SchemaVersion != interpreterFixtureSchema || len(definition.Contracts) == 0 {
		return manifest{}, fmt.Errorf("invalid interpreter fixture contract header")
	}
	return definition, nil
}

func runCase(manifestDirectory string, fixture caseDefinition) []Failure {
	if fixture.Expect.Status != "pass" || fixture.Expect.CompletedThrough != "interpret" {
		return []Failure{{CaseID: fixture.ID, Detail: "case does not declare pass through interpretation"}}
	}
	if fixture.Expect.Outcome == interpreter.OutcomeReturn && (fixture.Expect.Value == nil || fixture.Expect.Category != nil) {
		return []Failure{{CaseID: fixture.ID, Detail: "return expectation must contain only value"}}
	}
	if fixture.Expect.Outcome == interpreter.OutcomeTrap && (fixture.Expect.Category == nil || fixture.Expect.Value != nil) {
		return []Failure{{CaseID: fixture.ID, Detail: "trap expectation must contain only category"}}
	}
	if fixture.Expect.Outcome != interpreter.OutcomeReturn && fixture.Expect.Outcome != interpreter.OutcomeTrap {
		return []Failure{{CaseID: fixture.ID, Detail: fmt.Sprintf("unknown interpreter outcome %q", fixture.Expect.Outcome)}}
	}
	if !sortedUnique(fixture.Sources) {
		return []Failure{{CaseID: fixture.ID, Detail: "source list is empty, unsorted, or duplicated"}}
	}
	sourceRoot, err := resolveWithin(manifestDirectory, manifestDirectory, fixture.SourceRoot)
	if err != nil {
		return []Failure{{CaseID: fixture.ID, Detail: err.Error()}}
	}
	files := make([]semantic.File, 0, len(fixture.Sources))
	resourceTypes := make([]hir.Type, 0)
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
		for _, resource := range result.AST.Resources {
			resourceTypes = append(resourceTypes, hir.Type(result.AST.Path+"::"+resource.Name))
		}
		files = append(files, semantic.File{Path: relativePath, Module: result.AST})
	}
	semanticResult := semantic.Analyze(files)
	if len(semanticResult.Unsupported) != 0 {
		return []Failure{{CaseID: fixture.ID, Detail: fmt.Sprintf("fixture setup uses unsupported semantics: %+v", semanticResult.Unsupported)}}
	}
	if semanticResult.CompletedThrough != semantic.PhaseFlow || len(semanticResult.Diagnostics) != 0 {
		return []Failure{{CaseID: fixture.ID, Detail: fmt.Sprintf("fixture did not complete Flow: phase=%q diagnostics=%+v", semanticResult.CompletedThrough, semanticResult.Diagnostics)}}
	}
	program, err := mir.Lower(semanticResult.HIR, mir.NewTypeProperties(resourceTypes))
	if err != nil {
		return []Failure{{CaseID: fixture.ID, Detail: err.Error()}}
	}
	outcome, err := interpreter.Execute(program, fixture.Entry)
	if err != nil {
		return []Failure{{CaseID: fixture.ID, Detail: err.Error()}}
	}
	if outcome.Kind != fixture.Expect.Outcome {
		return []Failure{{CaseID: fixture.ID, Detail: fmt.Sprintf("outcome: want %q, got %+v", fixture.Expect.Outcome, outcome)}}
	}
	if outcome.Kind == interpreter.OutcomeReturn && outcome.Value != *fixture.Expect.Value {
		return []Failure{{CaseID: fixture.ID, Detail: fmt.Sprintf("return value: want %d, got %d", *fixture.Expect.Value, outcome.Value)}}
	}
	if outcome.Kind == interpreter.OutcomeTrap && outcome.TrapCategory != *fixture.Expect.Category {
		return []Failure{{CaseID: fixture.ID, Detail: fmt.Sprintf("trap category: want %q, got %q", *fixture.Expect.Category, outcome.TrapCategory)}}
	}
	return nil
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
