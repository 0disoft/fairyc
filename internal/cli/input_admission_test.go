package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadSourceAdmitsOrdinaryFilesAndRejectsSpecialPaths(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "main.fy"), []byte("module main;\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "directory"), 0o700); err != nil {
		t.Fatal(err)
	}
	loaded, err := loadSource(root, "main.fy", 1024)
	if err != nil || loaded.file.RelativePath != "main.fy" || string(loaded.file.Bytes) != "module main;\n" {
		t.Fatalf("loadSource() = %+v, %v", loaded, err)
	}
	if _, err := loadSource(root, "directory", 1024); err == nil || !strings.Contains(err.Error(), "not a regular file") {
		t.Fatalf("directory input error = %v", err)
	}
	if err := os.Symlink("main.fy", filepath.Join(root, "link.fy")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := loadSource(root, "link.fy", 1024); err == nil || !strings.Contains(err.Error(), "not a regular file") {
		t.Fatalf("symlink input error = %v", err)
	}
}
