package pecoff

import (
	"encoding/binary"
	"strings"
	"testing"
)

func TestInspectObjectRejectsSymbolBudgetBeforeParsing(t *testing.T) {
	for _, count := range []uint32{maxObjectSymbols + 1, ^uint32(0)} {
		// Only the header is present: rejection must precede debug/pe allocation
		// or any attempt to read the declared symbol table.
		object := make([]byte, coffFileHeaderBytes)
		binary.LittleEndian.PutUint16(object, machineAMD64)
		binary.LittleEndian.PutUint16(object[2:], 1)
		binary.LittleEndian.PutUint32(object[12:], count)
		if _, err := InspectObject(object); err == nil || !strings.Contains(err.Error(), "invalid symbol count") {
			t.Fatalf("symbol count %d: %v", count, err)
		}
	}
}

func TestInspectObjectRejectsPopulatedOverBudgetSymbolTable(t *testing.T) {
	if _, err := InspectObject(coffSymbolBudgetObject(maxObjectSymbols + 1)); err == nil || !strings.Contains(err.Error(), "invalid symbol count") {
		t.Fatalf("populated symbol table exceeded budget: %v", err)
	}
}

func TestInspectObjectSymbolBudgetBoundaryStillParses(t *testing.T) {
	object := make([]byte, 128)
	binary.LittleEndian.PutUint16(object, machineAMD64)
	binary.LittleEndian.PutUint16(object[2:], 1)
	binary.LittleEndian.PutUint32(object[8:], 96)
	binary.LittleEndian.PutUint32(object[12:], maxObjectSymbols)
	if _, err := InspectObject(object); err == nil || strings.Contains(err.Error(), "invalid symbol count") {
		t.Fatalf("in-budget header should reach truncated-table validation: %v", err)
	}
}

func BenchmarkInspectObjectOverSymbolBudget(b *testing.B) {
	object := coffSymbolBudgetObject(maxObjectSymbols + 1)
	b.ReportAllocs()
	for b.Loop() {
		if _, err := InspectObject(object); err != nil && !strings.Contains(err.Error(), "invalid symbol count") {
			b.Fatal(err)
		}
	}
}

func coffSymbolBudgetObject(count int) []byte {
	base := minimalAMD64COFF()
	end := testCOFFSymbolOffset + count*18
	object := make([]byte, end+4)
	copy(object, base[:testCOFFSymbolOffset])
	binary.LittleEndian.PutUint32(object[12:], uint32(count))
	for offset := testCOFFSymbolOffset; offset < end; offset += 18 {
		copy(object[offset:], base[testCOFFSymbolOffset:testCOFFSymbolOffset+18])
	}
	binary.LittleEndian.PutUint32(object[end:], 4)
	return object
}
