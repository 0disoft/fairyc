package prevrevision

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMigrateTreePlansBeforeWriting(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	valid := filepath.Join(root, "src", "main.fy")
	invalid := filepath.Join(root, "src", "broken.fy")
	if err := os.MkdirAll(filepath.Dir(valid), 0o755); err != nil {
		t.Fatal(err)
	}
	original := "module main;\nfn main() -> i32 { 0 }\n"
	if err := os.WriteFile(valid, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(invalid, []byte("module broken;\nfn broken(value: i32) -> i32 {\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	report, err := MigrateTree(root, true)
	if err == nil || len(report.Errors) != 1 {
		t.Fatalf("MigrateTree() = %+v, %v", report, err)
	}
	contents, readErr := os.ReadFile(valid)
	if readErr != nil || string(contents) != original {
		t.Fatalf("valid file changed on rejected plan: %q, %v", contents, readErr)
	}
}

func TestMigrateTreeWritesEligibleSourcesAndSkipsParserFailures(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	valid := filepath.Join(root, "examples", "main.fy")
	skipped := filepath.Join(root, "spec", "fixtures", "parser", "cases", "fail", "broken", "src", "main.fy")
	for _, path := range []string{valid, skipped} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(valid, []byte("module main;\nfn main() -> i32 { 0 }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	broken := "this is intentionally invalid"
	if err := os.WriteFile(skipped, []byte(broken), 0o644); err != nil {
		t.Fatal(err)
	}
	report, err := MigrateTree(root, true)
	if err != nil || report.Changed != 1 || report.Written != 1 || report.Skipped != 1 {
		t.Fatalf("MigrateTree() = %+v, %v", report, err)
	}
	contents, _ := os.ReadFile(valid)
	if !strings.Contains(string(contents), "fn i32 main() { return 0; }") {
		t.Fatalf("migrated source = %s", contents)
	}
	skippedContents, _ := os.ReadFile(skipped)
	if string(skippedContents) != broken {
		t.Fatalf("skipped source = %s", skippedContents)
	}
}
