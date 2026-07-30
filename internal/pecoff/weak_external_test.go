package pecoff

import (
	"bytes"
	"encoding/binary"
	"math"
	"strings"
	"testing"
)

const (
	testWeakRawOffset   = coffFileHeaderBytes + coffSectionHeaderBytes
	testWeakRelocOffset = testWeakRawOffset + 22
	testWeakSymbols     = testWeakRelocOffset + 2*coffRelocationBytes
	testWeakSourceRVA   = uint32(0x1080)
	testWeakSourceRaw   = testREL32ImageRaw + int(testWeakSourceRVA-testREL32SourceRVA)
)

func TestBuildAMD64WeakExternalProbeObjectEmitsExactAliases(t *testing.T) {
	object := testWeakObject()
	report, err := InspectObject(object)
	if err != nil {
		t.Fatal(err)
	}
	missing, err := exactObjectSymbol(report.SymbolRecords, "fy_wmiss")
	if err != nil {
		t.Fatal(err)
	}
	override, err := exactObjectSymbol(report.SymbolRecords, "fy_wovr")
	if err != nil {
		t.Fatal(err)
	}
	if missing.TableIndex != 2 || missing.WeakFallbackIndex != 6 || missing.WeakFallback != "fy_mfbk" || missing.WeakSearch != ObjectWeakSearchAlias ||
		override.TableIndex != 4 || override.WeakFallbackIndex != 7 || override.WeakFallback != "fy_ofbk" || override.WeakSearch != ObjectWeakSearchAlias {
		t.Fatalf("weak symbols = %+v, %+v", missing, override)
	}
	if len(report.RelocationRecords) != 2 || report.RelocationRecords[0].SymbolIndex != 2 || report.RelocationRecords[1].SymbolIndex != 4 {
		t.Fatalf("weak relocations = %+v", report.RelocationRecords)
	}
}

func TestInspectAMD64WeakExternalLinkProvesFallbackAndOverride(t *testing.T) {
	report, err := InspectAMD64WeakExternalLink(testWeakObject(), testWeakImage(), testWeakMap(), testWeakSpec())
	if err != nil {
		t.Fatal(err)
	}
	if !report.Fallback.UsedFallback || report.Fallback.SelectedSymbol != "fy_mfbk" || report.Fallback.SelectedTargetRVA != testWeakSourceRVA+10 ||
		report.Override.UsedFallback || report.Override.SelectedSymbol != "fy_wovr" || report.Override.SelectedTargetRVA != testVariantTargetRVA ||
		report.Fallback.LinkedDisplacement != report.Fallback.ExpectedDisplacement || report.Override.LinkedDisplacement != report.Override.ExpectedDisplacement {
		t.Fatalf("weak-external report = %+v", report)
	}
}

