package pecoff

import (
	"encoding/binary"
	"fmt"
	"strings"
	"testing"
)

func TestBigObjectLongNameScanOrder(t *testing.T) {
	const text = "first_long_name\x00second_long_name\x00unterminated"
	for _, offsets := range [][]uint32{
		{4, 5, 4, 20, 21, 20},
		{21, 5, 20, 4, 21, 4},
		{20, 4, 5, 21, 4, 20},
	} {
		object := bigObjectWithSharedNames(len(offsets), text)
		symbolStart := int(binary.LittleEndian.Uint32(object[48:]))
		for index, offset := range offsets {
			binary.LittleEndian.PutUint32(object[symbolStart+index*coffBigObjectSymbolBytes+4:], offset)
		}
		report, err := InspectObject(object)
		if err != nil {
			t.Fatal(err)
		}
		for index, offset := range offsets {
			start := int(offset) - 4
			want := text[start : start+strings.IndexByte(text[start:], 0)]
			if report.SymbolRecords[index].Name != want || report.SymbolRecords[index].TableIndex != uint32(index) {
				t.Fatalf("offsets=%v symbol %d: %+v, want %q", offsets, index, report.SymbolRecords[index], want)
			}
		}
	}
}

func TestBigObjectLongNameScansShareOneSpan(t *testing.T) {
	const count, nameBytes = 16384, 1 << 20
	object := bigObjectWithSharedNames(count, strings.Repeat("n", nameBytes))
	symbolStart := int(binary.LittleEndian.Uint32(object[48:]))
	symbolEnd := symbolStart + count*coffBigObjectSymbolBytes
	for index := range count {
		binary.LittleEndian.PutUint32(object[symbolStart+index*coffBigObjectSymbolBytes+4:], uint32(4+(index*7919)%count))
	}
	spans := indexBigCOFFNameSpans(object[symbolStart:symbolEnd], string(object[symbolEnd:]))
	if len(spans) != 1 || spans[0] != (bigCOFFNameSpan{start: 4, end: 4 + nameBytes}) {
		t.Fatalf("suffix references should scan one %d-byte span, got %v", nameBytes, spans)
	}
}

func TestBigObjectLongNameErrorsRemainInSymbolOrder(t *testing.T) {
	for _, test := range []struct {
		name    string
		offsets [2]uint32
		want    string
	}{
		{"range before unterminated", [2]uint32{1000, 9}, "string-table offset 1000 is out of range"},
		{"unterminated before range", [2]uint32{9, 1000}, "string-table name at offset 9 is unterminated"},
		{"empty name", [2]uint32{8, 4}, "symbol 0 has no name"},
	} {
		t.Run(test.name, func(t *testing.T) {
			object := bigObjectWithSharedNames(2, "name\x00tail")
			object[len(object)-1] = 'x'
			symbolStart := int(binary.LittleEndian.Uint32(object[48:]))
			for index, offset := range test.offsets {
				binary.LittleEndian.PutUint32(object[symbolStart+index*coffBigObjectSymbolBytes+4:], offset)
			}
			if _, err := InspectObject(object); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("name error: %v, want %q", err, test.want)
			}
		})
	}
}

func BenchmarkBigObjectOverlappingNameScans(b *testing.B) {
	const nameBytes = 1 << 20
	for _, count := range []int{4096, 16384} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			object := bigObjectWithSharedNames(count, strings.Repeat("n", nameBytes))
			symbolStart := int(binary.LittleEndian.Uint32(object[48:]))
			for index := range count {
				// Alternate offsets so caching only the last result cannot fix it.
				offset := uint32(4 + (index*7919)%count)
				binary.LittleEndian.PutUint32(object[symbolStart+index*coffBigObjectSymbolBytes+4:], offset)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				report, err := InspectObject(object)
				if err != nil || len(report.SymbolRecords) != count {
					b.Fatalf("long-name inspection: %v", err)
				}
			}
		})
	}
}
