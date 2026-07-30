package pecoff

import (
	"bytes"
	"encoding/binary"
	"strings"
	"testing"
)

const (
	testHandlerTextRaw    = coffFileHeaderBytes + 3*coffSectionHeaderBytes
	testHandlerPDataRaw   = testHandlerTextRaw + 1
	testHandlerXDataRaw   = testHandlerPDataRaw + runtimeFunctionBytes
	testHandlerPDataReloc = testHandlerXDataRaw + 8
	testHandlerXDataReloc = testHandlerPDataReloc + 3*coffRelocationBytes
	testHandlerSymbols    = testHandlerXDataReloc + coffRelocationBytes
)

func TestInspectAMD64HandlerLinkProvesExactADDR32NBTarget(t *testing.T) {
	report, err := InspectAMD64HandlerLink(testHandlerObject(), testHandlerImage(), testHandlerMap(), testHandlerSpec())
	if err != nil {
		t.Fatal(err)
	}
	if report.SourceRVA != 0x1000 || report.UnwindInfoRVA != 0x3000 ||
		report.ObjectFlags != unwindFlagUnwindHandler || report.ObjectHandlerAddend != 0 ||
		report.LinkedHandlerRVA != 0x1100 || report.ExpectedHandlerRVA != 0x1100 ||
		report.HandlerMapOwner != "libvcruntime.lib(gs_handler.obj)" || report.UnwindPrefixBytes != 4 {
		t.Fatalf("handler link report = %+v", report)
	}
}

func TestInspectAMD64HandlerLinkRejectsIndependentMutations(t *testing.T) {
	tests := []struct {
		name string
		edit func(*[]byte, *[]byte, *[]byte)
		want string
	}{
		{name: "handler relocation type", edit: func(object, _, _ *[]byte) {
			binary.LittleEndian.PutUint16((*object)[testHandlerXDataReloc+8:], ObjectRelocationAMD64Addr64)
		}, want: "want ADDR32NB"},
		{name: "duplicate handler relocation", edit: func(object, _, _ *[]byte) {
			original := *object
			mutated := make([]byte, len(original)+coffRelocationBytes)
			copy(mutated, original[:testHandlerSymbols])
			copy(mutated[testHandlerSymbols:], original[testHandlerXDataReloc:testHandlerXDataReloc+coffRelocationBytes])
			copy(mutated[testHandlerSymbols+coffRelocationBytes:], original[testHandlerSymbols:])
			binary.LittleEndian.PutUint32(mutated[8:], testHandlerSymbols+coffRelocationBytes)
			binary.LittleEndian.PutUint16(mutated[coffFileHeaderBytes+2*coffSectionHeaderBytes+32:], 2)
			*object = mutated
		}, want: "want exactly one"},
		{name: "handler relocation target", edit: func(object, _, _ *[]byte) {
			binary.LittleEndian.PutUint32((*object)[testHandlerXDataReloc+4:], 0)
		}, want: "tail targets"},
		{name: "handler becomes defined", edit: func(object, _, _ *[]byte) {
			binary.LittleEndian.PutUint16((*object)[testHandlerSymbols+3*18+12:], 1)
		}, want: "not undefined"},
		{name: "handler loses function type", edit: func(object, _, _ *[]byte) {
			binary.LittleEndian.PutUint16((*object)[testHandlerSymbols+3*18+14:], 0)
		}, want: "not an exact external function"},
		{name: "handler addend drifts", edit: func(object, _, _ *[]byte) {
			binary.LittleEndian.PutUint32((*object)[testHandlerXDataRaw+4:], 1)
		}, want: "linked handler RVA"},
		{name: "handler map address drifts", edit: func(_, _, linkMap *[]byte) {
			*linkMap = bytes.Replace(*linkMap, []byte("0000000140001100"), []byte("0000000140001101"), 1)
		}, want: "linked handler RVA"},
		{name: "handler map row disappears", edit: func(_, _, linkMap *[]byte) {
			*linkMap = bytes.Replace(*linkMap, []byte("__GSHandlerCheck_SEH"), []byte("__GSHandlerCheck_SEX"), 1)
		}, want: "no owned handler target"},
		{name: "linked handler drifts", edit: func(_, image, _ *[]byte) {
			binary.LittleEndian.PutUint32((*image)[testUnwindXDataRawOffset+4:], 0x1101)
		}, want: "linked handler RVA"},
		{name: "unwind prefix drifts", edit: func(_, image, _ *[]byte) {
			(*image)[testUnwindXDataRawOffset+1] = 1
		}, want: "unwind prefix drifted"},
		{name: "object loses handler flag", edit: func(object, _, _ *[]byte) {
			(*object)[testHandlerXDataRaw] = 1
		}, want: "invalid version/flags"},
		{name: "source owner drifts", edit: func(_, _, linkMap *[]byte) {
			*linkMap = bytes.Replace(*linkMap, []byte("source.obj"), []byte("otherx.obj"), 1)
		}, want: "source owner"},
		{name: "resolved handler is not executable", edit: func(_, image, linkMap *[]byte) {
			binary.LittleEndian.PutUint32((*image)[testUnwindXDataRawOffset+4:], 0x3000)
			*linkMap = bytes.Replace(*linkMap, []byte("0000000140001100"), []byte("0000000140003000"), 1)
		}, want: "invalid handler RVA"},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			object := testHandlerObject()
			image := testHandlerImage()
			linkMap := testHandlerMap()
			testCase.edit(&object, &image, &linkMap)
			if _, err := InspectAMD64HandlerLink(object, image, linkMap, testHandlerSpec()); err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("InspectAMD64HandlerLink() error = %v, want %q", err, testCase.want)
			}
		})
	}
}

