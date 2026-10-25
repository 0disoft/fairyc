package pecoff

import (
	"bytes"
	"debug/pe"
	"encoding/binary"
	"strings"
	"testing"
)

func TestObjectLongNamesSurviveAuxiliaryRecordsAndInputMutation(t *testing.T) {
	const name = "fairyc_shared_long_symbol"
	object := ordinaryObjectWithSharedNames(4, name)
	// The first primary skips one auxiliary record; the next uses a suffix.
	object[testCOFFSymbolOffset+17] = 1
	clear(object[testCOFFSymbolOffset+18 : testCOFFSymbolOffset+36])
	binary.LittleEndian.PutUint32(object[testCOFFSymbolOffset+2*18+4:], 4+7)
	report, err := InspectObject(object)
	if err != nil {
		t.Fatal(err)
	}
	clear(object)
	if len(report.SymbolRecords) != 3 {
		t.Fatalf("primary symbol count = %d, want 3", len(report.SymbolRecords))
	}
	for index, want := range []struct {
		tableIndex uint32
		name       string
	}{{0, name}, {2, name[7:]}, {3, name}} {
		if report.Symbols[index] != want.name || report.SymbolRecords[index].Name != want.name ||
			report.SymbolRecords[index].TableIndex != want.tableIndex {
			t.Fatalf("primary symbol %d = %+v", index, report.SymbolRecords[index])
		}
	}
}

func TestPrimaryCOFFSymbolsReuseDecodedNames(t *testing.T) {
	const count = 256
	object := ordinaryObjectWithSharedNames(count, strings.Repeat("long_name", 32))
	file, err := pe.NewFile(bytes.NewReader(object))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	allocations := testing.AllocsPerRun(1, func() {
		symbols, err := inspectPrimaryCOFFSymbols(file, object)
		if err != nil || len(symbols.records) != count {
			t.Fatalf("primary symbol inspection: %v", err)
		}
	})
	// Parsing owns name allocation; receipt construction must not repeat it.
	if allocations >= count/2 {
		t.Fatalf("decoded names copied again: %.0f allocations for %d symbols", allocations, count)
	}
}

func TestObjectLongNamesStillRejectInvalidOffsets(t *testing.T) {
	for _, offset := range []uint32{1, 3, 1000} {
		object := ordinaryObjectWithSharedNames(1, "fairyc_shared_long_symbol")
		binary.LittleEndian.PutUint32(object[testCOFFSymbolOffset+4:], offset)
		if _, err := InspectObject(object); err == nil || !strings.Contains(err.Error(), "string table") {
			t.Fatalf("long-name offset %d was not rejected: %v", offset, err)
		}
	}
}

func BenchmarkOrdinaryObjectSharedLongNames(b *testing.B) {
	const count = 2048
	name := strings.Repeat("n", 64<<10)
	object := ordinaryObjectWithSharedNames(count, name)
	b.ReportAllocs()
	b.SetBytes(int64(len(object)))
	b.ResetTimer()
	for range b.N {
		report, err := InspectObject(object)
		if err != nil || len(report.Symbols) != count || report.SymbolRecords[count-1].Name != name {
			b.Fatalf("ordinary long-name inspection: %v", err)
		}
	}
}

func ordinaryObjectWithSharedNames(count int, name string) []byte {
	object := coffSymbolBudgetObject(count)
	stringOffset := testCOFFSymbolOffset + count*18
	object = append(object, []byte(name)...)
	object = append(object, 0)
	binary.LittleEndian.PutUint32(object[stringOffset:], uint32(4+len(name)+1))
	for index := range count {
		symbol := object[testCOFFSymbolOffset+index*18:]
		clear(symbol[:8])
		binary.LittleEndian.PutUint32(symbol[4:], 4)
	}
	return object
}
