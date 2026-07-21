package packagefixture

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/semantic"
)

func TestPackageManifestV1IncludesV0(t *testing.T) {
	t.Parallel()
	manifestPath := filepath.Join("..", "..", "spec", "fixtures", "package", "manifest-v1.json")
	report, err := Run(manifestPath)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if report.CasesRun != 9 {
		t.Fatalf("Run() cases = %d, want 9", report.CasesRun)
	}
	for _, failure := range report.Failures {
		t.Errorf("%s: %s", failure.CaseID, failure.Detail)
	}
}

func TestPackageManifestV0RemainsReadable(t *testing.T) {
	t.Parallel()
	manifestPath := filepath.Join("..", "..", "spec", "fixtures", "package", "manifest-v0.json")
	report, err := Run(manifestPath)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if report.CasesRun != 8 || len(report.Failures) != 0 {
		t.Fatalf("Run() report = %+v", report)
	}
}

func TestResolveCaseReturnsValidatedStdlibGraph(t *testing.T) {
	t.Parallel()
	manifestPath := filepath.Join("..", "..", "spec", "fixtures", "package", "manifest-v1.json")
	resolved, err := ResolveCase(manifestPath, "pass.package.stdlib-arena-region")
	if err != nil {
		t.Fatalf("ResolveCase() error = %v", err)
	}
	stdlib, found := resolved.Node("stdlib")
	if !found || stdlib.Role != packagegraph.RoleStandardLibrary || !slices.Equal(stdlib.Sources, []string{"std/alloc.fy"}) {
		t.Fatalf("ResolveCase() stdlib = %+v, found = %t", stdlib, found)
	}
	if resolved.ExpectedStatus != "pass" || resolved.Rejection != nil || len(resolved.Resolution.Lookups) != 1 {
		t.Fatalf("ResolveCase() = %+v", resolved)
	}
}

func TestResolvedModulePathIsolationReachesSemanticAnalysis(t *testing.T) {
	t.Parallel()
	manifestPath := filepath.Join("..", "..", "spec", "fixtures", "package", "manifest-v1.json")
	resolved, err := ResolveCase(manifestPath, "pass.package.module-path-isolation")
	if err != nil {
		t.Fatalf("ResolveCase() error = %v", err)
	}
	input, err := resolved.SemanticInput()
	if err != nil {
		t.Fatalf("SemanticInput() error = %v", err)
	}
	result, err := compiler.CheckPackages(input)
	if err != nil {
		t.Fatalf("CheckPackages() error = %v", err)
	}
	if !result.Verified || result.Semantic.CompletedThrough != semantic.PhaseFlow || result.Semantic.FailedPhase != semantic.PhaseNone || len(result.Semantic.Diagnostics) != 0 || len(result.Semantic.Unsupported) != 0 {
		t.Fatalf("CheckPackages() result = %+v", result)
	}
	if len(result.Semantic.HIR.Modules) != 4 || len(result.MIR.Functions) != 0 {
		t.Fatalf("CheckPackages() HIR modules = %d, MIR functions = %d", len(result.Semantic.HIR.Modules), len(result.MIR.Functions))
	}
	packages := make(map[uint32]struct{})
	for _, module := range result.Semantic.HIR.Modules {
		packages[uint32(module.Package)] = struct{}{}
	}
	if len(packages) != 4 {
		t.Fatalf("CheckPackages() package identities = %v", packages)
	}
}

func TestValidatePassExpectationRejectsOverlappingClasses(t *testing.T) {
	t.Parallel()
	err := validatePassExpectation(expectation{InstanceClasses: []packageInstanceClass{
		{Label: "first", Members: []string{"left", "shared"}},
		{Label: "second", Members: []string{"right", "shared"}},
	}})
	if err == nil || !strings.Contains(err.Error(), "appears in classes") {
		t.Fatalf("validatePassExpectation() error = %v", err)
	}
}

func TestDecodeStrictJSONRejectsUnknownDuplicateAndMultipleValues(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name     string
		contents string
		contains string
	}{
		{name: "unknown", contents: `{"known":"one","unknown":true}`, contains: "unknown field"},
		{name: "duplicate", contents: `{"known":"one","known":"two"}`, contains: "duplicate JSON key"},
		{name: "multiple", contents: `{"known":"one"} {"known":"two"}`, contains: "unexpected JSON token"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			var target struct {
				Known string `json:"known"`
			}
			if err := decodeStrictJSON([]byte(test.contents), &target); err == nil || !strings.Contains(err.Error(), test.contains) {
				t.Fatalf("decodeStrictJSON() error = %v, want containing %q", err, test.contains)
			}
		})
	}
}

func TestLoadPackageModulesRejectsUnlistedSource(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "main.fy"), []byte("module main;\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "hidden.fy"), []byte("module hidden;\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	total := int64(0)
	if _, _, err := loadPackageModules(root, []string{"main.fy"}, &total); err == nil || !strings.Contains(err.Error(), "source list is incomplete") {
		t.Fatalf("loadPackageModules() error = %v", err)
	}
}

func TestResolveWithinRejectsSymlinkEscape(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	outside := t.TempDir()
	link := filepath.Join(root, "link")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if _, err := resolveWithin(root, root, "link"); err == nil {
		t.Fatal("resolveWithin() accepted a symlink outside the fixture boundary")
	}
}
