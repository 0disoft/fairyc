package workspaceloader

import (
	"testing"

	"github.com/0disoft/fairyc/internal/sdkstdlib"
)

func TestLoadVectorCandidateIdentityAndOverlay(t *testing.T) {
	for _, testsOnly := range []bool{false, true} {
		name := "production-import"
		if testsOnly {
			name = "test-only-import"
		}
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			main := "module main;\n"
			if !testsOnly {
				main += "import std::vec as vec;\n"
			}
			writePackage(t, root, ".", rootManifest(""), map[string]string{"src/main.fy": main})
			workspace, err := LoadVectorCandidate(root)
			if err != nil {
				t.Fatal(err)
			}
			if testsOnly && len(workspace.Packages) != 1 {
				t.Fatal("unused candidate SDK injected")
			}
			writeWorkspaceLoaderFile(t, root, "tests/check.fy", "module tests::check;\nimport std::vec as vec;\n")
			withTests, err := WithRootTests(workspace)
			if err != nil {
				t.Fatal(err)
			}
			sdk, exists := withTests.Package(sdkstdlib.PackagePath)
			definition, err := sdkstdlib.LoadVectorCandidate()
			if err != nil || !exists || sdk.SDKIdentity != sdkstdlib.VectorCandidateIdentity || sdk.Digest != definition.Digest || len(sdk.Sources) != 7 {
				t.Fatal("test overlay lost candidate identity")
			}
			if len(workspace.Packages[0].Sources) != 1 {
				t.Fatal("test overlay mutated production source")
			}
			if _, err := withTests.SemanticInput(); err != nil {
				t.Fatal(err)
			}
			ordinary, err := Load(root)
			if testsOnly {
				if err != nil {
					t.Fatal(err)
				}
				if _, err := WithRootTests(ordinary); err == nil {
					t.Fatal("ordinary test overlay admitted candidate")
				}
			} else if err == nil {
				t.Fatal("ordinary loader admitted candidate")
			}
		})
	}
}