func TestInspectAMD64WeakExternalLinkRejectsIndependentMutations(t *testing.T) {
	tests := []struct {
		name string
		edit func(*[]byte, *[]byte, *[]byte)
		want string
	}{
		{name: "invalid weak search", edit: func(object, _, _ *[]byte) {
			binary.LittleEndian.PutUint32((*object)[testWeakSymbols+3*18+4:], 4)
		}, want: "invalid search characteristic"},
		{name: "weak storage class becomes ordinary external", edit: func(object, _, _ *[]byte) {
			(*object)[testWeakSymbols+2*18+16] = coffStorageClassExternal
		}, want: "uses unsupported auxiliary records"},
		{name: "weak fallback targets auxiliary", edit: func(object, _, _ *[]byte) {
			binary.LittleEndian.PutUint32((*object)[testWeakSymbols+3*18:], 3)
		}, want: "auxiliary fallback symbol"},
		{name: "weak reserved bytes are nonzero", edit: func(object, _, _ *[]byte) {
			(*object)[testWeakSymbols+3*18+8] = 1
		}, want: "nonzero reserved bytes"},
		{name: "entry value overflows its section", edit: func(object, _, _ *[]byte) {
			binary.LittleEndian.PutUint32((*object)[testWeakSymbols+8:], math.MaxUint32)
		}, want: "exceeds its object section"},
		{name: "fallback relocation target drifts", edit: func(object, _, _ *[]byte) {
			binary.LittleEndian.PutUint32((*object)[testWeakRelocOffset+4:], 4)
		}, want: "want \"fy_wmiss\" REL32"},
		{name: "fallback owner drifts", edit: func(_, _, linkMap *[]byte) {
			*linkMap = bytes.Replace(*linkMap, []byte("fy_mfbk 000000014000108a f weak.obj"), []byte("fy_mfbk 000000014000108a f xxxx.obj"), 1)
		}, want: "selected target owner"},
		{name: "override owner drifts", edit: func(_, _, linkMap *[]byte) {
			*linkMap = bytes.Replace(*linkMap, []byte("fy_wovr 0000000140001100 f target.obj"), []byte("fy_wovr 0000000140001100 f otherx.obj"), 1)
		}, want: "selected target owner"},
		{name: "linked fallback displacement drifts", edit: func(_, image, _ *[]byte) {
			(*image)[testWeakSourceRaw+1]++
		}, want: "linked weak displacement"},
		{name: "linked override displacement drifts", edit: func(_, image, _ *[]byte) {
			(*image)[testWeakSourceRaw+6]++
		}, want: "linked weak displacement"},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			object := testWeakObject()
			image := testWeakImage()
			linkMap := testWeakMap()
			testCase.edit(&object, &image, &linkMap)
			if _, err := InspectAMD64WeakExternalLink(object, image, linkMap, testWeakSpec()); err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("InspectAMD64WeakExternalLink() error = %v, want %q", err, testCase.want)
			}
		})
	}
}

func TestBuildAMD64WeakExternalProbeObjectRejectsInvalidNames(t *testing.T) {
	if _, err := BuildAMD64WeakExternalProbeObject("same", "same", "weak1", "weak2", "fall1", "fall2"); err == nil {
		t.Fatal("BuildAMD64WeakExternalProbeObject() accepted duplicate names")
	}
}

func testWeakObject() []byte {
	object, err := BuildAMD64WeakExternalProbeObject("fy_wfb_e", "fy_wov_e", "fy_wmiss", "fy_wovr", "fy_mfbk", "fy_ofbk")
	if err != nil {
		panic(err)
	}
	return object
}

func testWeakImage() []byte {
	image := testREL32Image()
	object := testWeakObject()
	copy(image[testWeakSourceRaw:testWeakSourceRaw+22], object[testWeakRawOffset:testWeakRawOffset+22])
	for _, value := range []struct {
		site   uint32
		target uint32
	}{{site: 1, target: testWeakSourceRVA + 10}, {site: 6, target: testVariantTargetRVA}} {
		displacement, err := amd64REL32Displacement(value.target, 0, testWeakSourceRVA+value.site)
		if err != nil {
			panic(err)
		}
		binary.LittleEndian.PutUint32(image[testWeakSourceRaw+int(value.site):], uint32(displacement))
	}
	return image
}

func testWeakMap() []byte {
	return []byte(" 0001:00000080 fy_wfb_e 0000000140001080 f weak.obj\r\n" +
		" 0001:00000085 fy_wov_e 0000000140001085 f weak.obj\r\n" +
		" 0001:0000008a fy_mfbk 000000014000108a f weak.obj\r\n" +
		" 0001:00000100 fy_wovr 0000000140001100 f target.obj\r\n")
}

func testWeakSpec() AMD64WeakExternalSpec {
	return AMD64WeakExternalSpec{
		ObjectName: "weak.obj", TargetObject: "target.obj",
		FallbackEntry: "fy_wfb_e", OverrideEntry: "fy_wov_e",
		MissingWeak: "fy_wmiss", OverrideWeak: "fy_wovr",
		MissingFallback: "fy_mfbk", OverrideFallback: "fy_ofbk",
	}
}
