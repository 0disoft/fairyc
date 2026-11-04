//go:build linux

package cli

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestCLIInputAdmissionRejectsFIFOWithoutBlocking(t *testing.T) {
	root := t.TempDir()
	if err := syscall.Mkfifo(filepath.Join(root, "input.fy"), 0o600); err != nil {
		t.Fatal(err)
	}
	fixtureDirectory := filepath.Join(root, "fixtures")
	if err := os.Mkdir(fixtureDirectory, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := syscall.Mkfifo(filepath.Join(fixtureDirectory, "input.bin"), 0o600); err != nil {
		t.Fatal(err)
	}
	for name, read := range map[string]func() error{
		"source": func() error {
			_, err := loadSource(root, "input.fy", 1024)
			return err
		},
		"native test stdin": func() error {
			_, err := readStableNativeTestInput(root, "fixtures/input.bin")
			return err
		},
	} {
		t.Run(name, func(t *testing.T) {
			result := make(chan error, 1)
			go func() { result <- read() }()
			select {
			case err := <-result:
				if err == nil || !strings.Contains(err.Error(), "regular file") {
					t.Fatalf("FIFO admission error = %v", err)
				}
			case <-time.After(time.Second):
				t.Fatal("FIFO input blocked before type admission")
			}
		})
	}
}
