package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStablePackageRejectsArgumentsBeforeFilesystem(t *testing.T) {
	for _, args := range [][]string{{"--release"}, {"--json", "--json"}, {"--locked", "--locked"}, {"--output", "dist"}, {"--"}, {"--json=true"}} {
		code, out, diag := runForTest(append([]string{"package"}, args...), filepath.Join(t.TempDir(), "missing"))
		if code != ExitInvalidInvocation || out != "" || diag == "" {
			t.Fatalf("%v: %d %q %q", args, code, out, diag)
		}
	}
	code, out, diag := runForTest([]string{"package", "--help"}, filepath.Join(t.TempDir(), "missing"))
	if code != 0 || diag != "" || !strings.Contains(out, "fairyc package [--locked] [--json]") {
		t.Fatal(code, out, diag)
	}
}
func TestDesktopPackageReportChecksArtifactAndContainment(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "dist"), 0755); err != nil {
		t.Fatal(err)
	}
	body := []byte("archive bytes")
	file := filepath.Join(root, "dist", "app.zip")
	if err := os.WriteFile(file, body, 0644); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(body)
	receipt := map[string]any{"ok": true, "archive": file, "zipSha256": hex.EncodeToString(sum[:]), "exeBytes": 100, "zipBytes": len(body)}
	encode := func() []byte {
		data, err := json.Marshal(receipt)
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	report, err := parseDesktopPackageReport(encode(), root)
	if err != nil || report.Archive != "dist/app.zip" || report.Schema != "fairyc.package-report/v1" {
		t.Fatal(report, err)
	}
	for key, value := range map[string]any{"archive": filepath.Join(root, "escape.zip"), "ok": false, "published": true, "signed": true, "guiLaunched": true, "inputInjected": true, "zipSha256": strings.Repeat("0", 64), "zipBytes": 1} {
		original, exists := receipt[key]
		receipt[key] = value
		if _, err := parseDesktopPackageReport(encode(), root); err == nil {
			t.Fatalf("accepted invalid %s", key)
		}
		if exists {
			receipt[key] = original
		} else {
			delete(receipt, key)
		}
	}
}
func TestDesktopPythonMissingHasActionableDiagnostic(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	if _, err := findDesktopPython(); err == nil || !strings.Contains(err.Error(), "Python 3.11+") {
		t.Fatalf("missing Python: %v", err)
	}
}
