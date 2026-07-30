package pecoff

import (
	"bytes"
	"debug/pe"
	"encoding/binary"
	"strings"
	"testing"
)

func TestInspectAssociativeCOMDATUnwindLinkBindsObjectMapAndImage(t *testing.T) {
	report, err := InspectAssociativeCOMDATUnwindLink(testAssociativeLinkObject(), testAddressImage(), testAssociativeLinkMap(), testAssociativeLinkSpec())
	if err != nil {
		t.Fatal(err)
	}
	if report.ParentSectionIndex != 1 || report.PDataSectionIndex != 3 || report.XDataSectionIndex != 4 ||
		report.BeginAddend != 0 || report.EndAddend != 1 || report.UnwindAddend != 0 ||
		report.FinalBeginRVA != testAddressSourceRVA || report.FinalEndRVA != testAddressSourceRVA+1 ||
		report.FinalUnwindRVA != 0x3000 || report.ExpectedEndRVA != testAddressSourceRVA+1 || report.ExpectedUnwindRVA != 0x3000 {
		t.Fatalf("associative COMDAT unwind report = %+v", report)
	}
}

func TestInspectAssociativeCOMDATUnwindLinkRejectsIndependentLinkMutations(t *testing.T) {
	tests := []struct {
		name string
		edit func(*[]byte, *[]byte, *[]byte)
		want string
	}{
		{name: "begin addend drift", edit: func(object, _, _ *[]byte) {
			binary.LittleEndian.PutUint32((*object)[testAddressPDataRaw:], 1)
		}, want: "0 entries beginning"},
		{name: "end addend drift", edit: func(object, _, _ *[]byte) {
			binary.LittleEndian.PutUint32((*object)[testAddressPDataRaw+4:], 2)
		}, want: "final end RVA"},
		{name: "unwind addend drift", edit: func(object, _, _ *[]byte) {
			binary.LittleEndian.PutUint32((*object)[testAddressPDataRaw+8:], 4)
		}, want: "final unwind-info RVA"},
		{name: "archive owner spoof", edit: func(_, _, linkMap *[]byte) {
			*linkMap = bytes.Replace(*linkMap, []byte("choice:chosen.obj"), []byte("evilll:chosen.obj"), 1)
		}, want: "parent owner"},
		{name: "unwind owner drift", edit: func(_, _, linkMap *[]byte) {
			index := bytes.LastIndex(*linkMap, []byte("choice:chosen.obj"))
			copy((*linkMap)[index:], []byte("evilll:chosen.obj"))
		}, want: "unwind-info owner"},
		{name: "source map address drift", edit: func(_, _, linkMap *[]byte) {
			*linkMap = bytes.Replace(*linkMap, []byte("0000000140001000"), []byte("0000000140001001"), 1)
		}, want: "0 entries beginning"},
		{name: "unwind map address drift", edit: func(_, _, linkMap *[]byte) {
			*linkMap = bytes.Replace(*linkMap, []byte("0000000140003000"), []byte("0000000140003004"), 1)
		}, want: "final unwind-info RVA"},
		{name: "missing final begin", edit: func(_, image, _ *[]byte) {
			binary.LittleEndian.PutUint32((*image)[testUnwindPDataRawOffset:], testAddressSourceRVA+2)
			binary.LittleEndian.PutUint32((*image)[testUnwindPDataRawOffset+4:], testAddressSourceRVA+3)
		}, want: "0 entries beginning"},
		{name: "duplicate final begin", edit: func(_, image, _ *[]byte) {
			copy((*image)[testUnwindPDataRawOffset+runtimeFunctionBytes:], (*image)[testUnwindPDataRawOffset:testUnwindPDataRawOffset+runtimeFunctionBytes])
			binary.LittleEndian.PutUint32((*image)[testUnwindDirectoryOffset+4:], 2*runtimeFunctionBytes)
		}, want: "unwind"},
		{name: "valid but wrong final unwind", edit: func(_, image, _ *[]byte) {
			binary.LittleEndian.PutUint32((*image)[testUnwindPDataRawOffset+8:], 0x3004)
			copy((*image)[testUnwindXDataRawOffset+4:testUnwindXDataRawOffset+8], (*image)[testUnwindXDataRawOffset:testUnwindXDataRawOffset+4])
		}, want: "final unwind-info RVA"},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			object := testAssociativeLinkObject()
			image := testAddressImage()
			linkMap := testAssociativeLinkMap()
			testCase.edit(&object, &image, &linkMap)
			if _, err := InspectAssociativeCOMDATUnwindLink(object, image, linkMap, testAssociativeLinkSpec()); err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("InspectAssociativeCOMDATUnwindLink() error = %v, want %q", err, testCase.want)
			}
		})
	}
}

