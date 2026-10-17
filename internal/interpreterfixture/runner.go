package interpreterfixture

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

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/interpreter"
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
	if fixture.Expect.Outcome == interpreter.OutcomeLimit && (fixture.Expect.Category == nil || fixture.Expect.Value != nil) {
		return []Failure{{CaseID: fixture.ID, Detail: "limit expectation must contain only category"}}
	}
	if fixture.Expect.Outcome != interpreter.OutcomeReturn && fixture.Expect.Outcome != interpreter.OutcomeTrap && fixture.Expect.Outcome != interpreter.OutcomeLimit {
		return []Failure{{CaseID: fixture.ID, Detail: fmt.Sprintf("unknown interpreter outcome %q", fixture.Expect.Outcome)}}
	}
	if !sortedUnique(fixture.Sources) {
		return []Failure{{CaseID: fixture.ID, Detail: "source list is empty, unsorted, or duplicated"}}
	}
	testIntrinsics, err := testIntrinsicOptions(fixture)
	if err != nil {
		return []Failure{{CaseID: fixture.ID, Detail: err.Error()}}
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
	checked, err := compiler.Check(files)
	if err != nil {
		return []Failure{{CaseID: fixture.ID, Detail: err.Error()}}
	}
	semanticResult := checked.Semantic
	if len(semanticResult.Unsupported) != 0 {
		return []Failure{{CaseID: fixture.ID, Detail: fmt.Sprintf("fixture setup uses unsupported semantics: %+v", semanticResult.Unsupported)}}
	}
	if semanticResult.CompletedThrough != semantic.PhaseFlow || len(semanticResult.Diagnostics) != 0 {
		return []Failure{{CaseID: fixture.ID, Detail: fmt.Sprintf("fixture did not complete Flow: phase=%q diagnostics=%+v", semanticResult.CompletedThrough, semanticResult.Diagnostics)}}
	}
	if err := validateTestIntrinsicSignatures(semanticResult.HIR, fixture.TestIntrinsics); err != nil {
		return []Failure{{CaseID: fixture.ID, Detail: err.Error()}}
	}
	if !checked.Verified {
		return []Failure{{CaseID: fixture.ID, Detail: "fixture did not produce verified MIR"}}
	}
	outcome, err := interpreter.ExecuteWithOptions(checked.MIR, fixture.Entry, interpreter.Options{
		TestIntrinsics: testIntrinsics,
		OutputLimit:    fixture.OutputLimit,
	})
	if fixture.Expect.Outcome == interpreter.OutcomeLimit {
		var capability interpreter.CapabilityError
		if !errors.As(err, &capability) {
			return []Failure{{CaseID: fixture.ID, Detail: fmt.Sprintf("limit outcome: want %q, got outcome=%+v error=%v", *fixture.Expect.Category, outcome, err)}}
		}
		if capability.Kind != *fixture.Expect.Category {
			return []Failure{{CaseID: fixture.ID, Detail: fmt.Sprintf("limit category: want %q, got %q", *fixture.Expect.Category, capability.Kind)}}
		}
		if outcome.Kind != interpreter.OutcomeLimit {
			return []Failure{{CaseID: fixture.ID, Detail: fmt.Sprintf("limit outcome kind: want %q, got %q", interpreter.OutcomeLimit, outcome.Kind)}}
		}
		if outputFailure := compareOutput(fixture, outcome); outputFailure != "" {
			return []Failure{{CaseID: fixture.ID, Detail: outputFailure}}
		}
		return nil
	}
	if err != nil {
		return []Failure{{CaseID: fixture.ID, Detail: err.Error()}}
	}
	if outcome.Kind != fixture.Expect.Outcome {
		return []Failure{{CaseID: fixture.ID, Detail: fmt.Sprintf("outcome: want %q, got %+v", fixture.Expect.Outcome, outcome)}}
	}
	if outcome.Kind == interpreter.OutcomeReturn && outcome.Value != *fixture.Expect.Value {
		return []Failure{{CaseID: fixture.ID, Detail: fmt.Sprintf("return value: want %d, got %d", *fixture.Expect.Value, outcome.Value)}}
	}
	if !dropTraceEqual(outcome.DropTrace, fixture.Expect.Drops) {
		return []Failure{{CaseID: fixture.ID, Detail: fmt.Sprintf("drop trace: want %+v, got %+v", fixture.Expect.Drops, outcome.DropTrace)}}
	}
	if !regionTraceEqual(outcome.RegionTrace, fixture.Expect.Regions) {
		return []Failure{{CaseID: fixture.ID, Detail: fmt.Sprintf("region trace: want %+v, got %+v", fixture.Expect.Regions, outcome.RegionTrace)}}
	}
	if outputFailure := compareOutput(fixture, outcome); outputFailure != "" {
		return []Failure{{CaseID: fixture.ID, Detail: outputFailure}}
	}
	if outcome.Kind == interpreter.OutcomeTrap && outcome.TrapCategory != *fixture.Expect.Category {
		return []Failure{{CaseID: fixture.ID, Detail: fmt.Sprintf("trap category: want %q, got %q", *fixture.Expect.Category, outcome.TrapCategory)}}
	}
	return nil
}

