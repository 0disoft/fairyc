package pecoff

import (
	"bytes"
	"debug/pe"
	"encoding/binary"
	"math"
	"strings"
	"testing"
)

const (
	testAddressTextRaw      = 180
	testAddressReceiptRaw   = 184
	testAddressReceiptReloc = 216
	testAddressPDataRaw     = 228
	testAddressPDataReloc   = 240
	testAddressXDataRaw     = 284
	testAddressSymbols      = 288
	testAddressImageReceipt = 0xa00
	testAddressSourceRVA    = 0x1000
	testAddressTargetRVA    = 0x1020
	testAddressReceiptRVA   = 0x4000
)

var testAddressMagic = [16]byte{'F', 'Y', 'A', 'D', 'D', 'R', '-', 'T', 'E', 'S', 'T', '-', 'V', '0', '0', '!'}

func TestInspectAMD64AddressLinkProvesADDR64AndPDataADDR32NB(t *testing.T) {
	report, err := InspectAMD64AddressLink(testAddressObject(), testAddressImage(), testAddressMap(), testAddressSpec())
	if err != nil {
		t.Fatal(err)
	}
	if report.ADDR64ObjectAddend != 0 || report.ADDR64LinkedValue != 0x140001020 || report.ADDR64ExpectedValue != 0x140001020 ||
		report.ADDR32NBObjectAddend != 0 || report.ADDR32NBLinkedBeginRVA != testAddressSourceRVA ||
		report.ADDR32NBObjectTarget != "$begin" ||
		report.ADDR32NBExpectedBeginRVA != testAddressSourceRVA || report.RuntimeFunctionEndRVA != testAddressSourceRVA+1 ||
		report.ADDR32NBEndObjectAddend != 1 || report.ADDR32NBEndObjectTarget != "$begin" ||
		report.ExpectedFunctionEndRVA != testAddressSourceRVA+1 || report.ADDR32NBUnwindAddend != 0 ||
		report.ADDR32NBUnwindTarget != "$unwind" || report.UnwindInfoRVA != 0x3000 || report.ExpectedUnwindInfoRVA != 0x3000 {
		t.Fatalf("InspectAMD64AddressLink() report = %+v", report)
	}
}

func TestInspectAMD64AddressLinkAcceptsIndependentNonzeroAddends(t *testing.T) {
	object := testAddressObject()
	image := testAddressImage()
	binary.LittleEndian.PutUint64(object[testAddressReceiptRaw+amd64AddressReceiptTargetOffset:], 4)
	binary.LittleEndian.PutUint32(object[testAddressPDataRaw:], 4)
	binary.LittleEndian.PutUint32(object[testAddressPDataRaw+4:], 6)
	binary.LittleEndian.PutUint32(object[testAddressPDataRaw+8:], 8)
	binary.LittleEndian.PutUint64(image[testAddressImageReceipt+amd64AddressReceiptTargetOffset:], 0x140001024)
	binary.LittleEndian.PutUint32(image[testUnwindPDataRawOffset:], testAddressSourceRVA+4)
	binary.LittleEndian.PutUint32(image[testUnwindPDataRawOffset+4:], testAddressSourceRVA+6)
	binary.LittleEndian.PutUint32(image[testUnwindPDataRawOffset+8:], 0x3008)
	copy(image[testUnwindXDataRawOffset+8:testUnwindXDataRawOffset+12], image[testUnwindXDataRawOffset:testUnwindXDataRawOffset+4])
	report, err := InspectAMD64AddressLink(object, image, testAddressMap(), testAddressSpec())
	if err != nil {
		t.Fatal(err)
	}
	if report.ADDR64ObjectAddend != 4 || report.ADDR64LinkedValue != 0x140001024 || report.ADDR64ExpectedValue != 0x140001024 ||
		report.ADDR32NBObjectAddend != 4 || report.ADDR32NBLinkedBeginRVA != testAddressSourceRVA+4 ||
		report.ADDR32NBExpectedBeginRVA != testAddressSourceRVA+4 || report.ADDR32NBEndObjectAddend != 6 ||
		report.RuntimeFunctionEndRVA != testAddressSourceRVA+6 || report.ExpectedFunctionEndRVA != testAddressSourceRVA+6 ||
		report.ADDR32NBUnwindAddend != 8 || report.UnwindInfoRVA != 0x3008 || report.ExpectedUnwindInfoRVA != 0x3008 {
		t.Fatalf("nonzero-addend report = %+v", report)
	}
}

