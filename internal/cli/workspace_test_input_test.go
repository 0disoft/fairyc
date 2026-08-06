package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStableNativeTestInputReadsBoundedContainedFile(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeWorkspaceFile(t, root, "fixtures/input.bin", "Fairy\x00input")
	contents, err := readStableNativeTestInput(root, "fixtures/input.bin")
	if err != nil || string(contents) != "Fairy\x00input" {
		t.Fatalf("native test input = %q, %v", contents, err)
	}
}

func TestStableNativeTestInputRejectsUnsafeAndOversizedFiles(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeWorkspaceFile(t, root, "fixtures/large.bin", strings.Repeat("x", stableNativeTestInputMaximum+1))
	if _, err := readStableNativeTestInput(root, "../outside"); err == nil {
		t.Fatal("parent traversal succeeded")
	}
	if _, err := readStableNativeTestInput(root, `fixtures\large.bin`); err == nil {
		t.Fatal("host separator succeeded")
	}
	if _, err := readStableNativeTestInput(root, "fixtures"); err == nil {
		t.Fatal("directory input succeeded")
	}
	if _, err := readStableNativeTestInput(root, "fixtures/large.bin"); err == nil || !strings.Contains(err.Error(), "65536 bytes") {
		t.Fatalf("oversized input error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "fixtures", "large.bin")); err != nil {
		t.Fatal(err)
	}
}

func TestStableNativeTestInputOptionDomain(t *testing.T) {
	t.Parallel()
	options, err := parseStableTestOptions([]string{"--native", "--stdin-file", "fixtures/input.bin", "--", "--stdin-file"})
	if err != nil || options.stdinFile != "fixtures/input.bin" || len(options.programArgs) != 1 || options.programArgs[0] != "--stdin-file" {
		t.Fatalf("stdin-file options = %+v, %v", options, err)
	}
	for _, arguments := range [][]string{
		{"--stdin-file", "fixtures/input.bin"},
		{"--native", "--stdin", "--stdin-file", "fixtures/input.bin"},
		{"--list", "--stdin-file", "fixtures/input.bin"},
		{"--native", "--stdin-file"},
		{"--native", "--stdin-file", "fixtures/a", "--stdin-file", "fixtures/b"},
	} {
		if _, err := parseStableTestOptions(arguments); err == nil {
			t.Fatalf("invalid stdin-file options succeeded: %q", arguments)
		}
	}
}
