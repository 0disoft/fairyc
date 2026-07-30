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
	if len(report.SymbolRecords) != 1 || report.SymbolRecords[0] != (ObjectSymbol{
		TableIndex: 0, Name: "sample", SectionNumber: 1, Type: 0x20, StorageClass: 2,
	}) {
		t.Fatalf("COFF symbol records = %+v", report.SymbolRecords)
	}
	if len(report.RelocationRecords) != 1 || report.RelocationRecords[0] != (ObjectRelocation{
		SectionIndex: 1, Section: ".text", Symbol: "sample", Type: ObjectRelocationAMD64Rel32,
	}) {
		t.Fatalf("COFF relocation records = %+v", report.RelocationRecords)
	}
}

func TestInspectObjectReportsExactCOMDATAndRelocationMetadata(t *testing.T) {
	t.Parallel()
	report, err := InspectObject(comdatAMD64COFF())
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Sections) != 1 || report.Sections[0].Name != ".text$mn" ||
		report.Sections[0].COMDATSelection != ObjectCOMDATAny || report.Sections[0].AssociativeSection != 0 {
		t.Fatalf("COFF COMDAT section = %+v", report.Sections)
	}
	if len(report.SymbolRecords) != 3 || report.SymbolRecords[0] != (ObjectSymbol{
		TableIndex: 0, Name: ".text$mn", SectionNumber: 1, StorageClass: coffStorageClassStatic, AuxiliaryRecords: 1,
	}) || report.SymbolRecords[1] != (ObjectSymbol{
		TableIndex: 2, Name: "textkey", SectionNumber: 1, StorageClass: 2,
	}) || report.SymbolRecords[2] != (ObjectSymbol{
		TableIndex: 3, Name: "callee", StorageClass: 2,
	}) {
		t.Fatalf("COFF COMDAT symbols = %+v", report.SymbolRecords)
	}
	if len(report.RelocationRecords) != 1 || report.RelocationRecords[0] != (ObjectRelocation{
		SectionIndex: 1, Section: ".text$mn", SymbolIndex: 3, Symbol: "callee", Type: ObjectRelocationAMD64Rel32,
	}) {
		t.Fatalf("COFF COMDAT relocations = %+v", report.RelocationRecords)
	}
}

func TestInspectObjectReportsAssociativeCOMDATParent(t *testing.T) {
	t.Parallel()
	report, err := InspectObject(associativeCOMDATAMD64COFF())
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Sections) != 2 || report.Sections[0].COMDATSelection != ObjectCOMDATAny ||
		report.Sections[1].Name != ".rdata$x" ||
		report.Sections[1].COMDATSelection != ObjectCOMDATAssociative ||
		report.Sections[1].AssociativeSection != 1 {
		t.Fatalf("COFF associative COMDAT sections = %+v", report.Sections)
	}
}

func TestInspectObjectPreservesRelocationVirtualAddress(t *testing.T) {
	t.Parallel()
	object := minimalAMD64COFF()
	binary.LittleEndian.PutUint32(object[coffFileHeaderBytes+12:], 0x1000)
	binary.LittleEndian.PutUint32(object[testCOFFRelocOffset:], 0x1007)
	report, err := InspectObject(object)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.RelocationRecords) != 1 || report.RelocationRecords[0].VirtualAddress != 0x1007 {
		t.Fatalf("COFF relocation records = %+v", report.RelocationRecords)
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

func TestInspectObjectAcceptsMSVCFunctionLevelSectionCount(t *testing.T) {
	t.Parallel()
	const observedSections = 154
	report, err := InspectObject(manySectionAMD64COFF(observedSections))
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Sections) != observedSections || report.Sections[0].Name != ".text$mn" {
		t.Fatalf("COFF section receipt = %d, %q", len(report.Sections), report.Sections[0].Name)
	}
}

