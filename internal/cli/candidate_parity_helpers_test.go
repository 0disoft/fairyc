package cli

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/source"
)

// Native candidate help covers the shared CLI commands. Registry resolution,
// registry update, and Git snapshots currently belong to the Go frontend.
func candidateUsage() string {
	lines := strings.Split(usage, "\n")
	lines = slices.DeleteFunc(lines, func(line string) bool {
		return strings.HasPrefix(line, "  fairyc experimental registry-resolve ") ||
			strings.HasPrefix(line, "  fairyc experimental registry-update ") ||
			strings.HasPrefix(line, "  fairyc experimental git-snapshot ")
	})
	return strings.Join(lines, "\n")
}

// Copy only the actual authored import closure. Whole-compiler rebuilding is a
// separate acceptance gate, not a prerequisite for every small parity probe.
func copyCandidateParityModules(t *testing.T, root string, names ...string) {
	t.Helper()
	seen := map[string]bool{}
	for len(names) > 0 {
		name := names[len(names)-1]
		names = names[:len(names)-1]
		if seen[name] || strings.HasPrefix(name, "std::") {
			continue
		}
		seen[name] = true
		if strings.ContainsAny(name, "/\\:") {
			t.Fatalf("unexpected private module %q", name)
		}
		data, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "src", name+".fy"))
		if err != nil {
			t.Fatal(err)
		}
		parsed, err := frontend.AnalyzeSyntax(source.NewFile(name+".fy", data), source.DefaultLimits())
		if err != nil || parsed.AST == nil || len(parsed.Tree.Diagnostics) != 0 {
			t.Fatalf("parse authored module %s: %v; diagnostics=%+v", name, err, parsed.Tree.Diagnostics)
		}
		for _, imported := range parsed.AST.Imports {
			names = append(names, imported.Path)
		}
		writeTestSource(t, root, "src/"+name+".fy", string(data))
	}
}
