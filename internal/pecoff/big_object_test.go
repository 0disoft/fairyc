package pecoff

import (
	"encoding/binary"
	"strings"
	"testing"
)

func TestBuildAMD64BigObjectProbeEmitsExactVersion2Object(t *testing.T) {
	t.Parallel()
	object := testBigObject()
	if len(object) != 161 {
		t.Fatalf("bigobj bytes = %d, want 161", len(object))
	}
	report, err := InspectObject(object)
	if err != nil {
		t.Fatal(err)
	}
	if !report.BigObject || report.BigObjectVersion != 2 || report.Machine != machineAMD64 || report.Characteristics != 0 ||
		len(report.Sections) != 1 || report.Sections[0].Name != ".text" || report.Sections[0].Relocations != 1 {
		t.Fatalf("bigobj report = %+v", report)
	}
	if len(report.SymbolRecords) != 2 || report.SymbolRecords[0] != (ObjectSymbol{
		TableIndex: 0, Name: "fy_bigob", SectionNumber: 1, Type: coffSymbolTypeFunction, StorageClass: coffStorageClassExternal,
	}) || report.SymbolRecords[1] != (ObjectSymbol{
		TableIndex: 1, Name: "fy_bgdat", Value: 7, SectionNumber: 1, StorageClass: coffStorageClassStatic,
	}) {
		t.Fatalf("bigobj symbols = %+v", report.SymbolRecords)
	}
	if len(report.RelocationRecords) != 1 || report.RelocationRecords[0] != (ObjectRelocation{
		SectionIndex: 1, Section: ".text", VirtualAddress: 2, SymbolIndex: 1,
		Symbol: "fy_bgdat", Type: ObjectRelocationAMD64Rel32,
	}) {
		t.Fatalf("bigobj relocations = %+v", report.RelocationRecords)
	}
}

