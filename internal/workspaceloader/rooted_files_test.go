package workspaceloader

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSourceBoundaryKeepsOpenedDirectoryAfterReplacement(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "src")
	if err := os.Mkdir(source, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "main.fy"), []byte("original"), 0600); err != nil {
		t.Fatal(err)
	}
	boundary, err := openWorkspaceRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	defer boundary.Close()
	opened, err := openContainedDirectory(boundary, source)
	if err != nil {
		t.Fatal(err)
	}
	defer opened.Close()
	if err := os.Rename(source, filepath.Join(root, "old")); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(source, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "main.fy"), []byte("replacement"), 0600); err != nil {
		t.Fatal(err)
	}
	got, err := readBoundedRootFile(opened, "main.fy", 32)
	if err != nil || string(got) != "original" {
		t.Fatalf("opened source redirected: %q, %v", got, err)
	}
	for _, name := range []string{"../src/main.fy", filepath.Join(root, "src", "main.fy")} {
		if _, err := readBoundedRootFile(opened, name, 32); err == nil {
			t.Fatalf("accepted escaping file %q", name)
		}
	}
}

func TestSourceBoundaryRejectsExternalDirectoryLink(t *testing.T) {
	root, outside := t.TempDir(), t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "secret.fy"), []byte("outside"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "src")); err != nil {
		t.Skipf("directory symlinks unavailable: %v", err)
	}
	boundary, err := openWorkspaceRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	defer boundary.Close()
	if child, err := openContainedDirectory(boundary, filepath.Join(root, "src")); err == nil {
		child.Close()
		t.Fatal("accepted external directory link")
	}
	if _, err := readBoundedRootFile(boundary, filepath.Join("src", "secret.fy"), 32); err == nil {
		t.Fatal("read outside opened workspace")
	}
}
