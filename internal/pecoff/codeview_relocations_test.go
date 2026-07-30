package pecoff

import (
	"encoding/binary"
	"strings"
	"testing"
)

const (
	testCodeViewSectionTable = coffFileHeaderBytes
	testCodeViewTextRaw      = coffFileHeaderBytes + 2*coffSectionHeaderBytes
	testCodeViewDebugRaw     = testCodeViewTextRaw + 8
	testCodeViewRelocations  = testCodeViewDebugRaw + 6
	testCodeViewSymbols      = testCodeViewRelocations + 2*coffRelocationBytes
)

func TestInspectCodeViewRelocationsComputesClosedSectionOffsetPair(t *testing.T) {
	report, err := InspectCodeViewRelocations(testCodeViewObject())
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Pairs) != 1 || report.Pairs[0] != (CodeViewSectionOffsetPair{
		Symbol: "dbgfunc", SymbolSection: 1, SymbolValue: 2,
		SecRelSite: 0, SectionSite: 4, SecRelAddend: 3, SectionAddend: 0,
		ResolvedOffset: 5, ResolvedSection: 1,
	}) {
		t.Fatalf("CodeView relocation report = %+v", report)
	}
}

func TestInspectCodeViewRelocationsRejectsPairMutations(t *testing.T) {
	tests := []struct {
		name string
		edit func([]byte)
		want string
	}{
		{name: "section type missing", edit: func(object []byte) {
			binary.LittleEndian.PutUint16(object[testCodeViewRelocations+coffRelocationBytes+8:], ObjectRelocationAMD64SecRel)
		}, want: "2 SECREL and 0 SECTION"},
		{name: "section is not adjacent", edit: func(object []byte) {
			binary.LittleEndian.PutUint32(object[testCodeViewRelocations+coffRelocationBytes:], 5)
		}, want: "0 adjacent SECTION"},
		{name: "target is undefined", edit: func(object []byte) {
			binary.LittleEndian.PutUint16(object[testCodeViewSymbols+12:], 0)
		}, want: "not defined"},
		{name: "adjacent sites exceed debug bytes", edit: func(object []byte) {
			const debugSection = testCodeViewSectionTable + coffSectionHeaderBytes
			binary.LittleEndian.PutUint32(object[debugSection+16:], 5)
		}, want: "relocation site exceeds section bytes"},
		{name: "adjacent site overflows", edit: func(object []byte) {
			binary.LittleEndian.PutUint32(object[testCodeViewRelocations:], ^uint32(0)-2)
		}, want: "overflows its adjacent"},
		{name: "resolved offset overflows", edit: func(object []byte) {
			binary.LittleEndian.PutUint32(object[testCodeViewSymbols+8:], ^uint32(0))
		}, want: "overflows"},
		{name: "resolved section escapes object", edit: func(object []byte) {
			binary.LittleEndian.PutUint16(object[testCodeViewDebugRaw+4:], 2)
		}, want: "section 3 outside the object"},
		{name: "resolved offset escapes section", edit: func(object []byte) {
			binary.LittleEndian.PutUint32(object[testCodeViewDebugRaw:], 7)
		}, want: "outside section"},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			object := testCodeViewObject()
			testCase.edit(object)
			if _, err := InspectCodeViewRelocations(object); err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("InspectCodeViewRelocations() error = %v, want %q", err, testCase.want)
			}
		})
	}
}

func TestInspectCodeViewRelocationsRejectsDuplicateReusedAndUnpairedSections(t *testing.T) {
	tests := []struct {
		name      string
		typeValue uint16
		site      uint32
		want      string
	}{
		{name: "duplicate SECTION", typeValue: ObjectRelocationAMD64Section, site: 4, want: "2 adjacent SECTION"},
		{name: "reused SECTION", typeValue: ObjectRelocationAMD64SecRel, site: 0, want: "is reused"},
		{name: "unpaired SECTION", typeValue: ObjectRelocationAMD64Section, site: 5, want: "1 unpaired SECTION"},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			object := testCodeViewObjectWithExtraRelocation(testCase.site, testCase.typeValue)
			if _, err := InspectCodeViewRelocations(object); err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("InspectCodeViewRelocations() error = %v, want %q", err, testCase.want)
			}
		})
	}
}

func testCodeViewObject() []byte {
	object := make([]byte, testCodeViewSymbols+18+4)
	binary.LittleEndian.PutUint16(object, machineAMD64)
	binary.LittleEndian.PutUint16(object[2:], 2)
	binary.LittleEndian.PutUint32(object[8:], testCodeViewSymbols)
	binary.LittleEndian.PutUint32(object[12:], 1)

	text := testCodeViewSectionTable
	copy(object[text:], ".text")
	binary.LittleEndian.PutUint32(object[text+16:], 8)
	binary.LittleEndian.PutUint32(object[text+20:], testCodeViewTextRaw)
	binary.LittleEndian.PutUint32(object[text+36:], sectionContainsCode|sectionExecutable|sectionReadable)
	object[testCodeViewTextRaw+2] = 0xc3

	debug := text + coffSectionHeaderBytes
	copy(object[debug:], ".debug$S")
	binary.LittleEndian.PutUint32(object[debug+16:], 6)
	binary.LittleEndian.PutUint32(object[debug+20:], testCodeViewDebugRaw)
	binary.LittleEndian.PutUint32(object[debug+24:], testCodeViewRelocations)
	binary.LittleEndian.PutUint16(object[debug+32:], 2)
	binary.LittleEndian.PutUint32(object[debug+36:], sectionInitializedData|sectionReadable)
	binary.LittleEndian.PutUint32(object[testCodeViewDebugRaw:], 3)
	binary.LittleEndian.PutUint16(object[testCodeViewDebugRaw+4:], 0)

	binary.LittleEndian.PutUint32(object[testCodeViewRelocations:], 0)
	binary.LittleEndian.PutUint16(object[testCodeViewRelocations+8:], ObjectRelocationAMD64SecRel)
	second := testCodeViewRelocations + coffRelocationBytes
	binary.LittleEndian.PutUint32(object[second:], 4)
	binary.LittleEndian.PutUint16(object[second+8:], ObjectRelocationAMD64Section)

	copy(object[testCodeViewSymbols:], "dbgfunc")
	binary.LittleEndian.PutUint32(object[testCodeViewSymbols+8:], 2)
	binary.LittleEndian.PutUint16(object[testCodeViewSymbols+12:], 1)
	binary.LittleEndian.PutUint16(object[testCodeViewSymbols+14:], coffSymbolTypeFunction)
	object[testCodeViewSymbols+16] = coffStorageClassExternal
	binary.LittleEndian.PutUint32(object[testCodeViewSymbols+18:], 4)
	return object
}

func testCodeViewObjectWithExtraRelocation(site uint32, relocationType uint16) []byte {
	base := testCodeViewObject()
	object := make([]byte, len(base)+coffRelocationBytes)
	copy(object[:testCodeViewSymbols], base[:testCodeViewSymbols])
	copy(object[testCodeViewSymbols+coffRelocationBytes:], base[testCodeViewSymbols:])
	binary.LittleEndian.PutUint32(object[8:], testCodeViewSymbols+coffRelocationBytes)
	const debugSection = testCodeViewSectionTable + coffSectionHeaderBytes
	binary.LittleEndian.PutUint16(object[debugSection+32:], 3)
	extra := testCodeViewSymbols
	binary.LittleEndian.PutUint32(object[extra:], site)
	binary.LittleEndian.PutUint16(object[extra+8:], relocationType)
	return object
}