func TestInspectAMD64AddressLinkRejectsIndependentObjectMapAndImageMutations(t *testing.T) {
	tests := []struct {
		name string
		edit func(*[]byte, *[]byte, *[]byte)
		want string
	}{
		{name: "ADDR64 type drift", edit: func(object, _, _ *[]byte) {
			binary.LittleEndian.PutUint16((*object)[testAddressReceiptReloc+8:], ObjectRelocationAMD64Addr32)
		}, want: "ADDR64: object has no exact relocation"},
		{name: "ADDR64 addend drift", edit: func(object, _, _ *[]byte) {
			binary.LittleEndian.PutUint64((*object)[testAddressReceiptRaw+amd64AddressReceiptTargetOffset:], 1)
		}, want: "ADDR64 linked value"},
		{name: "ADDR64 linked value drift", edit: func(_, image, _ *[]byte) {
			binary.LittleEndian.PutUint64((*image)[testAddressImageReceipt+amd64AddressReceiptTargetOffset:], 0x140001021)
		}, want: "ADDR64 linked value"},
		{name: "target map owner drift", edit: func(_, _, linkMap *[]byte) {
			*linkMap = bytes.Replace(*linkMap, []byte("target.obj"), []byte("otherx.obj"), 1)
		}, want: "map target owner"},
		{name: "target map address drift", edit: func(_, _, linkMap *[]byte) {
			*linkMap = bytes.Replace(*linkMap, []byte("0000000140001020"), []byte("0000000140001021"), 1)
		}, want: "ADDR64 linked value"},
		{name: "ADDR32NB type drift", edit: func(object, _, _ *[]byte) {
			binary.LittleEndian.PutUint16((*object)[testAddressPDataReloc+8:], ObjectRelocationAMD64Addr32)
		}, want: "no exact source .pdata begin relocation"},
		{name: "ADDR32NB begin duplicate site", edit: func(object, _, _ *[]byte) {
			binary.LittleEndian.PutUint32((*object)[testAddressPDataReloc+10:], 0)
		}, want: ".pdata begin: object has 2 relocations"},
		{name: "ADDR32NB begin alias value drift", edit: func(object, _, _ *[]byte) {
			binary.LittleEndian.PutUint32((*object)[testAddressSymbols+3*18+8:], 1)
		}, want: "no exact source .pdata begin relocation"},
		{name: "ADDR32NB addend drift", edit: func(object, _, _ *[]byte) {
			binary.LittleEndian.PutUint32((*object)[testAddressPDataRaw:], 1)
		}, want: "0 entries beginning"},
		{name: "runtime function begin drift", edit: func(_, image, _ *[]byte) {
			binary.LittleEndian.PutUint32((*image)[testUnwindPDataRawOffset:], testAddressSourceRVA+1)
			binary.LittleEndian.PutUint32((*image)[testUnwindPDataRawOffset+4:], testAddressSourceRVA+2)
		}, want: "0 entries beginning"},
		{name: "ADDR32NB end type drift", edit: func(object, _, _ *[]byte) {
			binary.LittleEndian.PutUint16((*object)[testAddressPDataReloc+10+8:], ObjectRelocationAMD64Addr32)
		}, want: ".pdata end: relocation at"},
		{name: "ADDR32NB end duplicate site", edit: func(object, _, _ *[]byte) {
			binary.LittleEndian.PutUint32((*object)[testAddressPDataReloc+20:], 4)
		}, want: ".pdata end: object has 2 relocations"},
		{name: "ADDR32NB end target drift", edit: func(object, _, _ *[]byte) {
			binary.LittleEndian.PutUint32((*object)[testAddressPDataReloc+10+4:], 4)
		}, want: ".pdata end does not target the exact function alias"},
		{name: "ADDR32NB end addend drift", edit: func(object, _, _ *[]byte) {
			binary.LittleEndian.PutUint32((*object)[testAddressPDataRaw+4:], 2)
		}, want: "linked end RVA"},
		{name: "runtime function end drift", edit: func(_, image, _ *[]byte) {
			binary.LittleEndian.PutUint32((*image)[testUnwindPDataRawOffset+4:], testAddressSourceRVA+2)
		}, want: "linked end RVA"},
		{name: "ADDR32NB unwind type drift", edit: func(object, _, _ *[]byte) {
			binary.LittleEndian.PutUint16((*object)[testAddressPDataReloc+20+8:], ObjectRelocationAMD64Addr32)
		}, want: ".pdata unwind-info: relocation at"},
		{name: "ADDR32NB unwind duplicate site", edit: func(object, _, _ *[]byte) {
			pdata := coffFileHeaderBytes + 2*coffSectionHeaderBytes
			binary.LittleEndian.PutUint16((*object)[pdata+32:], 4)
			duplicate := testAddressPDataReloc + 30
			binary.LittleEndian.PutUint32((*object)[duplicate:], 8)
			binary.LittleEndian.PutUint32((*object)[duplicate+4:], 4)
			binary.LittleEndian.PutUint16((*object)[duplicate+8:], ObjectRelocationAMD64Addr32NB)
		}, want: ".pdata unwind-info: object has 2 relocations"},
		{name: "ADDR32NB unwind target section drift", edit: func(object, _, _ *[]byte) {
			binary.LittleEndian.PutUint16((*object)[testAddressSymbols+4*18+12:], 1)
		}, want: "not bounded read-only .xdata"},
		{name: "ADDR32NB unwind symbol value outside xdata", edit: func(object, _, _ *[]byte) {
			binary.LittleEndian.PutUint32((*object)[testAddressSymbols+4*18+8:], 4)
		}, want: "not bounded read-only .xdata"},
		{name: "ADDR32NB unwind xdata becomes writable", edit: func(object, _, _ *[]byte) {
			xdata := coffFileHeaderBytes + 3*coffSectionHeaderBytes
			binary.LittleEndian.PutUint32((*object)[xdata+36:], 0x40|sectionReadable|sectionWritable)
		}, want: "not bounded read-only .xdata"},
		{name: "ADDR32NB unwind xdata becomes uninitialized", edit: func(object, _, _ *[]byte) {
			xdata := coffFileHeaderBytes + 3*coffSectionHeaderBytes
			binary.LittleEndian.PutUint32((*object)[xdata+20:], 0)
			binary.LittleEndian.PutUint32((*object)[xdata+36:], sectionUninitializedData|sectionReadable)
		}, want: "not bounded read-only .xdata"},
		{name: "ADDR32NB unwind addend drift", edit: func(object, _, _ *[]byte) {
			binary.LittleEndian.PutUint32((*object)[testAddressPDataRaw+8:], 4)
		}, want: "linked unwind-info RVA"},
		{name: "unwind map address drift", edit: func(_, _, linkMap *[]byte) {
			*linkMap = bytes.Replace(*linkMap, []byte("0000000140003000"), []byte("0000000140003001"), 1)
		}, want: "linked unwind-info RVA"},
		{name: "unwind map owner drift", edit: func(_, _, linkMap *[]byte) {
			*linkMap = bytes.Replace(*linkMap, []byte("$unwind 0000000140003000 source.obj"), []byte("$unwind 0000000140003000 otherx.obj"), 1)
		}, want: "map unwind-info owner"},
		{name: "runtime unwind-info drift", edit: func(_, image, _ *[]byte) {
			binary.LittleEndian.PutUint32((*image)[testUnwindPDataRawOffset+8:], 0x3004)
			copy((*image)[testUnwindXDataRawOffset+4:testUnwindXDataRawOffset+8], (*image)[testUnwindXDataRawOffset:testUnwindXDataRawOffset+4])
		}, want: "linked unwind-info RVA"},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			object := testAddressObject()
			image := testAddressImage()
			linkMap := testAddressMap()
			testCase.edit(&object, &image, &linkMap)
			if _, err := InspectAMD64AddressLink(object, image, linkMap, testAddressSpec()); err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("InspectAMD64AddressLink() error = %v, want %q", err, testCase.want)
			}
		})
	}
}

