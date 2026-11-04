package durablefile

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestOpenRegularRejectsSpecialPathsAndChangedIdentity(t *testing.T) {
	root, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if err := root.WriteFile("record", []byte("record bytes"), 0600); err != nil {
		t.Fatal(err)
	}
	file, err := OpenRegular(root, "record")
	if err != nil {
		t.Fatal(err)
	}
	got, err := io.ReadAll(file)
	file.Close()
	if err != nil || string(got) != "record bytes" {
		t.Fatalf("read = %q, %v", got, err)
	}
	if err := root.Mkdir("directory", 0700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"directory", "missing", "../record"} {
		if file, err := OpenRegular(root, name); err == nil {
			file.Close()
			t.Fatalf("accepted %q", name)
		}
	}
	want := errors.New("open failure")
	if _, err := openRegular(root, "record", func(*os.Root, string) (*os.File, error) { return nil, want }); !errors.Is(err, want) {
		t.Fatalf("open error = %v", err)
	}
	var replaced *os.File
	_, err = openRegular(root, "record", func(root *os.Root, name string) (*os.File, error) {
		if err := root.Rename(name, "old"); err != nil {
			return nil, err
		}
		if err := root.WriteFile(name, []byte("replacement"), 0600); err != nil {
			return nil, err
		}
		var err error
		replaced, err = openReadFile(root, name)
		return replaced, err
	})
	if err == nil {
		t.Fatal("accepted replaced identity")
	}
	if _, err := replaced.Read(make([]byte, 1)); !errors.Is(err, os.ErrClosed) {
		t.Fatalf("rejected handle not closed: %v", err)
	}
}

func TestOpenRegularRejectsSymlinks(t *testing.T) {
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "target"), []byte("bytes"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("target", filepath.Join(directory, "link")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if file, err := OpenRegular(root, "link"); err == nil {
		file.Close()
		t.Fatal("accepted symlink")
	}
}

func TestOpenDirectoryAdmitsOnlyStableDirectories(t *testing.T) {
	root, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if err := root.Mkdir("directory", 0o700); err != nil {
		t.Fatal(err)
	}
	if err := root.WriteFile("record", []byte("record bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	directory, err := OpenDirectory(root, "directory")
	if err != nil {
		t.Fatal(err)
	}
	info, err := directory.Stat()
	closeErr := directory.Close()
	if err != nil || closeErr != nil || !info.IsDir() {
		t.Fatalf("directory stat = %+v, %v, %v", info, err, closeErr)
	}
	for _, name := range []string{"record", "missing", "../directory"} {
		if directory, err := OpenDirectory(root, name); err == nil {
			directory.Close()
			t.Fatalf("accepted %q", name)
		}
	}
	want := errors.New("open failure")
	if _, err := openDirectory(root, "directory", func(*os.Root, string) (*os.File, error) { return nil, want }); !errors.Is(err, want) {
		t.Fatalf("open error = %v", err)
	}
	var replaced *os.File
	_, err = openDirectory(root, "directory", func(root *os.Root, name string) (*os.File, error) {
		if err := root.Rename(name, "old"); err != nil {
			return nil, err
		}
		if err := root.Mkdir(name, 0o700); err != nil {
			return nil, err
		}
		var err error
		replaced, err = openReadDirectory(root, name)
		return replaced, err
	})
	if !errors.Is(err, ErrIdentityChanged) {
		t.Fatalf("replaced identity error = %v", err)
	}
	if _, err := replaced.Read(make([]byte, 1)); !errors.Is(err, os.ErrClosed) {
		t.Fatalf("rejected handle not closed: %v", err)
	}
}

func TestOpenDirectoryOpensExistingRootIdentity(t *testing.T) {
	root, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	before, err := root.Lstat(".")
	if err != nil {
		t.Fatal(err)
	}
	directory, err := OpenDirectory(root, ".")
	if err != nil {
		t.Fatalf("open root directory: %v", err)
	}
	after, err := directory.Stat()
	closeErr := directory.Close()
	if err != nil || closeErr != nil {
		t.Fatalf("root directory stat/close = %v, %v", err, closeErr)
	}
	if !os.SameFile(before, after) {
		t.Fatal("root directory identity changed")
	}
}

func TestOpenDirectoryRejectsSymlinks(t *testing.T) {
	directory := t.TempDir()
	if err := os.Mkdir(filepath.Join(directory, "target"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("target", filepath.Join(directory, "link")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if opened, err := OpenDirectory(root, "link"); err == nil {
		opened.Close()
		t.Fatal("accepted symlink")
	}
}
