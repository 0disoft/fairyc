package workspaceloader

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDirectoryResolverReadsSharedParentsOnce(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"a", "b", "c"} {
		if err := os.MkdirAll(filepath.Join(root, "libs", name), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	resolver := newDirectoryResolver()
	reads := make(map[string]int)
	resolver.readDir = func(path string) ([]os.DirEntry, error) {
		reads[path]++
		return os.ReadDir(path)
	}
	for _, name := range []string{"a", "b", "c", "a"} {
		got, err := resolver.resolve(root, "libs/"+name)
		if err != nil || got != filepath.Join(root, "libs", name) {
			t.Fatalf("resolve %q = %q, %v", name, got, err)
		}
	}
	if len(reads) != 2 || reads[root] != 1 || reads[filepath.Join(root, "libs")] != 1 {
		t.Fatalf("directory reads = %v", reads)
	}
	// A cached spelling must not hide removal or replacement by a regular file.
	path := filepath.Join(root, "libs", "a")
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	_, err := resolver.resolve(root, "libs/a")
	var failure *LoadError
	if !errors.As(err, &failure) || failure.Reason != ReasonPath || !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("removed directory = %v", err)
	}
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	_, err = resolver.resolve(root, "libs/a")
	if !errors.As(err, &failure) || failure.Reason != ReasonPath || failure.Detail != "path is not a directory" {
		t.Fatalf("directory replaced by file = %v", err)
	}
}

type namedDirectoryEntry struct {
	os.DirEntry
	name string
}

func (e namedDirectoryEntry) Name() string { return e.name }

func TestDirectoryResolverPreservesFoldCollisions(t *testing.T) {
	// Synthetic listings exercise case-sensitive-host collisions on Windows too.
	for _, pair := range [][2]string{{"Net", "net"}, {"Σ", "ς"}, {"K", "K"}, {"S", "ſ"}} {
		resolver := newDirectoryResolver()
		resolver.readDir = func(string) ([]os.DirEntry, error) {
			return []os.DirEntry{namedDirectoryEntry{name: pair[0]}, namedDirectoryEntry{name: pair[1]}}, nil
		}
		for _, name := range pair {
			_, err := resolver.resolve(t.TempDir(), name)
			var failure *LoadError
			if !errors.As(err, &failure) || failure.Reason != ReasonPathCase || failure.Path != name {
				t.Fatalf("collision %v, resolve %q: %v", pair, name, err)
			}
		}
	}
	for _, left := range []string{"Net", "net", "Σ", "σ", "ς", "K", "k", "K", "S", "s", "ſ", "İ", "i", "ı", "ß", "ss"} {
		for _, right := range []string{"Net", "net", "Σ", "σ", "ς", "K", "k", "K", "S", "s", "ſ", "İ", "i", "ı", "ß", "ss"} {
			if (foldDirectoryName(left) == foldDirectoryName(right)) != strings.EqualFold(left, right) {
				t.Fatalf("fold mismatch for %q and %q", left, right)
			}
		}
	}
}

func TestDirectoryResolverDoesNotCacheReadErrors(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "src"), 0o700); err != nil {
		t.Fatal(err)
	}
	resolver := newDirectoryResolver()
	reads := 0
	resolver.readDir = func(path string) ([]os.DirEntry, error) {
		reads++
		if reads == 1 {
			return nil, os.ErrPermission
		}
		return os.ReadDir(path)
	}
	if _, err := resolver.resolve(root, "src"); !errors.Is(err, os.ErrPermission) {
		t.Fatalf("read error = %v", err)
	}
	if _, err := resolver.resolve(root, "src"); err != nil || reads != 2 {
		t.Fatalf("retry = %v, reads %d", err, reads)
	}
}

func TestDirectoryResolverRechecksSymlinkAfterCacheHit(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "src")
	if err := os.Mkdir(path, 0o700); err != nil {
		t.Fatal(err)
	}
	resolver := newDirectoryResolver()
	if _, err := resolver.resolve(root, "src"); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(t.TempDir(), path); err != nil {
		t.Skipf("host cannot create directory symlink: %v", err)
	}
	_, err := resolver.resolve(root, "src")
	var failure *LoadError
	if !errors.As(err, &failure) || failure.Reason != ReasonSymlink {
		t.Fatalf("cached path replaced by symlink = %v", err)
	}
}

func TestLoadRefreshesDirectoryListings(t *testing.T) {
	root := t.TempDir()
	writePackage(t, root, ".", rootManifest(""), validRootSource())
	if _, err := Load(root); err != nil {
		t.Fatal(err)
	}
	writePackage(t, root, "libs/net", memberManifest("example.net", "net", "1.0.0", "fairyc-language/20", ""), validMemberSource("net"))
	writePackage(t, root, ".", rootManifest("\n[workspace]\nmembers = [\"libs/net\"]\n"), validRootSource())
	workspace, err := Load(root)
	if err != nil || len(workspace.Packages) != 2 {
		t.Fatalf("reload = %v, packages %d", err, len(workspace.Packages))
	}
	writePackage(t, root, ".", rootManifest("\n[workspace]\nmembers = [\"libs/Net\"]\n"), validRootSource())
	_, err = Load(root)
	var failure *LoadError
	if !errors.As(err, &failure) || failure.Reason != ReasonPathCase {
		t.Fatalf("reload with wrong case = %v", err)
	}
}

func BenchmarkResolveSiblingDirectories(b *testing.B) {
	for _, count := range []int{8, 1023} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			root := b.TempDir()
			paths := make([]string, count)
			for index := range paths {
				paths[index] = fmt.Sprintf("libs/p%04d", index)
				if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(paths[index])), 0o700); err != nil {
					b.Fatal(err)
				}
			}
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				resolver := newDirectoryResolver()
				for _, path := range paths {
					if _, err := resolver.resolve(root, path); err != nil {
						b.Fatal(err)
					}
				}
			}
		})
	}
}

func BenchmarkLoadSiblingWorkspace(b *testing.B) {
	workspace := batchTestWorkspace(b, 512)
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		loaded, err := Load(workspace.Root)
		if err != nil || len(loaded.Packages) != 512 {
			b.Fatalf("Load: %v, packages %d", err, len(loaded.Packages))
		}
	}
}
