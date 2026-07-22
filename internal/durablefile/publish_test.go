package durablefile

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestPublishImmutableCreatesConvergesAndRepairsInsideOpenedRoot(t *testing.T) {
	t.Parallel()
	rootPath := t.TempDir()
	root, err := os.OpenRoot(rootPath)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if err := root.Mkdir("maps", 0o700); err != nil {
		t.Fatal(err)
	}
	if err := SyncDirectory(root, "."); err != nil {
		t.Fatal(err)
	}
	payload := []byte("canonical-payload")
	publishTestPayload(t, root, payload)
	if got, err := root.ReadFile(filepath.Join("maps", "final")); err != nil || string(got) != string(payload) {
		t.Fatalf("published bytes = %q, %v", got, err)
	}
	publishTestPayload(t, root, payload)
	if err := root.WriteFile(filepath.Join("maps", "final"), []byte("corrupt"), 0o600); err != nil {
		t.Fatal(err)
	}
	publishTestPayload(t, root, payload)
	if got, err := root.ReadFile(filepath.Join("maps", "final")); err != nil || string(got) != string(payload) {
		t.Fatalf("repaired bytes = %q, %v", got, err)
	}
}

func TestPublicationRejectsInvalidBoundaries(t *testing.T) {
	t.Parallel()
	if _, _, err := CreateTemporary(nil, "."); err == nil {
		t.Fatal("nil root temporary succeeded")
	}
	root, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if _, _, err := CreateTemporary(root, "../escape"); err == nil {
		t.Fatal("escaping temporary directory succeeded")
	}
	if err := root.Mkdir("left", 0o700); err != nil {
		t.Fatal(err)
	}
	if err := root.Mkdir("right", 0o700); err != nil {
		t.Fatal(err)
	}
	if err := PublishImmutable(root, filepath.Join("left", "temp"), filepath.Join("right", "final"), []byte("x")); err == nil {
		t.Fatal("cross-directory publication succeeded")
	}
	if err := SyncDirectory(nil, "."); err == nil {
		t.Fatal("nil root sync succeeded")
	}
}

func publishTestPayload(t *testing.T, root *os.Root, payload []byte) {
	t.Helper()
	file, temporaryPath, err := CreateTemporary(root, "maps")
	if err != nil {
		t.Fatal(err)
	}
	closed := false
	defer func() {
		if !closed {
			_ = file.Close()
		}
		if err := root.Remove(temporaryPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			t.Errorf("remove temporary: %v", err)
		}
	}()
	if _, err := file.Write(payload); err != nil {
		t.Fatal(err)
	}
	if err := file.Sync(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	closed = true
	if err := PublishImmutable(root, temporaryPath, filepath.Join("maps", "final"), payload); err != nil {
		t.Fatal(err)
	}
}
