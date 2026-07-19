package windowsabiprobe

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/target"
)

func TestGenerateRepositoryWindowsABIManifest(t *testing.T) {
	t.Parallel()
	manifest := filepath.Join("..", "..", "spec", "fixtures", "c-abi", "manifest-windows-msvc-v0.json")
	unit, err := Generate(manifest)
	if err != nil {
		t.Fatal(err)
	}
	repeated, err := Generate(manifest)
	if err != nil || repeated.CallerSource != unit.CallerSource || repeated.CalleeSource != unit.CalleeSource || !slices.Equal(repeated.Cases, unit.Cases) {
		t.Fatalf("repeated Windows ABI probe drifted: %v", err)
	}
	if unit.Target != target.X8664WindowsMSVC || len(unit.Cases) != 12 || unit.Cases[0] != "pass.fifth-float-stack" || unit.Cases[len(unit.Cases)-1] != "pass.void-return" {
		t.Fatalf("Windows ABI probe unit = %+v", unit)
	}
	for _, required := range []string{
		"fy_windows_abi_target_0(",
		"fy_windows_abi_target_11(",
		"_Static_assert(sizeof(",
		"fy_windows_abi_callback",
		"int main(void)",
	} {
		if !strings.Contains(unit.CallerSource, required) && !strings.Contains(unit.CalleeSource, required) {
			t.Errorf("Windows ABI probe is missing %q", required)
		}
	}
	for _, caseID := range unit.Cases {
		if strings.Contains(unit.CallerSource, caseID) || strings.Contains(unit.CalleeSource, caseID) {
			t.Errorf("Windows ABI probe source leaks fixture case ID %q", caseID)
		}
	}
	cloned := CloneCases(unit)
	cloned[0] = "mutated"
	if unit.Cases[0] == "mutated" {
		t.Fatal("CloneCases() leaked caller mutation")
	}
}

func TestGenerateRejectsLinuxManifest(t *testing.T) {
	t.Parallel()
	manifest := filepath.Join("..", "..", "spec", "fixtures", "c-abi", "manifest-v0.json")
	if _, err := Generate(manifest); err == nil || !strings.Contains(err.Error(), "want \"x86_64-windows-msvc\"") {
		t.Fatalf("Generate() error = %v", err)
	}
}
