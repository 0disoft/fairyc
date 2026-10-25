package pecoff

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"
	"testing"
)

func TestArchiveSymbolDefinitionsRemainExact(t *testing.T) {
	for _, names := range [][]string{{"same", "same"}, {"same", "same", "same"}} {
		if _, err := InspectArchive(indexedSymbolArchive(names)); err == nil || !strings.Contains(err.Error(), "not defined externally") {
			t.Fatalf("%d duplicate definitions: %v", len(names), err)
		}
	}
	archive := indexedSymbolArchive([]string{"present"})
	firstSize := 4 + 4 + len("present") + 1
	secondHeader := nextSyntheticArchiveMemberOffset(len(coffArchiveSignature), firstSize)
	secondNames := secondHeader + coffArchiveHeaderBytes + 12 + 2
	copy(archive[secondNames:], "missing")
	if _, err := InspectArchive(archive); err == nil || !strings.Contains(err.Error(), "second linker member symbol") {
		t.Fatalf("second index must verify its own target: %v", err)
	}
}

func TestArchiveSymbolDefinitionsIgnoreIneligibleDuplicates(t *testing.T) {
	for _, test := range []struct {
		name    string
		section uint16
		value   uint32
		storage uint8
	}{
		{"undefined", 0, 0, coffStorageClassExternal},
		{"static", 1, 1, coffStorageClassStatic},
		{"missing-section", 2, 1, coffStorageClassExternal},
		{"absolute", 0xffff, 1, coffStorageClassExternal},
		{"past-section", 1, 2, coffStorageClassExternal},
	} {
		t.Run(test.name, func(t *testing.T) {
			archive := indexedSymbolArchive([]string{"same", "same"})
			members, err := parseRawArchiveMembers(archive)
			if err != nil {
				t.Fatal(err)
			}
			object := members[2].data
			secondSymbol := binary.LittleEndian.Uint32(object[8:]) + 18
			symbol := object[secondSymbol:]
			binary.LittleEndian.PutUint32(symbol[8:], test.value)
			binary.LittleEndian.PutUint16(symbol[12:], test.section)
			symbol[16] = test.storage
			if report, err := InspectArchive(archive); err != nil || len(report.Symbols) != 2 {
				t.Fatalf("one valid definition and an ineligible namesake: %v", err)
			}
		})
	}
}

func BenchmarkInspectArchiveSymbols(b *testing.B) {
	for _, count := range []int{4096, 16384} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			names := make([]string, count)
			for index := range names {
				names[index] = fmt.Sprintf("s%07d", index)
			}
			archive := indexedSymbolArchive(names)
			b.ReportAllocs()
			b.SetBytes(int64(len(archive)))
			b.ResetTimer()
			for range b.N {
				report, err := InspectArchive(archive)
				if err != nil || len(report.Symbols) != count {
					b.Fatalf("archive with %d symbols: %v", count, err)
				}
			}
		})
	}
}

func indexedSymbolArchive(names []string) []byte {
	var symbolNames bytes.Buffer
	for _, name := range names {
		symbolNames.WriteString(name)
		symbolNames.WriteByte(0)
	}
	first := make([]byte, 4+4*len(names)+symbolNames.Len())
	second := make([]byte, 12+2*len(names)+symbolNames.Len())
	secondHeader := nextSyntheticArchiveMemberOffset(len(coffArchiveSignature), len(first))
	objectHeader := nextSyntheticArchiveMemberOffset(secondHeader, len(second))
	binary.BigEndian.PutUint32(first, uint32(len(names)))
	binary.LittleEndian.PutUint32(second, 1)
	binary.LittleEndian.PutUint32(second[4:], uint32(objectHeader))
	binary.LittleEndian.PutUint32(second[8:], uint32(len(names)))
	for index := range names {
		binary.BigEndian.PutUint32(first[4+4*index:], uint32(objectHeader))
		binary.LittleEndian.PutUint16(second[12+2*index:], 1)
	}
	copy(first[4+4*len(names):], symbolNames.Bytes())
	copy(second[12+2*len(names):], symbolNames.Bytes())
	var archive bytes.Buffer
	archive.Write(coffArchiveSignature[:])
	writeSyntheticArchiveMember(&archive, "/", first)
	writeSyntheticArchiveMember(&archive, "/", second)
	writeSyntheticArchiveMember(&archive, "symbols.obj/", testWeakSearchArchiveObject(names...))
	return archive.Bytes()
}