func TestExactAssociativeCOMDATUnwindBindsDirectChildren(t *testing.T) {
	report, file, source := testAssociativeCOMDATUnwindMetadata()
	entry, err := exactAssociativeCOMDATUnwind(report, file, source)
	if err != nil {
		t.Fatal(err)
	}
	if entry.Begin.SectionIndex != 1 || entry.UnwindTarget.SectionNumber != 3 || entry.BeginTarget.Name != "$begin" {
		t.Fatalf("associative COMDAT unwind entry = %+v", entry)
	}
}

func TestExactAssociativeCOMDATUnwindRejectsIndependentRelationshipMutations(t *testing.T) {
	for _, testCase := range []struct {
		name string
		edit func(*ObjectReport)
		want string
	}{
		{name: "parent loses COMDAT", edit: func(report *ObjectReport) {
			report.Sections[1].COMDATSelection = 0
		}, want: "not an executable root COMDAT"},
		{name: "pdata loses association", edit: func(report *ObjectReport) {
			report.Sections[0].COMDATSelection = ObjectCOMDATAny
		}, want: ".pdata is not a direct associative child"},
		{name: "pdata changes parent", edit: func(report *ObjectReport) {
			report.Sections[0].AssociativeSection = 3
		}, want: ".pdata is not a direct associative child"},
		{name: "xdata loses association", edit: func(report *ObjectReport) {
			report.Sections[2].COMDATSelection = ObjectCOMDATAny
		}, want: ".xdata is not a direct associative child"},
		{name: "xdata changes parent", edit: func(report *ObjectReport) {
			report.Sections[2].AssociativeSection = 1
		}, want: ".xdata is not a direct associative child"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			report, file, source := testAssociativeCOMDATUnwindMetadata()
			testCase.edit(&report)
			if _, err := exactAssociativeCOMDATUnwind(report, file, source); err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("exactAssociativeCOMDATUnwind() error = %v, want %q", err, testCase.want)
			}
		})
	}
}

func testAssociativeCOMDATUnwindMetadata() (ObjectReport, *pe.File, ObjectSymbol) {
	file := &pe.File{Sections: []*pe.Section{
		{SectionHeader: pe.SectionHeader{Name: ".pdata", Size: runtimeFunctionBytes}},
		{SectionHeader: pe.SectionHeader{Name: ".text$fn", Size: 8}},
		{SectionHeader: pe.SectionHeader{Name: ".xdata", Size: 4}},
	}}
	source := ObjectSymbol{TableIndex: 1, Name: "source", SectionNumber: 2, Type: coffSymbolTypeFunction, StorageClass: coffStorageClassExternal}
	begin := ObjectSymbol{TableIndex: 2, Name: "$begin", SectionNumber: 2, StorageClass: coffStorageClassStatic}
	unwind := ObjectSymbol{TableIndex: 3, Name: "$unwind", SectionNumber: 3, StorageClass: coffStorageClassStatic}
	report := ObjectReport{
		Sections: []ObjectSection{
			{Name: ".pdata", Characteristics: sectionInitializedData | sectionReadable | sectionLinkCOMDAT, COMDATSelection: ObjectCOMDATAssociative, AssociativeSection: 2},
			{Name: ".text$fn", Characteristics: sectionContainsCode | sectionExecutable | sectionReadable | sectionLinkCOMDAT, COMDATSelection: ObjectCOMDATAny},
			{Name: ".xdata", RawOffset: 1, RawSize: 4, Characteristics: sectionInitializedData | sectionReadable | sectionLinkCOMDAT, COMDATSelection: ObjectCOMDATAssociative, AssociativeSection: 2},
		},
		SymbolRecords: []ObjectSymbol{source, begin, unwind},
		RelocationRecords: []ObjectRelocation{
			{SectionIndex: 1, Section: ".pdata", SymbolIndex: begin.TableIndex, Symbol: begin.Name, Type: ObjectRelocationAMD64Addr32NB},
			{SectionIndex: 1, Section: ".pdata", VirtualAddress: 4, SymbolIndex: begin.TableIndex, Symbol: begin.Name, Type: ObjectRelocationAMD64Addr32NB},
			{SectionIndex: 1, Section: ".pdata", VirtualAddress: 8, SymbolIndex: unwind.TableIndex, Symbol: unwind.Name, Type: ObjectRelocationAMD64Addr32NB},
		},
	}
	return report, file, source
}

