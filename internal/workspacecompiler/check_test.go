package workspacecompiler

import (
	"path/filepath"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
)

func TestCheckResolvedWorkspace(t *testing.T) {
	for _, generic := range []bool{false, true} {
		name, answer, main := "ordinary", testAnswer42, testRootSource
		if generic {
			name, answer, main = "generic", genericAnswer, genericMain
		}
		t.Run(name, func(t *testing.T) {
			root := writeCacheWorkspace(t, answer)
			writeFile(t, filepath.Join(root, "src", "main.fy"), main)
			workspace := loadCacheWorkspace(t, root)
			result, err := Check(workspace)
			if err != nil || !result.Verified {
				t.Fatalf("check: %+v %v", result.Semantic, err)
			}
			assertWorkspaceResult(t, workspace, result, 42)
			assertGenericWorkspaceNative(t, workspace, Result{Compiler: result})
			// Source expansion must not mutate the loader's retained AST.
			input, err := workspace.SemanticInput()
			if err != nil {
				t.Fatal(err)
			}
			ordinary, err := compiler.CheckPackages(input)
			if err != nil || ordinary.Verified == generic {
				t.Fatalf("retained source admission changed: %v %v", ordinary.Verified, err)
			}
		})
	}
}

func TestCheckWorkspaceGenericSourceDiagnostic(t *testing.T) {
	root := writeCacheWorkspace(t, genericAnswer)
	writeFile(t, filepath.Join(root, "src", "main.fy"), "module main; import answer as answer; fn i32 main() { return answer::value<Unknown>(42); }")
	result, err := Check(loadCacheWorkspace(t, root))
	if err != nil || result.Verified || len(result.MIR.Functions) != 0 || len(result.Semantic.Diagnostics) != 1 || result.Semantic.Diagnostics[0].Code != "E4502" {
		t.Fatalf("source diagnostic: %+v %v", result, err)
	}
}