func compareOutput(fixture caseDefinition, outcome interpreter.Outcome) string {
	if !slices.Equal(outcome.Stdout, fixture.Expect.StdoutBytes) {
		return fmt.Sprintf("stdout bytes: want %v, got %v", fixture.Expect.StdoutBytes, outcome.Stdout)
	}
	if !slices.Equal(outcome.Stderr, fixture.Expect.StderrBytes) {
		return fmt.Sprintf("stderr bytes: want %v, got %v", fixture.Expect.StderrBytes, outcome.Stderr)
	}
	if !ioTraceEqual(outcome.IOTrace, fixture.Expect.IOTrace) {
		return fmt.Sprintf("I/O trace: want %+v, got %+v", fixture.Expect.IOTrace, outcome.IOTrace)
	}
	return ""
}

func testIntrinsicOptions(fixture caseDefinition) (map[string]interpreter.TestIntrinsicKind, error) {
	if len(fixture.TestIntrinsics) == 0 {
		if fixture.OutputLimit != 0 {
			return nil, fmt.Errorf("outputLimit requires testIntrinsics")
		}
		return nil, nil
	}
	if fixture.OutputLimit <= 0 {
		return nil, fmt.Errorf("testIntrinsics require a positive outputLimit")
	}
	result := make(map[string]interpreter.TestIntrinsicKind, len(fixture.TestIntrinsics))
	previous := ""
	for _, binding := range fixture.TestIntrinsics {
		if binding.Function == "" || binding.Function <= previous {
			return nil, fmt.Errorf("testIntrinsics must be sorted by unique non-empty function name")
		}
		kind := interpreter.TestIntrinsicKind(binding.Kind)
		if kind != interpreter.TestIntrinsicWriteStdoutByte && kind != interpreter.TestIntrinsicWriteStderrByte {
			return nil, fmt.Errorf("test intrinsic %q has unsupported kind %q", binding.Function, binding.Kind)
		}
		result[binding.Function] = kind
		previous = binding.Function
	}
	return result, nil
}

func validateTestIntrinsicSignatures(pkg hir.Package, definitions []testIntrinsicDefinition) error {
	if len(definitions) == 0 {
		return nil
	}
	functions := make(map[string]*hir.Function)
	for moduleIndex := range pkg.Modules {
		module := &pkg.Modules[moduleIndex]
		for functionIndex := range module.Functions {
			function := &module.Functions[functionIndex]
			functions[module.Path+"::"+function.Name] = function
		}
	}
	for _, definition := range definitions {
		function := functions[definition.Function]
		if function == nil {
			return fmt.Errorf("test intrinsic function %q is missing from typed HIR", definition.Function)
		}
		if len(function.Parameters) != 1 || function.Parameters[0].Type != hir.TypeI32 || function.ResultType != hir.TypeI32 {
			return fmt.Errorf("test intrinsic function %q must have type fn(i32) -> i32", definition.Function)
		}
	}
	return nil
}

func ioTraceEqual(actual []interpreter.IOEvent, expected []ioExpectation) bool {
	if len(actual) != len(expected) {
		return false
	}
	for index := range actual {
		if actual[index].Stream != expected[index].Stream || actual[index].Byte != expected[index].Byte {
			return false
		}
	}
	return true
}

func regionTraceEqual(actual []interpreter.RegionEvent, expected []regionExpectation) bool {
	if len(actual) != len(expected) {
		return false
	}
	for index := range actual {
		if !regionSpanEqual(actual[index].Region, expected[index].Region) ||
			actual[index].ExitKind != expected[index].ExitKind ||
			actual[index].Placements != expected[index].Placements ||
			actual[index].DropCountBeforeReset != expected[index].DropCountBeforeReset {
			return false
		}
	}
	return true
}

func regionSpanEqual(actual hir.Span, expected spanExpectation) bool {
	return actual.Package == hir.DefaultPackage && actual.Source == expected.Source && actual.StartByte == expected.StartByte && actual.EndByte == expected.EndByte
}

func dropTraceEqual(actual []interpreter.DropEvent, expected []dropExpectation) bool {
	if len(actual) != len(expected) {
		return false
	}
	for index := range actual {
		if (actual[index].ResourceType).String() != expected[index].ResourceType ||
			actual[index].Binding.Package != hir.DefaultPackage ||
			actual[index].Binding.Source != expected[index].Binding.Source ||
			actual[index].Binding.StartByte != expected[index].Binding.StartByte ||
			actual[index].Binding.EndByte != expected[index].Binding.EndByte {
			return false
		}
	}
	return true
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
