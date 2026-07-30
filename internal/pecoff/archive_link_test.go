package pecoff

import (
	"bytes"
	"encoding/binary"
	"strings"
	"testing"
)

func TestClassifyMSVCMapOwnerAcceptsObservedArchiveAndDirectSpellings(t *testing.T) {
	spec := ArchiveCOMDATLinkSpec{
		ArchiveName: "choice.lib", SelectedMember: "chosen.obj", DirectObject: "direct.obj",
	}
	for _, testCase := range []struct {
		owner string
		want  string
	}{
		{owner: "choice:chosen.obj", want: "archive"},
		{owner: "choice.lib:chosen.obj", want: "archive"},
		{owner: `C:\work\choice.lib(chosen.obj)`, want: "archive"},
		{owner: `C:\work\direct.obj`, want: "direct"},
		{owner: "notchoice:chosen.obj", want: "notchoice:chosen.obj"},
		{owner: "other.obj", want: "other.obj"},
	} {
		if got := classifyMSVCMapOwner(testCase.owner, spec); got != testCase.want {
			t.Fatalf("classifyMSVCMapOwner(%q) = %q, want %q", testCase.owner, got, testCase.want)
		}
	}
}

func TestValidateArchiveCOMDATLinkSpecRejectsPathAndLongSectionSpellings(t *testing.T) {
	base := ArchiveCOMDATLinkSpec{
		ArchiveName: "choice.lib", SelectedMember: "chosen.obj", UnusedMember: "unused.obj",
		DirectObject: "direct.obj", DemandSymbol: "demand", UnusedSymbol: "unused",
		COMDATSymbol: "pickany", ReceiptSection: ".fysel", ReceiptMagic: testArchiveReceiptMagic,
		ArchiveCandidate: 17, DirectCandidate: 23,
	}
	for _, edit := range []func(*ArchiveCOMDATLinkSpec){
		func(spec *ArchiveCOMDATLinkSpec) { spec.ArchiveName = `dir\choice.lib` },
		func(spec *ArchiveCOMDATLinkSpec) { spec.SelectedMember = "chosen.o" },
		func(spec *ArchiveCOMDATLinkSpec) { spec.ReceiptSection = ".fyselect" },
		func(spec *ArchiveCOMDATLinkSpec) { spec.DemandSymbol = "bad symbol" },
	} {
		spec := base
		edit(&spec)
		if err := validateArchiveCOMDATLinkSpec(spec); err == nil {
			t.Fatalf("validateArchiveCOMDATLinkSpec() accepted %+v", spec)
		}
	}
}

func TestInspectMSVCMapSymbolsRequiresExactUniquePublicRows(t *testing.T) {
	data := []byte("  Address         Publics by Value              Rva+Base               Lib:Object\r\n" +
		" 0001:00000020       fy_archive_target          0000000140001020 f   choice:chosen.obj\r\n" +
		" 0002:00000010       fy_pick_any_value          0000000140002010     direct.obj\r\n")
	symbols, err := inspectMSVCMapSymbols(data, "fy_archive_target", "fy_pick_any_value", "fy_archive_unused")
	if err != nil {
		t.Fatalf("inspectMSVCMapSymbols() error = %v", err)
	}
	if len(symbols) != 2 || symbols["fy_archive_target"].Owner != "choice:chosen.obj" ||
		symbols["fy_pick_any_value"].PreferredAddress != 0x140002010 {
		t.Fatalf("map symbols = %+v", symbols)
	}
	duplicate := append(data, []byte(" 0001:00000024 fy_archive_target 0000000140001024 f other.obj\r\n")...)
	if _, err := inspectMSVCMapSymbols(duplicate, "fy_archive_target"); err == nil {
		t.Fatal("inspectMSVCMapSymbols() accepted a duplicate exact symbol")
	}
}

func TestInspectArchiveCOMDATLinkBindsIndexMapAndFinalImage(t *testing.T) {
	archive, direct, image, linkMap, spec := testArchiveCOMDATLinkInputs()
	report, err := InspectArchiveCOMDATLink(archive, direct, image, linkMap, spec)
	if err != nil {
		t.Fatalf("InspectArchiveCOMDATLink() error = %v", err)
	}
	if report.ArchiveMembers != 2 || report.ExtractedMember != spec.SelectedMember || report.DemandOwner != "archive" || report.COMDATOwner != "direct" ||
		report.TargetRVA != 0x1000 || report.COMDATRVA != 0x2040 || report.SelectedCOMDATValue != spec.DirectCandidate {
		t.Fatalf("archive COMDAT link report = %+v", report)
	}
}

