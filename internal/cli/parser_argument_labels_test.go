package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
)

func TestArgumentLabelsParserAdoption(t *testing.T) {
	root := t.TempDir()
	copyLexerStreamExample(t, root)
	for _, name := range []string{"syntax.fy", "integrity.fy"} {
		data, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "src", name))
		if err != nil {
			t.Fatal(err)
		}
		writeTestSource(t, root, "src/"+name, string(data))
	}
	writeTestSource(t, root, "src/main.fy", parserCorpusProgram)
	path := filepath.Join(root, "src", "bodies.fy")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	labeled := string(data)
	for _, label := range []string{".minimum = ", ".depth = ", ".allow_record = "} {
		if strings.Count(labeled, label) != 11 {
			t.Fatalf("expected 11 adopted calls for %q", label)
		}
	}
	plain := strings.NewReplacer(".minimum = ", "", ".depth = ", "", ".allow_record = ", "").Replace(labeled)
	var sources [2]string
	var functions [2]int
	for i, input := range []string{plain, labeled} {
		writeTestSource(t, root, "src/bodies.fy", input)
		var diagnostics strings.Builder
		lowered, code := lowerWorkspaceExecutable(root, false, workspaceCompilePolicy{stableDiagnostics: true}, "parser argument labels", &diagnostics)
		if code != ExitSuccess {
			t.Fatalf("variant %d: %s", i, diagnostics.String())
		}
		unit, err := cbackend.EmitStorageTransfers(lowered.program, lowered.entryName, cbackend.Stage0WindowsMSVCProfile())
		if err != nil {
			t.Fatal(err)
		}
		sources[i] = unit.Source
		functions[i] = len(lowered.program.Functions)
	}
	if functions[0] != functions[1] {
		t.Fatalf("function count changed: %d vs %d", functions[0], functions[1])
	}
	// Closed-graph nominal identities hash source ASTs, so labels can change
	// specialization order and disposable C names. Compare raw size without
	// erasing those changes; this is not a runtime or compile-time measurement.
	if len(sources[1]) > len(sources[0]) {
		t.Fatalf("generated C grew: %d vs %d", len(sources[0]), len(sources[1]))
	}
	t.Logf("11 adopted calls; C bytes plain=%d labeled=%d growth=%.4f%% functions=%d byte_identical=%t",
		len(sources[0]), len(sources[1]), 100*(float64(len(sources[1]))/float64(len(sources[0]))-1), functions[1], sources[0] == sources[1])
}
