package durablefile

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestOpenRegularRejectsFIFOIncludingReplacement(t *testing.T) {
	directory := t.TempDir()
	root, err := os.OpenRoot(directory)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if err := syscall.Mkfifo(filepath.Join(directory, "fifo"), 0600); err != nil {
		t.Fatal(err)
	}
	if file, err := OpenRegular(root, "fifo"); err == nil {
		file.Close()
		t.Fatal("accepted FIFO")
	}
	if err := root.WriteFile("regular", []byte("bytes"), 0600); err != nil {
		t.Fatal(err)
	}
	_, err = openRegular(root, "regular", func(root *os.Root, name string) (*os.File, error) {
		if err := root.Remove(name); err != nil {
			return nil, err
		}
		if err := syscall.Mkfifo(filepath.Join(directory, name), 0600); err != nil {
			return nil, err
		}
		return openReadFile(root, name)
	})
	if err == nil {
		t.Fatal("accepted FIFO replacing checked regular file")
	}
}

func TestOpenDirectoryRejectsFIFOIncludingReplacement(t *testing.T) {
	directory := t.TempDir()
	root, err := os.OpenRoot(directory)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if err := syscall.Mkfifo(filepath.Join(directory, "fifo"), 0o600); err != nil {
		t.Fatal(err)
	}
	if opened, err := OpenDirectory(root, "fifo"); err == nil {
		opened.Close()
		t.Fatal("accepted FIFO")
	}
	if err := root.Mkdir("directory", 0o700); err != nil {
		t.Fatal(err)
	}
	_, err = openDirectory(root, "directory", func(root *os.Root, name string) (*os.File, error) {
		if err := root.Remove(name); err != nil {
			return nil, err
		}
		if err := syscall.Mkfifo(filepath.Join(directory, name), 0o600); err != nil {
			return nil, err
		}
		return openReadDirectory(root, name)
	})
	if err == nil {
		t.Fatal("accepted FIFO replacing checked directory")
	}
}