func TestInspectArchiveCOMDATLinkRejectsIndependentSelectionMutations(t *testing.T) {
	baselineArchive, baselineDirect, baselineImage, baselineMap, spec := testArchiveCOMDATLinkInputs()
	archiveReport, err := InspectArchive(baselineArchive)
	if err != nil {
		t.Fatal(err)
	}
	chosen, err := exactArchiveMember(archiveReport.Members, spec.SelectedMember)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name string
		edit func(*[]byte, *[]byte, *[]byte, *[]byte)
		want string
	}{
		{name: "archive candidate is not pick-any", edit: func(archive, _, _, _ *[]byte) {
			(*archive)[int(chosen.DataOffset)+testArchiveCOMDATAuxSelection] = ObjectCOMDATNoDuplicates
		}, want: "not in a pick-any read-only section"},
		{name: "demand map owner drift", edit: func(_, _, _, linkMap *[]byte) {
			*linkMap = bytes.Replace(*linkMap, []byte("choice:chosen.obj"), []byte("direct.obj        "), 1)
		}, want: "want selected archive member"},
		{name: "unknown COMDAT owner", edit: func(_, _, _, linkMap *[]byte) {
			*linkMap = bytes.Replace(*linkMap, []byte("direct.obj"), []byte("otherx.obj"), 1)
		}, want: "not a declared candidate"},
		{name: "unused archive member retained", edit: func(_, _, _, linkMap *[]byte) {
			*linkMap = append(*linkMap, []byte(" 0001:00000020 unused 0000000140001020 f choice:unused.obj\r\n")...)
		}, want: "retained unused archive symbol"},
		{name: "demand address drift", edit: func(_, _, _, linkMap *[]byte) {
			*linkMap = bytes.Replace(*linkMap, []byte("0000000140001000"), []byte("0000000140001001"), 1)
		}, want: "does not match receipt"},
		{name: "selected value drift", edit: func(_, _, image, _ *[]byte) {
			binary.LittleEndian.PutUint32((*image)[testArchiveReceiptRaw+0x40:], 99)
		}, want: "selected COMDAT value is 99"},
		{name: "receipt becomes writable", edit: func(_, _, image, _ *[]byte) {
			section := testSectionHeaderOffset + coffSectionHeaderBytes
			binary.LittleEndian.PutUint32((*image)[section+36:], 0x00000040|sectionReadable|sectionWritable)
		}, want: "receipt section"},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			archive := bytes.Clone(baselineArchive)
			direct := bytes.Clone(baselineDirect)
			image := bytes.Clone(baselineImage)
			linkMap := bytes.Clone(baselineMap)
			testCase.edit(&archive, &direct, &image, &linkMap)
			if _, err := InspectArchiveCOMDATLink(archive, direct, image, linkMap, spec); err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("InspectArchiveCOMDATLink() error = %v, want %q", err, testCase.want)
			}
		})
	}
}

func TestInspectArchiveCOMDATLinkAcceptsArchiveCandidateOwner(t *testing.T) {
	archive, direct, image, linkMap, spec := testArchiveCOMDATLinkInputs()
	linkMap = bytes.Replace(linkMap, []byte("direct.obj"), []byte("choice:chosen.obj"), 1)
	binary.LittleEndian.PutUint32(image[testArchiveReceiptRaw+0x40:], uint32(spec.ArchiveCandidate))
	report, err := InspectArchiveCOMDATLink(archive, direct, image, linkMap, spec)
	if err != nil {
		t.Fatal(err)
	}
	if report.COMDATOwner != "archive" || report.SelectedCOMDATValue != spec.ArchiveCandidate {
		t.Fatalf("archive-owned COMDAT report = %+v", report)
	}
}

