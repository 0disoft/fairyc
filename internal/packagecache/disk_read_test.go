package packagecache

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
	"testing/iotest"
)

func TestDiskReadHintsDoNotReplaceByteBounds(t *testing.T) {
	for _, test := range []struct {
		name     string
		size     int
		hint     int64
		limit    uint64
		oversize bool
	}{
		{"exact limit", 256, 256, 256, false},
		{"grew after stat", 256, 128, 256, false},
		{"shrank after stat", 128, 256, 256, false},
		{"unknown size", 256, -1, 256, false},
		{"grew beyond limit", 257, 128, 256, true},
		{"oversize metadata", 257, 257, 256, true},
		{"huge stale hint", 2, 1 << 62, ^uint64(0), false},
		{"larger than preallocation", int(maxDiskReadHint) + 1024, maxDiskReadHint + 1024, uint64(maxDiskReadHint) + 1024, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			body := bytes.Repeat([]byte{'x'}, test.size)
			reader := bytes.NewReader(body)
			got, err := readDiskBytes(reader, test.hint, test.limit)
			if test.oversize {
				if !errors.Is(err, ErrRecordTooLarge) || got != nil {
					t.Fatalf("oversize read = %d bytes, %v", len(got), err)
				}
				if test.hint > int64(test.limit) && reader.Len() != len(body) {
					t.Fatal("oversized metadata caused a payload read")
				}
				return
			}
			if err != nil || !bytes.Equal(got, body) {
				t.Fatalf("read = %d bytes, %v; want %d bytes", len(got), err, len(body))
			}
		})
	}
}

func TestDiskReadPreservesErrorsAndUsesOpenedFile(t *testing.T) {
	wantErr := errors.New("read failed")
	if _, err := readDiskBytes(iotest.ErrReader(wantErr), 32, 32); !errors.Is(err, wantErr) {
		t.Fatalf("reader error = %v", err)
	}
	path := filepath.Join(t.TempDir(), "record")
	if err := os.WriteFile(path, []byte("record bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { file.Close() })
	got, err := readDiskRecordBytes(file, 64)
	if err != nil || string(got) != "record bytes" {
		t.Fatalf("opened file = %q, %v", got, err)
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	if got, err := readDiskRecordBytes(file, 2); !errors.Is(err, ErrRecordTooLarge) || got != nil {
		t.Fatalf("oversize opened file = %q, %v", got, err)
	}
	if offset, err := file.Seek(0, io.SeekCurrent); err != nil || offset != 0 {
		t.Fatalf("oversize file was read: offset=%d, err=%v", offset, err)
	}
	file.Close()
	if _, err := readDiskRecordBytes(file, 64); err == nil {
		t.Fatal("closed-file stat error was ignored")
	}
}