func TestInspectAMD64AddressLinkRejectsInvalidSpecs(t *testing.T) {
	base := testAddressSpec()
	for _, edit := range []func(*AMD64AddressLinkSpec){
		func(spec *AMD64AddressLinkSpec) { spec.SourceSymbol = spec.TargetSymbol },
		func(spec *AMD64AddressLinkSpec) { spec.ReceiptSymbol = "bad symbol" },
		func(spec *AMD64AddressLinkSpec) { spec.SourceObject = `dir\source.obj` },
		func(spec *AMD64AddressLinkSpec) { spec.TargetObject = "target.o" },
		func(spec *AMD64AddressLinkSpec) { spec.ReceiptSection = ".too-long" },
		func(spec *AMD64AddressLinkSpec) { spec.ReceiptMagic = [16]byte{} },
	} {
		spec := base
		edit(&spec)
		if _, err := InspectAMD64AddressLink(testAddressObject(), testAddressImage(), testAddressMap(), spec); err == nil {
			t.Fatalf("InspectAMD64AddressLink() accepted spec %+v", spec)
		}
	}
}

func TestExactAddressRelocationRejectsDuplicateAndWrongSite(t *testing.T) {
	relocation := ObjectRelocation{SectionIndex: 2, Section: ".rdata", VirtualAddress: 24, SymbolIndex: 7, Type: ObjectRelocationAMD64Addr64}
	if _, err := exactAddressRelocation([]ObjectRelocation{relocation}, 2, ".rdata", 7, ObjectRelocationAMD64Addr64, 24); err != nil {
		t.Fatal(err)
	}
	if _, err := exactAddressRelocation([]ObjectRelocation{relocation, relocation}, 2, ".rdata", 7, ObjectRelocationAMD64Addr64, 24); err == nil || !strings.Contains(err.Error(), "repeats") {
		t.Fatalf("duplicate exactAddressRelocation() error = %v", err)
	}
	if _, err := exactAddressRelocation([]ObjectRelocation{relocation}, 2, ".rdata", 7, ObjectRelocationAMD64Addr64, 25); err == nil {
		t.Fatal("exactAddressRelocation() accepted a wrong site")
	}
	decoy := relocation
	decoy.SectionIndex = 1
	selected, err := exactAddressRelocation([]ObjectRelocation{decoy, relocation}, 2, ".rdata", 7, ObjectRelocationAMD64Addr64, 24)
	if err != nil || selected.SectionIndex != 2 {
		t.Fatalf("section-indexed exactAddressRelocation() = %+v, %v", selected, err)
	}
}