func TestInspectObjectRejectsSectionCountOverBudget(t *testing.T) {
	t.Parallel()
	object := manySectionAMD64COFF(maxObjectSections + 1)
	if _, err := InspectObject(object); err == nil || !strings.Contains(err.Error(), "invalid section count") {
		t.Fatalf("InspectObject() error = %v", err)
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
		{name: "unknown relocation type", edit: func(object []byte) {
			binary.LittleEndian.PutUint16(object[testCOFFRelocOffset+8:], 0xffff)
		}, want: "unknown AMD64 relocation type"},
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

func TestInspectObjectRejectsRelocationToAuxiliarySymbolAndInvalidCOMDAT(t *testing.T) {
	t.Parallel()
	for _, testCase := range []struct {
		name string
		edit func([]byte)
		want string
	}{
		{name: "auxiliary relocation", edit: func(object []byte) {
			binary.LittleEndian.PutUint32(object[comdatCOFFRelocOffset+4:], 1)
		}, want: "references auxiliary symbol"},
		{name: "missing section definition", edit: func(object []byte) {
			object[comdatCOFFSymbolOffset+16] = 0
		}, want: "has no exact section definition"},
		{name: "invalid selection", edit: func(object []byte) {
			object[comdatCOFFSymbolOffset+18+14] = 7
		}, want: "invalid selection"},
		{name: "extra section-definition auxiliary record", edit: func(object []byte) {
			object[comdatCOFFSymbolOffset+17] = 2
		}, want: "want exactly 1"},
		{name: "section-definition header mismatch", edit: func(object []byte) {
			binary.LittleEndian.PutUint32(object[comdatCOFFSymbolOffset+18:], 2)
		}, want: "does not match its header"},
		{name: "missing following COMDAT symbol", edit: func(object []byte) {
			binary.LittleEndian.PutUint16(object[comdatCOFFSymbolOffset+36+12:], 0)
		}, want: "has no following COMDAT symbol"},
		{name: "non-COMDAT section with parent", edit: func(object []byte) {
			const section = coffFileHeaderBytes
			binary.LittleEndian.PutUint32(object[section+36:], sectionContainsCode|sectionExecutable|sectionReadable)
			object[comdatCOFFSymbolOffset+18+14] = 0
			binary.LittleEndian.PutUint16(object[comdatCOFFSymbolOffset+18+12:], 1)
		}, want: "non-COMDAT section"},
		{name: "self association", edit: func(object []byte) {
			object[comdatCOFFSymbolOffset+18+14] = ObjectCOMDATAssociative
			binary.LittleEndian.PutUint16(object[comdatCOFFSymbolOffset+18+12:], 1)
		}, want: "invalid parent"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			object := comdatAMD64COFF()
			testCase.edit(object)
			if _, err := InspectObject(object); err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("InspectObject() error = %v", err)
			}
		})
	}
}

