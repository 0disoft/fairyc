package pecoff

import (
	"bytes"
	"debug/pe"
	"encoding/binary"
	"strings"
	"testing"
)

const (
	testAddressTextRaw      = 140
	testAddressReceiptRaw   = 144
	testAddressReceiptReloc = 176
	testAddressPDataRaw     = 188
	testAddressPDataReloc   = 200
	testAddressSymbols      = 210
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
		report.UnwindInfoRVA != 0x3000 {
		t.Fatalf("InspectAMD64AddressLink() report = %+v", report)
	}
}

func TestInspectAMD64AddressLinkAcceptsIndependentNonzeroAddends(t *testing.T) {
	object := testAddressObject()
	image := testAddressImage()
	binary.LittleEndian.PutUint64(object[testAddressReceiptRaw+amd64AddressReceiptTargetOffset:], 4)
	binary.LittleEndian.PutUint32(object[testAddressPDataRaw:], 4)
	binary.LittleEndian.PutUint64(image[testAddressImageReceipt+amd64AddressReceiptTargetOffset:], 0x140001024)
	binary.LittleEndian.PutUint32(image[testUnwindPDataRawOffset:], testAddressSourceRVA+4)
	binary.LittleEndian.PutUint32(image[testUnwindPDataRawOffset+4:], testAddressSourceRVA+5)
	report, err := InspectAMD64AddressLink(object, image, testAddressMap(), testAddressSpec())
	if err != nil {
		t.Fatal(err)
	}
	if report.ADDR64ObjectAddend != 4 || report.ADDR64LinkedValue != 0x140001024 || report.ADDR64ExpectedValue != 0x140001024 ||
		report.ADDR32NBObjectAddend != 4 || report.ADDR32NBLinkedBeginRVA != testAddressSourceRVA+4 ||
		report.ADDR32NBExpectedBeginRVA != testAddressSourceRVA+4 {
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

func TestExactPDataBeginRelocationScopesDuplicateNamesBySectionIndex(t *testing.T) {
	file := &pe.File{Sections: []*pe.Section{
		{SectionHeader: pe.SectionHeader{Name: ".pdata", Size: runtimeFunctionBytes}},
		{SectionHeader: pe.SectionHeader{Name: ".pdata", Size: runtimeFunctionBytes}},
	}}
	source := ObjectSymbol{TableIndex: 1, Name: "source", SectionNumber: 3, Type: coffSymbolTypeFunction, StorageClass: coffStorageClassExternal}
	alias := ObjectSymbol{TableIndex: 2, Name: "$begin", SectionNumber: 3, StorageClass: coffStorageClassStatic}
	report := ObjectReport{
		Sections:      []ObjectSection{{Name: ".pdata"}, {Name: ".pdata"}},
		SymbolRecords: []ObjectSymbol{source, alias},
		RelocationRecords: []ObjectRelocation{{
			SectionIndex: 1, Section: ".pdata", SymbolIndex: alias.TableIndex,
			Symbol: alias.Name, Type: ObjectRelocationAMD64Addr32NB,
		}},
	}
	relocation, section, target, err := exactPDataBeginRelocation(report, file, source)
	if err != nil {
		t.Fatal(err)
	}
	if relocation.SectionIndex != 1 || section != file.Sections[0] || target.Name != alias.Name {
		t.Fatalf("section-indexed .pdata relocation = %+v, %p, %+v", relocation, section, target)
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
		" 0001:00000020 target 0000000140001020 f target.obj\r\n")
}

func testAddressObject() []byte {
	object := make([]byte, testAddressSymbols+4*18+4)
	binary.LittleEndian.PutUint16(object, machineAMD64)
	binary.LittleEndian.PutUint16(object[2:], 3)
	binary.LittleEndian.PutUint32(object[8:], testAddressSymbols)
	binary.LittleEndian.PutUint32(object[12:], 4)

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
	binary.LittleEndian.PutUint16(object[pdata+32:], 1)
	binary.LittleEndian.PutUint32(object[pdata+36:], 0x40|sectionReadable)
	binary.LittleEndian.PutUint32(object[testAddressPDataReloc+4:], 3)
	binary.LittleEndian.PutUint16(object[testAddressPDataReloc+8:], ObjectRelocationAMD64Addr32NB)

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