func TestInspectArchiveCOMDATLinkRejectsObjectCandidateMutations(t *testing.T) {
	baselineArchive, baselineDirect, image, linkMap, spec := testArchiveCOMDATLinkInputs()
	archiveReport, err := InspectArchive(baselineArchive)
	if err != nil {
		t.Fatal(err)
	}
	chosen, err := exactArchiveMember(archiveReport.Members, spec.SelectedMember)
	if err != nil {
		t.Fatal(err)
	}
	for _, testCase := range []struct {
		name string
		edit func(*[]byte, *[]byte)
		want string
	}{
		{name: "archive object value drift", edit: func(archive, _ *[]byte) {
			binary.LittleEndian.PutUint32((*archive)[int(chosen.DataOffset)+testArchiveObjectRaw+1:], 19)
		}, want: "archive COMDAT object value is 19"},
		{name: "direct object value drift", edit: func(_, direct *[]byte) {
			binary.LittleEndian.PutUint32((*direct)[testArchiveObjectRaw+1:], 29)
		}, want: "direct COMDAT object value is 29"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			archive := bytes.Clone(baselineArchive)
			direct := bytes.Clone(baselineDirect)
			testCase.edit(&archive, &direct)
			if _, err := InspectArchiveCOMDATLink(archive, direct, image, linkMap, spec); err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("InspectArchiveCOMDATLink() error = %v, want %q", err, testCase.want)
			}
		})
	}
}

func TestRequireDefinedExternalNonCOMDATFunctionRejectsPackagedFunction(t *testing.T) {
	report, err := InspectObject(namedMinimalAMD64COFF("unused"))
	if err != nil {
		t.Fatal(err)
	}
	report.Sections[0].COMDATSelection = ObjectCOMDATAny
	report.Sections[0].Characteristics |= sectionLinkCOMDAT
	if err := requireDefinedExternalNonCOMDATFunction(report, "unused", "archive unused"); err == nil ||
		!strings.Contains(err.Error(), "must be in a non-COMDAT section") {
		t.Fatalf("requireDefinedExternalNonCOMDATFunction() error = %v", err)
	}
}

const (
	testArchiveObjectRaw          = coffFileHeaderBytes + 2*coffSectionHeaderBytes
	testArchiveObjectSymbols      = testArchiveObjectRaw + 5
	testArchiveCOMDATAuxSelection = testArchiveObjectSymbols + 36 + 14
	testArchiveReceiptRaw         = 0x400
)

var testArchiveReceiptMagic = [16]byte{'F', 'Y', 'A', 'R', '-', 'T', 'E', 'S', 'T', '-', 'V', '0', '0', '0', '0', '!'}

func testArchiveCOMDATLinkInputs() ([]byte, []byte, []byte, []byte, ArchiveCOMDATLinkSpec) {
	chosen := testPickAnyObject("demand", "pickany", 17)
	unused := namedMinimalAMD64COFF("unused")
	direct := testPickAnyObject("anchor", "pickany", 23)
	archive := testIndexedArchive(chosen, unused)
	image := testArchiveSelectionImage(23)
	linkMap := []byte("  Address         Publics by Value              Rva+Base               Lib:Object\r\n" +
		" 0001:00000000 demand 0000000140001000 f choice:chosen.obj\r\n" +
		" 0002:00000040 pickany 0000000140002040 direct.obj\r\n")
	spec := ArchiveCOMDATLinkSpec{
		ArchiveName: "choice.lib", SelectedMember: "chosen.obj", UnusedMember: "unused.obj",
		DirectObject: "direct.obj", DemandSymbol: "demand", UnusedSymbol: "unused",
		COMDATSymbol: "pickany", ReceiptSection: ".fysel", ReceiptMagic: testArchiveReceiptMagic,
		ArchiveCandidate: 17, DirectCandidate: 23,
	}
	return archive, direct, image, linkMap, spec
}

func testPickAnyObject(function, key string, value int32) []byte {
	object := make([]byte, testArchiveObjectSymbols+4*18+4)
	binary.LittleEndian.PutUint16(object, machineAMD64)
	binary.LittleEndian.PutUint16(object[2:], 2)
	binary.LittleEndian.PutUint32(object[8:], testArchiveObjectSymbols)
	binary.LittleEndian.PutUint32(object[12:], 4)

	text := coffFileHeaderBytes
	copy(object[text:], ".text")
	binary.LittleEndian.PutUint32(object[text+16:], 1)
	binary.LittleEndian.PutUint32(object[text+20:], testArchiveObjectRaw)
	binary.LittleEndian.PutUint32(object[text+36:], sectionContainsCode|sectionExecutable|sectionReadable)
	object[testArchiveObjectRaw] = 0xc3

	data := text + coffSectionHeaderBytes
	copy(object[data:], ".rdata$x")
	binary.LittleEndian.PutUint32(object[data+16:], 4)
	binary.LittleEndian.PutUint32(object[data+20:], testArchiveObjectRaw+1)
	binary.LittleEndian.PutUint32(object[data+36:], 0x00000040|sectionReadable|sectionLinkCOMDAT)
	binary.LittleEndian.PutUint32(object[testArchiveObjectRaw+1:], uint32(value))

	symbol := testArchiveObjectSymbols
	copy(object[symbol:], function)
	binary.LittleEndian.PutUint16(object[symbol+12:], 1)
	binary.LittleEndian.PutUint16(object[symbol+14:], coffSymbolTypeFunction)
	object[symbol+16] = coffStorageClassExternal

	sectionDefinition := symbol + 18
	copy(object[sectionDefinition:], ".rdata$x")
	binary.LittleEndian.PutUint16(object[sectionDefinition+12:], 2)
	object[sectionDefinition+16] = coffStorageClassStatic
	object[sectionDefinition+17] = 1
	auxiliary := sectionDefinition + 18
	binary.LittleEndian.PutUint32(object[auxiliary:], 4)
	object[auxiliary+14] = ObjectCOMDATAny

	dataKey := auxiliary + 18
	copy(object[dataKey:], key)
	binary.LittleEndian.PutUint16(object[dataKey+12:], 2)
	object[dataKey+16] = coffStorageClassExternal
	binary.LittleEndian.PutUint32(object[dataKey+18:], 4)
	return object
}

