package directoryscan

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"slices"
	"testing"
	"testing/fstest"
)

type countingDirectory struct {
	fs.ReadDirFile
	requested []int
	read      int
}

func (directory *countingDirectory) ReadDir(n int) ([]fs.DirEntry, error) {
	directory.requested = append(directory.requested, n)
	entries, err := directory.ReadDirFile.ReadDir(n)
	directory.read += len(entries)
	return entries, err
}

func TestReadSortedEnforcesBoundaryBeforeMaterialization(t *testing.T) {
	for _, size := range []int{255, 256, 257} {
		files := fstest.MapFS{}
		for i := 0; i < size; i++ {
			files[string(rune(0x1000+i))] = &fstest.MapFile{}
		}
		file, err := files.Open(".")
		if err != nil {
			t.Fatal(err)
		}
		directory := &countingDirectory{ReadDirFile: file.(fs.ReadDirFile)}
		budget := &Budget{Remaining: 256}
		entries, err := ReadSorted(directory, budget)
		file.Close()
		if len(entries) != min(size, 256) || errors.Is(err, ErrLimit) != (size > 256) {
			t.Fatalf("size=%d admitted=%d err=%v", size, len(entries), err)
		}
		if directory.read > 257 {
			t.Fatalf("read beyond overflow probe: %d", directory.read)
		}
		for _, n := range directory.requested {
			if n < 1 || n > BatchSize {
				t.Fatalf("unbounded batch %d", n)
			}
		}
	}
}

type emptyDirectory struct{ fs.ReadDirFile }

func (emptyDirectory) ReadDir(int) ([]fs.DirEntry, error) { return nil, nil }

type failingDirectory struct {
	fs.ReadDirFile
	err error
}

func (directory failingDirectory) ReadDir(n int) ([]fs.DirEntry, error) {
	entries, _ := directory.ReadDirFile.ReadDir(n)
	return entries, directory.err
}

func TestReadSortedCancellationAndNoProgress(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := ReadSorted(emptyDirectory{}, &Budget{Context: ctx, Remaining: 1}); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := ReadSorted(emptyDirectory{}, &Budget{Remaining: 1}); !errors.Is(err, io.ErrNoProgress) {
		t.Fatal(err)
	}
}

func TestReadSortedReturnsSortedPartialEntriesOnReadError(t *testing.T) {
	files := fstest.MapFS{
		"entries/z": &fstest.MapFile{},
		"entries/a": &fstest.MapFile{},
	}
	original, err := files.Open("entries")
	if err != nil {
		t.Fatal(err)
	}
	want := errors.New("read failure")
	entries, readErr := ReadSorted(failingDirectory{ReadDirFile: original.(fs.ReadDirFile), err: want}, &Budget{Remaining: 4})
	original.Close()
	if !errors.Is(readErr, want) || len(entries) != 2 || entries[0].Name() != "a" || entries[1].Name() != "z" {
		t.Fatalf("entries = %v, %v", entries, readErr)
	}
}

func TestReadSortedSharesOneBudgetAcrossDirectories(t *testing.T) {
	files := fstest.MapFS{
		"first/b":  &fstest.MapFile{},
		"first/a":  &fstest.MapFile{},
		"second/d": &fstest.MapFile{},
		"second/c": &fstest.MapFile{},
	}
	budget := &Budget{Remaining: 3}
	first, err := files.Open("first")
	if err != nil {
		t.Fatal(err)
	}
	firstEntries, firstErr := ReadSorted(first.(fs.ReadDirFile), budget)
	first.Close()
	second, err := files.Open("second")
	if err != nil {
		t.Fatal(err)
	}
	secondEntries, secondErr := ReadSorted(second.(fs.ReadDirFile), budget)
	second.Close()
	if firstErr != nil || len(firstEntries) != 2 || firstEntries[0].Name() != "a" || firstEntries[1].Name() != "b" {
		t.Fatalf("first directory = %v, %v", firstEntries, firstErr)
	}
	if !errors.Is(secondErr, ErrLimit) || len(secondEntries) != 1 || secondEntries[0].Name() != "c" || budget.Remaining != 0 {
		t.Fatalf("second directory = %v, %v, remaining=%d", secondEntries, secondErr, budget.Remaining)
	}
}

type reversedDirectory struct{ fs.ReadDirFile }

func (directory reversedDirectory) ReadDir(n int) ([]fs.DirEntry, error) {
	entries, err := directory.ReadDirFile.ReadDir(n)
	slices.Reverse(entries)
	return entries, err
}

func TestReadSortedSortsAdmittedEntriesBeforeReturningPartial(t *testing.T) {
	files := fstest.MapFS{
		"entries/z": &fstest.MapFile{},
		"entries/a": &fstest.MapFile{},
		"entries/m": &fstest.MapFile{},
	}
	directory, err := files.Open("entries")
	if err != nil {
		t.Fatal(err)
	}
	entries, readErr := ReadSorted(reversedDirectory{directory.(fs.ReadDirFile)}, &Budget{Remaining: 2})
	directory.Close()
	// Enumeration is z,m,a: a is only the overflow probe, not an admitted entry.
	if !errors.Is(readErr, ErrLimit) || len(entries) != 2 || entries[0].Name() != "m" || entries[1].Name() != "z" {
		t.Fatalf("entries = %v, %v", entries, readErr)
	}
}
