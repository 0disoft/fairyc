package pecoff

import (
	"bytes"
	"encoding/binary"
	"strings"
	"testing"
)

const (
	testVariantSourceRVA = uint32(0x1080)
	testVariantTargetRVA = uint32(0x1100)
	testVariantSourceRaw = testREL32ImageRaw + int(testVariantSourceRVA-testREL32SourceRVA)
)

func TestBuildAMD64REL32VariantProbeObjectEmitsClosedSixTypeTable(t *testing.T) {
	object, err := BuildAMD64REL32VariantProbeObject("fy_vsrc", "fy_vdst")
	if err != nil {
		t.Fatal(err)
	}
	report, err := InspectObject(object)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.RelocationRecords) != amd64REL32VariantCount {
		t.Fatalf("variant relocations = %+v", report.RelocationRecords)
	}
	for index, relocation := range report.RelocationRecords {
		if relocation.Type != ObjectRelocationAMD64Rel32+uint16(index) || relocation.Symbol != "fy_vdst" {
			t.Fatalf("variant relocation %d = %+v", index, relocation)
		}
	}
}

func TestInspectAMD64REL32VariantsLinkProvesAllSixFormulas(t *testing.T) {
	report, err := InspectAMD64REL32VariantsLink(testVariantObject(), testVariantImage(), testVariantMap(), testVariantSpec())
	if err != nil {
		t.Fatal(err)
	}
	if report.SourceRVA != testVariantSourceRVA || report.TargetRVA != testVariantTargetRVA || len(report.Variants) != amd64REL32VariantCount {
		t.Fatalf("REL32 variants report = %+v", report)
	}
	for index, variant := range report.Variants {
		if variant.Type != ObjectRelocationAMD64Rel32+uint16(index) || variant.TrailingBytes != uint32(index) ||
			variant.LinkedDisplacement != variant.ExpectedDisplacement {
			t.Fatalf("REL32 variant %d = %+v", index, variant)
		}
	}
}

func TestInspectAMD64REL32VariantsLinkRejectsIndependentMutations(t *testing.T) {
	tests := []struct {
		name string
		edit func(*[]byte, *[]byte, *[]byte)
		want string
	}{
		{name: "variant type drifts", edit: func(object, _, _ *[]byte) {
			relocations := 60 + 47
			binary.LittleEndian.PutUint16((*object)[relocations+3*coffRelocationBytes+8:], ObjectRelocationAMD64Rel32_2)
		}, want: "type 0x6 has 2 exact"},
		{name: "variant target drifts", edit: func(object, _, _ *[]byte) {
			relocations := 60 + 47
			binary.LittleEndian.PutUint32((*object)[relocations+2*coffRelocationBytes+4:], 0)
		}, want: "type 0x6 has 0 exact"},
		{name: "object addend drifts", edit: func(object, _, _ *[]byte) {
			(*object)[60+2]++
		}, want: "linked displacement"},
		{name: "linked displacement drifts", edit: func(_, image, _ *[]byte) {
			(*image)[testVariantSourceRaw+2]++
		}, want: "linked displacement"},
		{name: "trailing bytes drift", edit: func(_, image, _ *[]byte) {
			(*image)[testVariantSourceRaw+10] ^= 0xff
		}, want: "trailing bytes drifted"},
		{name: "source owner drifts", edit: func(_, _, linkMap *[]byte) {
			*linkMap = bytes.Replace(*linkMap, []byte("variants.obj"), []byte("otherxxx.obj"), 1)
		}, want: "source owner"},
		{name: "target owner drifts", edit: func(_, _, linkMap *[]byte) {
			*linkMap = bytes.Replace(*linkMap, []byte("target.obj"), []byte("otherx.obj"), 1)
		}, want: "target owner"},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			object := testVariantObject()
			image := testVariantImage()
			linkMap := testVariantMap()
			testCase.edit(&object, &image, &linkMap)
			if _, err := InspectAMD64REL32VariantsLink(object, image, linkMap, testVariantSpec()); err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("InspectAMD64REL32VariantsLink() error = %v, want %q", err, testCase.want)
			}
		})
	}
}

func TestAMD64REL32VariantDisplacementUsesTrailingByteCount(t *testing.T) {
	for trailing := uint32(0); trailing < amd64REL32VariantCount; trailing++ {
		value, err := amd64REL32VariantDisplacement(0x2000, -7, 0x1000, trailing)
		if err != nil || value != int32(0x2000-7-0x1000-4-trailing) {
			t.Fatalf("trailing %d displacement = %d, %v", trailing, value, err)
		}
	}
	if _, err := amd64REL32VariantDisplacement(0, 0, 0, amd64REL32VariantCount); err == nil {
		t.Fatal("amd64REL32VariantDisplacement() accepted REL32_6")
	}
}

func testVariantObject() []byte {
	object, err := BuildAMD64REL32VariantProbeObject("fy_vsrc", "fy_vdst")
	if err != nil {
		panic(err)
	}
	return object
}

func testVariantImage() []byte {
	image := testREL32Image()
	object := testVariantObject()
	copy(image[testVariantSourceRaw:testVariantSourceRaw+47], object[60:60+47])
	site := uint32(2)
	for trailing := uint32(0); trailing < amd64REL32VariantCount; trailing++ {
		addend := int32(binary.LittleEndian.Uint32(object[60+int(site):]))
		displacement, err := amd64REL32VariantDisplacement(testVariantTargetRVA, addend, testVariantSourceRVA+site, trailing)
		if err != nil {
			panic(err)
		}
		binary.LittleEndian.PutUint32(image[testVariantSourceRaw+int(site):], uint32(displacement))
		site += 4 + trailing
	}
	image[testREL32ImageRaw+0x100] = 0xc3
	return image
}

func testVariantMap() []byte {
	return []byte(" 0001:00000080 fy_vsrc 0000000140001080 f variants.obj\r\n" +
		" 0001:00000100 fy_vdst 0000000140001100 f target.obj\r\n")
}

func testVariantSpec() AMD64REL32VariantsSpec {
	return AMD64REL32VariantsSpec{
		SourceSymbol: "fy_vsrc", TargetSymbol: "fy_vdst", SourceObject: "variants.obj", TargetObject: "target.obj",
	}
}