func TestExactPDataEntryRelocationsScopesDuplicateNamesBySectionIndex(t *testing.T) {
	file := &pe.File{Sections: []*pe.Section{
		{SectionHeader: pe.SectionHeader{Name: ".pdata", Size: runtimeFunctionBytes}},
		{SectionHeader: pe.SectionHeader{Name: ".pdata", Size: runtimeFunctionBytes}},
	}}
	source := ObjectSymbol{TableIndex: 1, Name: "source", SectionNumber: 3, Type: coffSymbolTypeFunction, StorageClass: coffStorageClassExternal}
	alias := ObjectSymbol{TableIndex: 2, Name: "$begin", SectionNumber: 3, StorageClass: coffStorageClassStatic}
	unwind := ObjectSymbol{TableIndex: 3, Name: "$unwind", SectionNumber: 4, StorageClass: coffStorageClassStatic}
	report := ObjectReport{
		Sections: []ObjectSection{{Name: ".pdata"}, {Name: ".pdata"}, {Name: ".text"}, {
			Name: ".xdata", RawOffset: 1, RawSize: 4, Characteristics: sectionInitializedData | sectionReadable,
		}},
		SymbolRecords: []ObjectSymbol{source, alias, unwind},
		RelocationRecords: []ObjectRelocation{{
			SectionIndex: 1, Section: ".pdata", SymbolIndex: alias.TableIndex,
			Symbol: alias.Name, Type: ObjectRelocationAMD64Addr32NB,
		}, {
			SectionIndex: 1, Section: ".pdata", VirtualAddress: 4, SymbolIndex: alias.TableIndex,
			Symbol: alias.Name, Type: ObjectRelocationAMD64Addr32NB,
		}, {
			SectionIndex: 1, Section: ".pdata", VirtualAddress: 8, SymbolIndex: unwind.TableIndex,
			Symbol: unwind.Name, Type: ObjectRelocationAMD64Addr32NB,
		}, {
			SectionIndex: 2, Section: ".pdata", SymbolIndex: unwind.TableIndex,
			Symbol: unwind.Name, Type: ObjectRelocationAMD64Addr32NB,
		}, {
			SectionIndex: 2, Section: ".pdata", VirtualAddress: 4, SymbolIndex: unwind.TableIndex,
			Symbol: unwind.Name, Type: ObjectRelocationAMD64Addr32NB,
		}, {
			SectionIndex: 2, Section: ".pdata", VirtualAddress: 8, SymbolIndex: unwind.TableIndex,
			Symbol: unwind.Name, Type: ObjectRelocationAMD64Addr32NB,
		}},
	}
	entry, err := exactPDataEntryRelocations(report, file, source)
	if err != nil {
		t.Fatal(err)
	}
	if entry.Begin.SectionIndex != 1 || entry.End.SectionIndex != 1 || entry.Unwind.SectionIndex != 1 ||
		entry.Section != file.Sections[0] || entry.BeginTarget.Name != alias.Name || entry.UnwindTarget.Name != unwind.Name {
		t.Fatalf("section-indexed .pdata entry = %+v", entry)
	}
}

