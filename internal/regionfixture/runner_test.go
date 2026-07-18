package regionfixture

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRegionManifestV1(t *testing.T) {
	t.Parallel()
	manifestPath := filepath.Join("..", "..", "spec", "fixtures", "region", "manifest-v1.json")
	report, err := Run(manifestPath)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if report.CasesRun != 6 {
		t.Fatalf("Run() cases = %d, want 6", report.CasesRun)
	}
	for _, failure := range report.Failures {
		t.Errorf("%s: %s", failure.CaseID, failure.Detail)
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

func TestResolveWithinRejectsFixtureEscape(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if _, err := resolveWithin(root, root, "../escape.fy"); err == nil {
		t.Fatal("resolveWithin() accepted a parent traversal outside the fixture boundary")
	}
}

func TestStdlibLookupRejectsNonStdlibProvider(t *testing.T) {
	t.Parallel()
	fixture := packageCase{
		Dependencies: []packageDependency{{Owner: "app", Binding: "stdlib", Target: "fake"}},
		Expect: packageExpectation{
			InstanceClasses: []packageInstanceClass{{Label: "provider", Members: []string{"fake"}}},
			Lookups:         []packageLookup{{From: "app", ImportPath: "std::alloc", ProviderClass: "provider", ModulePath: "std::alloc"}},
		},
	}
	if _, _, err := stdlibLookup(fixture, "app", map[string]packageNode{"fake": {Node: "fake", Role: "dependency"}}); err == nil {
		t.Fatal("stdlibLookup() accepted a non-standard-library provider")
	}
}