func TestInspectAMD64HandlerLinkRejectsInvalidSpec(t *testing.T) {
	for _, spec := range []AMD64HandlerLinkSpec{
		{},
		{SourceSymbol: "same", HandlerSymbol: "same", SourceObject: "source.obj"},
		{SourceSymbol: "source", HandlerSymbol: "bad symbol", SourceObject: "source.obj"},
		{SourceSymbol: "source", HandlerSymbol: "handler", SourceObject: `dir\source.obj`},
	} {
		if _, err := InspectAMD64HandlerLink(testHandlerObject(), testHandlerImage(), testHandlerMap(), spec); err == nil {
			t.Fatalf("InspectAMD64HandlerLink() accepted %+v", spec)
		}
	}
}

func testHandlerSpec() AMD64HandlerLinkSpec {
	return AMD64HandlerLinkSpec{SourceSymbol: "source", SourceObject: "source.obj", HandlerSymbol: "__GSHandlerCheck_SEH"}
}

func testHandlerMap() []byte {
	return []byte("  Address         Publics by Value              Rva+Base               Lib:Object\r\n" +
		" 0001:00000000 source 0000000140001000 f source.obj\r\n" +
		" 0001:00000100 __GSHandlerCheck_SEH 0000000140001100 f libvcruntime.lib(gs_handler.obj)\r\n" +
		" Static symbols\r\n" +
		" 0003:00000000 $unwind 0000000140003000 source.obj\r\n")
}

