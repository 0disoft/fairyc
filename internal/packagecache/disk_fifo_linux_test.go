//go:build linux

package packagecache

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func TestDiskStoreRejectsFIFOKeyDirectoryWithoutBlocking(t *testing.T) {
	root := t.TempDir()
	limits := DiskLimits{MaxEntries: 2, MaxBytes: 4096, MaxRecordBytes: 1024, TempMaxAge: time.Minute}
	store, err := NewDiskStore(root, limits)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	key := Key{1}
	if err := syscall.Mkfifo(filepath.Join(root, "v1", store.keyDirectoryName(key)), 0o600); err != nil {
		t.Fatal(err)
	}
	found := make(chan bool, 1)
	go func() {
		_, ok := store.Get(key)
		found <- ok
	}()
	select {
	case ok := <-found:
		if ok {
			t.Fatal("FIFO key directory produced a cache hit")
		}
	case <-time.After(time.Second):
		t.Fatal("FIFO key directory blocked cache lookup")
	}
	info, err := os.Lstat(filepath.Join(root, "v1", store.keyDirectoryName(key)))
	if err != nil || info.Mode()&os.ModeNamedPipe == 0 {
		t.Fatalf("FIFO was replaced: %v, %v", info, err)
	}
}
