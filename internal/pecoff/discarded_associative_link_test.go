package pecoff

import (
	"bytes"
	"encoding/binary"
	"strings"
	"testing"
)

const testDiscardedSymbols = testAddressSymbols + 4

func TestInspectDiscardedAssociativeCOMDATLinkBindsArchiveSelectionAndDiscard(t *testing.T) {
	report, err := InspectDiscardedAssociativeCOMDATLink(
		testDiscardedArchiveObject(), testDiscardedDirectObject(), testDiscardedImage(0x02), testAssociativeLinkMap(), testDiscardedAssociativeSpec(),
	)
	if err != nil {
		t.Fatal(err)
	}
	if report.SelectedOwner != "archive" || report.DiscardedOwner != "direct" ||
		report.SelectedBeginRVA != testAddressSourceRVA || report.SelectedEndRVA != testAddressSourceRVA+1 ||
		report.SelectedUnwindRVA != 0x3000 || report.DiscardedPayloadMatches != 0 || report.DiscardedRawMatches != 0 || report.OrphanPDataBytes != 0 ||
		report.SelectedUnwindDigest == report.DiscardedUnwindDigest {
		t.Fatalf("discarded associative COMDAT report = %+v", report)
	}
}

func TestInspectDiscardedAssociativeCOMDATLinkAcceptsDirectSelection(t *testing.T) {
	image := testDiscardedImage(0x12)
	linkMap := bytes.ReplaceAll(testAssociativeLinkMap(), []byte("choice:chosen.obj"), []byte("direct.obj       "))
	report, err := InspectDiscardedAssociativeCOMDATLink(
		testDiscardedArchiveObject(), testDiscardedDirectObject(), image, linkMap, testDiscardedAssociativeSpec(),
	)
	if err != nil {
		t.Fatal(err)
	}
	if report.SelectedOwner != "direct" || report.DiscardedOwner != "archive" || report.DiscardedPayloadMatches != 0 || report.DiscardedRawMatches != 0 || report.OrphanPDataBytes != 0 {
		t.Fatalf("direct-selected associative COMDAT report = %+v", report)
	}
}

func TestInspectDiscardedAssociativeCOMDATLinkRejectsIndependentDiscardMutations(t *testing.T) {
	tests := []struct {
		name string
		edit func(*[]byte, *[]byte, *[]byte, *[]byte)
		want string
	}{
		{name: "candidates lose distinct payloads", edit: func(_, direct, _, _ *[]byte) {
			(*direct)[testAddressXDataRaw+5] = 0x02
		}, want: "indistinguishable unwind payloads"},
		{name: "parent owner is undeclared", edit: func(_, _, _, linkMap *[]byte) {
			*linkMap = bytes.Replace(*linkMap, []byte("choice:chosen.obj"), []byte("evilll:chosen.obj"), 1)
		}, want: "want a declared candidate"},
		{name: "unwind owner disagrees", edit: func(_, _, _, linkMap *[]byte) {
			index := bytes.LastIndex(*linkMap, []byte("choice:chosen.obj"))
			copy((*linkMap)[index:], []byte("direct.obj       "))
		}, want: "selected associative unwind owner"},
		{name: "selected final payload drifts", edit: func(_, _, image, _ *[]byte) {
			(*image)[testUnwindXDataRawOffset+5] = 0x22
		}, want: "final unwind payload does not match"},
		{name: "discarded payload remains referenced", edit: func(_, direct, image, _ *[]byte) {
			binary.LittleEndian.PutUint32((*image)[testUnwindDirectoryOffset+4:], 2*runtimeFunctionBytes)
			second := testUnwindPDataRawOffset + runtimeFunctionBytes
			binary.LittleEndian.PutUint32((*image)[second:], testAddressSourceRVA+0x20)
			binary.LittleEndian.PutUint32((*image)[second+4:], testAddressSourceRVA+0x21)
			binary.LittleEndian.PutUint32((*image)[second+8:], 0x3008)
			copy((*image)[testUnwindXDataRawOffset+8:testUnwindXDataRawOffset+16], (*direct)[testAddressXDataRaw:testAddressXDataRaw+8])
		}, want: "references the discarded associative unwind payload"},
		{name: "discarded payload remains unreferenced", edit: func(_, direct, image, _ *[]byte) {
			copy((*image)[testUnwindXDataRawOffset+8:testUnwindXDataRawOffset+16], (*direct)[testAddressXDataRaw:testAddressXDataRaw+8])
		}, want: "retains the discarded associative unwind payload"},
		{name: "direct parent is not ANY", edit: func(_, direct, _, _ *[]byte) {
			(*direct)[testDiscardedSymbols+18+14] = ObjectCOMDATLargest
		}, want: "want ANY"},
		{name: "direct parent becomes static", edit: func(_, direct, _, _ *[]byte) {
			(*direct)[testDiscardedSymbols+2*18+16] = coffStorageClassStatic
		}, want: "not an exact defined external function"},
		{name: "direct parent loses function type", edit: func(_, direct, _, _ *[]byte) {
			binary.LittleEndian.PutUint16((*direct)[testDiscardedSymbols+2*18+14:], 0)
		}, want: "not an exact defined external function"},
		{name: "orphan pdata bytes remain", edit: func(_, _, image, _ *[]byte) {
			(*image)[testUnwindPDataRawOffset+runtimeFunctionBytes] = 0x7f
		}, want: "nonzero .pdata bytes outside the exception directory"},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			archive := testDiscardedArchiveObject()
			direct := testDiscardedDirectObject()
			image := testDiscardedImage(0x02)
			linkMap := testAssociativeLinkMap()
			testCase.edit(&archive, &direct, &image, &linkMap)
			if _, err := InspectDiscardedAssociativeCOMDATLink(archive, direct, image, linkMap, testDiscardedAssociativeSpec()); err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("InspectDiscardedAssociativeCOMDATLink() error = %v, want %q", err, testCase.want)
			}
		})
	}
}

func testDiscardedDirectObject() []byte {
	return testDiscardedObject(0x12)
}

func testDiscardedArchiveObject() []byte {
	return testDiscardedObject(0x02)
}

func testDiscardedObject(operation byte) []byte {
	base := testAssociativeLinkObject()
	const symbolShift = testDiscardedSymbols - testAddressSymbols
	newSymbols := testDiscardedSymbols
	object := make([]byte, len(base)+symbolShift)
	copy(object[:testAddressSymbols], base[:testAddressSymbols])
	copy(object[newSymbols:], base[testAddressSymbols:])
	binary.LittleEndian.PutUint32(object[8:], uint32(newSymbols))
	xdataHeader := coffFileHeaderBytes + 3*coffSectionHeaderBytes
	binary.LittleEndian.PutUint32(object[xdataHeader+16:], 8)
	xdataAuxiliary := newSymbols + 8*18
	binary.LittleEndian.PutUint32(object[xdataAuxiliary:], 8)
	copy(object[testAddressXDataRaw:testAddressXDataRaw+8], []byte{1, 1, 1, 0, 1, operation, 0, 0})
	return object
}

func testDiscardedImage(operation byte) []byte {
	image := testAddressImage()
	copy(image[testUnwindXDataRawOffset:testUnwindXDataRawOffset+8], []byte{1, 1, 1, 0, 1, operation, 0, 0})
	return image
}

func testDiscardedAssociativeSpec() DiscardedAssociativeCOMDATSpec {
	return DiscardedAssociativeCOMDATSpec{
		ArchiveName: "choice.lib", ArchiveMember: "chosen.obj", DirectObject: "direct.obj", SourceSymbol: "source",
	}
}
