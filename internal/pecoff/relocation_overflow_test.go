package pecoff

import (
	"encoding/binary"
	"strings"
	"testing"
)

func TestBuildAMD64RelocationOverflowProbeObjectEmitsExactExtendedTable(t *testing.T) {
	t.Parallel()
	object, err := BuildAMD64RelocationOverflowProbeObject("fy_ovfl")
	if err != nil {
		t.Fatal(err)
	}
	if len(object) != 655471 {
		t.Fatalf("relocation-overflow object bytes = %d, want 655471", len(object))
	}
	report, err := InspectObject(object)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Sections) != 1 || report.Sections[0].Relocations != amd64RelocationOverflowProbeCount ||
		report.Relocations != amd64RelocationOverflowProbeCount || len(report.RelocationRecords) != amd64RelocationOverflowProbeCount {
		t.Fatalf("relocation-overflow report = sections %+v, count %d, records %d", report.Sections, report.Relocations, len(report.RelocationRecords))
	}
	for index, relocation := range report.RelocationRecords[:len(report.RelocationRecords)-1] {
		if relocation.SectionIndex != 1 || relocation.Section != ".text" || relocation.VirtualAddress != 0 ||
			relocation.SymbolIndex != 0 || relocation.Symbol != "fy_ovfl" || relocation.Type != ObjectRelocationAMD64Absolute {
			t.Fatalf("relocation %d = %+v", index, relocation)
		}
	}
	last := report.RelocationRecords[len(report.RelocationRecords)-1]
	if last.SectionIndex != 1 || last.Section != ".text" || last.VirtualAddress != 2 ||
		last.SymbolIndex != 1 || last.Symbol != "fy_ovdat" || last.Type != ObjectRelocationAMD64Rel32 {
		t.Fatalf("final relocation = %+v", last)
	}
}

func TestInspectObjectRejectsMalformedRelocationOverflow(t *testing.T) {
	t.Parallel()
	for _, testCase := range []struct {
		name string
		edit func([]byte)
		want string
	}{
		{name: "wrong header marker", edit: func(object []byte) {
			binary.LittleEndian.PutUint16(object[coffFileHeaderBytes+32:], 0xfffe)
		}, want: "count marker"},
		{name: "count below threshold", edit: func(object []byte) {
			binary.LittleEndian.PutUint32(object[coffFileHeaderBytes+coffSectionHeaderBytes+amd64RelocationOverflowRawSize:], 0xffff)
		}, want: "does not exceed"},
		{name: "nonzero sentinel symbol", edit: func(object []byte) {
			binary.LittleEndian.PutUint32(object[coffFileHeaderBytes+coffSectionHeaderBytes+amd64RelocationOverflowRawSize+4:], 1)
		}, want: "nonzero symbol or type"},
		{name: "nonzero sentinel type", edit: func(object []byte) {
			binary.LittleEndian.PutUint16(object[coffFileHeaderBytes+coffSectionHeaderBytes+amd64RelocationOverflowRawSize+8:], 1)
		}, want: "nonzero symbol or type"},
		{name: "truncated table", edit: func(object []byte) {
			binary.LittleEndian.PutUint32(object[coffFileHeaderBytes+coffSectionHeaderBytes+amd64RelocationOverflowRawSize:], uint32(len(object)))
		}, want: "invalid relocation range"},
		{name: "over budget", edit: func(object []byte) {
			binary.LittleEndian.PutUint32(object[coffFileHeaderBytes+coffSectionHeaderBytes+amd64RelocationOverflowRawSize:], maxObjectRelocations+2)
		}, want: "limit is"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			object, err := BuildAMD64RelocationOverflowProbeObject("fy_ovfl")
			if err != nil {
				t.Fatal(err)
			}
			testCase.edit(object)
			if _, err := InspectObject(object); err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("InspectObject() error = %v, want %q", err, testCase.want)
			}
		})
	}
}

