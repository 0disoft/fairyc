package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
)

// Compare the two real lexer-candidate source edits under the existing native
// parser workload; this does not measure Fairy against a C implementation.
func TestResultBranchReviewIsOkNativeSizes(t *testing.T) {
	root, _, _ := parserLabelWorkspace(t)
	data, err := os.ReadFile(filepath.Join(root, "src", "body_state.fy"))
	if err != nil {
		t.Fatal(err)
	}
	changed := string(data)
	plain := changed
	for _, edit := range []struct{ property, previous string }{
		{"v::push<Node>(nodes, node).is_ok", "switch (v::push<Node>(nodes, node)) { case .ok(_): { true } case .err(_): { false } }"},
		{"v::push<Statement>(statements, value).is_ok", "switch (v::push<Statement>(statements, value)) { case .ok(_): { true } case .err(_): { false } }"},
	} {
		if strings.Count(plain, edit.property) != 1 {
			t.Fatalf("expected one adopted %q", edit.property)
		}
		plain = strings.Replace(plain, edit.property, edit.previous, 1)
	}
	var cBytes, functions [2]int
	for index, source := range []string{plain, changed} {
		writeTestSource(t, root, "src/body_state.fy", source)
		var diagnostics strings.Builder
		lowered, code := lowerWorkspaceExecutable(root, false, workspaceCompilePolicy{stableDiagnostics: true}, "Result is_ok C size", &diagnostics)
		if code != ExitSuccess {
			t.Fatalf("variant %d: %s", index, diagnostics.String())
		}
		unit, err := cbackend.EmitStorageTransfers(lowered.program, lowered.entryName, cbackend.Stage0WindowsMSVCProfile())
		if err != nil {
			t.Fatal(err)
		}
		cBytes[index] = len(unit.Source)
		functions[index] = len(lowered.program.Functions)
	}
	if functions[0] != functions[1] {
		t.Fatalf("MIR function count %d vs %d", functions[0], functions[1])
	}
	t.Logf("Result is_ok C bytes plain=%d changed=%d; MIR functions=%d", cBytes[0], cBytes[1], functions[1])
	compareParserNativeEdit(t, root, "src/body_state.fy", plain, changed, "RESULT_IS_OK_NATIVE_SIZES")
}