func testAssociativeLinkSpec() AssociativeCOMDATUnwindSpec {
	return AssociativeCOMDATUnwindSpec{ArchiveName: "choice.lib", SourceMember: "chosen.obj", SourceSymbol: "source"}
}

func testAssociativeLinkMap() []byte {
	return []byte(" 0001:00000000 source 0000000140001000 f choice:chosen.obj\r\n" +
		" 0003:00000000 $unwind 0000000140003000 f choice:chosen.obj\r\n")
}

func testAssociativeLinkObject() []byte {
	object := make([]byte, testAddressSymbols+12*18+4)
	copy(object, testAddressObject()[:testAddressSymbols])
	binary.LittleEndian.PutUint32(object[12:], 12)

	text := coffFileHeaderBytes
	pdata := text + 2*coffSectionHeaderBytes
	xdata := text + 3*coffSectionHeaderBytes
	binary.LittleEndian.PutUint32(object[text+36:], sectionContainsCode|sectionExecutable|sectionReadable|sectionLinkCOMDAT)
	binary.LittleEndian.PutUint32(object[pdata+36:], sectionInitializedData|sectionReadable|sectionLinkCOMDAT)
	binary.LittleEndian.PutUint32(object[xdata+36:], sectionInitializedData|sectionReadable|sectionLinkCOMDAT)
	binary.LittleEndian.PutUint32(object[testAddressReceiptReloc+4:], 11)
	binary.LittleEndian.PutUint32(object[testAddressPDataReloc+4:], 6)
	binary.LittleEndian.PutUint32(object[testAddressPDataReloc+10+4:], 6)
	binary.LittleEndian.PutUint32(object[testAddressPDataReloc+20+4:], 9)

	symbol := testAddressSymbols
	emitAssociativeSectionDefinition(object[symbol:], ".text", 1, 1, 0, ObjectCOMDATAny)
	symbol += 36
	emitAssociativeTestSymbol(object[symbol:], "source", 1, coffSymbolTypeFunction, coffStorageClassExternal)
	symbol += 18
	emitAssociativeSectionDefinition(object[symbol:], ".pdata", 3, runtimeFunctionBytes, 1, ObjectCOMDATAssociative)
	binary.LittleEndian.PutUint16(object[symbol+18+4:], 3)
	symbol += 36
	emitAssociativeTestSymbol(object[symbol:], "pdata", 3, 0, coffStorageClassExternal)
	symbol += 18
	emitAssociativeTestSymbol(object[symbol:], "$begin", 1, 0, coffStorageClassStatic)
	symbol += 18
	emitAssociativeSectionDefinition(object[symbol:], ".xdata", 4, 4, 1, ObjectCOMDATAssociative)
	symbol += 36
	emitAssociativeTestSymbol(object[symbol:], "$unwind", 4, 0, coffStorageClassStatic)
	symbol += 18
	emitAssociativeTestSymbol(object[symbol:], "receipt", 2, 0, coffStorageClassExternal)
	symbol += 18
	emitAssociativeTestSymbol(object[symbol:], "target", 0, coffSymbolTypeFunction, coffStorageClassExternal)
	binary.LittleEndian.PutUint32(object[testAddressSymbols+12*18:], 4)
	return object
}

func emitAssociativeSectionDefinition(target []byte, name string, section int16, length uint32, parent uint16, selection uint8) {
	copy(target, name)
	binary.LittleEndian.PutUint16(target[12:], uint16(section))
	target[16] = coffStorageClassStatic
	target[17] = 1
	binary.LittleEndian.PutUint32(target[18:], length)
	binary.LittleEndian.PutUint16(target[18+12:], parent)
	target[18+14] = selection
}

func emitAssociativeTestSymbol(target []byte, name string, section int16, symbolType uint16, storage uint8) {
	copy(target, name)
	binary.LittleEndian.PutUint16(target[12:], uint16(section))
	binary.LittleEndian.PutUint16(target[14:], symbolType)
	target[16] = storage
}
