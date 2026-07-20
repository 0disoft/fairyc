package semanticfixture

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

const semanticFixtureSchema = "fairyc.semantic-fixtures/v0"

// Run loads every source in one semantic manifest and executes its resolve, type, and flow cases.
func Run(manifestPath string) (Report, error) {
	absoluteManifest, err := filepath.Abs(manifestPath)
	if err != nil {
		return Report{}, fmt.Errorf("resolve manifest: %w", err)
	}
	definition, err := loadManifest(absoluteManifest)
	if err != nil {
		return Report{}, err
	}
	report := Report{}
	manifestDirectory := filepath.Dir(absoluteManifest)
	seen := make(map[string]struct{}, len(definition.Cases))
	for _, fixture := range definition.Cases {
		if _, duplicate := seen[fixture.ID]; duplicate {
			report.Failures = append(report.Failures, Failure{CaseID: fixture.ID, Detail: "duplicate case id"})
			continue
		}
		seen[fixture.ID] = struct{}{}
		files, failures := parseCase(manifestDirectory, fixture)
		report.CasesParsed++
		if len(failures) != 0 {
			report.Failures = append(report.Failures, failures...)
			continue
		}
		report.CasesRun++
		report.Failures = append(report.Failures, compareCase(fixture, semantic.Analyze(files))...)
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
	if err := ensureJSONEOF(decoder); err != nil {
		return manifest{}, fmt.Errorf("decode manifest: %w", err)
	}
	if definition.Schema == "" {
		return manifest{}, fmt.Errorf("semantic fixture schema reference is empty")
	}
	if definition.SchemaVersion != semanticFixtureSchema {
		return manifest{}, fmt.Errorf("unsupported semantic fixture schema %q", definition.SchemaVersion)
	}
	if len(definition.Contracts) == 0 {
		return manifest{}, fmt.Errorf("semantic fixture contract list is empty")
	}
	return definition, nil
}

func ensureJSONEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); err == io.EOF {
		return nil
	} else if err != nil {
		return err
	}
	return fmt.Errorf("multiple JSON values")
}

func parseCase(manifestDirectory string, fixture caseDefinition) ([]semantic.File, []Failure) {
	if fixture.ID == "" {
		return nil, []Failure{{CaseID: fixture.ID, Detail: "empty case id"}}
	}
	if len(fixture.Sources) == 0 {
		return nil, []Failure{{CaseID: fixture.ID, Detail: "source list is empty"}}
	}
	if !slices.IsSorted(fixture.Sources) {
		return nil, []Failure{{CaseID: fixture.ID, Detail: "source list is not sorted"}}
	}
	if adjacentDuplicate(fixture.Sources) {
		return nil, []Failure{{CaseID: fixture.ID, Detail: "source list contains a duplicate"}}
	}
	sourceRoot, err := containedPath(manifestDirectory, fixture.SourceRoot)
	if err != nil {
		return nil, []Failure{{CaseID: fixture.ID, Detail: err.Error()}}
	}
	files := make([]semantic.File, 0, len(fixture.Sources))
	for _, relativePath := range fixture.Sources {
		if filepath.ToSlash(filepath.Clean(filepath.FromSlash(relativePath))) != relativePath {
			return nil, []Failure{{CaseID: fixture.ID, Detail: fmt.Sprintf("source path %q is not canonical", relativePath)}}
		}
		path, err := containedPath(sourceRoot, relativePath)
		if err != nil {
			return nil, []Failure{{CaseID: fixture.ID, Detail: err.Error()}}
		}
		contents, err := os.ReadFile(path)
		if err != nil {
			return nil, []Failure{{CaseID: fixture.ID, Detail: fmt.Sprintf("read source %q: %v", relativePath, err)}}
		}
		result, err := frontend.AnalyzeSyntax(source.NewFile(relativePath, contents), source.DefaultLimits())
		if err != nil {
			return nil, []Failure{{CaseID: fixture.ID, Detail: fmt.Sprintf("analyze syntax %q: %v", relativePath, err)}}
		}
		if len(result.Tree.Diagnostics) != 0 || result.AST == nil {
			return nil, []Failure{{CaseID: fixture.ID, Detail: fmt.Sprintf("source %q is not parser-valid: %+v", relativePath, result.Tree.Diagnostics)}}
		}
		files = append(files, semantic.File{Path: relativePath, Module: result.AST})
	}
	return files, nil
}

func compareCase(fixture caseDefinition, actual semantic.Result) []Failure {
	switch fixture.Expect.Status {
	case "pass":
		return comparePass(fixture, actual)
	case "fail":
		return compareFail(fixture, actual)
	default:
		return []Failure{{CaseID: fixture.ID, Detail: fmt.Sprintf("unsupported expected status %q", fixture.Expect.Status)}}
	}
}