func TestInspectObjectRejectsMalformedBigObject(t *testing.T) {
	t.Parallel()
	for _, testCase := range []struct {
		name string
		edit func([]byte)
		want string
	}{
		{name: "version", edit: func(object []byte) { binary.LittleEndian.PutUint16(object[4:], 3) }, want: "version"},
		{name: "machine", edit: func(object []byte) { binary.LittleEndian.PutUint16(object[6:], 0x14c) }, want: "want AMD64"},
		{name: "class ID", edit: func(object []byte) { object[12]++ }, want: "class ID"},
		{name: "reserved metadata", edit: func(object []byte) { object[28] = 1 }, want: "reserved metadata"},
		{name: "zero sections", edit: func(object []byte) { binary.LittleEndian.PutUint32(object[44:], 0) }, want: "section count"},
		{name: "section table range", edit: func(object []byte) { binary.LittleEndian.PutUint32(object[44:], 3) }, want: "section table exceeds"},
		{name: "sections over budget", edit: func(object []byte) { binary.LittleEndian.PutUint32(object[44:], maxObjectSections+1) }, want: "section count"},
		{name: "zero symbols", edit: func(object []byte) { binary.LittleEndian.PutUint32(object[52:], 0) }, want: "symbol count"},
		{name: "symbols over budget", edit: func(object []byte) { binary.LittleEndian.PutUint32(object[52:], maxObjectSymbols+1) }, want: "symbol count"},
		{name: "symbol range", edit: func(object []byte) { binary.LittleEndian.PutUint32(object[48:], uint32(len(object))) }, want: "symbol-table range"},
		{name: "string table", edit: func(object []byte) { binary.LittleEndian.PutUint32(object[len(object)-4:], 3) }, want: "string table"},
		{name: "symbol auxiliary", edit: func(object []byte) { object[amd64BigObjectSymbolOffset+19] = 1 }, want: "unsupported auxiliary"},
		{name: "symbol name padding", edit: func(object []byte) { copy(object[amd64BigObjectSymbolOffset:], "fy\x00junk\x00") }, want: "nonzero padding"},
		{name: "symbol section", edit: func(object []byte) { binary.LittleEndian.PutUint32(object[amd64BigObjectSymbolOffset+12:], 2) }, want: "invalid section"},
		{name: "negative symbol section", edit: func(object []byte) { binary.LittleEndian.PutUint32(object[amd64BigObjectSymbolOffset+12:], 0xfffffffd) }, want: "invalid section"},
		{name: "long symbol offset", edit: func(object []byte) {
			clear(object[amd64BigObjectSymbolOffset : amd64BigObjectSymbolOffset+8])
			binary.LittleEndian.PutUint32(object[amd64BigObjectSymbolOffset+4:], 4)
		}, want: "string-table offset"},
		{name: "COMDAT", edit: func(object []byte) {
			binary.LittleEndian.PutUint32(object[coffBigObjectHeaderBytes+36:], sectionContainsCode|sectionExecutable|sectionReadable|sectionLinkCOMDAT)
		}, want: "unsupported COMDAT"},
		{name: "long section name", edit: func(object []byte) { copy(object[coffBigObjectHeaderBytes:], "/4\x00\x00\x00\x00\x00\x00") }, want: "unsupported long name"},
		{name: "symbol-style section name", edit: func(object []byte) {
			clear(object[coffBigObjectHeaderBytes : coffBigObjectHeaderBytes+8])
			binary.LittleEndian.PutUint32(object[coffBigObjectHeaderBytes+4:], 4)
		}, want: "unsupported long name"},
		{name: "section name padding", edit: func(object []byte) { copy(object[coffBigObjectHeaderBytes:], ".text\x00x\x00") }, want: "nonzero padding"},
		{name: "raw range", edit: func(object []byte) {
			binary.LittleEndian.PutUint32(object[coffBigObjectHeaderBytes+20:], uint32(len(object)))
		}, want: "invalid raw range"},
		{name: "relocation range", edit: func(object []byte) {
			binary.LittleEndian.PutUint32(object[coffBigObjectHeaderBytes+24:], uint32(len(object)))
		}, want: "invalid relocation range"},
		{name: "raw overlaps symbols", edit: func(object []byte) {
			binary.LittleEndian.PutUint32(object[coffBigObjectHeaderBytes+20:], amd64BigObjectSymbolOffset)
		}, want: "overlaps symbol and string tables"},
		{name: "relocations overlap symbols", edit: func(object []byte) {
			binary.LittleEndian.PutUint32(object[coffBigObjectHeaderBytes+24:], amd64BigObjectSymbolOffset)
		}, want: "overlaps symbol and string tables"},
		{name: "relocations overlap raw", edit: func(object []byte) {
			binary.LittleEndian.PutUint32(object[coffBigObjectHeaderBytes+24:], amd64BigObjectRawOffset)
		}, want: "overlaps section .text raw data"},
		{name: "missing relocation symbol", edit: func(object []byte) { binary.LittleEndian.PutUint32(object[amd64BigObjectRelocOffset+4:], 2) }, want: "missing symbol"},
		{name: "unknown relocation", edit: func(object []byte) { binary.LittleEndian.PutUint16(object[amd64BigObjectRelocOffset+8:], 0xffff) }, want: "unknown AMD64"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			object := testBigObject()
			testCase.edit(object)
			if _, err := InspectObject(object); err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("InspectObject() error = %v, want %q", err, testCase.want)
			}
		})
	}
}

func TestInspectObjectHandlesBoundedBigObjectLongSymbolName(t *testing.T) {
	t.Parallel()
	const longName = "fairyc_bigobj_long_symbol"
	object := testBigObject()
	stringOffset := len(object) - 4
	object = append(object, []byte(longName+"\x00")...)
	binary.LittleEndian.PutUint32(object[stringOffset:], uint32(4+len(longName)+1))
	clear(object[amd64BigObjectSymbolOffset : amd64BigObjectSymbolOffset+8])
	binary.LittleEndian.PutUint32(object[amd64BigObjectSymbolOffset+4:], 4)
	report, err := InspectObject(object)
	if err != nil {
		t.Fatal(err)
	}
	if report.SymbolRecords[0].Name != longName {
		t.Fatalf("bigobj long symbol = %q", report.SymbolRecords[0].Name)
	}

	unterminated := testBigObject()
	stringOffset = len(unterminated) - 4
	unterminated = append(unterminated, []byte(longName)...)
	binary.LittleEndian.PutUint32(unterminated[stringOffset:], uint32(4+len(longName)))
	clear(unterminated[amd64BigObjectSymbolOffset : amd64BigObjectSymbolOffset+8])
	binary.LittleEndian.PutUint32(unterminated[amd64BigObjectSymbolOffset+4:], 4)
	if _, err := InspectObject(unterminated); err == nil || !strings.Contains(err.Error(), "unterminated") {
		t.Fatalf("InspectObject() unterminated long-name error = %v", err)
	}
}

