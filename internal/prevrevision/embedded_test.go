package prevrevision

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMigrateEmbeddedGoTreeRewritesCompletePrograms(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	path := filepath.Join(root, "fixture_test.go")
	input := "package fixture\n\nconst source = `module main;\nfn main() -> i32 { 42 }\n`\n"
	if err := os.WriteFile(path, []byte(input), 0o644); err != nil {
		t.Fatal(err)
	}
	check, err := MigrateEmbeddedGoTree(root, false)
	if err != nil || check.FilesChanged != 1 || check.LiteralsChanged != 1 || check.FilesWritten != 0 {
		t.Fatalf("check = %+v, %v", check, err)
	}
	write, err := MigrateEmbeddedGoTree(root, true)
	if err != nil || write.FilesWritten != 1 {
		t.Fatalf("write = %+v, %v", write, err)
	}
	output, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(output), "fn i32 main() { return 42; }") {
		t.Fatalf("migrated Go source:\n%s", output)
	}
}

func TestMigrateEmbeddedGoTreeSkipsInvalidCompleteCandidate(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	path := filepath.Join(root, "fixture_test.go")
	input := "package fixture\n\nconst source = `module main;\nfn main() -> i32 {\n`\n"
	if err := os.WriteFile(path, []byte(input), 0o644); err != nil {
		t.Fatal(err)
	}
	report, err := MigrateEmbeddedGoTree(root, true)
	if err != nil || report.LiteralsSkipped != 1 || len(report.Skipped) != 1 || report.FilesWritten != 0 {
		t.Fatalf("report = %+v, %v", report, err)
	}
	output, readErr := os.ReadFile(path)
	if readErr != nil || string(output) != input {
		t.Fatalf("source changed after rejected plan: %v\n%s", readErr, output)
	}
}
