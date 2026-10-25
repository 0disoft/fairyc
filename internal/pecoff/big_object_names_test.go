package pecoff

import (
	"encoding/binary"
	"strings"
	"testing"
)

func TestBigObjectSharedLongNamesAreIndependentOfInput(t *testing.T) {
	const name = "fairyc_shared_long_symbol"
	object := bigObjectWithSharedNames(4, name)
	symbolOffset := int(binary.LittleEndian.Uint32(object[48:]))
	// COFF may reference the middle of another string-table name.
	binary.LittleEndian.PutUint32(object[symbolOffset+coffBigObjectSymbolBytes+4:], 4+7)
	report, err := InspectObject(object)
	if err != nil {
		t.Fatal(err)
	}
	clear(object)
	for index, want := range []string{name, name[7:], name, name} {
		if report.Symbols[index] != want || report.SymbolRecords[index].Name != want {
			t.Fatalf("symbol %d changed after input mutation", index)
		}
	}
}

func TestBigObjectSharedLongNamesDoNotAllocatePerSymbol(t *testing.T) {
	const count = 256
	object := bigObjectWithSharedNames(count, strings.Repeat("long_name", 32))
	allocations := testing.AllocsPerRun(1, func() {
		report, err := InspectObject(object)
		if err != nil || len(report.Symbols) != count {
			t.Fatalf("shared name inspection: %v", err)
		}
	})
	if allocations >= count/2 {
		t.Fatalf("shared names allocate per symbol: %.0f allocations for %d symbols", allocations, count)
	}
}

func BenchmarkBigObjectSharedLongNames(b *testing.B) {
	const count = 2048
	name := strings.Repeat("n", 64<<10)
	object := bigObjectWithSharedNames(count, name)
	b.ReportAllocs()
	b.SetBytes(int64(len(object)))
	b.ResetTimer()
	for range b.N {
		report, err := InspectObject(object)
		if err != nil || len(report.Symbols) != count || report.SymbolRecords[count-1].Name != name {
			b.Fatalf("shared long-name inspection: %v", err)
		}
	}
}

func bigObjectWithSharedNames(count int, name string) []byte {
	const rawOffset = coffBigObjectHeaderBytes + coffSectionHeaderBytes
	const symbolOffset = rawOffset + 1
	stringOffset := symbolOffset + count*coffBigObjectSymbolBytes
	object := make([]byte, stringOffset+4+len(name)+1)
	binary.LittleEndian.PutUint16(object[2:], 0xffff)
	binary.LittleEndian.PutUint16(object[4:], 2)
	binary.LittleEndian.PutUint16(object[6:], machineAMD64)
	copy(object[12:28], coffBigObjectClassID[:])
	binary.LittleEndian.PutUint32(object[44:], 1)
	binary.LittleEndian.PutUint32(object[48:], symbolOffset)
	binary.LittleEndian.PutUint32(object[52:], uint32(count))
	header := object[coffBigObjectHeaderBytes:]
	copy(header, ".text")
	binary.LittleEndian.PutUint32(header[16:], 1)
	binary.LittleEndian.PutUint32(header[20:], rawOffset)
	binary.LittleEndian.PutUint32(header[36:], sectionContainsCode|sectionExecutable|sectionReadable)
	object[rawOffset] = 0xc3
	for index := range count {
		symbol := object[symbolOffset+index*coffBigObjectSymbolBytes:]
		binary.LittleEndian.PutUint32(symbol[4:], 4)
		binary.LittleEndian.PutUint32(symbol[12:], 1)
		symbol[18] = coffStorageClassStatic
	}
	binary.LittleEndian.PutUint32(object[stringOffset:], uint32(4+len(name)+1))
	copy(object[stringOffset+4:], name)
	return object
}
