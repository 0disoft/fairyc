package cli

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/atlasmap"
)

func impactArgs(uid, symbol string) []string {
	return []string{"--package-uid", uid, "--symbol", symbol, "--max-packages", "8", "--max-calls", "128", "--max-depth", "8", "--max-symbols", "64", "--max-output-bytes", "100000"}
}

func TestAtlasImpactLockedCurrentAndReadOnly(t *testing.T) {
	root := writeWorkspaceSessionFixture(t, workspaceSessionRootSource)
	s := atlasFunctionSymbolForPackage(t, root, "example.session.answer", "value")
	args := append([]string{"experimental", "atlas-impact"}, impactArgs("example.session.answer", s.Visible)...)
	code, out, err := runForTest(args, root)
	if code != 0 || err != "" || strings.Contains(out, root) {
		t.Fatalf("impact %d %q %q", code, out, err)
	}
	var report atlasmap.ImpactResult
	if json.Unmarshal([]byte(out), &report) != nil || len(report.Nodes) != 2 || report.State == "exact" {
		t.Fatal(out)
	}
	writeWorkspaceFile(t, root, "src/main.fy", workspaceSessionRootSource+"\n")
	if code, out, _ := runForTest(args, root); code != ExitPackageFailure || out != "" {
		t.Fatal("stale lock admitted")
	}
}
