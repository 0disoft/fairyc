package languageserver

import (
	"fmt"
	"strings"
	"testing"
)

func TestRequestPositionsAcrossLongLineCheckpoints(t *testing.T) {
	for name, text := range map[string][]byte{
		"ascii":    []byte(strings.Repeat("x", 64<<10)),
		"unicode":  []byte(strings.Repeat("한😀x", 8192)),
		"crlf":     []byte(strings.Repeat("한😀x", 8192) + "\r\nnext"),
		"lone-cr":  []byte(strings.Repeat("x", 2048) + "\r" + strings.Repeat("y", 64<<10) + "\nnext"),
		"cr-reset": []byte("first\r\n" + strings.Repeat("한😀x", 8192)),
	} {
		t.Run(name, func(t *testing.T) {
			index := newPositionIndex(text)
			check := func(position Position) {
				t.Helper()
				got, ok := index.positionByte(position)
				want, valid := positionByte(text, position)
				if ok != valid || valid && got != want {
					t.Fatalf("%+v = %d/%t, want %d/%t", position, got, ok, want, valid)
				}
			}
			for _, offset := range []int{len(text), 2048, 1024, 1023, 1, 0} {
				position := bytePosition(text, offset)
				check(position)
				check(Position{Line: position.Line, Character: position.Character + 1})
			}
			index.bytePosition(len(text))
			for _, position := range []Position{{0, 2048}, {0, 2049}, {0, 8193}, {1, 1}, {1, 2}, {0, 1 << 20}} {
				check(position)
			}
		})
	}
}

func TestRequestPositionDoesNotIndexPastRejectedCRColumn(t *testing.T) {
	text := []byte("\r" + strings.Repeat("x", 1<<20))
	index := newPositionIndex(text)
	if _, ok := index.positionByte(Position{Character: len(text)}); ok {
		t.Fatal("accepted a column past the first CR")
	}
	if index.scanned > positionCheckpointBytes {
		t.Fatal("rejected column indexed text beyond the first chunk")
	}
}

func BenchmarkRequestPositionLongLine(b *testing.B) {
	for _, size := range []int{64 << 10, 1 << 20} {
		b.Run(fmt.Sprint(size), func(b *testing.B) {
			text := []byte(strings.Repeat("x", size))
			index := newPositionIndex(text)
			position := Position{Character: size - 17}
			if offset, ok := index.positionByte(position); !ok || offset != position.Character {
				b.Fatal("invalid position fixture")
			}
			b.ReportAllocs()
			for b.Loop() {
				if offset, ok := index.positionByte(position); !ok || offset != position.Character {
					b.Fatal("request position changed")
				}
			}
		})
	}
}

func BenchmarkRequestPositionLongLineCold(b *testing.B) {
	text := []byte(strings.Repeat("x", 1<<20))
	position := Position{Character: len(text) - 17}
	b.ReportAllocs()
	for b.Loop() {
		index := newPositionIndex(text)
		if offset, ok := index.positionByte(position); !ok || offset != position.Character {
			b.Fatal("request position changed")
		}
	}
}
