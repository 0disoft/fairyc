package packagesource

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestReadRegularMatchesBoundariesAndOwnership(t *testing.T) {
	dir := t.TempDir()
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	for _, size := range []int{0, 1, sourceCompareBufferBytes - 1, sourceCompareBufferBytes, sourceCompareBufferBytes + 1, MaxFileBytes, MaxFileBytes + 1} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			contents := bytes.Repeat([]byte{'x'}, size)
			if err := root.WriteFile("data.bin", contents, 0600); err != nil {
				t.Fatal(err)
			}
			// Compare to the existing bounded reader, including identical errors.
			actual, readErr := readRegular(root, "data.bin", MaxFileBytes)
			variants := [][]byte{contents, append(bytes.Clone(contents), 'y')}
			if size > 0 {
				changed := bytes.Clone(contents)
				changed[size-1] = 'y'
				variants = append(variants, contents[:size-1], changed)
			}
			for _, expected := range variants {
				before := bytes.Clone(expected)
				matches, err := readRegularMatches(context.Background(), root, "data.bin", MaxFileBytes, expected)
				if readErr != nil {
					if err == nil || err.Error() != readErr.Error() {
						t.Fatalf("size=%d: read error=%v, compare error=%v", size, readErr, err)
					}
				} else if err != nil || matches != bytes.Equal(actual, expected) {
					t.Fatalf("size=%d expected=%d: matches=%v error=%v", size, len(expected), matches, err)
				}
				if !bytes.Equal(before, expected) {
					t.Fatal("comparison modified the expected bytes")
				}
			}
		})
	}
}

func TestReadRegularMatchesPreservesPathAdmission(t *testing.T) {
	dir := t.TempDir()
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if err := root.Mkdir("folder", 0700); err != nil {
		t.Fatal(err)
	}
	if err := root.WriteFile("data.bin", []byte("xx"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"folder", "missing", "../escape", "DATA.bin"} {
		_, want := readRegular(root, name, MaxFileBytes)
		_, got := readRegularMatches(context.Background(), root, name, MaxFileBytes, nil)
		if want == nil || got == nil || got.Error() != want.Error() {
			t.Fatalf("path %q: compare=%v, read=%v", name, got, want)
		}
	}
}

type comparisonCheckContext struct {
	context.Context
	onCheck func(int)
	checks  int
}

func (ctx *comparisonCheckContext) Err() error {
	ctx.checks++
	ctx.onCheck(ctx.checks)
	return ctx.Context.Err()
}

func TestVerifyTreeStreamMismatchAndCancellation(t *testing.T) {
	dir := t.TempDir()
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if err := root.Mkdir("package", 0700); err != nil {
		t.Fatal(err)
	}
	contents := bytes.Repeat([]byte{'x'}, sourceCompareBufferBytes*2+1)
	if err := root.WriteFile("package/data.bin", contents, 0600); err != nil {
		t.Fatal(err)
	}
	changed := bytes.Clone(contents)
	changed[0] = 'y'
	if err := verifyTree(context.Background(), root, "package", bundle{files: map[string][]byte{"data.bin": changed}}); err == nil || err.Error() != "existing package differs from pinned archive" {
		t.Fatalf("mismatch error changed: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	during := &comparisonCheckContext{Context: ctx, onCheck: func(check int) {
		if check == 2 {
			cancel()
		}
	}}
	if _, err := readRegularMatches(during, root, "package/data.bin", MaxFileBytes, contents); !errors.Is(err, context.Canceled) || during.checks != 2 {
		t.Fatalf("cancellation between chunks: checks=%d error=%v", during.checks, err)
	}
	if err := verifyTree(ctx, root, "package", bundle{files: map[string][]byte{"data.bin": contents}}); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled tree verification succeeded: %v", err)
	}
	// A completed or cancelled comparison must release its file before return.
	if err := root.Remove("package/data.bin"); err != nil {
		t.Fatalf("remove file after comparison: %v", err)
	}
}

func TestReadRegularMatchesGrowthLimitPrecedesMismatch(t *testing.T) {
	root, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if err := root.WriteFile("data.bin", []byte("xxxxxxxx"), 0600); err != nil {
		t.Fatal(err)
	}
	ctx := &comparisonCheckContext{Context: context.Background(), onCheck: func(check int) {
		if check != 1 {
			return
		}
		// Both identity checks have completed. Grow the same opened file
		// before its first read, without a timing-dependent goroutine.
		f, err := root.OpenFile("data.bin", os.O_WRONLY|os.O_APPEND, 0600)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		if _, err := f.Write([]byte("x")); err != nil {
			t.Fatal(err)
		}
	}}
	if _, err := readRegularMatches(ctx, root, "data.bin", 8, []byte("yyyyyyyy")); err == nil || err.Error() != "source file byte limit exceeded" {
		t.Fatalf("growth byte limit must precede content mismatch: %v", err)
	}
}

func BenchmarkVerifyInstalledPackage(b *testing.B) {
	dir := b.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "package"), 0700); err != nil {
		b.Fatal(err)
	}
	contents := bytes.Repeat([]byte{'x'}, MaxFileBytes)
	if err := os.WriteFile(filepath.Join(dir, "package", "data.bin"), contents, 0600); err != nil {
		b.Fatal(err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() { root.Close() })
	bundle := bundle{files: map[string][]byte{"data.bin": contents}, bytes: len(contents)}
	b.SetBytes(int64(len(contents)))
	b.ReportAllocs()
	for b.Loop() {
		if err := verifyTree(context.Background(), root, "package", bundle); err != nil {
			b.Fatal(err)
		}
	}
}
