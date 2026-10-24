package languageserver

import (
	"strings"
	"testing"
)

func TestPositionIndexMatchesLinearPositions(t *testing.T) {
	for name, text := range map[string][]byte{
		"empty":     nil,
		"unicode":   []byte(strings.Repeat("한😀x\r\nsecond\n", 180)),
		"long-line": []byte(strings.Repeat("x", 1022) + "😀" + strings.Repeat("한x", 600)),
		"newlines":  []byte(strings.Repeat("\r\r\n\n", 600)),
		"invalid":   append([]byte(strings.Repeat("x", 1023)), 0xff, 0xc0, 0x80, 0xf0, 0x9f),
	} {
		t.Run(name, func(t *testing.T) {
			index := newPositionIndex(text)
			check := func(offset int) {
				t.Helper()
				if got, want := index.bytePosition(offset), bytePosition(text, offset); got != want {
					t.Fatalf("offset %d: got %+v, want %+v", offset, got, want)
				}
			}
			// Cover partial UTF-8 offsets while extending, then checkpoint lookup
			// in reverse order, including positions inside supplementary runes.
			for offset := -1; offset <= len(text)+1; offset++ {
				check(offset)
			}
			for offset := len(text) + 1; offset >= -1; offset-- {
				check(offset)
			}
			if index.scanned != len(text) || len(index.checkpoints) > len(text)/positionCheckpointBytes+1 {
				t.Fatal("index exceeded source or sparse-checkpoint bound")
			}
		})
	}
}

func TestPositionIndexIsLazyAndSnapshotOwned(t *testing.T) {
	text := []byte(strings.Repeat("x", 1<<20))
	index := newPositionIndex(text)
	if got := index.bytePosition(0); got != (Position{}) || index.scanned != 0 || len(index.checkpoints) != 1 {
		t.Fatal("byte-zero lookup scanned or indexed the source")
	}
	old := &analysis{}
	previous := old.sourceRange("main.fy", text, 1024, 2048)
	if old.sourceRange("main.fy", text, 1024, 2048) != previous || len(old.positions) != 1 {
		t.Fatal("same immutable source did not reuse its index")
	}
	currentText := []byte("한😀\nnext")
	current := &analysis{}
	if got, want := current.sourceRange("main.fy", currentText, 7, len(currentText)), textRange(currentText, 7, len(currentText)); got != want {
		t.Fatalf("new snapshot reused stale positions: %+v, want %+v", got, want)
	}
	if old.positions["main.fy"] == current.positions["main.fy"] {
		t.Fatal("index escaped its analysis snapshot")
	}
	otherText := []byte("\nother")
	if got, want := current.sourceRange("other.fy", otherText, 1, len(otherText)), textRange(otherText, 1, len(otherText)); got != want {
		t.Fatalf("different file reused another index: %+v, want %+v", got, want)
	}
}
