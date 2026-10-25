package pecoff

import (
	"encoding/binary"
	"strings"
	"testing"
)

func TestInspectObjectRejectsRelocationBudgetBeforeParsing(t *testing.T) {
	object := sharedRelocationBudgetObject(17)
	// Retain all section headers, but omit the symbol and relocation tables.
	object = object[:coffFileHeaderBytes+17*coffSectionHeaderBytes]
	if _, err := InspectObject(object); err == nil || !strings.Contains(err.Error(), "more than 1048576 relocations") {
		t.Fatalf("budget rejection did not precede parser allocation: %v", err)
	}
}

func TestInspectObjectRelocationBudgetBoundary(t *testing.T) {
	object := sharedRelocationBudgetObject(17)
	last := coffFileHeaderBytes + 16*coffSectionHeaderBytes
	// 16*65,535 + 16 = 1,048,576: preserve admission at the exact limit.
	binary.LittleEndian.PutUint16(object[last+32:], 16)
	if _, err := InspectObject(object); err != nil {
		t.Fatal(err)
	}
	binary.LittleEndian.PutUint16(object[last+32:], 17)
	if _, err := InspectObject(object); err == nil || !strings.Contains(err.Error(), "more than 1048576 relocations") {
		t.Fatalf("accepted relocation count above the limit: %v", err)
	}
}

func TestInspectObjectExtendedRelocationBudgetBeforeParsing(t *testing.T) {
	object, err := BuildAMD64RelocationOverflowProbeObject("fy_bud")
	if err != nil {
		t.Fatal(err)
	}
	header := coffFileHeaderBytes
	pointer := binary.LittleEndian.Uint32(object[header+24:])
	binary.LittleEndian.PutUint32(object[pointer:], maxObjectRelocations+2)
	if _, err := InspectObject(object); err == nil || !strings.Contains(err.Error(), "limit is 1048576") {
		t.Fatalf("extended count was not bounded before parsing: %v", err)
	}
}

func TestInspectObjectInvalidExtendedCountStillBoundsParserAllocations(t *testing.T) {
	for _, stored := range []uint32{0, 1} {
		object := sharedRelocationBudgetObject(64)
		for index := range 64 {
			header := object[coffFileHeaderBytes+index*coffSectionHeaderBytes:]
			flags := binary.LittleEndian.Uint32(header[36:40])
			binary.LittleEndian.PutUint32(header[36:], flags|sectionRelocationOverflow)
		}
		pointer := binary.LittleEndian.Uint32(object[coffFileHeaderBytes+24:])
		binary.LittleEndian.PutUint32(object[pointer:], stored)
		if _, err := InspectObject(object); err == nil || !strings.Contains(err.Error(), "more than 1048576 relocations") {
			t.Fatalf("invalid extended count %d hid parser allocations: %v", stored, err)
		}
	}
}

func BenchmarkInspectObjectOverRelocationBudget(b *testing.B) {
	object := sharedRelocationBudgetObject(64)
	b.ReportAllocs()
	for b.Loop() {
		if _, err := InspectObject(object); err == nil || !strings.Contains(err.Error(), "more than 1048576 relocations") {
			b.Fatal(err)
		}
	}
}

func sharedRelocationBudgetObject(sections int) []byte {
	const count = 65535
	headerEnd := coffFileHeaderBytes + sections*coffSectionHeaderBytes
	relocationOffset := headerEnd + 1
	symbolOffset := relocationOffset + count*coffRelocationBytes
	object := make([]byte, symbolOffset+18+4)
	binary.LittleEndian.PutUint16(object, machineAMD64)
	binary.LittleEndian.PutUint16(object[2:], uint16(sections))
	binary.LittleEndian.PutUint32(object[8:], uint32(symbolOffset))
	binary.LittleEndian.PutUint32(object[12:], 1)
	for index := range sections {
		header := object[coffFileHeaderBytes+index*coffSectionHeaderBytes:]
		copy(header, ".bss")
		binary.LittleEndian.PutUint32(header[36:], sectionUninitializedData|sectionReadable)
		if index == 0 {
			copy(header[:8], []byte(".text\x00\x00\x00"))
			binary.LittleEndian.PutUint32(header[16:], 1)
			binary.LittleEndian.PutUint32(header[20:], uint32(headerEnd))
			binary.LittleEndian.PutUint32(header[36:], sectionContainsCode|sectionExecutable|sectionReadable)
		}
		binary.LittleEndian.PutUint32(header[24:], uint32(relocationOffset))
		binary.LittleEndian.PutUint16(header[32:], count)
	}
	object[headerEnd] = 0xc3
	// Every section references the same valid ABSOLUTE relocation records.
	copy(object[symbolOffset:], minimalAMD64COFF()[testCOFFSymbolOffset:testCOFFSymbolOffset+18])
	binary.LittleEndian.PutUint32(object[symbolOffset+18:], 4)
	return object
}
