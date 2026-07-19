package pecoff

import (
	"encoding/binary"
	"strings"
	"testing"
)

const (
	testCOFFSectionOffset = coffFileHeaderBytes
	testCOFFRawOffset     = coffFileHeaderBytes + coffSectionHeaderBytes
	testCOFFRelocOffset   = testCOFFRawOffset + 1
	testCOFFSymbolOffset  = testCOFFRelocOffset + coffRelocationBytes
)

func TestInspectObjectAcceptsClosedAMD64COFF(t *testing.T) {
	t.Parallel()
	report, err := InspectObject(minimalAMD64COFF())
	if err != nil {
		t.Fatal(err)
	}
	if report.Machine != machineAMD64 || len(report.Sections) != 1 || report.Sections[0].Name != ".text" || report.Relocations != 1 {
		t.Fatalf("COFF report = %+v", report)
	}
	if len(report.Symbols) != 1 || report.Symbols[0] != "sample" {
		t.Fatalf("COFF symbols = %q", report.Symbols)
	}
}

func TestInspectObjectAcceptsMSVCTextSubsection(t *testing.T) {
	t.Parallel()
	object := minimalAMD64COFF()
	copy(object[testCOFFSectionOffset:testCOFFSectionOffset+8], []byte(".text$mn"))
	report, err := InspectObject(object)
	if err != nil {
		t.Fatal(err)
	}
	if report.Sections[0].Name != ".text$mn" {
		t.Fatalf("COFF section = %q", report.Sections[0].Name)
	}
}

func TestInspectObjectAcceptsUninitializedBSSWithoutRawStorage(t *testing.T) {
	t.Parallel()
	report, err := InspectObject(minimalAMD64COFFWithBSS())
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Sections) != 2 || report.Sections[1].Name != ".bss" || report.Sections[1].RawOffset != 0 || report.Sections[1].RawSize != 8 {
		t.Fatalf("COFF BSS report = %+v", report)
	}
}

func TestInspectObjectRejectsMalformedIdentityAndRanges(t *testing.T) {
	t.Parallel()
	for _, testCase := range []struct {
		name string
		edit func([]byte)
		want string
	}{
		{name: "linked PE", edit: func(object []byte) { copy(object, "MZ") }, want: "must not be a linked PE image"},
		{name: "x86 machine", edit: func(object []byte) { binary.LittleEndian.PutUint16(object, 0x014c) }, want: "want AMD64"},
		{name: "linked characteristic", edit: func(object []byte) { binary.LittleEndian.PutUint16(object[18:], executableImage) }, want: "linked-image characteristics"},
		{name: "entry section not executable", edit: func(object []byte) {
			binary.LittleEndian.PutUint32(object[testCOFFSectionOffset+36:], sectionContainsCode|sectionReadable)
		}, want: ".text section is not readable executable code"},
		{name: "raw range overlaps headers", edit: func(object []byte) {
			binary.LittleEndian.PutUint32(object[testCOFFSectionOffset+20:], 0)
		}, want: "invalid zero raw pointer"},
		{name: "relocation references missing symbol", edit: func(object []byte) {
			binary.LittleEndian.PutUint32(object[testCOFFRelocOffset+4:], 1)
		}, want: "references missing symbol"},
	} {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			object := minimalAMD64COFF()
			testCase.edit(object)
			if _, err := InspectObject(object); err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("InspectObject() error = %v", err)
			}
		})
	}
}

func TestInspectObjectRejectsTruncatedObject(t *testing.T) {
	t.Parallel()
	if _, err := InspectObject(make([]byte, coffFileHeaderBytes-1)); err == nil || !strings.Contains(err.Error(), "invalid size") {
		t.Fatalf("InspectObject() error = %v", err)
	}
}

func minimalAMD64COFF() []byte {
	// debug/pe reads a 96-byte discriminator block before deciding whether the
	// input is a linked PE image or a raw COFF object.
	object := make([]byte, 128)
	binary.LittleEndian.PutUint16(object, machineAMD64)
	binary.LittleEndian.PutUint16(object[2:], 1)
	binary.LittleEndian.PutUint32(object[8:], testCOFFSymbolOffset)
	binary.LittleEndian.PutUint32(object[12:], 1)

	section := testCOFFSectionOffset
	copy(object[section:], ".text")
	binary.LittleEndian.PutUint32(object[section+16:], 1)
	binary.LittleEndian.PutUint32(object[section+20:], testCOFFRawOffset)
	binary.LittleEndian.PutUint32(object[section+24:], testCOFFRelocOffset)
	binary.LittleEndian.PutUint16(object[section+32:], 1)
	binary.LittleEndian.PutUint32(object[section+36:], sectionContainsCode|sectionExecutable|sectionReadable)
	object[testCOFFRawOffset] = 0xc3

	relocation := testCOFFRelocOffset
	binary.LittleEndian.PutUint32(object[relocation:], 0)
	binary.LittleEndian.PutUint32(object[relocation+4:], 0)
	binary.LittleEndian.PutUint16(object[relocation+8:], 4)

	symbol := testCOFFSymbolOffset
	copy(object[symbol:], "sample")
	binary.LittleEndian.PutUint16(object[symbol+12:], 1)
	binary.LittleEndian.PutUint16(object[symbol+14:], 0x20)
	object[symbol+16] = 2
	binary.LittleEndian.PutUint32(object[symbol+18:], 4)
	return object
}

func minimalAMD64COFFWithBSS() []byte {
	const (
		secondSection = coffFileHeaderBytes + coffSectionHeaderBytes
		rawOffset     = coffFileHeaderBytes + 2*coffSectionHeaderBytes
		relocOffset   = rawOffset + 1
		symbolOffset  = relocOffset + coffRelocationBytes
	)
	object := make([]byte, 160)
	binary.LittleEndian.PutUint16(object, machineAMD64)
	binary.LittleEndian.PutUint16(object[2:], 2)
	binary.LittleEndian.PutUint32(object[8:], symbolOffset)
	binary.LittleEndian.PutUint32(object[12:], 1)

	copy(object[testCOFFSectionOffset:], ".text")
	binary.LittleEndian.PutUint32(object[testCOFFSectionOffset+16:], 1)
	binary.LittleEndian.PutUint32(object[testCOFFSectionOffset+20:], rawOffset)
	binary.LittleEndian.PutUint32(object[testCOFFSectionOffset+24:], relocOffset)
	binary.LittleEndian.PutUint16(object[testCOFFSectionOffset+32:], 1)
	binary.LittleEndian.PutUint32(object[testCOFFSectionOffset+36:], sectionContainsCode|sectionExecutable|sectionReadable)
	object[rawOffset] = 0xc3

	copy(object[secondSection:], ".bss")
	binary.LittleEndian.PutUint32(object[secondSection+16:], 8)
	binary.LittleEndian.PutUint32(object[secondSection+36:], sectionUninitializedData|sectionReadable|0x80000000)

	binary.LittleEndian.PutUint32(object[relocOffset+4:], 0)
	binary.LittleEndian.PutUint16(object[relocOffset+8:], 4)
	copy(object[symbolOffset:], "sample")
	binary.LittleEndian.PutUint16(object[symbolOffset+12:], 1)
	binary.LittleEndian.PutUint16(object[symbolOffset+14:], 0x20)
	object[symbolOffset+16] = 2
	binary.LittleEndian.PutUint32(object[symbolOffset+18:], 4)
	return object
}