func TestExactPDataEntryRelocationsRejectsTargetIdentityDrift(t *testing.T) {
	file := &pe.File{Sections: []*pe.Section{{SectionHeader: pe.SectionHeader{Name: ".pdata", Size: runtimeFunctionBytes}}}}
	source := ObjectSymbol{TableIndex: 1, Name: "source", SectionNumber: 2, Type: coffSymbolTypeFunction, StorageClass: coffStorageClassExternal}
	beginAlias := ObjectSymbol{TableIndex: 2, Name: "$begin", SectionNumber: 2, StorageClass: coffStorageClassStatic}
	baseEndAlias := ObjectSymbol{TableIndex: 3, Name: "$end", SectionNumber: 2, StorageClass: coffStorageClassStatic}
	baseUnwind := ObjectSymbol{TableIndex: 4, Name: "$unwind", SectionNumber: 3, StorageClass: coffStorageClassStatic}
	for _, testCase := range []struct {
		name string
		edit func(*ObjectSymbol, *ObjectSymbol)
	}{
		{name: "end value", edit: func(alias, _ *ObjectSymbol) { alias.Value = 1 }},
		{name: "end auxiliary", edit: func(alias, _ *ObjectSymbol) { alias.AuxiliaryRecords = 1 }},
		{name: "unwind section", edit: func(_, unwind *ObjectSymbol) { unwind.SectionNumber = 2 }},
		{name: "unwind auxiliary", edit: func(_, unwind *ObjectSymbol) { unwind.AuxiliaryRecords = 1 }},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			endAlias, unwind := baseEndAlias, baseUnwind
			testCase.edit(&endAlias, &unwind)
			report := ObjectReport{
				Sections:      []ObjectSection{{Name: ".pdata"}, {Name: ".text"}, {Name: ".xdata", RawOffset: 1, RawSize: 4, Characteristics: sectionInitializedData | sectionReadable}},
				SymbolRecords: []ObjectSymbol{source, beginAlias, endAlias, unwind},
				RelocationRecords: []ObjectRelocation{
					{SectionIndex: 1, Section: ".pdata", SymbolIndex: beginAlias.TableIndex, Symbol: beginAlias.Name, Type: ObjectRelocationAMD64Addr32NB},
					{SectionIndex: 1, Section: ".pdata", VirtualAddress: 4, SymbolIndex: endAlias.TableIndex, Symbol: endAlias.Name, Type: ObjectRelocationAMD64Addr32NB},
					{SectionIndex: 1, Section: ".pdata", VirtualAddress: 8, SymbolIndex: unwind.TableIndex, Symbol: unwind.Name, Type: ObjectRelocationAMD64Addr32NB},
				},
			}
			if _, err := exactPDataEntryRelocations(report, file, source); err == nil {
				t.Fatal("exactPDataEntryRelocations() accepted target identity drift")
			}
		})
	}
}

