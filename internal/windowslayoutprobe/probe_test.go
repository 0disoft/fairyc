package windowslayoutprobe

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/target"
)

func TestGenerateRepositoryWindowsManifest(t *testing.T) {
	t.Parallel()
	manifest := filepath.Join("..", "..", "spec", "fixtures", "layout", "manifest-windows-msvc-v0.json")
	unit, err := Generate(manifest)
	if err != nil {
		t.Fatal(err)
	}
	repeated, err := Generate(manifest)
	if err != nil || repeated.Source != unit.Source || !slices.Equal(repeated.Cases, unit.Cases) {
		t.Fatalf("repeated Windows layout probe drifted: %v", err)
	}
	if unit.Target != target.X8664WindowsMSVC || len(unit.Cases) != 8 || !strings.HasSuffix(unit.Source, "int main(void) { return 0; }\n") {
		t.Fatalf("Windows layout probe unit = %+v", unit)
	}
	for _, required := range []string{
		"typedef long",
		"typedef unsigned long",
		"typedef size_t",
		"typedef intptr_t",
		"typedef void *",
		"typedef void (*",
		"_Static_assert(sizeof(",
		"_Static_assert(_Alignof(",
		"_Static_assert(offsetof(",
	} {
		if !strings.Contains(unit.Source, required) {
			t.Errorf("Windows layout probe is missing %q", required)
		}
	}
	for _, forbidden := range []string{"_POSIX_C_SOURCE", "sys/types.h", "ssize_t"} {
		if strings.Contains(unit.Source, forbidden) {
			t.Errorf("Windows layout probe contains forbidden %q", forbidden)
		}
	}
	for _, caseID := range unit.Cases {
		if strings.Contains(unit.Source, caseID) {
			t.Errorf("Windows layout probe source leaks fixture case ID %q", caseID)
		}
	}
}

func TestGenerateRejectsLinuxManifest(t *testing.T) {
	t.Parallel()
	manifest := filepath.Join("..", "..", "spec", "fixtures", "layout", "manifest-v0.json")
	if _, err := Generate(manifest); err == nil || !strings.Contains(err.Error(), "want \"x86_64-windows-msvc\"") {
		t.Fatalf("Generate() error = %v", err)
	}
}
