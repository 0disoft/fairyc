package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/workspacecompiler"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

// Verify the authored example's storage path rather than a second implementation
// of its lexer: scanning and normalization must not copy the fixed token field.
func TestTokenPageCLIStorage(t *testing.T) {
	root := t.TempDir()
	example := filepath.Join("..", "..", "examples", "token_page_cli")
	for _, name := range []string{"fairy.toml", "src/main.fy", "src/tokens.fy", "src/evaluator.fy", "src/arithmetic.fy", "src/input_file.fy"} {
		source, err := os.ReadFile(filepath.Join(example, name))
		if err != nil {
			t.Fatal(err)
		}
		writeTestSource(t, root, name, string(source))
	}
	workspace, err := workspaceloader.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	checked, err := workspacecompiler.Check(workspace)
	if err != nil || !checked.Verified {
		t.Fatalf("example check: %v %+v %+v", err, checked.Semantic.Diagnostics, checked.Semantic.Unsupported)
	}
	functions, views, stores := 0, 0, 0
	for _, function := range checked.MIR.Functions {
		if !strings.Contains(function.Name, "tokens::") {
			continue
		}
		functions++
		for _, block := range function.Blocks {
			for _, operation := range block.Operations {
				if operation.Kind == mir.OperationFieldRead {
					if _, _, array := hir.FixedArrayParts(operation.Type); array {
						t.Fatalf("%s copies the whole token field", function.Name)
					}
				}
				if operation.Kind == mir.OperationArrayIndex {
					t.Fatalf("%s indexes a copied token array", function.Name)
				}
				if operation.Kind == mir.OperationArrayBorrowView {
					views++
				}
				if operation.Kind == mir.OperationViewIndexStore || operation.Kind == mir.OperationFieldStore {
					stores++
				}
			}
		}
	}
	if functions == 0 || views == 0 || stores == 0 {
		t.Fatalf("example storage path not exercised: functions=%d views=%d stores=%d", functions, views, stores)
	}
}