func TestInspectAMD64RelocationOverflowLinkBindsMapAndImage(t *testing.T) {
	t.Parallel()
	report, err := InspectAMD64RelocationOverflowLink(testOverflowObject(), testOverflowImage(), testOverflowMap(), testOverflowSpec())
	if err != nil {
		t.Fatal(err)
	}
	if report.Relocations != amd64RelocationOverflowProbeCount || report.SourceRVA != 0x1180 ||
		report.LinkedDisplacement != 1 || report.ExpectedDisplacement != 1 {
		t.Fatalf("relocation-overflow link report = %+v", report)
	}
}

func TestInspectAMD64RelocationOverflowLinkRejectsIndependentMutations(t *testing.T) {
	t.Parallel()
	for _, testCase := range []struct {
		name string
		edit func(*[]byte, *[]byte, *[]byte)
		want string
	}{
		{name: "wrong count", edit: func(object, _, _ *[]byte) {
			symbolOffset := int(binary.LittleEndian.Uint32((*object)[8:]))
			expanded := make([]byte, len(*object)+coffRelocationBytes)
			copy(expanded, (*object)[:symbolOffset])
			copy(expanded[symbolOffset+coffRelocationBytes:], (*object)[symbolOffset:])
			binary.LittleEndian.PutUint32(expanded[8:], uint32(symbolOffset+coffRelocationBytes))
			binary.LittleEndian.PutUint32(expanded[coffFileHeaderBytes+coffSectionHeaderBytes+amd64RelocationOverflowRawSize:], 0x10001)
			*object = expanded
		}, want: "65536 relocations"},
		{name: "wrong map owner", edit: func(_, _, linkMap *[]byte) {
			*linkMap = bytesReplace(*linkMap, "overflow.obj", "xxxxflow.obj")
		}, want: "source owner"},
		{name: "REL32 is not final", edit: func(object, _, _ *[]byte) {
			relocationOffset := int(binary.LittleEndian.Uint32((*object)[coffFileHeaderBytes+24:]))
			first := relocationOffset + coffRelocationBytes
			last := relocationOffset + amd64RelocationOverflowProbeCount*coffRelocationBytes
			firstRecord := append([]byte(nil), (*object)[first:first+coffRelocationBytes]...)
			copy((*object)[first:first+coffRelocationBytes], (*object)[last:last+coffRelocationBytes])
			copy((*object)[last:last+coffRelocationBytes], firstRecord)
		}, want: "want final record"},
		{name: "linked bytes drift", edit: func(_, image, _ *[]byte) {
			(*image)[0x380+1]++
		}, want: "bytes drifted"},
		{name: "linked displacement drift", edit: func(_, image, _ *[]byte) {
			binary.LittleEndian.PutUint32((*image)[0x380+2:], 2)
		}, want: "linked displacement"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			object, image, linkMap := testOverflowObject(), testOverflowImage(), testOverflowMap()
			testCase.edit(&object, &image, &linkMap)
			if _, err := InspectAMD64RelocationOverflowLink(object, image, linkMap, testOverflowSpec()); err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("InspectAMD64RelocationOverflowLink() error = %v, want %q", err, testCase.want)
			}
		})
	}
}

func testOverflowObject() []byte {
	object, err := BuildAMD64RelocationOverflowProbeObject("fy_ovfl")
	if err != nil {
		panic(err)
	}
	return object
}

func testOverflowImage() []byte {
	image := testREL32Image()
	object := testOverflowObject()
	copy(image[0x380:0x38b], object[coffFileHeaderBytes+coffSectionHeaderBytes:coffFileHeaderBytes+coffSectionHeaderBytes+11])
	binary.LittleEndian.PutUint32(image[0x380+2:], 1)
	return image
}

func testOverflowMap() []byte {
	return []byte(" 0001:00000180 fy_ovfl 0000000140001180 f overflow.obj\r\n")
}

func testOverflowSpec() AMD64RelocationOverflowSpec {
	return AMD64RelocationOverflowSpec{SourceSymbol: "fy_ovfl", SourceObject: "overflow.obj"}
}

func bytesReplace(source []byte, old, replacement string) []byte {
	return []byte(strings.Replace(string(source), old, replacement, 1))
}
