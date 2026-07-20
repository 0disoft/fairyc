package mirfixture

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
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
)

const mirFixtureSchema = "fairyc.mir-fixtures/v0"

// Run loads and executes every case in one source-to-MIR manifest.
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
	if definition.Schema == "" || definition.SchemaVersion != mirFixtureSchema || len(definition.Contracts) == 0 {
		return manifest{}, fmt.Errorf("invalid MIR fixture contract header")
	}
	return definition, nil
}

func runCase(manifestDirectory string, fixture caseDefinition) []Failure {
	if fixture.Expect.Status != "pass" || fixture.Expect.CompletedThrough != "mir-verify" {
		return []Failure{{CaseID: fixture.ID, Detail: "case does not declare pass through mir-verify"}}
	}
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

	semanticResult := semantic.Analyze(files)
	if len(semanticResult.Unsupported) != 0 {
		return []Failure{{CaseID: fixture.ID, Detail: fmt.Sprintf("fixture setup uses unsupported semantics: %+v", semanticResult.Unsupported)}}
	}
	if semanticResult.CompletedThrough != semantic.PhaseFlow || len(semanticResult.Diagnostics) != 0 {
		return []Failure{{CaseID: fixture.ID, Detail: fmt.Sprintf("fixture did not complete Flow: phase=%q diagnostics=%+v", semanticResult.CompletedThrough, semanticResult.Diagnostics)}}
	}
	program, err := mir.Lower(semanticResult.HIR)
	if err != nil {
		return []Failure{{CaseID: fixture.ID, Detail: err.Error()}}
	}
	if failures := mir.Verify(program); len(failures) != 0 {
		return []Failure{{CaseID: fixture.ID, Detail: fmt.Sprintf("MIR verifier rejected lowered program: %+v", failures)}}
	}
	return compareObservations(fixture, program)
}

func compareObservations(fixture caseDefinition, program mir.Program) []Failure {
	var failures []Failure
	for index, expected := range fixture.Expect.Observations {
		function, found := findFunction(program, expected.Function)
		if !found {
			failures = append(failures, Failure{CaseID: fixture.ID, Detail: fmt.Sprintf("observation %d function %q is missing", index, expected.Function)})
			continue
		}
		if !hasObservation(function, expected) {
			failures = append(failures, Failure{CaseID: fixture.ID, Detail: fmt.Sprintf("observation %d is missing: %+v", index, expected)})
		}
	}
	return failures
}

func findFunction(program mir.Program, name string) (mir.Function, bool) {
	for _, function := range program.Functions {
		if function.Name == name {
			return function, true
		}
	}
	return mir.Function{}, false
}

func hasObservation(function mir.Function, expected observationAssertion) bool {
	switch expected.Kind {
	case "ordered-evaluation":
		for _, actual := range function.OrderedEvaluations {
			if spanEqual(actual.Expression, expected.Expression) && spansEqual(actual.Sequence, expected.Sequence) && actual.MaterializedBeforeParent == expected.MaterializedBeforeParent {
				return true
			}
		}
	case "call":
		for _, actual := range function.Calls {
			if spanEqual(actual.Call, expected.Call) && spansEqual(actual.ArgumentOrder, expected.ArgumentOrder) && actual.NormalSuccessor == expected.NormalSuccessor && actual.UnwindSuccessor == expected.UnwindSuccessor {
				return true
			}
		}
	case "move":
		for _, actual := range function.Moves {
			if spanEqual(actual.Binding, expected.Binding) && spanEqual(actual.Operand, expected.Operand) && actual.Destination == expected.Destination && actual.CleanupObligationTransferred == expected.CleanupTransferred {
				return true
			}
		}
	case "resource-reassignment":
		for _, actual := range function.Reassignments {
			if spanEqual(actual.Binding, expected.Binding) && spanEqual(actual.Target, expected.Target) &&
				spanEqual(actual.Value, expected.Value) && actual.ValueBeforeDrop == expected.ValueBeforeDrop &&
				actual.DroppedOld == expected.DroppedOld && actual.InstalledNew == expected.InstalledNew {
				return true
			}
		}
	case "region-scope":
		for _, actual := range function.RegionScopes {
			if spanEqual(actual.Region, expected.Region) && spansEqual(actual.Placements, expected.Placements) &&
				actionsEqual(actual.CleanupBeforeReset, expected.CleanupBeforeReset) && spanEqual(actual.Reset, expected.Reset) {
				return true
			}
		}
	case "region-early-reset":
		for _, region := range function.RegionScopes {
			if !spanEqual(region.Region, expected.Region) || !spansEqual(region.Placements, expected.Placements) {
				continue
			}
			for _, actual := range region.EarlyResets {
				if spanEqual(actual.Exit, expected.Exit) && actual.ExitKind == expected.ExitKind &&
					actionsEqual(actual.CleanupBeforeReset, expected.CleanupBeforeReset) && spanEqual(actual.Reset, expected.Reset) {
					return true
				}
			}
		}
	case "cleanup-path":
		for _, actual := range function.CleanupPaths {
			if spanEqual(actual.Exit, expected.Exit) && actual.ExitKind == expected.ExitKind && actionsEqual(actual.Actions, expected.Actions) {
				return true
			}
		}
	case "result-branch":
		for _, actual := range function.ResultBranches {
			if spanEqual(actual.Question, expected.Question) && spanEqual(actual.ResultExpression, expected.ResultExpression) && actual.EvaluationCount == expected.EvaluationCount && actual.SuccessPayload == expected.SuccessPayload && actual.ErrorPayload == expected.ErrorPayload && slices.Equal(actual.ResultTemporaryDeadOn, expected.ResultTemporaryDeadOn) && actionsEqual(actual.ErrorCleanupActions, expected.ErrorCleanupActions) {
				return true
			}
		}
	case "trap-path":
		for _, actual := range function.TrapPaths {
			if spanEqual(actual.Operation, expected.Operation) && slices.Equal(actual.Categories, expected.Categories) && actionsEqual(actual.CleanupActions, expected.CleanupActions) && actual.HasSuccessor == expected.HasSuccessor {
				return true
			}
		}
	case "eliminated-form":
		for _, actual := range function.EliminatedForms {
			if actual.Form == expected.Form && actual.RemainingCount == expected.RemainingCount {
				return true
			}
		}
	}
	return false
}

func spansEqual(actual []hir.Span, expected []spanAssertion) bool {
	if len(actual) != len(expected) {
		return false
	}
	for index := range actual {
		if !spanEqual(actual[index], expected[index]) {
			return false
		}
	}
	return true
}

func actionsEqual(actual []mir.CleanupAction, expected []cleanupActionAssertion) bool {
	if len(actual) != len(expected) {
		return false
	}
	for index := range actual {
		if actual[index].Kind != expected[index].Kind || !spanEqual(actual[index].Binding, expected[index].Binding) {
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
