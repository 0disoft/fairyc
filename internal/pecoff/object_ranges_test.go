package pecoff

import (
	"encoding/binary"
	"fmt"
	"strings"
	"testing"
)

func TestInspectObjectRawRanges(t *testing.T) {
	for _, test := range []struct {
		name    string
		offsets []int
		sizes   []int
		overlap bool
	}{
		{"adjacent", []int{0, 2, 4}, []int{2, 2, 2}, false},
		{"unordered", []int{4, 0, 2}, []int{2, 2, 2}, false},
		{"gap", []int{6, 0, 3}, []int{2, 2, 2}, false},
		{"duplicate", []int{0, 0, 4}, []int{2, 2, 2}, true},
		{"partial", []int{0, 1, 4}, []int{2, 2, 2}, true},
		{"contained", []int{2, 0, 6}, []int{2, 6, 2}, true},
		{"equal end", []int{0, 1, 6}, []int{4, 3, 2}, true},
		{"unordered overlap", []int{6, 0, 2}, []int{2, 6, 2}, true},
		{"empty", []int{0, 0, 4}, []int{0, 2, 2}, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			object := storedSectionAMD64COFF(len(test.offsets))
			base := coffFileHeaderBytes + len(test.offsets)*coffSectionHeaderBytes
			for index, offset := range test.offsets {
				header := object[coffFileHeaderBytes+index*coffSectionHeaderBytes:]
				binary.LittleEndian.PutUint32(header[16:], uint32(test.sizes[index]))
				binary.LittleEndian.PutUint32(header[20:], uint32(base+offset))
			}
			report, err := InspectObject(object)
			if test.overlap {
				if err == nil || !strings.Contains(err.Error(), "overlaps another raw range") {
					t.Fatalf("overlapping ranges accepted or misdiagnosed: %v", err)
				}
			} else if err != nil {
				t.Fatalf("valid ranges rejected: %v", err)
			}
			if err == nil {
				for index, offset := range test.offsets {
					if report.Sections[index].RawOffset != uint32(base+offset) {
						t.Fatalf("section %d reordered", index)
					}
				}
			}
		})
	}
}

func BenchmarkInspectObjectStoredSections(b *testing.B) {
	for _, count := range []int{4096, maxObjectSections} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			object := storedSectionAMD64COFF(count)
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				report, err := InspectObject(object)
				if err != nil || len(report.Sections) != count {
					b.Fatalf("section inspection: sections=%d err=%v", len(report.Sections), err)
				}
			}
		})
	}
}

// Unlike the existing mostly-BSS fanout fixture, every section owns raw bytes.
func storedSectionAMD64COFF(count int) []byte {
	const rawSize = 8
	headerEnd := coffFileHeaderBytes + count*coffSectionHeaderBytes
	symbolOffset := headerEnd + count*rawSize
	object := make([]byte, symbolOffset+18+4)
	binary.LittleEndian.PutUint16(object, machineAMD64)
	binary.LittleEndian.PutUint16(object[2:], uint16(count))
	binary.LittleEndian.PutUint32(object[8:], uint32(symbolOffset))
	binary.LittleEndian.PutUint32(object[12:], 1)
	for index := range count {
		header := object[coffFileHeaderBytes+index*coffSectionHeaderBytes:]
		copy(header, ".text")
		binary.LittleEndian.PutUint32(header[16:], rawSize)
		binary.LittleEndian.PutUint32(header[20:], uint32(headerEnd+index*rawSize))
		binary.LittleEndian.PutUint32(header[36:], sectionContainsCode|sectionExecutable|sectionReadable)
	}
	copy(object[symbolOffset:], "sample")
	binary.LittleEndian.PutUint16(object[symbolOffset+12:], 1)
	binary.LittleEndian.PutUint16(object[symbolOffset+14:], coffSymbolTypeFunction)
	object[symbolOffset+16] = coffStorageClassExternal
	binary.LittleEndian.PutUint32(object[symbolOffset+18:], 4)
	return object
}
