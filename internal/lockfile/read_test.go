package lockfile

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/0disoft/fairyc/internal/workspaceloader"
)

func lockReadFixture(t *testing.T) []byte {
	t.Helper()
	workspace, err := workspaceloader.Load(filepath.Join("..", "..", "spec", "fixtures", "workspace-loader", "cases", "pass-feature-distinct"))
	if err != nil {
		t.Fatal(err)
	}
	document, err := FromWorkspace(workspace)
	if err != nil {
		t.Fatal(err)
	}
	contents, err := Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	return contents
}

type observedLockReadFile struct {
	info                 os.FileInfo
	reader               io.Reader
	read                 func([]byte) (int, error)
	statErr, closeErr    error
	onClose              func()
	closes, reads, bytes int
	maxRequest           int
}

func (file *observedLockReadFile) Stat() (fs.FileInfo, error) { return file.info, file.statErr }
func (file *observedLockReadFile) Close() error {
	file.closes++
	if file.onClose != nil {
		file.onClose()
	}
	return file.closeErr
}
func (file *observedLockReadFile) Read(buffer []byte) (int, error) {
	file.reads++
	file.maxRequest = max(file.maxRequest, len(buffer))
	read := file.read
	if read == nil {
		read = file.reader.Read
	}
	n, err := read(buffer)
	file.bytes += n
	return n, err
}

func lockReadMetadata(t *testing.T, contents []byte) (string, os.FileInfo) {
	t.Helper()
	path := filepath.Join(t.TempDir(), FileName)
	if err := os.WriteFile(path, contents, 0600); err != nil {
		t.Fatal(err)
	}
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatal(err)
	}
	return path, info
}

func TestReadLockSizeBoundaries(t *testing.T) {
	canonical := lockReadFixture(t)
	for _, size := range []int{MaxBytes - 1, MaxBytes, MaxBytes + 1} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			contents := append(bytes.Clone(canonical), bytes.Repeat([]byte{' '}, size-len(canonical))...)
			path, _ := lockReadMetadata(t, contents)
			document, got, err := Read(path)
			if size > MaxBytes {
				if err == nil || got != nil || document.Schema != "" {
					t.Fatalf("oversized lock accepted: bytes=%d error=%v", len(got), err)
				}
			} else if err != nil || !bytes.Equal(got, contents) || document.Schema != SchemaVersion {
				t.Fatalf("bounded lock changed: bytes=%d error=%v", len(got), err)
			}
		})
	}
}

func TestReadRejectsGrowthBeforeReading(t *testing.T) {
	canonical := lockReadFixture(t)
	path, _ := lockReadMetadata(t, canonical)
	var observed *observedLockReadFile
	document, got, err := ReadWithOptions(path, ReadOptions{
		Lstat: func(path string) (os.FileInfo, error) {
			info, err := os.Lstat(path)
			if err == nil {
				err = os.Truncate(path, MaxBytes+1)
			}
			return info, err
		},
		Open: func(path string) (fs.File, error) {
			file, err := os.Open(path)
			if err != nil {
				return nil, err
			}
			info, err := file.Stat()
			if err != nil {
				file.Close()
				return nil, err
			}
			observed = &observedLockReadFile{info: info, reader: file, onClose: func() {
				if err := file.Close(); err != nil {
					t.Error(err)
				}
			}}
			return observed, nil
		},
	})
	if observed == nil {
		t.Fatal("file was not opened")
	}
	if err == nil || got != nil || document.Schema != "" || observed.bytes != 0 || observed.closes != 1 {
		t.Fatalf("growth not rejected before I/O: read=%d closes=%d result=%d error=%v", observed.bytes, observed.closes, len(got), err)
	}
}

func TestReadLockCancellationAndIOFailures(t *testing.T) {
	canonical := lockReadFixture(t)
	_, info := lockReadMetadata(t, canonical)
	for _, phase := range []string{"inspect", "open", "stat", "read", "close", "cancel-open", "cancel-eof", "cancel-close", "cancel-before", "nil-file"} {
		t.Run(phase, func(t *testing.T) {
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			want := errors.New("injected " + phase)
			file := &observedLockReadFile{info: info, reader: bytes.NewReader(canonical)}
			if phase == "stat" {
				file.statErr = want
			}
			if phase == "close" {
				file.closeErr = want
			}
			if phase == "read" {
				file.read = func(buffer []byte) (int, error) { return copy(buffer, canonical), want }
			}
			if phase == "cancel-eof" {
				reader := bytes.NewReader(canonical)
				file.read = func(buffer []byte) (int, error) {
					n, err := reader.Read(buffer)
					if reader.Len() == 0 {
						cancel(want)
						err = io.EOF
					}
					return n, err
				}
			}
			if phase == "cancel-close" {
				file.onClose = func() { cancel(want) }
			}
			if phase == "cancel-before" {
				cancel(want)
			}
			inspections, opens := 0, 0
			document, got, err := ReadWithOptions("unused", ReadOptions{Context: ctx,
				Lstat: func(string) (os.FileInfo, error) {
					inspections++
					if phase == "inspect" {
						return nil, want
					}
					return info, nil
				},
				Open: func(string) (fs.File, error) {
					opens++
					if phase == "open" {
						return nil, want
					}
					if phase == "nil-file" {
						return nil, nil
					}
					if phase == "cancel-open" {
						cancel(want)
					}
					return file, nil
				},
			})
			if err == nil || got != nil || document.Schema != "" {
				t.Fatalf("failed read returned success data: bytes=%d error=%v", len(got), err)
			}
			if phase != "nil-file" && !errors.Is(err, want) {
				t.Fatalf("failure cause lost: %v", err)
			}
			wantCloses := 1
			if phase == "inspect" || phase == "open" || phase == "cancel-before" || phase == "nil-file" {
				wantCloses = 0
			}
			if file.closes != wantCloses || (phase == "cancel-before" && (inspections != 0 || opens != 0)) {
				t.Fatalf("resource/cancellation boundary failed: closes=%d inspections=%d opens=%d", file.closes, inspections, opens)
			}
			if phase == "cancel-before" || phase == "cancel-open" || phase == "cancel-eof" || phase == "cancel-close" {
				if !errors.Is(err, context.Canceled) {
					t.Fatalf("cancellation identity lost: %v", err)
				}
			}
		})
	}
}

