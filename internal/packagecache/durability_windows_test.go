//go:build windows

package packagecache

import (
	"os"
	"testing"
)

func TestWindowsCacheRenameStaysRelativeToOpenedRoot(t *testing.T) {
	rootPath := t.TempDir()
	root, err := os.OpenRoot(rootPath)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if err := root.WriteFile("temporary", []byte("payload"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := renameCacheGeneration(root, "temporary", "published"); err != nil {
		t.Fatal(err)
	}
	if err := syncPublishedCacheGeneration(root, "published", "."); err != nil {
		t.Fatal(err)
	}
	contents, err := os.ReadFile(rootPath + string(os.PathSeparator) + "published")
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) != "payload" {
		t.Fatalf("published contents = %q", contents)
	}
}