func TestExactPDataSiteRelocationRejectsAddressOverflowBeforeNarrowing(t *testing.T) {
	relocation := ObjectRelocation{SectionIndex: 1, Section: ".pdata", VirtualAddress: 0}
	if _, err := exactPDataSiteRelocation([]ObjectRelocation{relocation}, 1, ".pdata", uint64(math.MaxUint32)+1); err == nil || !strings.Contains(err.Error(), "overflows") {
		t.Fatalf("exactPDataSiteRelocation() overflow error = %v", err)
	}
}

func TestMSVCMapOwnerIsObjectRejectsSuffixSpoof(t *testing.T) {
	for _, owner := range []string{"source.obj", `C:\work\source.obj`, "C:/work/source.obj"} {
		if !msvcMapOwnerIsObject(owner, "source.obj") {
			t.Fatalf("msvcMapOwnerIsObject(%q) = false", owner)
		}
	}
	for _, owner := range []string{"notsource.obj", "source.obj.extra", "archive:source.obj"} {
		if msvcMapOwnerIsObject(owner, "source.obj") {
			t.Fatalf("msvcMapOwnerIsObject(%q) = true", owner)
		}
	}
}

func testAddressSpec() AMD64AddressLinkSpec {
	return AMD64AddressLinkSpec{
		SourceSymbol: "source", TargetSymbol: "target", ReceiptSymbol: "receipt",
		SourceObject: "source.obj", TargetObject: "target.obj",
		ReceiptSection: ".fyaddr", ReceiptMagic: testAddressMagic,
	}
}

func testAddressMap() []byte {
	return []byte("  Address         Publics by Value              Rva+Base               Lib:Object\r\n" +
		" 0001:00000000 source 0000000140001000 f source.obj\r\n" +
		" 0001:00000020 target 0000000140001020 f target.obj\r\n" +
		" Static symbols\r\n" +
		" 0003:00000000 $unwind 0000000140003000 source.obj\r\n")
}