func testIndexedArchive(chosen, unused []byte) []byte {
	firstSize := 4 + 3*4 + len("demand\x00pickany\x00unused\x00")
	secondSize := 4 + 2*4 + 4 + 3*2 + len("demand\x00pickany\x00unused\x00")
	firstHeader := len(coffArchiveSignature)
	secondHeader := nextSyntheticArchiveMemberOffset(firstHeader, firstSize)
	chosenHeader := nextSyntheticArchiveMemberOffset(secondHeader, secondSize)
	unusedHeader := nextSyntheticArchiveMemberOffset(chosenHeader, len(chosen))

	first := make([]byte, firstSize)
	binary.BigEndian.PutUint32(first, 3)
	binary.BigEndian.PutUint32(first[4:], uint32(chosenHeader))
	binary.BigEndian.PutUint32(first[8:], uint32(chosenHeader))
	binary.BigEndian.PutUint32(first[12:], uint32(unusedHeader))
	copy(first[16:], "demand\x00pickany\x00unused\x00")

	second := make([]byte, secondSize)
	binary.LittleEndian.PutUint32(second, 2)
	binary.LittleEndian.PutUint32(second[4:], uint32(chosenHeader))
	binary.LittleEndian.PutUint32(second[8:], uint32(unusedHeader))
	binary.LittleEndian.PutUint32(second[12:], 3)
	binary.LittleEndian.PutUint16(second[16:], 1)
	binary.LittleEndian.PutUint16(second[18:], 1)
	binary.LittleEndian.PutUint16(second[20:], 2)
	copy(second[22:], "demand\x00pickany\x00unused\x00")

	var result bytes.Buffer
	result.Write(coffArchiveSignature[:])
	writeSyntheticArchiveMember(&result, "/", first)
	writeSyntheticArchiveMember(&result, "/", second)
	writeSyntheticArchiveMember(&result, "chosen.obj/", chosen)
	writeSyntheticArchiveMember(&result, "unused.obj/", unused)
	return result.Bytes()
}

func testArchiveSelectionImage(value int32) []byte {
	base := minimalAMD64PE()
	image := make([]byte, 0x600)
	copy(image, base)
	coff := testPEOffset + 4
	binary.LittleEndian.PutUint16(image[coff+2:], 2)
	binary.LittleEndian.PutUint32(image[testOptionalHeaderOffset+56:], 0x3000)
	receipt := testSectionHeaderOffset + coffSectionHeaderBytes
	copy(image[receipt:], ".fysel")
	binary.LittleEndian.PutUint32(image[receipt+8:], 0x80)
	binary.LittleEndian.PutUint32(image[receipt+12:], 0x2000)
	binary.LittleEndian.PutUint32(image[receipt+16:], 0x200)
	binary.LittleEndian.PutUint32(image[receipt+20:], testArchiveReceiptRaw)
	binary.LittleEndian.PutUint32(image[receipt+36:], 0x00000040|sectionReadable)
	image[testREL32ImageRaw] = 0xc3
	copy(image[testArchiveReceiptRaw:], testArchiveReceiptMagic[:])
	binary.LittleEndian.PutUint64(image[testArchiveReceiptRaw+16:], 0x140001000)
	binary.LittleEndian.PutUint64(image[testArchiveReceiptRaw+24:], 0x140002040)
	binary.LittleEndian.PutUint32(image[testArchiveReceiptRaw+0x40:], uint32(value))
	return image
}
