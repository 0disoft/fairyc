package prevrevision

import (
	"os"
	"path/filepath"
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
	original := "module main;\nfn i32 main() { return 0; }\n"
	if err := os.WriteFile(valid, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(invalid, []byte("module broken;\nfn i32 broken(i32 value) {\n"), 0o644); err != nil {
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

func TestMigrateTreePreservesPreviousCompoundAssignment(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	path := filepath.Join(root, "src", "main.fy")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	input := "module main;\nfn i32 main() { var i32 value = 40; value += 2; return value; }\n"
	if err := os.WriteFile(path, []byte(input), 0o644); err != nil {
		t.Fatal(err)
	}
	report, err := MigrateTree(root, true)
	if err != nil || report.Scanned != 1 || report.Skipped != 0 || report.Changed != 0 || report.Written != 0 {
		t.Fatalf("MigrateTree() = %+v, %v", report, err)
	}
	contents, readErr := os.ReadFile(path)
	if readErr != nil || string(contents) != input {
		t.Fatalf("current source changed: %q, %v", contents, readErr)
	}
}

func TestMigrateTreeSkipsCurrentDeferSource(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	path := filepath.Join(root, "src", "main.fy")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	input := "module main;\nfn void record(i32 value) {}\nfn i32 main() { defer record(42); return 42; }\n"
	if err := os.WriteFile(path, []byte(input), 0o644); err != nil {
		t.Fatal(err)
	}
	report, err := MigrateTree(root, true)
	if err != nil || report.Scanned != 1 || report.Skipped != 1 || report.Changed != 0 || report.Written != 0 {
		t.Fatalf("MigrateTree() = %+v, %v", report, err)
	}
	contents, readErr := os.ReadFile(path)
	if readErr != nil || string(contents) != input {
		t.Fatalf("current source changed: %q, %v", contents, readErr)
	}
}

func TestMigrateTreeKeepsEligibleSourcesAndSkipsParserFailures(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	valid := filepath.Join(root, "examples", "main.fy")
	skipped := filepath.Join(root, "spec", "fixtures", "parser", "cases", "fail", "broken", "src", "main.fy")
	for _, path := range []string{valid, skipped} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	original := "module main;\nfn i32 main() { return 0; }\n"
	if err := os.WriteFile(valid, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}
	broken := "this is intentionally invalid"
	if err := os.WriteFile(skipped, []byte(broken), 0o644); err != nil {
		t.Fatal(err)
	}
	report, err := MigrateTree(root, true)
	if err != nil || report.Changed != 0 || report.Written != 0 || report.Skipped != 1 {
		t.Fatalf("MigrateTree() = %+v, %v", report, err)
	}
	contents, _ := os.ReadFile(valid)
	if string(contents) != original {
		t.Fatalf("identity migration changed source = %s", contents)
	}
	skippedContents, _ := os.ReadFile(skipped)
	if string(skippedContents) != broken {
		t.Fatalf("skipped source = %s", skippedContents)
	}
}