func TestInspectObjectRejectsInvalidAssociativeCOMDATChains(t *testing.T) {
	t.Parallel()
	for _, testCase := range []struct {
		name string
		edit func([]byte)
		want string
	}{
		{name: "non-COMDAT parent", edit: func(object []byte) {
			const textSection = coffFileHeaderBytes
			const textAuxiliary = associativeCOFFSymbolOffset + 18
			binary.LittleEndian.PutUint32(object[textSection+36:], sectionContainsCode|sectionExecutable|sectionReadable)
			object[textAuxiliary+14] = 0
		}, want: "is not COMDAT"},
		{name: "association cycle", edit: func(object []byte) {
			const textAuxiliary = associativeCOFFSymbolOffset + 18
			object[textAuxiliary+14] = ObjectCOMDATAssociative
			binary.LittleEndian.PutUint16(object[textAuxiliary+12:], 2)
		}, want: "contains a cycle"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			object := associativeCOMDATAMD64COFF()
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

const (
	comdatCOFFRawOffset    = coffFileHeaderBytes + coffSectionHeaderBytes
	comdatCOFFRelocOffset  = comdatCOFFRawOffset + 1
	comdatCOFFSymbolOffset = comdatCOFFRelocOffset + coffRelocationBytes
)

func comdatAMD64COFF() []byte {
	object := make([]byte, comdatCOFFSymbolOffset+4*18+4)
	binary.LittleEndian.PutUint16(object, machineAMD64)
	binary.LittleEndian.PutUint16(object[2:], 1)
	binary.LittleEndian.PutUint32(object[8:], comdatCOFFSymbolOffset)
	binary.LittleEndian.PutUint32(object[12:], 4)

	section := coffFileHeaderBytes
	copy(object[section:], ".text$mn")
	binary.LittleEndian.PutUint32(object[section+16:], 1)
	binary.LittleEndian.PutUint32(object[section+20:], comdatCOFFRawOffset)
	binary.LittleEndian.PutUint32(object[section+24:], comdatCOFFRelocOffset)
	binary.LittleEndian.PutUint16(object[section+32:], 1)
	binary.LittleEndian.PutUint32(object[section+36:], sectionContainsCode|sectionExecutable|sectionReadable|sectionLinkCOMDAT)
	object[comdatCOFFRawOffset] = 0xe8

	binary.LittleEndian.PutUint32(object[comdatCOFFRelocOffset+4:], 3)
	binary.LittleEndian.PutUint16(object[comdatCOFFRelocOffset+8:], ObjectRelocationAMD64Rel32)

	symbol := comdatCOFFSymbolOffset
	copy(object[symbol:], ".text$mn")
	binary.LittleEndian.PutUint16(object[symbol+12:], 1)
	object[symbol+16] = coffStorageClassStatic
	object[symbol+17] = 1
	auxiliary := symbol + 18
	binary.LittleEndian.PutUint32(object[auxiliary:], 1)
	binary.LittleEndian.PutUint16(object[auxiliary+4:], 1)
	object[auxiliary+14] = ObjectCOMDATAny

	key := symbol + 36
	copy(object[key:], "textkey")
	binary.LittleEndian.PutUint16(object[key+12:], 1)
	object[key+16] = 2

	callee := symbol + 54
	copy(object[callee:], "callee")
	object[callee+16] = 2
	binary.LittleEndian.PutUint32(object[callee+18:], 4)
	return object
}

const associativeCOFFSymbolOffset = coffFileHeaderBytes + 2*coffSectionHeaderBytes + 2 + coffRelocationBytes

func associativeCOMDATAMD64COFF() []byte {
	const (
		sectionTableEnd = coffFileHeaderBytes + 2*coffSectionHeaderBytes
		textRaw         = sectionTableEnd
		dataRaw         = textRaw + 1
		relocation      = dataRaw + 1
		symbolTable     = associativeCOFFSymbolOffset
	)
	object := make([]byte, symbolTable+7*18+4)
	binary.LittleEndian.PutUint16(object, machineAMD64)
	binary.LittleEndian.PutUint16(object[2:], 2)
	binary.LittleEndian.PutUint32(object[8:], symbolTable)
	binary.LittleEndian.PutUint32(object[12:], 7)

	textSection := coffFileHeaderBytes
	copy(object[textSection:], ".text$mn")
	binary.LittleEndian.PutUint32(object[textSection+16:], 1)
	binary.LittleEndian.PutUint32(object[textSection+20:], textRaw)
	binary.LittleEndian.PutUint32(object[textSection+24:], relocation)
	binary.LittleEndian.PutUint16(object[textSection+32:], 1)
	binary.LittleEndian.PutUint32(object[textSection+36:], sectionContainsCode|sectionExecutable|sectionReadable|sectionLinkCOMDAT)
	object[textRaw] = 0xe8

	dataSection := textSection + coffSectionHeaderBytes
	copy(object[dataSection:], ".rdata$x")
	binary.LittleEndian.PutUint32(object[dataSection+16:], 1)
	binary.LittleEndian.PutUint32(object[dataSection+20:], dataRaw)
	binary.LittleEndian.PutUint32(object[dataSection+36:], 0x00000040|sectionReadable|sectionLinkCOMDAT)
	object[dataRaw] = 0x2a

	binary.LittleEndian.PutUint32(object[relocation+4:], 6)
	binary.LittleEndian.PutUint16(object[relocation+8:], ObjectRelocationAMD64Rel32)

	textSymbol := symbolTable
	copy(object[textSymbol:], ".text$mn")
	binary.LittleEndian.PutUint16(object[textSymbol+12:], 1)
	object[textSymbol+16] = coffStorageClassStatic
	object[textSymbol+17] = 1
	textAuxiliary := textSymbol + 18
	binary.LittleEndian.PutUint32(object[textAuxiliary:], 1)
	binary.LittleEndian.PutUint16(object[textAuxiliary+4:], 1)
	object[textAuxiliary+14] = ObjectCOMDATAny

	textKey := textSymbol + 36
	copy(object[textKey:], "textkey")
	binary.LittleEndian.PutUint16(object[textKey+12:], 1)
	object[textKey+16] = 2

	dataSymbol := textSymbol + 54
	copy(object[dataSymbol:], ".rdata$x")
	binary.LittleEndian.PutUint16(object[dataSymbol+12:], 2)
	object[dataSymbol+16] = coffStorageClassStatic
	object[dataSymbol+17] = 1
	dataAuxiliary := dataSymbol + 18
	binary.LittleEndian.PutUint32(object[dataAuxiliary:], 1)
	binary.LittleEndian.PutUint16(object[dataAuxiliary+12:], 1)
	object[dataAuxiliary+14] = ObjectCOMDATAssociative

	dataKey := dataSymbol + 36
	copy(object[dataKey:], "datakey")
	binary.LittleEndian.PutUint16(object[dataKey+12:], 2)
	object[dataKey+16] = 2

	callee := dataSymbol + 54
	copy(object[callee:], "callee")
	object[callee+16] = 2
	binary.LittleEndian.PutUint32(object[callee+18:], 4)
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

func manySectionAMD64COFF(sectionCount int) []byte {
	headerEnd := coffFileHeaderBytes + sectionCount*coffSectionHeaderBytes
	rawOffset := headerEnd
	relocOffset := rawOffset + 1
	symbolOffset := relocOffset + coffRelocationBytes
	object := make([]byte, symbolOffset+18+4)
	binary.LittleEndian.PutUint16(object, machineAMD64)
	binary.LittleEndian.PutUint16(object[2:], uint16(sectionCount))
	binary.LittleEndian.PutUint32(object[8:], uint32(symbolOffset))
	binary.LittleEndian.PutUint32(object[12:], 1)

	for index := 0; index < sectionCount; index++ {
		section := coffFileHeaderBytes + index*coffSectionHeaderBytes
		if index == 0 {
			copy(object[section:], ".text$mn")
			binary.LittleEndian.PutUint32(object[section+16:], 1)
			binary.LittleEndian.PutUint32(object[section+20:], uint32(rawOffset))
			binary.LittleEndian.PutUint32(object[section+24:], uint32(relocOffset))
			binary.LittleEndian.PutUint16(object[section+32:], 1)
			binary.LittleEndian.PutUint32(object[section+36:], sectionContainsCode|sectionExecutable|sectionReadable)
			continue
		}
		copy(object[section:], ".bss")
		binary.LittleEndian.PutUint32(object[section+36:], sectionUninitializedData|sectionReadable|0x80000000)
	}
	object[rawOffset] = 0xc3
	binary.LittleEndian.PutUint32(object[relocOffset+4:], 0)
	binary.LittleEndian.PutUint16(object[relocOffset+8:], 4)
	copy(object[symbolOffset:], "sample")
	binary.LittleEndian.PutUint16(object[symbolOffset+12:], 1)
	binary.LittleEndian.PutUint16(object[symbolOffset+14:], 0x20)
	object[symbolOffset+16] = 2
	binary.LittleEndian.PutUint32(object[symbolOffset+18:], 4)
	return object
}
