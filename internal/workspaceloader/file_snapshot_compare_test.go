package workspaceloader

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSnapshotStreamDetectsTailAndFileSetChanges(t *testing.T) {
	for _, change := range []string{"tail", "truncate", "extra-empty", "deleted"} {
		t.Run(change, func(t *testing.T) {
			root := t.TempDir()
			sources := validRootSource()
			original := sources["src/main.fy"] + "\n//" + strings.Repeat(" ", snapshotCompareBufferBytes*2) + "tail\n"
			sources["src/main.fy"] = original
			writePackage(t, root, ".", rootManifest(""), sources)
			workspace, err := Load(root)
			if err != nil {
				t.Fatal(err)
			}
			snapshot, err := CaptureFileSnapshot(t.Context(), workspace, nil)
			if err != nil {
				t.Fatal(err)
			}
			if same, err := snapshot.Matches(t.Context(), nil); err != nil || !same {
				t.Fatalf("unchanged: %v %v", same, err)
			}
			name := filepath.Join(root, "src", "main.fy")
			info, err := os.Stat(name)
			if err != nil {
				t.Fatal(err)
			}
			switch change {
			case "tail":
				data := []byte(original)
				data[len(data)-2] = 'X'
				err = os.WriteFile(name, data, 0600)
				if err == nil {
					err = os.Chtimes(name, info.ModTime(), info.ModTime())
				}
			case "truncate":
				err = os.WriteFile(name, nil, 0600)
			case "extra-empty":
				err = os.WriteFile(filepath.Join(root, "src", "added.fy"), nil, 0600)
			case "deleted":
				err = os.Remove(name)
			}
			if err != nil {
				t.Fatal(err)
			}
			if same, err := snapshot.Matches(t.Context(), nil); err != nil || same {
				t.Fatalf("%s: %v %v", change, same, err)
			}
		})
	}
}

type snapshotReaderFunc func([]byte) (int, error)

func (f snapshotReaderFunc) Read(buffer []byte) (int, error) { return f(buffer) }

func TestSnapshotStreamErrorPrecedenceAndBounds(t *testing.T) {
	for _, tc := range []struct {
		actual, expected string
		maximum          int
		same             bool
	}{{"", "", 0, true}, {"abc", "abc", 3, true}, {"abc", "abd", 3, false}, {"abc", "ab", 3, false}, {"ab", "abc", 3, false}} {
		same, err := matchBoundedSourceReader(t.Context(), strings.NewReader(tc.actual), tc.maximum, []byte(tc.expected), make([]byte, 2))
		if err != nil || same != tc.same {
			t.Fatalf("%+v: %v %v", tc, same, err)
		}
	}
	read := 0
	overflow := snapshotReaderFunc(func(buffer []byte) (int, error) { read += len(buffer); clear(buffer); return len(buffer), nil })
	if _, err := matchBoundedSourceReader(t.Context(), overflow, 3, []byte("abc"), make([]byte, 2)); err == nil || read != 4 {
		t.Fatalf("overflow: read=%d err=%v", read, err)
	}
	failure := errors.New("late read failure")
	input := io.MultiReader(strings.NewReader("different"), snapshotReaderFunc(func([]byte) (int, error) { return 0, failure }))
	if same, err := matchBoundedSourceReader(t.Context(), input, 20, []byte("expected"), make([]byte, 2)); same || !errors.Is(err, failure) {
		t.Fatalf("mismatch masked I/O failure: %v %v", same, err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	canceling := snapshotReaderFunc(func(buffer []byte) (int, error) { buffer[0] = 'a'; cancel(); return 1, nil })
	if _, err := matchBoundedSourceReader(ctx, canceling, 20, []byte("a"), make([]byte, 2)); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation: %v", err)
	}
	buffer := []byte("stable")
	if _, err := matchBoundedSourceReader(t.Context(), bytes.NewReader(nil), 0, nil, buffer); err != nil {
		t.Fatal(err)
	}
}
