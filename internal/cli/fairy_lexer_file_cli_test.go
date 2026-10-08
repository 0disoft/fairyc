package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/0disoft/fairyc/internal/workspacecompiler"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

// The legacy corpus uses the authored main. Keep its new file-mode dependencies
// in the same workspace, even when a test exercises only the stdin branch.
func copyLexerFileCLIExample(t *testing.T, root string) {
	t.Helper()
	example := filepath.Join("..", "..", "examples", "lexer_candidate")
	manifest, err := os.ReadFile(filepath.Join(example, "fairy.toml"))
	if err != nil {
		t.Fatal(err)
	}
	writeTestSource(t, root, "fairy.toml", string(manifest))
	sources, err := filepath.Glob(filepath.Join(example, "src", "*.fy"))
	if err != nil || len(sources) == 0 {
		t.Fatalf("candidate source inventory: %v (%d files)", err, len(sources))
	}
	for _, source := range sources {
		data, err := os.ReadFile(source)
		if err != nil {
			t.Fatal(err)
		}
		writeTestSource(t, root, filepath.Join("src", filepath.Base(source)), string(data))
	}
}

func TestLexerFileCLIWorkspace(t *testing.T) {
	root := t.TempDir()
	copyLexerFileCLIExample(t, root)
	workspace, err := workspaceloader.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	checked, err := workspacecompiler.Check(workspace)
	if err != nil || !checked.Verified {
		t.Fatalf("legacy corpus CLI dependency check: %v %+v %+v", err, checked.Semantic.Diagnostics, checked.Semantic.Unsupported)
	}
}