func TestInspectAMD64BigObjectLinkBindsMapRelocationAndImage(t *testing.T) {
	t.Parallel()
	report, err := InspectAMD64BigObjectLink(testBigObject(), testBigObjectImage(), testBigObjectMap(), testBigObjectSpec())
	if err != nil {
		t.Fatal(err)
	}
	if report.Version != 2 || report.SourceRVA != 0x11c0 || report.LinkedDisplacement != 1 || report.ExpectedDisplacement != 1 {
		t.Fatalf("bigobj link report = %+v", report)
	}
}

func TestInspectAMD64BigObjectLinkRejectsIndependentMutations(t *testing.T) {
	t.Parallel()
	for _, testCase := range []struct {
		name string
		edit func(*[]byte, *[]byte, *[]byte)
		want string
	}{
		{name: "object class", edit: func(object, _, _ *[]byte) { (*object)[12]++ }, want: "class ID"},
		{name: "object opcode", edit: func(object, _, _ *[]byte) { (*object)[amd64BigObjectRawOffset]++ }, want: "source encoding"},
		{name: "object addend", edit: func(object, _, _ *[]byte) {
			binary.LittleEndian.PutUint32((*object)[amd64BigObjectRawOffset+2:], 1)
		}, want: "linked displacement"},
		{name: "data value", edit: func(object, _, _ *[]byte) {
			binary.LittleEndian.PutUint32((*object)[amd64BigObjectSymbolOffset+coffBigObjectSymbolBytes+8:], 8)
		}, want: "data symbol"},
		{name: "source type", edit: func(object, _, _ *[]byte) {
			binary.LittleEndian.PutUint16((*object)[amd64BigObjectSymbolOffset+16:], 0)
		}, want: "source symbol"},
		{name: "source class", edit: func(object, _, _ *[]byte) {
			(*object)[amd64BigObjectSymbolOffset+18] = coffStorageClassStatic
		}, want: "source symbol"},
		{name: "relocation site", edit: func(object, _, _ *[]byte) {
			binary.LittleEndian.PutUint32((*object)[amd64BigObjectRelocOffset:], 3)
		}, want: "relocation is invalid"},
		{name: "map owner", edit: func(_, _, linkMap *[]byte) {
			*linkMap = []byte(strings.Replace(string(*linkMap), "bigobj.obj", "xxxxxx.obj", 1))
		}, want: "source owner"},
		{name: "source opcode", edit: func(_, image, _ *[]byte) { (*image)[0x3c0]++ }, want: "bytes drifted"},
		{name: "source data", edit: func(_, image, _ *[]byte) { (*image)[0x3c0+7]++ }, want: "bytes drifted"},
		{name: "source section not executable", edit: func(_, image, _ *[]byte) {
			binary.LittleEndian.PutUint32((*image)[testSectionHeaderOffset+36:], sectionContainsCode|sectionReadable)
		}, want: "not readable executable code"},
		{name: "displacement", edit: func(_, image, _ *[]byte) {
			binary.LittleEndian.PutUint32((*image)[0x3c0+2:], 2)
		}, want: "linked displacement"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			object, image, linkMap := testBigObject(), testBigObjectImage(), testBigObjectMap()
			testCase.edit(&object, &image, &linkMap)
			if _, err := InspectAMD64BigObjectLink(object, image, linkMap, testBigObjectSpec()); err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("InspectAMD64BigObjectLink() error = %v, want %q", err, testCase.want)
			}
		})
	}
}

func testBigObject() []byte {
	object, err := BuildAMD64BigObjectProbe("fy_bigob")
	if err != nil {
		panic(err)
	}
	return object
}

func testBigObjectImage() []byte {
	image := testREL32Image()
	object := testBigObject()
	copy(image[0x3c0:0x3cb], object[amd64BigObjectRawOffset:amd64BigObjectRawOffset+amd64BigObjectRawSize])
	binary.LittleEndian.PutUint32(image[0x3c0+2:], 1)
	return image
}

func testBigObjectMap() []byte {
	return []byte(" 0001:000001c0 fy_bigob 00000001400011c0 f bigobj.obj\r\n")
}

func testBigObjectSpec() AMD64BigObjectSpec {
	return AMD64BigObjectSpec{SourceSymbol: "fy_bigob", SourceObject: "bigobj.obj"}
}