func testAddressObject() []byte {
	object := make([]byte, testAddressSymbols+5*18+4)
	binary.LittleEndian.PutUint16(object, machineAMD64)
	binary.LittleEndian.PutUint16(object[2:], 4)
	binary.LittleEndian.PutUint32(object[8:], testAddressSymbols)
	binary.LittleEndian.PutUint32(object[12:], 5)

	text := coffFileHeaderBytes
	copy(object[text:], ".text")
	binary.LittleEndian.PutUint32(object[text+16:], 1)
	binary.LittleEndian.PutUint32(object[text+20:], testAddressTextRaw)
	binary.LittleEndian.PutUint32(object[text+36:], sectionContainsCode|sectionExecutable|sectionReadable)
	object[testAddressTextRaw] = 0xc3

	receipt := text + coffSectionHeaderBytes
	copy(object[receipt:], ".fyaddr")
	binary.LittleEndian.PutUint32(object[receipt+16:], 32)
	binary.LittleEndian.PutUint32(object[receipt+20:], testAddressReceiptRaw)
	binary.LittleEndian.PutUint32(object[receipt+24:], testAddressReceiptReloc)
	binary.LittleEndian.PutUint16(object[receipt+32:], 1)
	binary.LittleEndian.PutUint32(object[receipt+36:], 0x40|sectionReadable)
	copy(object[testAddressReceiptRaw:], testAddressMagic[:])
	binary.LittleEndian.PutUint32(object[testAddressReceiptReloc:], amd64AddressReceiptTargetOffset)
	binary.LittleEndian.PutUint32(object[testAddressReceiptReloc+4:], 2)
	binary.LittleEndian.PutUint16(object[testAddressReceiptReloc+8:], ObjectRelocationAMD64Addr64)

	pdata := receipt + coffSectionHeaderBytes
	copy(object[pdata:], ".pdata")
	binary.LittleEndian.PutUint32(object[pdata+16:], runtimeFunctionBytes)
	binary.LittleEndian.PutUint32(object[pdata+20:], testAddressPDataRaw)
	binary.LittleEndian.PutUint32(object[pdata+24:], testAddressPDataReloc)
	binary.LittleEndian.PutUint16(object[pdata+32:], 3)
	binary.LittleEndian.PutUint32(object[pdata+36:], 0x40|sectionReadable)
	binary.LittleEndian.PutUint32(object[testAddressPDataRaw+4:], 1)
	binary.LittleEndian.PutUint32(object[testAddressPDataReloc+4:], 3)
	binary.LittleEndian.PutUint16(object[testAddressPDataReloc+8:], ObjectRelocationAMD64Addr32NB)
	binary.LittleEndian.PutUint32(object[testAddressPDataReloc+10:], 4)
	binary.LittleEndian.PutUint32(object[testAddressPDataReloc+10+4:], 3)
	binary.LittleEndian.PutUint16(object[testAddressPDataReloc+10+8:], ObjectRelocationAMD64Addr32NB)
	binary.LittleEndian.PutUint32(object[testAddressPDataReloc+20:], 8)
	binary.LittleEndian.PutUint32(object[testAddressPDataReloc+20+4:], 4)
	binary.LittleEndian.PutUint16(object[testAddressPDataReloc+20+8:], ObjectRelocationAMD64Addr32NB)

	xdata := pdata + coffSectionHeaderBytes
	copy(object[xdata:], ".xdata")
	binary.LittleEndian.PutUint32(object[xdata+16:], 4)
	binary.LittleEndian.PutUint32(object[xdata+20:], testAddressXDataRaw)
	binary.LittleEndian.PutUint32(object[xdata+36:], 0x40|sectionReadable)
	copy(object[testAddressXDataRaw:], []byte{1, 0, 0, 0})

	symbol := testAddressSymbols
	copy(object[symbol:], "receipt")
	binary.LittleEndian.PutUint16(object[symbol+12:], 2)
	object[symbol+16] = coffStorageClassExternal
	symbol += 18
	copy(object[symbol:], "source")
	binary.LittleEndian.PutUint16(object[symbol+12:], 1)
	binary.LittleEndian.PutUint16(object[symbol+14:], coffSymbolTypeFunction)
	object[symbol+16] = coffStorageClassExternal
	symbol += 18
	copy(object[symbol:], "target")
	binary.LittleEndian.PutUint16(object[symbol+14:], coffSymbolTypeFunction)
	object[symbol+16] = coffStorageClassExternal
	symbol += 18
	copy(object[symbol:], "$begin")
	binary.LittleEndian.PutUint16(object[symbol+12:], 1)
	object[symbol+16] = coffStorageClassStatic
	symbol += 18
	copy(object[symbol:], "$unwind")
	binary.LittleEndian.PutUint16(object[symbol+12:], 4)
	object[symbol+16] = coffStorageClassStatic
	binary.LittleEndian.PutUint32(object[symbol+18:], 4)
	return object
}

func testAddressImage() []byte {
	base := minimalAMD64PEWithUnwind()
	image := make([]byte, 0xc00)
	copy(image, base)
	coff := testUnwindPEOffset + 4
	binary.LittleEndian.PutUint16(image[coff+2:], 4)
	binary.LittleEndian.PutUint32(image[testUnwindOptionalOffset+56:], 0x5000)
	emitPESection(image, testUnwindSectionsOffset, ".text", 0x40, 0x1000, 0x200, testUnwindTextRawOffset, sectionContainsCode|sectionExecutable|sectionReadable)
	emitPESection(image, testUnwindSectionsOffset+3*coffSectionHeaderBytes, ".fyaddr", 32, testAddressReceiptRVA, 0x200, testAddressImageReceipt, 0x40|sectionReadable)
	image[testUnwindTextRawOffset+0x20] = 0xc3
	copy(image[testAddressImageReceipt:], testAddressMagic[:])
	binary.LittleEndian.PutUint64(image[testAddressImageReceipt+16:], 0x140001000)
	binary.LittleEndian.PutUint64(image[testAddressImageReceipt+24:], 0x140001020)
	return image
}
