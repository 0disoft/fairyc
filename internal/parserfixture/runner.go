package parserfixture

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/diagnostic"
	"github.com/0disoft/fairyc/internal/source"
	"github.com/0disoft/fairyc/internal/syntax"
)

// Run loads and executes one parser manifest without changing repository state.
func Run(manifestPath string) (Report, error) {
	absoluteManifest, err := filepath.Abs(manifestPath)
	if err != nil {
		return Report{}, fmt.Errorf("resolve manifest: %w", err)
	}
	manifestBytes, err := os.ReadFile(absoluteManifest)
	if err != nil {
		return Report{}, fmt.Errorf("read manifest: %w", err)
	}
	var definition manifest
	decoder := json.NewDecoder(bytes.NewReader(manifestBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&definition); err != nil {
		return Report{}, fmt.Errorf("decode manifest: %w", err)
	}
	if definition.Schema == "" {
		return Report{}, fmt.Errorf("parser fixture schema reference is empty")
	}
	if definition.SchemaVersion != "fairyc.parser-fixtures/v0" {
		return Report{}, fmt.Errorf("unsupported parser fixture schema %q", definition.SchemaVersion)
	}
	manifestDirectory := filepath.Dir(absoluteManifest)
	specRoot := filepath.Clean(filepath.Join(manifestDirectory, "..", ".."))
	grammarPath, err := containedCandidate(specRoot, filepath.Join(manifestDirectory, filepath.FromSlash(definition.Grammar)))
	if err != nil {
		return Report{}, fmt.Errorf("resolve grammar: %w", err)
	}
	if _, err := os.Stat(grammarPath); err != nil {
		return Report{}, fmt.Errorf("grammar unavailable: %w", err)
	}
	report := Report{CasesRun: len(definition.Cases)}
	seen := make(map[string]struct{}, len(definition.Cases))
	for _, fixture := range definition.Cases {
		if _, duplicate := seen[fixture.ID]; duplicate {
			report.Failures = append(report.Failures, Failure{CaseID: fixture.ID, Detail: "duplicate case id"})
			continue
		}
		seen[fixture.ID] = struct{}{}
		report.Failures = append(report.Failures, runCase(manifestDirectory, fixture)...)
	}
	return report, nil
}

func runCase(manifestDirectory string, fixture caseDefinition) []Failure {
	sourceRoot, err := containedPath(manifestDirectory, fixture.SourceRoot)
	if err != nil {
		return []Failure{{CaseID: fixture.ID, Detail: err.Error()}}
	}
	sourcePath, err := containedPath(sourceRoot, fixture.Source)
	if err != nil {
		return []Failure{{CaseID: fixture.ID, Detail: err.Error()}}
	}
	sourceBytes, err := os.ReadFile(sourcePath)
	if err != nil {
		return []Failure{{CaseID: fixture.ID, Detail: fmt.Sprintf("read source: %v", err)}}
	}
	tree := syntax.Parse(source.NewFile(fixture.Source, sourceBytes), source.DefaultLimits())
	if fixture.Expect.Status == "fail" {
		return compareDiagnostics(fixture.ID, fixture.Expect.Diagnostics, tree.Diagnostics)
	}
	if len(tree.Diagnostics) != 0 {
		return []Failure{{CaseID: fixture.ID, Detail: fmt.Sprintf("unexpected diagnostics: %+v", tree.Diagnostics)}}
	}
	reconstructed, ok := tree.Reconstruct()
	if !ok || !slices.Equal(reconstructed, sourceBytes) {
		return []Failure{{CaseID: fixture.ID, Detail: "lossless token reconstruction differs from source"}}
	}
	module, err := ast.Lower(tree)
	if err != nil {
		return []Failure{{CaseID: fixture.ID, Detail: fmt.Sprintf("lower AST: %v", err)}}
	}
	var failures []Failure
	if module.Path != fixture.Expect.Module {
		failures = append(failures, Failure{CaseID: fixture.ID, Detail: fmt.Sprintf("module: want %q, got %q", fixture.Expect.Module, module.Path)})
	}
	bindings := make(map[string]*ast.Expression, len(module.Bindings))
	for _, binding := range module.Bindings {
		bindings[binding.Name] = binding.Expression
	}
	for _, expected := range fixture.Expect.Expressions {
		expression, found := bindings[expected.Binding]
		if !found {
			failures = append(failures, Failure{CaseID: fixture.ID, Detail: fmt.Sprintf("binding %q missing", expected.Binding)})
			continue
		}
		actual := ast.NormalizeExpression(expression)
		if actual != expected.Tree {
			failures = append(failures, Failure{CaseID: fixture.ID, Detail: fmt.Sprintf("binding %s: want %q, got %q", expected.Binding, expected.Tree, actual)})
		}
	}
	if fixture.Expect.Literals != nil {
		if len(module.Literals) != len(fixture.Expect.Literals) {
			failures = append(failures, Failure{CaseID: fixture.ID, Detail: fmt.Sprintf("literal count: want %d, got %d", len(fixture.Expect.Literals), len(module.Literals))})
		} else {
			for index, expected := range fixture.Expect.Literals {
				actual := module.Literals[index]
				if actual.Raw != expected.Raw || actual.Kind != expected.Kind || actual.Cooked != expected.Cooked {
					failures = append(failures, Failure{CaseID: fixture.ID, Detail: fmt.Sprintf("literal %d: want %+v, got %+v", index, expected, actual)})
				}
			}
		}
	}
	return failures
}

func compareDiagnostics(caseID string, expected []diagnosticAssertion, actual []diagnostic.Diagnostic) []Failure {
	if len(expected) != len(actual) {
		return []Failure{{CaseID: caseID, Detail: fmt.Sprintf("diagnostic count: want %d, got %d (%+v)", len(expected), len(actual), actual)}}
	}
	var failures []Failure
	for index := range expected {
		want := expected[index]
		got := actual[index]
		if got.Code != want.Code || got.Span != want.Span || string(got.Repair) != want.Repair || got.Token != want.Token {
			failures = append(failures, Failure{CaseID: caseID, Detail: fmt.Sprintf("diagnostic %d: want %+v, got %+v", index, want, got)})
		}
	}
	return failures
}

func containedPath(root, relative string) (string, error) {
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	candidate, err := filepath.Abs(filepath.Join(absoluteRoot, filepath.FromSlash(relative)))
	if err != nil {
		return "", err
	}
	return containedCandidate(absoluteRoot, candidate)
}

func containedCandidate(root, candidate string) (string, error) {
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	absoluteCandidate, err := filepath.Abs(candidate)
	if err != nil {
		return "", err
	}
	relativeToRoot, err := filepath.Rel(absoluteRoot, absoluteCandidate)
	if err != nil {
		return "", err
	}
	if relativeToRoot == ".." || filepath.IsAbs(relativeToRoot) || len(relativeToRoot) >= 3 && relativeToRoot[:3] == ".."+string(filepath.Separator) {
		return "", fmt.Errorf("path %q escapes %q", candidate, root)
	}
	return absoluteCandidate, nil
}
