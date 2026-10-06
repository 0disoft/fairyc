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
	for _, name := range []string{"fairy.toml", "src/main.fy", "src/scanner.fy", "src/numbers.fy", "src/diagnostics.fy",
		"src/utf8.fy", "src/strings.fy", "src/budget.fy", "src/file_cli.fy", "src/token_output.fy", "src/declaration_output.fy", "src/name_cli.fy", "src/name_resolution.fy", "src/binding_output.fy",
		"src/type_control.fy", "src/type_state.fy", "src/array_types.fy", "src/literal_types.fy", "src/nominal_types.fy", "src/type_check.fy", "src/type_cli.fy",
		"src/module_registry.fy", "src/module_cli.fy", "src/module_check.fy",
		"src/hir_model.fy", "src/hir_shapes.fy", "src/hir_shape_lower.fy", "src/hir_state.fy", "src/hir_lower.fy", "src/hir_view_check.fy", "src/hir_cli.fy", "src/hir_output.fy",
		"src/mir_model.fy", "src/mir_state.fy", "src/mir_nominal.fy", "src/mir_arrays.fy", "src/mir_cleanup.fy", "src/mir_verify.fy", "src/mir_verify_shape.fy", "src/mir_verify_nominal.fy", "src/mir_verify_arrays.fy", "src/mir_verify_literals.fy", "src/mir_verify_flow.fy", "src/mir_lower.fy", "src/mir_cli.fy", "src/mir_output.fy",
		"src/c_flow.fy", "src/c_text.fy", "src/c_nominal.fy", "src/c_arrays.fy", "src/c_literals.fy", "src/c_scalar.fy", "src/c_guard.fy", "src/c_emit.fy", "src/c_cli.fy",
		"src/syntax.fy", "src/types.fy", "src/records.fy", "src/bodies.fy", "src/body_state.fy", "src/attributes.fy", "src/cleanup_syntax.fy", "src/parse_tokens.fy"} {
		data, err := os.ReadFile(filepath.Join(example, name))
		if err != nil {
			t.Fatal(err)
		}
		writeTestSource(t, root, name, string(data))
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
