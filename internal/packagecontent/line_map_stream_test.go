package packagecontent

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"testing"
)

func TestLineMapStreamingMatchesCanonicalEncodingAcrossBlocks(t *testing.T) {
	for _, count := range []int{0, 1, 7, 8, 9, 15, 16, 17, 63, 64, 65, 1023, 1024, 1025} {
		for _, ending := range []string{"", "\n"} {
			t.Run(fmt.Sprintf("lines-%d/ending-%q", count, ending), func(t *testing.T) {
				contents := append(bytes.Repeat([]byte("\x00é\r\n"), count), []byte("tail"+ending)...)
				want := LineMapID(sha256.Sum256(EncodeLineMapInput(contents)))
				if got := ComputeLineMapID(contents); got != want {
					t.Fatalf("line-map hash = %s, want %s", got, want)
				}
			})
		}
	}
}

func BenchmarkComputeLineMapStreaming(b *testing.B) {
	for _, workload := range []struct {
		name     string
		contents []byte
	}{
		{"single-line-64KiB", bytes.Repeat([]byte("x"), 64<<10)},
		{"source-66KiB", bytes.Repeat([]byte("fn i32 value() { return 12345; }\n"), 2048)},
		{"dense-lines-128KiB", bytes.Repeat([]byte("\n"), 128<<10)},
	} {
		b.Run(workload.name, func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(workload.contents)))
			for b.Loop() {
				if got := ComputeLineMapID(workload.contents); got == (LineMapID{}) {
					b.Fatal("empty line-map hash")
				}
			}
		})
	}
}
