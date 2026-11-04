package packagesource

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/0disoft/fairyc/internal/directoryscan"
	"github.com/0disoft/fairyc/internal/durablefile"
)

func snapshotTreeNames(t testing.TB, root *os.Root, name string, index *treeNameIndex) {
	t.Helper()
	directory, err := durablefile.OpenDirectory(root, name)
	if err != nil {
		t.Fatal(err)
	}
	defer directory.Close()
	identity, err := directory.Stat()
	if err != nil {
		t.Fatal(err)
	}
	entries, err := directoryscan.ReadSorted(directory, &directoryscan.Budget{Remaining: MaxFiles * 2})
	if err != nil {
		t.Fatal(err)
	}
	index.remember(name, identity, entries)
}

func TestTreeNameIndexPreservesCaseAndIdentity(t *testing.T) {
	for _, change := range []string{"unchanged", "rename-case", "extra", "replace-parent"} {
		t.Run(change, func(t *testing.T) {
			root, err := os.OpenRoot(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			defer root.Close()
			if err := root.Mkdir("src", 0700); err != nil {
				t.Fatal(err)
			}
			if err := root.WriteFile("src/Module.fy", []byte("data"), 0600); err != nil {
				t.Fatal(err)
			}
			index := &treeNameIndex{root: root, dirs: make(map[string]treeDirectoryNames)}
			snapshotTreeNames(t, root, ".", index)
			snapshotTreeNames(t, root, "src", index)
			for _, name := range []string{"src/Module.fy", "src/module.fy", "SRC/Module.fy", "../Module.fy"} {
				actual, indexedErr := readRegularMatchesWithNames(context.Background(), root, name, MaxFileBytes, []byte("data"), index)
				want, oldErr := readRegularMatches(context.Background(), root, name, MaxFileBytes, []byte("data"))
				if actual != want || (indexedErr == nil) != (oldErr == nil) {
					t.Fatalf("path=%s indexed=%t/%v old=%t/%v", name, actual, indexedErr, want, oldErr)
				}
			}
			switch change {
			case "rename-case":
				// Two steps also force a case-only rename on insensitive hosts.
				if err := root.Rename("src/Module.fy", "src/temporary"); err != nil {
					t.Fatal(err)
				}
				if err := root.Rename("src/temporary", "src/module.fy"); err != nil {
					t.Fatal(err)
				}
			case "extra":
				if err := root.WriteFile("src/extra", nil, 0600); err != nil {
					t.Fatal(err)
				}
			case "replace-parent":
				if err := root.Rename("src", "old-src"); err != nil {
					t.Fatal(err)
				}
				if err := root.Mkdir("src", 0700); err != nil {
					t.Fatal(err)
				}
				if _, err := openRegularWithNames(root, "src/Module.fy", MaxFileBytes, index); err == nil {
					t.Fatal("replaced directory accepted through old index")
				}
			}
			err = index.verifyStable(context.Background(), 16)
			if (err == nil) != (change == "unchanged") {
				t.Fatalf("change=%s stable verification=%v", change, err)
			}
			// A new invocation must observe the new spelling, without stale names.
			if change == "rename-case" {
				fresh := &treeNameIndex{root: root, dirs: make(map[string]treeDirectoryNames)}
				snapshotTreeNames(t, root, ".", fresh)
				snapshotTreeNames(t, root, "src", fresh)
				if err := fresh.exactName("src", "Module.fy"); err == nil {
					t.Fatal("fresh verification retained stale spelling")
				}
				if err := fresh.exactName("src", "module.fy"); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkInstalledTreeNameAdmission(b *testing.B) {
	root, err := os.OpenRoot(b.TempDir())
	if err != nil {
		b.Fatal(err)
	}
	defer root.Close()
	const count = 512
	paths := make([]string, count)
	for i := range paths {
		paths[i] = fmt.Sprintf("module-%04d.fy", i)
		if err := root.WriteFile(paths[i], nil, 0600); err != nil {
			b.Fatal(err)
		}
	}
	for _, indexed := range []bool{false, true} {
		b.Run(fmt.Sprintf("indexed_%t", indexed), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				var index *treeNameIndex
				if indexed {
					index = &treeNameIndex{root: root, dirs: make(map[string]treeDirectoryNames)}
					snapshotTreeNames(b, root, ".", index)
				}
				for _, name := range paths {
					file, err := openRegularWithNames(root, name, MaxFileBytes, index)
					if err != nil {
						b.Fatal(err)
					}
					file.Close()
				}
				if index != nil {
					if err := index.verifyStable(context.Background(), count+1); err != nil {
						b.Fatal(err)
					}
				}
			}
		})
	}
}
