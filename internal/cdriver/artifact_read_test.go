package cdriver

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/iotest"
)

func TestReadArtifactContentsTreatsSizeAsHint(t *testing.T) {
	for _, test := range []struct {
		name string
		hint int64
		data string
	}{
		{"shrunk", 4096, "image"},
		{"grown", 1, strings.Repeat("image", 1024)},
		{"short-reads", 5, "image"},
	} {
		t.Run(test.name, func(t *testing.T) {
			contents, err := readArtifactContents(iotest.OneByteReader(strings.NewReader(test.data)), test.hint)
			if err != nil || string(contents) != test.data {
				t.Fatalf("read with size hint %d: length %d, error %v", test.hint, len(contents), err)
			}
		})
	}
	if _, err := readArtifactContents(strings.NewReader(""), 5); err == nil || !strings.Contains(err.Error(), "invalid size 0") {
		t.Fatalf("file shrunk to empty: %v", err)
	}
	failure := errors.New("artifact read failure")
	reader := io.MultiReader(strings.NewReader("image"), iotest.ErrReader(failure))
	if contents, err := readArtifactContents(reader, 5); !errors.Is(err, failure) || contents != nil {
		t.Fatalf("partial failed read: %q, %v", contents, err)
	}
}

func TestReadArtifactContentsPreservesGrowthLimit(t *testing.T) {
	for _, extra := range []int64{0, 2} {
		reader := &artifactCountingReader{remaining: maxArtifactBytes + extra}
		contents, err := readArtifactContents(reader, maxArtifactBytes)
		if extra == 0 {
			if err != nil || len(contents) != maxArtifactBytes {
				t.Fatalf("exact byte limit: length %d, error %v", len(contents), err)
			}
		} else if err == nil || contents != nil || reader.remaining != 1 || !strings.Contains(err.Error(), "invalid size") {
			t.Fatalf("growth past limit: length %d, remaining %d, error %v", len(contents), reader.remaining, err)
		}
	}
}

type artifactCountingReader struct{ remaining int64 }

func (reader *artifactCountingReader) Read(data []byte) (int, error) {
	if reader.remaining == 0 {
		return 0, io.EOF
	}
	n := int(min(int64(len(data)), reader.remaining))
	clear(data[:n])
	reader.remaining -= int64(n)
	return n, nil
}

func BenchmarkReadArtifact(b *testing.B) {
	for _, size := range []int{64 << 10, 4 << 20, 16 << 20} {
		b.Run(fmt.Sprint(size), func(b *testing.B) {
			path := filepath.Join(b.TempDir(), "artifact")
			if err := os.WriteFile(path, bytes.Repeat([]byte{0x5a}, size), 0o600); err != nil {
				b.Fatal(err)
			}
			b.SetBytes(int64(size))
			b.ReportAllocs()
			for b.Loop() {
				contents, err := readArtifact(path)
				if err != nil || len(contents) != size || contents[0] != 0x5a || contents[size-1] != 0x5a {
					b.Fatalf("read artifact: length %d, error %v", len(contents), err)
				}
			}
		})
	}
}
