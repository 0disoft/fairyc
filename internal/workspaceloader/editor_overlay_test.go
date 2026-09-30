package workspaceloader

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEditorOverlayPreservesDiskAndLoadsNewSources(t *testing.T) {
	root := t.TempDir()
	manifest := "schema = \"fairyc.manifest/v1\"\n[package]\nuid = \"editor.test\"\nname = \"editor-test\"\nversion = \"1.0.0\"\nlanguage = \"fairyc-language/20\"\nsource = \"src\"\n"
	if err := os.WriteFile(filepath.Join(root, "fairy.toml"), []byte(manifest), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "src"), 0755); err != nil {
		t.Fatal(err)
	}
	filename := filepath.Join(root, "src", "main.fy")
	original := []byte("module main; fn i32 main() { return 1; }")
	if err := os.WriteFile(filename, original, 0644); err != nil {
		t.Fatal(err)
	}
	workspace, err := LoadEditorOverlays(root, map[string][]byte{filename: []byte("module main; fn i32 main() { return 2; }"), filepath.Join(root, "src", "other.fy"): []byte("module other; pub fn i32 value() { return 3; }")})
	if err != nil {
		t.Fatal(err)
	}
	loaded, _ := workspace.Package(".")
	if len(loaded.Sources) != 2 {
		t.Fatalf("sources: %d", len(loaded.Sources))
	}
	unchanged, _ := os.ReadFile(filename)
	if string(unchanged) != string(original) {
		t.Fatal("overlay wrote source")
	}
	if _, err := os.Stat(filepath.Join(root, "src", "other.fy")); !os.IsNotExist(err) {
		t.Fatal("new source written")
	}
	if _, err := LoadEditorOverlays(root, map[string][]byte{filepath.Join(root, "..", "escape.fy"): original}); err == nil {
		t.Fatal("outside overlay accepted")
	}
	if _, err := LoadEditorOverlays(root, map[string][]byte{filepath.Join(root, "fairy.toml"): original}); err == nil {
		t.Fatal("manifest overlay accepted")
	}
}
