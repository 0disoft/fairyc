package workspaceloader

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestSourceWalkerPreservesLexicalOrderAndCountsIgnoredEntries(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"z.txt", "b/main.fy", "a.fy", "b/ignored.txt"} {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(root, name)), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, name), nil, 0600); err != nil {
			t.Fatal(err)
		}
	}
	var want []string
	if err := fs.WalkDir(os.DirFS(root), ".", func(name string, entry fs.DirEntry, err error) error { want = append(want, name); return err }); err != nil {
		t.Fatal(err)
	}
	for _, maximum := range []int{len(want) - 1, len(want), len(want) + 1} {
		var got []string
		budget := &limits{}
		err := walkSourceTree(os.DirFS(root), budget, maximum, func(name string, entry fs.DirEntry, err error) error {
			if err == nil {
				got = append(got, name)
			}
			return err
		})
		if maximum < len(want) {
			var load *LoadError
			if !errors.As(err, &load) || load.Reason != ReasonSourceLimit {
				t.Fatalf("entry bound: %v", err)
			}
		} else if err != nil || !reflect.DeepEqual(got, want) || budget.entries != len(want) {
			t.Fatalf("walk=%v/%d, %v; want %v", got, budget.entries, err, want)
		}
	}
}

func TestSourceWalkerStopsWideDirectoriesAndCancellation(t *testing.T) {
	root := t.TempDir()
	for i := range sourceDirectoryBatch + 5 {
		if err := os.WriteFile(filepath.Join(root, fmt.Sprintf("%04d.txt", i)), nil, 0600); err != nil {
			t.Fatal(err)
		}
	}
	budget := &limits{}
	err := walkSourceTree(os.DirFS(root), budget, sourceDirectoryBatch, func(_ string, _ fs.DirEntry, err error) error { return err })
	var load *LoadError
	if !errors.As(err, &load) || load.Reason != ReasonSourceLimit || budget.entries > sourceDirectoryBatch {
		t.Fatalf("wide directory: entries=%d err=%v", budget.entries, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := walkSourceTree(os.DirFS(root), &limits{ctx: ctx}, MaxSourceEntries, func(_ string, _ fs.DirEntry, err error) error { return err }); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation: %v", err)
	}
}

func TestDirectorySpellingEnumerationSharesEntryGuard(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"src", "other"} {
		if err := os.Mkdir(filepath.Join(root, name), 0700); err != nil {
			t.Fatal(err)
		}
	}
	resolver := newDirectoryResolver()
	resolver.budget.entries = MaxSourceEntries - 1
	_, err := resolver.resolve(root, "src")
	var load *LoadError
	if !errors.As(err, &load) || load.Reason != ReasonSourceLimit {
		t.Fatalf("spelling scan bypassed entry guard: %v", err)
	}
}
