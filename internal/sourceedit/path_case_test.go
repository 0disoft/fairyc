package sourceedit

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/projectmanifest"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

func editFile(t *testing.T, root, name, text string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestSnapshotCaseIndexReadsSharedParentsOnce(t *testing.T) {
	const members = 256
	root := t.TempDir()
	editFile(t, root, "fairy.toml", "root")
	editFile(t, root, "fairy.lock", "lock")
	opened, err := os.OpenRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	defer opened.Close()
	index := newPathCaseIndex(opened)
	open := index.open
	opens := map[string]int{}
	index.open = func(parent string) (pathCaseDirectory, error) {
		opens[parent]++
		return open(parent)
	}
	paths := []string{"fairy.toml"}
	for i := range members {
		name := fmt.Sprintf("libs/p%04d/fairy.toml", i)
		editFile(t, root, name, "member")
		paths = append(paths, name)
	}
	paths = append(paths, "fairy.lock")
	for _, name := range paths {
		want, err := ReadFile(opened, name, projectmanifest.MaxBytes)
		if err != nil {
			t.Fatal(err)
		}
		got, err := readFile(opened, name, projectmanifest.MaxBytes, index)
		if err != nil || string(got) != string(want) {
			t.Fatalf("%s: %q, %v", name, got, err)
		}
	}
	if len(opens) != members+2 || index.entries != 2*members+3 {
		t.Fatalf("opens=%d, retained entries=%d", len(opens), index.entries)
	}
	for parent, count := range opens {
		if count != 1 {
			t.Fatalf("directory %s enumerated %d times", parent, count)
		}
	}
	t.Logf("directory enumerations: %d -> %d; shared-parent enumerations: %d -> 2", 3*members+2, len(opens), 2*members+2)
}

func TestSnapshotCaseIndexPreservesReadAdmission(t *testing.T) {
	root := t.TempDir()
	editFile(t, root, "libs/a/fairy.toml", "data")
	opened, err := os.OpenRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	defer opened.Close()
	index := newPathCaseIndex(opened)
	for _, max := range []int64{3, 4, 5} {
		got, err := readFile(opened, "libs/a/fairy.toml", max, index)
		if max < 4 {
			if err == nil || got != nil {
				t.Fatal("oversized input admitted")
			}
		} else if err != nil || string(got) != "data" {
			t.Fatalf("size %d: %q, %v", max, got, err)
		}
	}
	for _, name := range []string{"Libs/a/fairy.toml", "libs/A/fairy.toml", "libs/a/FAIRY.toml", "libs/a", "../outside", "/absolute", "libs/../fairy.toml", "libs\\a\\fairy.toml"} {
		if _, err := readFile(opened, name, 100, index); err == nil {
			t.Fatalf("invalid input admitted: %s", name)
		}
	}
	if err := os.Remove(filepath.Join(root, "libs", "a", "fairy.toml")); err != nil {
		t.Fatal(err)
	}
	if _, err := readFile(opened, "libs/a/fairy.toml", 100, index); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("cached name hid removed file: %v", err)
	}
	if err := os.Mkdir(filepath.Join(root, "libs", "a", "fairy.toml"), 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := readFile(opened, "libs/a/fairy.toml", 100, index); err == nil {
		t.Fatal("cached file replaced by directory admitted")
	}
}

func TestSnapshotCaseIndexRechecksSymlinks(t *testing.T) {
	for _, directory := range []bool{false, true} {
		t.Run(fmt.Sprint(directory), func(t *testing.T) {
			root := t.TempDir()
			editFile(t, root, "libs/a/fairy.toml", "data")
			editFile(t, root, "other/fairy.toml", "other")
			opened, err := os.OpenRoot(root)
			if err != nil {
				t.Fatal(err)
			}
			defer opened.Close()
			index := newPathCaseIndex(opened)
			if _, err := readFile(opened, "libs/a/fairy.toml", 100, index); err != nil {
				t.Fatal(err)
			}
			link := filepath.Join(root, "libs", "a", "fairy.toml")
			target := filepath.Join(root, "other", "fairy.toml")
			if directory {
				link, target = filepath.Dir(link), filepath.Dir(target)
				if err := os.Remove(filepath.Join(link, "fairy.toml")); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.Remove(link); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(target, link); err != nil {
				t.Skipf("symlink unavailable: %v", err)
			}
			if _, err := readFile(opened, "libs/a/fairy.toml", 100, index); err == nil || !strings.Contains(err.Error(), "symlink") {
				t.Fatalf("cached path replacement admitted: %v", err)
			}
		})
	}
}

type caseEntry struct{ name string }

func (e caseEntry) Name() string             { return e.name }
func (caseEntry) IsDir() bool                { return false }
func (caseEntry) Type() os.FileMode          { return 0 }
func (caseEntry) Info() (os.FileInfo, error) { return nil, os.ErrInvalid }

type caseDirectory struct {
	remaining, reads, closes int
	failure, closeError      error
}

func (d *caseDirectory) ReadDir(n int) ([]os.DirEntry, error) {
	d.reads++
	count := min(n, d.remaining)
	entries := make([]os.DirEntry, count)
	for i := range entries {
		// Repeated names deliberately still consume the enumeration allowance.
		entries[i] = caseEntry{name: "fairy.toml"}
	}
	d.remaining -= count
	if d.failure != nil {
		return entries, d.failure
	}
	if d.remaining == 0 {
		return entries, io.EOF
	}
	return entries, nil
}

func (d *caseDirectory) Close() error { d.closes++; return d.closeError }

func TestSnapshotCaseIndexEntryBoundAndCleanup(t *testing.T) {
	for _, count := range []int{workspaceloader.MaxSourceEntries - 1, workspaceloader.MaxSourceEntries, workspaceloader.MaxSourceEntries + 1} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			d := &caseDirectory{remaining: count}
			index := &pathCaseIndex{listings: map[string]map[string]struct{}{}, open: func(string) (pathCaseDirectory, error) { return d, nil }}
			found, err := index.contains(".", "fairy.toml")
			if count <= workspaceloader.MaxSourceEntries {
				if err != nil || !found || index.entries != count {
					t.Fatalf("exact admission: %t, %v, %d", found, err, index.entries)
				}
			} else if err == nil || found || len(index.listings) != 0 {
				t.Fatalf("overflow cached or accepted: %t, %v", found, err)
			}
			if d.closes != 1 {
				t.Fatalf("directory closed %d times", d.closes)
			}
		})
	}
	// The ceiling is shared between parent directories, not reset per parent.
	d := &caseDirectory{remaining: 1}
	index := &pathCaseIndex{entries: workspaceloader.MaxSourceEntries, listings: map[string]map[string]struct{}{}, open: func(string) (pathCaseDirectory, error) { return d, nil }}
	if _, err := index.contains("next", "fairy.toml"); err == nil || d.closes != 1 || len(index.listings) != 0 {
		t.Fatalf("shared ceiling: %v, closes %d", err, d.closes)
	}
}

func TestSnapshotCaseIndexDoesNotCacheFailures(t *testing.T) {
	for _, duringClose := range []bool{false, true} {
		t.Run(fmt.Sprint(duringClose), func(t *testing.T) {
			failure := errors.New("directory I/O failed")
			d := &caseDirectory{remaining: 1, failure: failure}
			if duringClose {
				d.failure, d.closeError = nil, failure
			}
			index := &pathCaseIndex{listings: map[string]map[string]struct{}{}, open: func(string) (pathCaseDirectory, error) { return d, nil }}
			if found, err := index.contains(".", "fairy.toml"); found || !errors.Is(err, failure) || len(index.listings) != 0 || d.closes != 1 {
				t.Fatalf("partial/error listing retained: %t, %v, closes %d", found, err, d.closes)
			}
			d.remaining, d.failure, d.closeError = 1, nil, nil
			if found, err := index.contains(".", "fairy.toml"); !found || err != nil || d.closes != 2 {
				t.Fatalf("retry: %t, %v, closes %d", found, err, d.closes)
			}
		})
	}
}
