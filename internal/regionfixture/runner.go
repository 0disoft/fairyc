package regionfixture

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
	"github.com/0disoft/fairyc/internal/packagefixture"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
)

const regionFixtureSchema = "fairyc.region-fixtures/v1"

// Run loads and executes every case in one Region manifest.
func Run(manifestPath string) (Report, error) {
	absoluteManifest, err := filepath.Abs(manifestPath)
	if err != nil {
		return Report{}, fmt.Errorf("resolve manifest: %w", err)
	}
	definition, err := loadRegionManifest(absoluteManifest)
	if err != nil {
		return Report{}, err
	}
	report := Report{CasesRun: len(definition.Cases)}
	manifestDirectory := filepath.Dir(absoluteManifest)
	fixtureRoot := filepath.Dir(manifestDirectory)
	seen := make(map[string]struct{}, len(definition.Cases))
	for _, fixture := range definition.Cases {
		if _, duplicate := seen[fixture.ID]; duplicate {
			report.Failures = append(report.Failures, Failure{CaseID: fixture.ID, Detail: "duplicate case id"})
			continue
		}
		seen[fixture.ID] = struct{}{}
		report.Failures = append(report.Failures, runCase(fixtureRoot, manifestDirectory, fixture)...)
	}
	return report, nil
}

func loadRegionManifest(path string) (manifest, error) {
	var definition manifest
	if err := decodeStrictFile(path, &definition); err != nil {
		return manifest{}, fmt.Errorf("decode Region manifest: %w", err)
	}
	if definition.Schema == "" || definition.SchemaVersion != regionFixtureSchema || len(definition.Contracts) == 0 {
		return manifest{}, fmt.Errorf("invalid Region fixture contract header")
	}
	return definition, nil
}

