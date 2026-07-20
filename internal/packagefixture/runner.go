package packagefixture

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
)

const (
	fixtureSchemaVersion = "fairyc.package-fixtures/v0"
	fixtureSchemaPath    = "./schema-v0.json"
	maxManifestBytes     = 2 << 20
	maxFixtureCases      = 1024
	maxCaseSources       = 4096
	maxCaseSourceBytes   = 64 << 20
)

var (
	caseIDPattern    = regexp.MustCompile(`^(pass|fail)\.package\.[a-z0-9]+(?:-[a-z0-9]+)*$`)
	kebabPattern     = regexp.MustCompile(`^[a-z][a-z0-9]*(?:-[a-z0-9]+)*$`)
	modulePattern    = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*(?:::[A-Za-z_][A-Za-z0-9_]*)*$`)
	sourcePattern    = regexp.MustCompile(`^[A-Za-z0-9_-]+(?:/[A-Za-z0-9_-]+)*\.fy$`)
	directoryPattern = regexp.MustCompile(`^[A-Za-z0-9_-]+(?:/[A-Za-z0-9_-]+)*$`)
)

// Run loads and executes every case in one resolved-package-graph manifest.
func Run(manifestPath string) (Report, error) {
	absoluteManifest, err := filepath.Abs(manifestPath)
	if err != nil {
		return Report{}, fmt.Errorf("resolve package manifest: %w", err)
	}
	definition, err := loadManifest(absoluteManifest)
	if err != nil {
		return Report{}, err
	}
	report := Report{CasesRun: len(definition.Cases)}
	manifestDirectory := filepath.Dir(absoluteManifest)
	for _, fixture := range definition.Cases {
		report.Failures = append(report.Failures, runCase(manifestDirectory, fixture)...)
	}
	return report, nil
}

// ResolveCase loads one named case, proves that its computed result matches the manifest
// expectation, and returns the validated graph for a dependent acceptance runner.
func ResolveCase(manifestPath, caseID string) (ResolvedCase, error) {
	absoluteManifest, err := filepath.Abs(manifestPath)
	if err != nil {
		return ResolvedCase{}, fmt.Errorf("resolve package manifest: %w", err)
	}
	definition, err := loadManifest(absoluteManifest)
	if err != nil {
		return ResolvedCase{}, err
	}
	var selected *caseDefinition
	for index := range definition.Cases {
		if definition.Cases[index].ID != caseID {
			continue
		}
		if selected != nil {
			return ResolvedCase{}, fmt.Errorf("duplicate package graph case %q", caseID)
		}
		selected = &definition.Cases[index]
	}
	if selected == nil {
		return ResolvedCase{}, fmt.Errorf("package graph case %q is missing", caseID)
	}
	resolved, err := resolveCase(filepath.Dir(absoluteManifest), *selected)
	if err != nil {
		return ResolvedCase{}, fmt.Errorf("package graph case %q: %w", caseID, err)
	}
	failures := compareResolved(*selected, resolved)
	if len(failures) != 0 {
		return ResolvedCase{}, fmt.Errorf("package graph case %q does not match its expectation: %s", caseID, failures[0].Detail)
	}
	return resolved, nil
}

func loadManifest(path string) (manifest, error) {
	contents, err := readBoundedRegularFile(path, maxManifestBytes)
	if err != nil {
		return manifest{}, fmt.Errorf("read package manifest: %w", err)
	}
	var definition manifest
	if err := decodeStrictJSON(contents, &definition); err != nil {
		return manifest{}, fmt.Errorf("decode package manifest: %w", err)
	}
	if definition.Schema != fixtureSchemaPath || definition.SchemaVersion != fixtureSchemaVersion {
		return manifest{}, fmt.Errorf("invalid package fixture contract header")
	}
	if len(definition.Cases) == 0 || len(definition.Cases) > maxFixtureCases {
		return manifest{}, fmt.Errorf("package fixture case count is outside the implementation guard")
	}
	if !nonemptyUnique(definition.Deferred, true) || !nonemptyUnique(definition.Contracts, false) {
		return manifest{}, fmt.Errorf("invalid package fixture contract lists")
	}
	seenCases := make(map[string]struct{}, len(definition.Cases))
	for _, fixture := range definition.Cases {
		if _, duplicate := seenCases[fixture.ID]; duplicate {
			return manifest{}, fmt.Errorf("duplicate package fixture case %q", fixture.ID)
		}
		seenCases[fixture.ID] = struct{}{}
	}
	return definition, nil
}

func runCase(manifestDirectory string, fixture caseDefinition) []Failure {
	resolved, err := resolveCase(manifestDirectory, fixture)
	if err != nil {
		return []Failure{{CaseID: fixture.ID, Detail: err.Error()}}
	}
	return compareResolved(fixture, resolved)
}

func resolveCase(manifestDirectory string, fixture caseDefinition) (ResolvedCase, error) {
	if err := validateCaseShape(fixture); err != nil {
		return ResolvedCase{}, err
	}
	caseRoot, err := resolveWithin(manifestDirectory, manifestDirectory, fixture.CaseRoot)
	if err != nil {
		return ResolvedCase{}, fmt.Errorf("case root: %v", err)
	}
	graph := packagegraph.Graph{
		Nodes:        make([]packagegraph.Node, 0, len(fixture.Packages)),
		Dependencies: make([]packagegraph.Dependency, 0, len(fixture.Dependencies)),
	}
	resolvedRoots := make(map[string]string, len(fixture.Packages))
	sourceLists := make(map[string][]string, len(fixture.Packages))
	resolvedNodes := make([]ResolvedNode, 0, len(fixture.Packages))
	totalBytes := int64(0)
	totalSources := 0
	for _, node := range fixture.Packages {
		if !directoryPattern.MatchString(node.SourceRoot) || !sortedUnique(node.Sources, false) {
			return ResolvedCase{}, fmt.Errorf("package node %q has an invalid source root or unsorted source list", node.Node)
		}
		for _, relative := range node.Sources {
			if !sourcePattern.MatchString(relative) {
				return ResolvedCase{}, fmt.Errorf("package node %q has invalid source path %q", node.Node, relative)
			}
		}
		totalSources += len(node.Sources)
		if totalSources > maxCaseSources {
			return ResolvedCase{}, fmt.Errorf("package case source count exceeds the implementation guard")
		}
		sourceRoot, err := resolveWithin(caseRoot, caseRoot, node.SourceRoot)
		if err != nil {
			return ResolvedCase{}, fmt.Errorf("package node %q source root: %v", node.Node, err)
		}
		modules, files, err := loadPackageModules(sourceRoot, node.Sources, &totalBytes)
		if err != nil {
			return ResolvedCase{}, fmt.Errorf("package node %q: %v", node.Node, err)
		}
		resolvedRoots[node.Node] = sourceRoot
		sourceLists[node.Node] = slices.Clone(node.Sources)
		graph.Nodes = append(graph.Nodes, packagegraph.Node{
			ID:   node.Node,
			Role: node.Role,
			Identity: packagegraph.Identity{
				PackageUID:      node.PackageUID,
				SourceIdentity:  node.SourceIdentity,
				VersionIdentity: node.VersionIdentity,
				Features:        slices.Clone(node.Features),
			},
			Modules: modules,
		})
		resolvedNodes = append(resolvedNodes, ResolvedNode{
			ID: node.Node, Role: node.Role, SourceRoot: sourceRoot,
			Sources: slices.Clone(node.Sources), Modules: slices.Clone(modules), Files: files,
		})
	}
	for _, dependency := range fixture.Dependencies {
		if !kebabPattern.MatchString(dependency.Owner) || !kebabPattern.MatchString(dependency.Binding) || !kebabPattern.MatchString(dependency.Target) {
			return ResolvedCase{}, fmt.Errorf("dependency contains an invalid node or binding id")
		}
		graph.Dependencies = append(graph.Dependencies, packagegraph.Dependency{
			Owner: dependency.Owner, Binding: dependency.Binding, Target: dependency.Target,
		})
	}
	actual, rejection, err := packagegraph.Resolve(graph)
	if err != nil {
		return ResolvedCase{}, err
	}
	if err := validateInstanceSources(actual.InstanceClasses, resolvedRoots, sourceLists); err != nil {
		return ResolvedCase{}, err
	}
	return ResolvedCase{
		ID: fixture.ID, ExpectedStatus: fixture.Expect.Status, Nodes: resolvedNodes,
		Dependencies: slices.Clone(graph.Dependencies), Resolution: actual, Rejection: rejection,
	}, nil
}

func compareResolved(fixture caseDefinition, resolved ResolvedCase) []Failure {
	if fixture.Expect.Status == "pass" {
		return comparePass(fixture, resolved.Resolution, resolved.Rejection)
	}
	return compareReject(fixture, resolved.Rejection)
}

func validateCaseShape(fixture caseDefinition) error {
	if !caseIDPattern.MatchString(fixture.ID) || !directoryPattern.MatchString(fixture.CaseRoot) || len(fixture.Packages) == 0 {
		return fmt.Errorf("invalid package case shape")
	}
	if strings.HasPrefix(fixture.ID, "pass.") != (fixture.Expect.Status == "pass") || strings.HasPrefix(fixture.ID, "fail.") != (fixture.Expect.Status == "reject") {
		return fmt.Errorf("case id and expected status disagree")
	}
	for _, node := range fixture.Packages {
		if !kebabPattern.MatchString(node.Node) || node.PackageUID == "" || node.SourceIdentity == "" || node.VersionIdentity == "" || !sortedUnique(node.Features, true) {
			return fmt.Errorf("package node %q has an invalid identity", node.Node)
		}
		for _, feature := range node.Features {
			if !kebabPattern.MatchString(feature) {
				return fmt.Errorf("package node %q has invalid feature %q", node.Node, feature)
			}
		}
	}
	if fixture.Expect.Status == "pass" {
		if len(fixture.Expect.InstanceClasses) == 0 || fixture.Expect.Reason != "" || len(fixture.Expect.Subjects) != 0 || fixture.Expect.ImportPath != "" {
			return fmt.Errorf("pass expectation has invalid fields")
		}
		return validatePassExpectation(fixture.Expect)
	}
	if len(fixture.Expect.InstanceClasses) != 0 || len(fixture.Expect.Lookups) != 0 || len(fixture.Expect.Subjects) == 0 || !sortedUniqueSet(fixture.Expect.Subjects) {
		return fmt.Errorf("reject expectation has invalid fields")
	}
	switch fixture.Expect.Reason {
	case packagegraph.ReasonAmbiguousVisibleModule, packagegraph.ReasonMissingVisibleModule:
		if !modulePattern.MatchString(fixture.Expect.ImportPath) {
			return fmt.Errorf("reject expectation has invalid import path")
		}
	case packagegraph.ReasonDuplicateBinding:
		if fixture.Expect.ImportPath != "" {
			return fmt.Errorf("duplicate-binding expectation must omit importPath")
		}
	default:
		return fmt.Errorf("unknown reject reason %q", fixture.Expect.Reason)
	}
	return nil
}

func validatePassExpectation(expected expectation) error {
	labels := make(map[string]struct{}, len(expected.InstanceClasses))
	members := make(map[string]string)
	classSets := make(map[string]struct{}, len(expected.InstanceClasses))
	for _, class := range expected.InstanceClasses {
		if !kebabPattern.MatchString(class.Label) || !sortedUniqueSet(class.Members) {
			return fmt.Errorf("invalid expected instance class %q", class.Label)
		}
		if _, duplicate := labels[class.Label]; duplicate {
			return fmt.Errorf("duplicate expected instance class %q", class.Label)
		}
		labels[class.Label] = struct{}{}
		setKey := canonicalSet(class.Members)
		if _, duplicate := classSets[setKey]; duplicate {
			return fmt.Errorf("duplicate expected instance member set in class %q", class.Label)
		}
		classSets[setKey] = struct{}{}
		for _, member := range class.Members {
			if previous, duplicate := members[member]; duplicate {
				return fmt.Errorf("expected package node %q appears in classes %q and %q", member, previous, class.Label)
			}
			members[member] = class.Label
		}
	}
	for _, lookup := range expected.Lookups {
		if !kebabPattern.MatchString(lookup.From) || !modulePattern.MatchString(lookup.ImportPath) || !kebabPattern.MatchString(lookup.ProviderClass) || !modulePattern.MatchString(lookup.ModulePath) {
			return fmt.Errorf("invalid expected lookup from %q", lookup.From)
		}
		if _, found := labels[lookup.ProviderClass]; !found {
			return fmt.Errorf("lookup references unknown provider class %q", lookup.ProviderClass)
		}
	}
	return nil
}

func loadPackageModules(root string, expectedSources []string, totalBytes *int64) ([]packagegraph.Module, []semantic.File, error) {
	actualSources, err := listFYFiles(root)
	if err != nil {
		return nil, nil, err
	}
	if !slices.Equal(actualSources, expectedSources) {
		return nil, nil, fmt.Errorf("source list is incomplete or contains unlisted files: want %v, found %v", expectedSources, actualSources)
	}
	modules := make([]packagegraph.Module, 0, len(expectedSources))
	files := make([]semantic.File, 0, len(expectedSources))
	for _, relative := range expectedSources {
		path, err := resolveWithin(root, root, relative)
		if err != nil {
			return nil, nil, err
		}
		contents, err := readBoundedRegularFile(path, source.DefaultLimits().MaxBytes)
		if err != nil {
			return nil, nil, fmt.Errorf("read source %q: %w", relative, err)
		}
		*totalBytes += int64(len(contents))
		if *totalBytes > maxCaseSourceBytes {
			return nil, nil, fmt.Errorf("package case source bytes exceed the implementation guard")
		}
		result, err := frontend.AnalyzeSyntax(source.NewFile(relative, contents), source.DefaultLimits())
		if err != nil {
			return nil, nil, fmt.Errorf("analyze syntax %q: %w", relative, err)
		}
		if len(result.Tree.Diagnostics) != 0 || result.AST == nil {
			return nil, nil, fmt.Errorf("source %q is not parser-valid: %+v", relative, result.Tree.Diagnostics)
		}
		imports := make([]string, len(result.AST.Imports))
		for index, imported := range result.AST.Imports {
			imports[index] = imported.Path
		}
		modules = append(modules, packagegraph.Module{Path: result.AST.Path, Imports: imports})
		files = append(files, semantic.File{Path: relative, Module: result.AST})
	}
	slices.SortFunc(modules, func(left, right packagegraph.Module) int { return strings.Compare(left.Path, right.Path) })
	return modules, files, nil
}

func listFYFiles(root string) ([]string, error) {
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return nil, fmt.Errorf("source root is not a directory")
	}
	var sources []string
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("source root contains symlink %q", path)
		}
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".fy" {
			return nil
		}
		info, err := entry.Info()
		if err != nil || !info.Mode().IsRegular() {
			return fmt.Errorf("source %q is not a regular file", path)
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		sources = append(sources, filepath.ToSlash(relative))
		if len(sources) > maxCaseSources {
			return fmt.Errorf("source root exceeds the implementation file-count guard")
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	slices.Sort(sources)
	return sources, nil
}

func validateInstanceSources(classes []packagegraph.InstanceClass, roots map[string]string, sources map[string][]string) error {
	for _, class := range classes {
		if len(class.Members) == 0 {
			return fmt.Errorf("resolved package instance class is empty")
		}
		reference := class.Members[0]
		for _, member := range class.Members[1:] {
			if !samePath(roots[reference], roots[member]) || !slices.Equal(sources[reference], sources[member]) {
				return fmt.Errorf("equal package instance nodes %q and %q do not own the same source root and source list", reference, member)
			}
		}
	}
	return nil
}

func comparePass(fixture caseDefinition, actual packagegraph.Result, rejection *packagegraph.Rejection) []Failure {
	if rejection != nil {
		return []Failure{{CaseID: fixture.ID, Detail: fmt.Sprintf("unexpected rejection: %+v", *rejection)}}
	}
	classLabels := make(map[string]string, len(fixture.Expect.InstanceClasses))
	expectedClasses := make(map[string]string, len(fixture.Expect.InstanceClasses))
	for _, class := range fixture.Expect.InstanceClasses {
		key := canonicalSet(class.Members)
		expectedClasses[key] = class.Label
		classLabels[key] = class.Label
	}
	actualClasses := make(map[string]struct{}, len(actual.InstanceClasses))
	for _, class := range actual.InstanceClasses {
		actualClasses[canonicalSet(class.Members)] = struct{}{}
	}
	if len(actualClasses) != len(expectedClasses) {
		return []Failure{{CaseID: fixture.ID, Detail: fmt.Sprintf("instance class count: want %d, got %d", len(expectedClasses), len(actualClasses))}}
	}
	for expected := range expectedClasses {
		if _, found := actualClasses[expected]; !found {
			return []Failure{{CaseID: fixture.ID, Detail: fmt.Sprintf("missing expected instance class %q", expectedClasses[expected])}}
		}
	}
	expectedLookups := make(map[string]int, len(fixture.Expect.Lookups))
	for _, lookup := range fixture.Expect.Lookups {
		expectedLookups[lookupKey(lookup.From, lookup.ImportPath, lookup.ProviderClass, lookup.ModulePath)]++
	}
	actualLookups := make(map[string]int, len(actual.Lookups))
	for _, lookup := range actual.Lookups {
		providerClass, found := classLabels[canonicalSet(lookup.ProviderMembers)]
		if !found {
			return []Failure{{CaseID: fixture.ID, Detail: fmt.Sprintf("lookup provider has no expected class: %+v", lookup)}}
		}
		actualLookups[lookupKey(lookup.From, lookup.ImportPath, providerClass, lookup.ModulePath)]++
	}
	if !mapsEqual(expectedLookups, actualLookups) {
		return []Failure{{CaseID: fixture.ID, Detail: fmt.Sprintf("lookup set: want %v, got %v", expectedLookups, actualLookups)}}
	}
	return nil
}

func compareReject(fixture caseDefinition, actual *packagegraph.Rejection) []Failure {
	if actual == nil {
		return []Failure{{CaseID: fixture.ID, Detail: "expected package graph rejection, got pass"}}
	}
	if actual.Reason != fixture.Expect.Reason || actual.ImportPath != fixture.Expect.ImportPath || canonicalSet(actual.Subjects) != canonicalSet(fixture.Expect.Subjects) {
		return []Failure{{CaseID: fixture.ID, Detail: fmt.Sprintf("rejection: want reason=%q subjects=%v import=%q, got %+v", fixture.Expect.Reason, fixture.Expect.Subjects, fixture.Expect.ImportPath, *actual)}}
	}
	return nil
}

func decodeStrictJSON(contents []byte, destination any) error {
	if err := rejectDuplicateJSONKeys(contents); err != nil {
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

func rejectDuplicateJSONKeys(contents []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(contents))
	if err := scanJSONValue(decoder); err != nil {
		return err
	}
	if token, err := decoder.Token(); err != io.EOF {
		if err == nil {
			return fmt.Errorf("unexpected JSON token %v", token)
		}
		return err
	}
	return nil
}

func scanJSONValue(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delimiter, composite := token.(json.Delim)
	if !composite {
		return nil
	}
	switch delimiter {
	case '{':
		seen := make(map[string]struct{})
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok {
				return fmt.Errorf("JSON object key is not a string")
			}
			if _, duplicate := seen[key]; duplicate {
				return fmt.Errorf("duplicate JSON key %q", key)
			}
			seen[key] = struct{}{}
			if err := scanJSONValue(decoder); err != nil {
				return err
			}
		}
		closing, err := decoder.Token()
		if err != nil || closing != json.Delim('}') {
			return fmt.Errorf("invalid JSON object closing delimiter")
		}
	case '[':
		for decoder.More() {
			if err := scanJSONValue(decoder); err != nil {
				return err
			}
		}
		closing, err := decoder.Token()
		if err != nil || closing != json.Delim(']') {
			return fmt.Errorf("invalid JSON array closing delimiter")
		}
	default:
		return fmt.Errorf("unexpected JSON delimiter %q", delimiter)
	}
	return nil
}

func readBoundedRegularFile(path string, maximum int) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("not a regular file")
	}
	if info.Size() > int64(maximum) {
		return nil, fmt.Errorf("file exceeds %d-byte implementation guard", maximum)
	}
	contents, err := io.ReadAll(io.LimitReader(file, int64(maximum)+1))
	if err != nil {
		return nil, err
	}
	if len(contents) > maximum {
		return nil, fmt.Errorf("file exceeds %d-byte implementation guard", maximum)
	}
	return contents, nil
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

func sortedUnique(values []string, allowEmpty bool) bool {
	if len(values) == 0 {
		return allowEmpty
	}
	if !slices.IsSorted(values) {
		return false
	}
	for index, value := range values {
		if value == "" || index > 0 && values[index-1] == value {
			return false
		}
	}
	return true
}

func sortedUniqueSet(values []string) bool {
	if len(values) == 0 {
		return false
	}
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if !kebabPattern.MatchString(value) {
			return false
		}
		if _, duplicate := seen[value]; duplicate {
			return false
		}
		seen[value] = struct{}{}
	}
	return true
}

func nonemptyUnique(values []string, allowEmptyList bool) bool {
	if len(values) == 0 {
		return allowEmptyList
	}
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if value == "" {
			return false
		}
		if _, duplicate := seen[value]; duplicate {
			return false
		}
		seen[value] = struct{}{}
	}
	return true
}

func canonicalSet(values []string) string {
	copyValues := slices.Clone(values)
	slices.Sort(copyValues)
	return strings.Join(copyValues, "\x00")
}

func lookupKey(from, importPath, providerClass, modulePath string) string {
	return strings.Join([]string{from, importPath, providerClass, modulePath}, "\x00")
}

func mapsEqual(left, right map[string]int) bool {
	if len(left) != len(right) {
		return false
	}
	for key, value := range left {
		if right[key] != value {
			return false
		}
	}
	return true
}