func TestReadLockNoProgressBoundaries(t *testing.T) {
	canonical := lockReadFixture(t)
	_, info := lockReadMetadata(t, canonical)
	for _, empty := range []int{maxLockEmptyReads - 1, maxLockEmptyReads, maxLockEmptyReads + 1} {
		t.Run(fmt.Sprint(empty), func(t *testing.T) {
			reader := bytes.NewReader(canonical)
			calls := 0
			file := &observedLockReadFile{info: info, read: func(buffer []byte) (int, error) {
				calls++
				if calls <= empty {
					return 0, nil
				}
				return reader.Read(buffer)
			}}
			document, got, err := ReadWithOptions("unused", ReadOptions{
				Lstat: func(string) (os.FileInfo, error) { return info, nil },
				Open:  func(string) (fs.File, error) { return file, nil },
			})
			if empty < maxLockEmptyReads {
				if err != nil || !bytes.Equal(got, canonical) || document.Schema != SchemaVersion {
					t.Fatalf("premature no-progress failure: %v", err)
				}
			} else if !errors.Is(err, io.ErrNoProgress) || got != nil || document.Schema != "" || calls != maxLockEmptyReads {
				t.Fatalf("empty read loop not bounded: calls=%d error=%v", calls, err)
			}
			if file.closes != 1 || file.maxRequest > lockReadChunkBytes {
				t.Fatalf("unbounded reader: closes=%d maxRequest=%d", file.closes, file.maxRequest)
			}
		})
	}
}

func TestReadRejectsReplacementAfterInspection(t *testing.T) {
	contents := lockReadFixture(t)
	for _, symlink := range []bool{false, true} {
		name := "regular"
		if symlink {
			name = "symlink"
		}
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), FileName)
			if err := os.WriteFile(path, contents, 0600); err != nil {
				t.Fatal(err)
			}
			foreign := filepath.Join(t.TempDir(), "foreign.lock")
			if err := os.WriteFile(foreign, contents, 0600); err != nil {
				t.Fatal(err)
			}
			document, data, err := ReadWithOptions(path, ReadOptions{Open: func(path string) (fs.File, error) {
				if err := os.Rename(path, path+".before"); err != nil {
					t.Fatal(err)
				}
				if symlink {
					if err := os.Symlink(foreign, path); err != nil {
						t.Skipf("symlink unavailable: %v", err)
					}
				} else if err := os.WriteFile(path, contents, 0600); err != nil {
					t.Fatal(err)
				}
				return openLockReadFile(path)
			}})
			if err == nil || data != nil || document.Schema != "" {
				t.Fatalf("uninspected replacement accepted: schema=%q bytes=%d error=%v", document.Schema, len(data), err)
			}
		})
	}
}

func TestReadRejectsGrowthDuringReading(t *testing.T) {
	canonical := lockReadFixture(t)
	_, info := lockReadMetadata(t, canonical)
	file := &observedLockReadFile{info: info, reader: io.MultiReader(bytes.NewReader(canonical),
		bytes.NewReader(bytes.Repeat([]byte{' '}, MaxBytes+1-len(canonical))))}
	document, got, err := ReadWithOptions("unused", ReadOptions{
		Lstat: func(string) (os.FileInfo, error) { return info, nil },
		Open:  func(string) (fs.File, error) { return file, nil },
	})
	if err == nil || got != nil || document.Schema != "" || file.bytes != MaxBytes+1 || file.closes != 1 || file.maxRequest > lockReadChunkBytes {
		t.Fatalf("growing input escaped limit: read=%d maxRequest=%d closes=%d error=%v", file.bytes, file.maxRequest, file.closes, err)
	}
}

func TestReadOptionsAndParseFailure(t *testing.T) {
	if _, _, err := ReadWithOptions("unused", ReadOptions{}, ReadOptions{}); err == nil {
		t.Fatal("multiple options values accepted")
	}
	if _, _, err := ReadWithOptions("unused", ReadOptions{Lstat: func(string) (os.FileInfo, error) { return nil, nil }}); err == nil {
		t.Fatal("missing inspection metadata accepted")
	}
	path, _ := lockReadMetadata(t, []byte("{"))
	document, contents, err := Read(path)
	if err == nil || document.Schema != "" || string(contents) != "{" {
		t.Fatalf("parse failure changed raw-byte contract: data=%q error=%v", contents, err)
	}
	if _, _, err := Read(filepath.Join(t.TempDir(), "missing.lock")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("missing-file identity lost: %v", err)
	}
}