func comparePass(fixture caseDefinition, actual semantic.Result) []Failure {
	if actual.FailedPhase != semantic.PhaseNone || len(actual.Diagnostics) != 0 {
		return []Failure{{CaseID: fixture.ID, Detail: fmt.Sprintf("unexpected semantic diagnostics: %+v", actual.Diagnostics)}}
	}
	if len(actual.Unsupported) != 0 {
		return []Failure{{CaseID: fixture.ID, Detail: fmt.Sprintf("implemented phase reported unsupported syntax: %+v", actual.Unsupported)}}
	}
	expectedPhase := semantic.Phase(fixture.Expect.CompletedThrough)
	if phaseRank(actual.CompletedThrough) < phaseRank(expectedPhase) {
		return []Failure{{CaseID: fixture.ID, Detail: fmt.Sprintf("completed through %q, want at least %q", actual.CompletedThrough, expectedPhase)}}
	}
	var failures []Failure
	for _, expected := range fixture.Expect.Resolutions {
		if !hasResolution(actual.Resolutions, expected) {
			failures = append(failures, Failure{CaseID: fixture.ID, Detail: fmt.Sprintf("resolution missing: %+v", expected)})
		}
	}
	for _, expected := range fixture.Expect.Types {
		if !hasType(actual.Types, expected) {
			failures = append(failures, Failure{CaseID: fixture.ID, Detail: fmt.Sprintf("type fact missing: %+v", expected)})
		}
	}
	for _, expected := range fixture.Expect.Borrows {
		if !hasBorrow(actual.Borrows, expected) {
			failures = append(failures, Failure{CaseID: fixture.ID, Detail: fmt.Sprintf("borrow fact missing: %+v", expected)})
		}
	}
	for _, expected := range fixture.Expect.Effects {
		if !hasEffect(actual.Effects, expected) {
			failures = append(failures, Failure{CaseID: fixture.ID, Detail: fmt.Sprintf("effect fact missing: %+v", expected)})
		}
	}
	return failures
}

func compareFail(fixture caseDefinition, actual semantic.Result) []Failure {
	expectedPhase := semantic.Phase(fixture.Expect.Phase)
	if actual.FailedPhase != expectedPhase {
		return []Failure{{CaseID: fixture.ID, Detail: fmt.Sprintf("failed phase %q, want %q", actual.FailedPhase, expectedPhase)}}
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

func hasResolution(actual []semantic.Resolution, expected resolutionAssertion) bool {
	for _, value := range actual {
		if spanEqual(value.Use, expected.Use) &&
			value.Target.Module == expected.Target.Module &&
			value.Target.Name == expected.Target.Name &&
			value.Target.Kind == expected.Target.Kind &&
			spanEqual(value.Target.Declaration, expected.Target.Declaration) {
			return true
		}
	}
	return false
}

func hasType(actual []semantic.TypeFact, expected typeAssertion) bool {
	for _, value := range actual {
		if spanEqual(value.Expression, expected.Expression) && string(value.CanonicalType) == expected.CanonicalType {
			return true
		}
	}
	return false
}

func hasBorrow(actual []semantic.BorrowFact, expected borrowAssertion) bool {
	for _, value := range actual {
		if !spanEqual(value.Borrow, expected.Borrow) || value.Kind != expected.Kind || value.Origin != expected.Origin || value.Escapes != expected.Escapes {
			continue
		}
		if value.LastUse == nil && expected.LastUse == nil {
			return true
		}
		if value.LastUse != nil && expected.LastUse != nil && spanEqual(*value.LastUse, *expected.LastUse) {
			return true
		}
	}
	return false
}

func hasEffect(actual []semantic.EffectFact, expected effectAssertion) bool {
	for _, value := range actual {
		if !spanEqual(value.Declaration, expected.Declaration) || len(value.Effects) != len(expected.Effects) {
			continue
		}
		matched := true
		for index := range value.Effects {
			if string(value.Effects[index]) != expected.Effects[index] {
				matched = false
				break
			}
		}
		if matched {
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

func phaseRank(phase semantic.Phase) int {
	switch phase {
	case semantic.PhaseResolve:
		return 1
	case semantic.PhaseType:
		return 2
	case semantic.PhaseFlow:
		return 3
	default:
		return 0
	}
}

func adjacentDuplicate(values []string) bool {
	for index := 1; index < len(values); index++ {
		if values[index] == values[index-1] {
			return true
		}
	}
	return false
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
