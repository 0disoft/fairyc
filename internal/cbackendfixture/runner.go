package cbackendfixture

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

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
	"github.com/0disoft/fairyc/internal/target"
)

const cEmissionFixtureSchema = "fairyc.c-emission-fixtures/v0"

// Run executes every source through Flow, verified MIR, the interpreter, and deterministic C emission.
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
		var emitted EmittedCase
		failures := runCase(directory, fixture, &emitted)
		report.Failures = append(report.Failures, failures...)
		if len(failures) == 0 {
			report.Emitted = append(report.Emitted, emitted)
		}
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
	if definition.Schema == "" || definition.SchemaVersion != cEmissionFixtureSchema || len(definition.Contracts) == 0 || len(definition.Deferred) == 0 {
		return manifest{}, fmt.Errorf("invalid C emission fixture contract header")
	}
	return definition, nil
}

func runCase(manifestDirectory string, fixture caseDefinition, emitted *EmittedCase) []Failure {
	if fixture.Expect.Status != "pass" || fixture.Expect.CompletedThrough != "c-emission" {
		return []Failure{{CaseID: fixture.ID, Detail: "case does not declare pass through c-emission"}}
	}
	if !sortedUnique(fixture.Sources) || !sortedUnique(fixture.Expect.Functions) || !sortedUnique(fixture.Expect.Features) {
		return []Failure{{CaseID: fixture.ID, Detail: "sources, functions, and features must be non-empty sorted unique lists"}}
	}
	if fixture.Expect.Outcome == string(interpreter.OutcomeReturn) {
		if fixture.Expect.Value == nil || fixture.Expect.Category != nil {
			return []Failure{{CaseID: fixture.ID, Detail: "return outcome must declare only value"}}
		}
	} else if fixture.Expect.Outcome == string(interpreter.OutcomeTrap) {
		if fixture.Expect.Category == nil || fixture.Expect.Value != nil {
			return []Failure{{CaseID: fixture.ID, Detail: "trap outcome must declare only category"}}
		}
	} else {
		return []Failure{{CaseID: fixture.ID, Detail: fmt.Sprintf("unsupported expected outcome %q", fixture.Expect.Outcome)}}
	}
	program, setupFailure := lowerFixture(manifestDirectory, fixture)
	if setupFailure != nil {
		return []Failure{*setupFailure}
	}
	outcome, err := interpreter.Execute(program, fixture.Entry)
	if err != nil {
		var capability interpreter.CapabilityError
		if errors.As(err, &capability) {
			return []Failure{{CaseID: fixture.ID, Detail: fmt.Sprintf("interpreter capability %q: %v", capability.Kind, capability)}}
		}
		return []Failure{{CaseID: fixture.ID, Detail: fmt.Sprintf("interpreter: %v", err)}}
	}
	if failure := compareOutcome(fixture, outcome); failure != "" {
		return []Failure{{CaseID: fixture.ID, Detail: failure}}
	}
	profile := cbackend.Profile{Target: target.ID(fixture.Expect.Target), Dialect: fixture.Expect.Dialect}
	unit, err := cbackend.Emit(program, fixture.Entry, profile)
	if err != nil {
		return []Failure{{CaseID: fixture.ID, Detail: fmt.Sprintf("emit C: %v", err)}}
	}
	if unit.Target != fixture.Expect.Target || unit.Dialect != fixture.Expect.Dialect || unit.Entry != fixture.Entry {
		return []Failure{{CaseID: fixture.ID, Detail: fmt.Sprintf("C unit identity: got target=%q dialect=%q entry=%q", unit.Target, unit.Dialect, unit.Entry)}}
	}
	if !slices.Equal(unit.Functions, fixture.Expect.Functions) {
		return []Failure{{CaseID: fixture.ID, Detail: fmt.Sprintf("C unit functions: want %v, got %v", fixture.Expect.Functions, unit.Functions)}}
	}
	if !slices.Equal(unit.Features, fixture.Expect.Features) {
		return []Failure{{CaseID: fixture.ID, Detail: fmt.Sprintf("C unit features: want %v, got %v", fixture.Expect.Features, unit.Features)}}
	}
	if unit.Source == "" || strings.Contains(unit.Source, "\r") || strings.Contains(unit.Source, manifestDirectory) {
		return []Failure{{CaseID: fixture.ID, Detail: "generated C is empty, non-LF, or leaks an absolute fixture path"}}
	}
	second, err := cbackend.Emit(program, fixture.Entry, profile)
	if err != nil || second.Source != unit.Source {
		return []Failure{{CaseID: fixture.ID, Detail: fmt.Sprintf("repeated C emission is not deterministic: %v", err)}}
	}
	traceUnit, err := cbackend.Emit(program, fixture.Entry, cbackend.Stage0DropTraceProfile())
	if err != nil {
		return []Failure{{CaseID: fixture.ID, Detail: fmt.Sprintf("emit drop-trace C: %v", err)}}
	}
	if traceUnit.Target != unit.Target || traceUnit.Dialect != unit.Dialect || traceUnit.Entry != unit.Entry ||
		traceUnit.Instrumentation != cbackend.InstrumentationDropTraceV0 ||
		!slices.Equal(traceUnit.Functions, unit.Functions) || !slices.Equal(traceUnit.Features, unit.Features) {
		return []Failure{{CaseID: fixture.ID, Detail: "drop-trace C unit identity drifted from production emission"}}
	}
	if traceUnit.Source == "" || traceUnit.Source == unit.Source || strings.Contains(traceUnit.Source, "\r") || strings.Contains(traceUnit.Source, manifestDirectory) {
		return []Failure{{CaseID: fixture.ID, Detail: "drop-trace C is empty, uninstrumented, non-LF, or leaks an absolute fixture path"}}
	}
	for _, event := range outcome.DropTrace {
		if !dropEventHasSite(event, traceUnit.DropSites) {
			return []Failure{{CaseID: fixture.ID, Detail: fmt.Sprintf("drop trace has no emitted site for %+v", event)}}
		}
	}
	secondTrace, err := cbackend.Emit(program, fixture.Entry, cbackend.Stage0DropTraceProfile())
	if err != nil || secondTrace.Source != traceUnit.Source || !slices.Equal(secondTrace.DropSites, traceUnit.DropSites) {
		return []Failure{{CaseID: fixture.ID, Detail: fmt.Sprintf("repeated drop-trace C emission is not deterministic: %v", err)}}
	}
	*emitted = EmittedCase{
		ID: fixture.ID, Outcome: fixture.Expect.Outcome, Unit: unit, TraceUnit: traceUnit,
		DropTrace: slices.Clone(outcome.DropTrace),
	}
	if fixture.Expect.Value != nil {
		emitted.Value = *fixture.Expect.Value
	}
	if fixture.Expect.Category != nil {
		emitted.TrapCategory = *fixture.Expect.Category
	}
	return nil
}