func testHandlerObject() []byte {
	object := make([]byte, testHandlerSymbols+4*18+4)
	binary.LittleEndian.PutUint16(object, machineAMD64)
	binary.LittleEndian.PutUint16(object[2:], 3)
	binary.LittleEndian.PutUint32(object[8:], testHandlerSymbols)
	binary.LittleEndian.PutUint32(object[12:], 4)

	text := coffFileHeaderBytes
	copy(object[text:], ".text")
	binary.LittleEndian.PutUint32(object[text+16:], 1)
	binary.LittleEndian.PutUint32(object[text+20:], testHandlerTextRaw)
	binary.LittleEndian.PutUint32(object[text+36:], sectionContainsCode|sectionExecutable|sectionReadable)
	object[testHandlerTextRaw] = 0xc3

	pdata := text + coffSectionHeaderBytes
	copy(object[pdata:], ".pdata")
	binary.LittleEndian.PutUint32(object[pdata+16:], runtimeFunctionBytes)
	binary.LittleEndian.PutUint32(object[pdata+20:], testHandlerPDataRaw)
	binary.LittleEndian.PutUint32(object[pdata+24:], testHandlerPDataReloc)
	binary.LittleEndian.PutUint16(object[pdata+32:], 3)
	binary.LittleEndian.PutUint32(object[pdata+36:], sectionInitializedData|sectionReadable)
	binary.LittleEndian.PutUint32(object[testHandlerPDataRaw+4:], 1)
	for index, site := range []uint32{0, 4, 8} {
		relocation := testHandlerPDataReloc + index*coffRelocationBytes
		binary.LittleEndian.PutUint32(object[relocation:], site)
		target := uint32(1)
		if index == 2 {
			target = 2
		}
		binary.LittleEndian.PutUint32(object[relocation+4:], target)
		binary.LittleEndian.PutUint16(object[relocation+8:], ObjectRelocationAMD64Addr32NB)
	}

	xdata := pdata + coffSectionHeaderBytes
	copy(object[xdata:], ".xdata")
	binary.LittleEndian.PutUint32(object[xdata+16:], 8)
	binary.LittleEndian.PutUint32(object[xdata+20:], testHandlerXDataRaw)
	binary.LittleEndian.PutUint32(object[xdata+24:], testHandlerXDataReloc)
	binary.LittleEndian.PutUint16(object[xdata+32:], 1)
	binary.LittleEndian.PutUint32(object[xdata+36:], sectionInitializedData|sectionReadable)
	object[testHandlerXDataRaw] = 1 | unwindFlagUnwindHandler<<3
	binary.LittleEndian.PutUint32(object[testHandlerXDataReloc:], 4)
	binary.LittleEndian.PutUint32(object[testHandlerXDataReloc+4:], 3)
	binary.LittleEndian.PutUint16(object[testHandlerXDataReloc+8:], ObjectRelocationAMD64Addr32NB)

	symbol := testHandlerSymbols
	copy(object[symbol:], "source")
	binary.LittleEndian.PutUint16(object[symbol+12:], 1)
	binary.LittleEndian.PutUint16(object[symbol+14:], coffSymbolTypeFunction)
	object[symbol+16] = coffStorageClassExternal
	symbol += 18
	copy(object[symbol:], "$begin")
	binary.LittleEndian.PutUint16(object[symbol+12:], 1)
	object[symbol+16] = coffStorageClassStatic
	symbol += 18
	copy(object[symbol:], "$unwind")
	binary.LittleEndian.PutUint16(object[symbol+12:], 3)
	object[symbol+16] = coffStorageClassStatic
	symbol += 18
	copy(object[symbol:], "__GSHand")
	// The full long name is stored in the string table below.
	binary.LittleEndian.PutUint32(object[symbol:], 0)
	binary.LittleEndian.PutUint32(object[symbol+4:], 4)
	binary.LittleEndian.PutUint16(object[symbol+14:], coffSymbolTypeFunction)
	object[symbol+16] = coffStorageClassExternal
	stringTable := testHandlerSymbols + 4*18
	name := []byte("__GSHandlerCheck_SEH\x00")
	binary.LittleEndian.PutUint32(object[stringTable:], uint32(4+len(name)))
	object = append(object, name...)
	return object
}

func testHandlerImage() []byte {
	image := minimalAMD64PEWithUnwind()
	emitPESection(image, testUnwindSectionsOffset, ".text", 0x200, 0x1000, 0x200, testUnwindTextRawOffset, sectionContainsCode|sectionExecutable|sectionReadable)
	image[testUnwindTextRawOffset+0x100] = 0xc3
	image[testUnwindXDataRawOffset] = 1 | unwindFlagUnwindHandler<<3
	binary.LittleEndian.PutUint32(image[testUnwindXDataRawOffset+4:], 0x1100)
	return image
}
