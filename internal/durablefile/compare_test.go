package durablefile

import (
	"bytes"
	"errors"
	"io"
	"os"
	"testing"
	"testing/iotest"
)

func TestMatchesFile(t *testing.T) {
	root, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	for _, size := range []int{1, 1024, 32768, 32769, 100000} {
		payload := bytes.Repeat([]byte("x"), size)
		if err := root.WriteFile("data", payload, 0600); err != nil {
			t.Fatal(err)
		}
		if !matchesFile(root, "data", payload) {
			t.Fatalf("identical file rejected at size %d", size)
		}
		for _, offset := range []int{0, size / 2, size - 1} {
			payload[offset] = 'y'
			if matchesFile(root, "data", payload) {
				t.Fatalf("mismatch accepted at size %d, offset %d", size, offset)
			}
			payload[offset] = 'x'
		}
		if matchesFile(root, "data", append(payload, 'x')) || matchesFile(root, "data", payload[:size-1]) {
			t.Fatalf("different lengths accepted at size %d", size)
		}
	}
	if matchesFile(root, "missing", []byte("x")) || matchesFile(root, ".", []byte("x")) {
		t.Fatal("missing file or directory accepted")
	}
}

func TestPublishConvergesWhenTemporaryIsGone(t *testing.T) {
	root, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	payload := []byte("canonical-payload")
	if err := root.WriteFile("final", payload, 0600); err != nil {
		t.Fatal(err)
	}
	if err := PublishImmutable(root, "missing-temp", "final", payload); err != nil {
		t.Fatalf("identical destination did not converge: %v", err)
	}
}

func TestMatchesReaderBoundsReadsAndRejectsChanges(t *testing.T) {
	expected := bytes.Repeat([]byte("x"), compareChunkBytes*3+7)
	for _, test := range []struct {
		name string
		data []byte
		want bool
	}{
		{"equal", expected, true},
		{"shrank", expected[:len(expected)-1], false},
		{"grew", append(bytes.Clone(expected), bytes.Repeat([]byte("x"), 1<<20)...), false},
		{"different", bytes.Repeat([]byte("y"), len(expected)), false},
	} {
		t.Run(test.name, func(t *testing.T) {
			reader := &countedReader{Reader: bytes.NewReader(test.data)}
			if got := matchesReader(reader, expected); got != test.want {
				t.Fatalf("match = %v, want %v", got, test.want)
			}
			if reader.largestRead > compareChunkBytes || reader.total > len(expected)+1 {
				t.Fatalf("unbounded read: largest=%d, total=%d", reader.largestRead, reader.total)
			}
		})
	}
	broken := iotest.ErrReader(errors.New("read failure"))
	if matchesReader(broken, expected) || matchesReader(io.MultiReader(bytes.NewReader(expected), broken), expected) {
		t.Fatal("read error accepted as an exact match")
	}
}

type countedReader struct {
	io.Reader
	largestRead int
	total       int
}

func (reader *countedReader) Read(buffer []byte) (int, error) {
	reader.largestRead = max(reader.largestRead, len(buffer))
	count, err := reader.Reader.Read(buffer)
	reader.total += count
	return count, err
}

func BenchmarkMatchesOversizedFile(b *testing.B) {
	root, err := os.OpenRoot(b.TempDir())
	if err != nil {
		b.Fatal(err)
	}
	defer root.Close()
	file, err := root.Create("oversized")
	if err != nil {
		b.Fatal(err)
	}
	if err := file.Truncate(32 << 20); err != nil {
		file.Close()
		b.Fatal(err)
	}
	if err := file.Close(); err != nil {
		b.Fatal(err)
	}
	expected := []byte("canonical-payload")
	b.ReportAllocs()
	for b.Loop() {
		if matchesFile(root, "oversized", expected) {
			b.Fatal("oversized file matched")
		}
	}
}