func dropEventHasSite(event interpreter.DropEvent, sites []cbackend.DropSite) bool {
	for _, site := range sites {
		if site.ResourceType == event.ResourceType.String() && site.Package == event.Binding.Package && site.Source == event.Binding.Source &&
			site.StartByte == event.Binding.StartByte && site.EndByte == event.Binding.EndByte {
			return true
		}
	}
	return false
}

func lowerFixture(manifestDirectory string, fixture caseDefinition) (mir.Program, *Failure) {
	sourceRoot, err := resolveWithin(manifestDirectory, manifestDirectory, fixture.SourceRoot)
	if err != nil {
		return mir.Program{}, &Failure{CaseID: fixture.ID, Detail: err.Error()}
	}
	files := make([]semantic.File, 0, len(fixture.Sources))
	for _, relativePath := range fixture.Sources {
		path, err := resolveWithin(sourceRoot, sourceRoot, relativePath)
		if err != nil {
			return mir.Program{}, &Failure{CaseID: fixture.ID, Detail: err.Error()}
		}
		contents, err := os.ReadFile(path)
		if err != nil {
			return mir.Program{}, &Failure{CaseID: fixture.ID, Detail: fmt.Sprintf("read source %q: %v", relativePath, err)}
		}
		parsed, err := frontend.AnalyzeSyntax(source.NewFile(relativePath, contents), source.DefaultLimits())
		if err != nil {
			return mir.Program{}, &Failure{CaseID: fixture.ID, Detail: fmt.Sprintf("analyze syntax %q: %v", relativePath, err)}
		}
		if len(parsed.Tree.Diagnostics) != 0 || parsed.AST == nil {
			return mir.Program{}, &Failure{CaseID: fixture.ID, Detail: fmt.Sprintf("source %q is not parser-valid: %+v", relativePath, parsed.Tree.Diagnostics)}
		}
		files = append(files, semantic.File{Path: relativePath, Module: parsed.AST})
	}
	checked, err := compiler.Check(files)
	if err != nil {
		return mir.Program{}, &Failure{CaseID: fixture.ID, Detail: err.Error()}
	}
	semanticResult := checked.Semantic
	if len(semanticResult.Unsupported) != 0 {
		return mir.Program{}, &Failure{CaseID: fixture.ID, Detail: fmt.Sprintf("fixture setup uses unsupported semantics: %+v", semanticResult.Unsupported)}
	}
	if semanticResult.CompletedThrough != semantic.PhaseFlow || len(semanticResult.Diagnostics) != 0 {
		return mir.Program{}, &Failure{CaseID: fixture.ID, Detail: fmt.Sprintf("fixture did not complete Flow: phase=%q diagnostics=%+v", semanticResult.CompletedThrough, semanticResult.Diagnostics)}
	}
	if !checked.Verified {
		return mir.Program{}, &Failure{CaseID: fixture.ID, Detail: "fixture did not produce verified MIR"}
	}
	return checked.MIR, nil
}

func compareOutcome(fixture caseDefinition, actual interpreter.Outcome) string {
	if string(actual.Kind) != fixture.Expect.Outcome {
		return fmt.Sprintf("interpreter outcome: want %q, got %q", fixture.Expect.Outcome, actual.Kind)
	}
	if actual.Kind == interpreter.OutcomeReturn && actual.Value != *fixture.Expect.Value {
		return fmt.Sprintf("interpreter value: want %d, got %d", *fixture.Expect.Value, actual.Value)
	}
	if actual.Kind == interpreter.OutcomeTrap && actual.TrapCategory != *fixture.Expect.Category {
		return fmt.Sprintf("interpreter trap: want %q, got %q", *fixture.Expect.Category, actual.TrapCategory)
	}
	return ""
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
