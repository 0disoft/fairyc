package workspaceloader

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestFileSnapshotChecksBytesEvenWithUnchangedSizeAndTimestamp(t *testing.T) {
	root := t.TempDir()
	writePackage(t, root, ".", rootManifest(""), validRootSource())
	workspace, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := CaptureFileSnapshot(context.Background(), workspace, nil)
	if err != nil {
		t.Fatal(err)
	}
	if current, err := snapshot.Matches(context.Background(), nil); err != nil || !current {
		t.Fatalf("unchanged = %t, %v", current, err)
	}
	name := filepath.Join(root, "src", "main.fy")
	info, err := os.Stat(name)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	changed := bytes.Replace(data, []byte("module "), []byte("module\t"), 1)
	if bytes.Equal(data, changed) {
		t.Fatal("fixture has no replaceable value")
	}
	if err := os.WriteFile(name, changed, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(name, info.ModTime(), info.ModTime()); err != nil {
		t.Fatal(err)
	}
	if current, err := snapshot.Matches(context.Background(), nil); err != nil || current {
		t.Fatalf("same-metadata mutation = %t, %v", current, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := snapshot.Matches(ctx, nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel cause = %v", err)
	}
}

func TestFileSnapshotBindsAllOverlayKeys(t *testing.T) {
	root := t.TempDir()
	writePackage(t, root, ".", rootManifest(""), validRootSource())
	workspace, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := CaptureFileSnapshot(context.Background(), workspace, nil)
	if err != nil {
		t.Fatal(err)
	}
	// Even a new document outside the package's source root changes editor
	// diagnostics and must invalidate an otherwise identical source closure.
	if current, err := snapshot.Matches(context.Background(), map[string][]byte{filepath.Join(root, "outside.fy"): []byte("module outside;")}); err != nil || current {
		t.Fatalf("new overlay = %t, %v", current, err)
	}
}
