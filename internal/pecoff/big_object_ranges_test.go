package pecoff

import (
	"encoding/binary"
	"fmt"
	"strings"
	"testing"
)

func TestBigObjectRangesRejectCrossSectionAliases(t *testing.T) {
	for _, test := range []struct {
		name string
		edit func([]byte)
	}{
		{"raw aliases raw", func(object []byte) {
			first := object[coffBigObjectHeaderBytes:]
			second := object[coffBigObjectHeaderBytes+coffSectionHeaderBytes:]
			copy(second[20:24], first[20:24])
		}},
		{"relocations alias relocations", func(object []byte) {
			first := object[coffBigObjectHeaderBytes:]
			second := object[coffBigObjectHeaderBytes+coffSectionHeaderBytes:]
			copy(second[24:28], first[24:28])
		}},
		{"raw overlaps relocations", func(object []byte) {
			first := object[coffBigObjectHeaderBytes:]
			second := object[coffBigObjectHeaderBytes+coffSectionHeaderBytes:]
			copy(second[20:24], first[24:28])
		}},
		{"raw contains relocations", func(object []byte) {
			binary.LittleEndian.PutUint32(object[coffBigObjectHeaderBytes+16:], 18)
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			object := storedSectionBigCOFF(3)
			test.edit(object)
			if _, err := InspectObject(object); err == nil || !strings.Contains(err.Error(), "overlaps") {
				t.Fatalf("overlapping storage accepted or misdiagnosed: %v", err)
			}
		})
	}
}

func TestBigObjectUnorderedAdjacentRangesPreserveSectionNumbers(t *testing.T) {
	object := storedSectionBigCOFF(3)
	first := object[coffBigObjectHeaderBytes:][:coffSectionHeaderBytes]
	last := object[coffBigObjectHeaderBytes+2*coffSectionHeaderBytes:][:coffSectionHeaderBytes]
	for index := range first {
		first[index], last[index] = last[index], first[index]
	}
	report, err := InspectObject(object)
	if err != nil {
		t.Fatal(err)
	}
	for index, section := range report.Sections {
		wantOffset := coffBigObjectHeaderBytes + 3*coffSectionHeaderBytes + (2-index)*18
		if section.RawOffset != uint32(wantOffset) || report.RelocationRecords[index].SectionIndex != index+1 {
			t.Fatalf("section %d or its relocation reordered", index)
		}
	}
}

func TestBigObjectStoredSectionFanout(t *testing.T) {
	for _, count := range []int{1, 4135, maxObjectSections} {
		object := storedSectionBigCOFF(count)
		report, err := InspectObject(object)
		if err != nil || len(report.Sections) != count || report.Relocations != count {
			t.Fatalf("sections=%d relocations=%d err=%v", len(report.Sections), report.Relocations, err)
		}
		for index, section := range report.Sections {
			wantOffset := coffBigObjectHeaderBytes + count*coffSectionHeaderBytes + index*18
			if section.RawOffset != uint32(wantOffset) || report.RelocationRecords[index].SectionIndex != index+1 {
				t.Fatalf("section %d or its relocation reordered", index)
			}
		}
	}
}

func BenchmarkInspectBigObjectStoredSections(b *testing.B) {
	for _, count := range []int{4096, maxObjectSections} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			object := storedSectionBigCOFF(count)
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				report, err := InspectObject(object)
				if err != nil || len(report.Sections) != count || report.Relocations != count {
					b.Fatalf("section inspection: %v", err)
				}
			}
		})
	}
}

func storedSectionBigCOFF(count int) []byte {
	headerEnd := coffBigObjectHeaderBytes + count*coffSectionHeaderBytes
	symbolOffset := headerEnd + count*18 // Eight raw bytes and one relocation.
	object := make([]byte, symbolOffset+coffBigObjectSymbolBytes+4)
	binary.LittleEndian.PutUint16(object[2:], 0xffff)
	binary.LittleEndian.PutUint16(object[4:], 2)
	binary.LittleEndian.PutUint16(object[6:], machineAMD64)
	copy(object[12:28], coffBigObjectClassID[:])
	binary.LittleEndian.PutUint32(object[44:], uint32(count))
	binary.LittleEndian.PutUint32(object[48:], uint32(symbolOffset))
	binary.LittleEndian.PutUint32(object[52:], 1)
	for index := range count {
		header := object[coffBigObjectHeaderBytes+index*coffSectionHeaderBytes:]
		copy(header, ".text")
		rawOffset := headerEnd + index*18
		binary.LittleEndian.PutUint32(header[16:], 8)
		binary.LittleEndian.PutUint32(header[20:], uint32(rawOffset))
		binary.LittleEndian.PutUint32(header[24:], uint32(rawOffset+8))
		binary.LittleEndian.PutUint16(header[32:], 1)
		binary.LittleEndian.PutUint32(header[36:], sectionContainsCode|sectionExecutable|sectionReadable)
		object[rawOffset] = 0xc3
		binary.LittleEndian.PutUint16(object[rawOffset+8+8:], ObjectRelocationAMD64Rel32)
	}
	copy(object[symbolOffset:], "sample")
	binary.LittleEndian.PutUint32(object[symbolOffset+12:], 1)
	binary.LittleEndian.PutUint16(object[symbolOffset+16:], coffSymbolTypeFunction)
	object[symbolOffset+18] = coffStorageClassExternal
	binary.LittleEndian.PutUint32(object[symbolOffset+coffBigObjectSymbolBytes:], 4)
	return object
}