func runCase(fixtureRoot, manifestDirectory string, fixture caseDefinition) []Failure {
	if !sortedUnique(fixture.Sources) {
		return []Failure{{CaseID: fixture.ID, Detail: "source list is empty or unsorted"}}
	}
	sourceRoot, err := resolveWithin(fixtureRoot, manifestDirectory, fixture.SourceRoot)
	if err != nil {
		return []Failure{{CaseID: fixture.ID, Detail: err.Error()}}
	}
	files, err := loadSemanticSources(sourceRoot, fixture.Sources)
	if err != nil {
		return []Failure{{CaseID: fixture.ID, Detail: err.Error()}}
	}
	if fixture.PackageGraph != nil {
		providerFiles, err := loadPackageProvider(fixtureRoot, manifestDirectory, sourceRoot, fixture.Sources, *fixture.PackageGraph)
		if err != nil {
			return []Failure{{CaseID: fixture.ID, Detail: err.Error()}}
		}
		files = append(files, providerFiles...)
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
	if len(actual.RegionOrigins) != len(fixture.Expect.Origins) {
		return append(failures, Failure{CaseID: fixture.ID, Detail: fmt.Sprintf("origin count: want %d, got %d (%+v)", len(fixture.Expect.Origins), len(actual.RegionOrigins), actual.RegionOrigins)})
	}
	for index, expected := range fixture.Expect.Origins {
		if !originEqual(actual.RegionOrigins[index], expected) {
			failures = append(failures, Failure{CaseID: fixture.ID, Detail: fmt.Sprintf("origin %d: want %+v, got %+v", index, expected, actual.RegionOrigins[index])})
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

func loadPackageProvider(fixtureRoot, regionDirectory, expectedRoot string, expectedSources []string, reference packageGraphReference) ([]semantic.File, error) {
	manifestPath, err := resolveWithin(fixtureRoot, regionDirectory, reference.Manifest)
	if err != nil {
		return nil, fmt.Errorf("package graph manifest: %w", err)
	}
	selected, err := packagefixture.ResolveCase(manifestPath, reference.CaseID)
	if err != nil {
		return nil, err
	}
	if selected.ExpectedStatus != "pass" || selected.Rejection != nil {
		return nil, fmt.Errorf("package graph case %q is not passing", reference.CaseID)
	}
	root, found := selected.Node(reference.RootNode)
	if !found || root.Role != packagegraph.RoleRoot {
		return nil, fmt.Errorf("package root node %q is missing or not root", reference.RootNode)
	}
	if !samePath(root.SourceRoot, expectedRoot) || !slices.Equal(root.Sources, expectedSources) {
		return nil, fmt.Errorf("package root source does not match Region sourceRoot")
	}
	providerNode, err := stdlibProvider(selected, reference.RootNode)
	if err != nil {
		return nil, err
	}
	files, err := loadSemanticSources(providerNode.SourceRoot, providerNode.Sources)
	if err != nil {
		return nil, err
	}
	if !exportsOpaqueArena(files) {
		return nil, fmt.Errorf("declared standard-library provider does not export std::alloc::Arena")
	}
	return files, nil
}

func stdlibProvider(selected packagefixture.ResolvedCase, rootNode string) (packagefixture.ResolvedNode, error) {
	provider := ""
	for _, lookup := range selected.Resolution.Lookups {
		if lookup.From != rootNode || lookup.ImportPath != "std::alloc" || lookup.ModulePath != "std::alloc" {
			continue
		}
		if provider != "" || len(lookup.ProviderMembers) != 1 {
			return packagefixture.ResolvedNode{}, fmt.Errorf("package graph has ambiguous std::alloc lookup from %q", rootNode)
		}
		provider = lookup.ProviderMembers[0]
	}
	if provider == "" {
		return packagefixture.ResolvedNode{}, fmt.Errorf("package graph has no std::alloc lookup from %q", rootNode)
	}
	linked := false
	for _, dependency := range selected.Dependencies {
		if dependency.Owner == rootNode && dependency.Binding == "stdlib" && dependency.Target == provider {
			linked = true
		}
	}
	if !linked {
		return packagefixture.ResolvedNode{}, fmt.Errorf("root package does not depend on std::alloc provider")
	}
	providerNode, found := selected.Node(provider)
	if !found || providerNode.Role != packagegraph.RoleStandardLibrary {
		return packagefixture.ResolvedNode{}, fmt.Errorf("std::alloc provider is not a declared standard-library node")
	}
	return providerNode, nil
}

func exportsOpaqueArena(files []semantic.File) bool {
	for _, file := range files {
		if file.Module == nil || file.Module.Path != "std::alloc" {
			continue
		}
		for _, resource := range file.Module.Resources {
			if resource.Name == "Arena" && resource.Public {
				return true
			}
		}
	}
	return false
}

func loadSemanticSources(root string, sources []string) ([]semantic.File, error) {
	if !sortedUnique(sources) {
		return nil, fmt.Errorf("source list is empty or unsorted")
	}
	files := make([]semantic.File, 0, len(sources))
	for _, relativePath := range sources {
		path, err := resolveWithin(root, root, relativePath)
		if err != nil {
			return nil, err
		}
		contents, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read source %q: %w", relativePath, err)
		}
		result, err := frontend.AnalyzeSyntax(source.NewFile(relativePath, contents), source.DefaultLimits())
		if err != nil {
			return nil, fmt.Errorf("analyze syntax %q: %w", relativePath, err)
		}
		if len(result.Tree.Diagnostics) != 0 || result.AST == nil {
			return nil, fmt.Errorf("source %q is not parser-valid: %+v", relativePath, result.Tree.Diagnostics)
		}
		files = append(files, semantic.File{Path: relativePath, Module: result.AST})
	}
	return files, nil
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

func decodeStrictFile(path string, destination any) error {
	contents, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(contents))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return fmt.Errorf("multiple JSON values")
		}
		return err
	}
	return nil
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

func samePath(left, right string) bool {
	leftInfo, leftErr := os.Stat(left)
	rightInfo, rightErr := os.Stat(right)
	return leftErr == nil && rightErr == nil && os.SameFile(leftInfo, rightInfo)
}

func originEqual(actual semantic.RegionOriginFact, expected originAssertion) bool {
	if actual.Operation != expected.Operation || actual.Origin != expected.Origin || actual.Escapes != expected.Escapes || !spanEqual(actual.Allocation, expected.Allocation) {
		return false
	}
	if actual.LastUse == nil || expected.LastUse == nil {
		return actual.LastUse == nil && expected.LastUse == nil
	}
	return spanEqual(*actual.LastUse, *expected.LastUse)
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
