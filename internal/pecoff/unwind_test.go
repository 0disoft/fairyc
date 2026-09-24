package pecoff

import (
	"encoding/binary"
	"strings"
	"testing"
)

const (
	testUnwindPEOffset        = 0x80
	testUnwindOptionalOffset  = testUnwindPEOffset + 4 + coffFileHeaderBytes
	testUnwindSectionsOffset  = testUnwindOptionalOffset + 240
	testUnwindTextRawOffset   = 0x400
	testUnwindPDataRawOffset  = 0x600
	testUnwindXDataRawOffset  = 0x800
	testUnwindDirectoryOffset = testUnwindOptionalOffset + 112 + exceptionDirectoryIndex*8
)

func TestInspectAMD64UnwindAcceptsClosedRuntimeFunction(t *testing.T) {
	t.Parallel()
	report, err := InspectAMD64Unwind(minimalAMD64PEWithUnwind())
	if err != nil {
		t.Fatal(err)
	}
	if report.RuntimeFunctions != 1 || report.UniqueUnwindInfos != 1 || report.HandlerInfos != 0 || report.ChainedInfos != 0 || report.MaximumUnwindCodes != 0 {
		t.Fatalf("unwind report = %+v", report)
	}
	if len(report.Functions) != 1 || report.Functions[0] != (RuntimeFunction{BeginRVA: 0x1000, EndRVA: 0x1001, UnwindInfoRVA: 0x3000}) {
		t.Fatalf("runtime function intervals = %+v", report.Functions)
	}
}

func TestInspectAMD64UnwindAcceptsUnusedSetFrameOperationInfo(t *testing.T) {
	t.Parallel()
	image := minimalAMD64PEWithUnwind()
	image[testUnwindXDataRawOffset+1] = 1
	image[testUnwindXDataRawOffset+2] = 1
	image[testUnwindXDataRawOffset+4] = 1
	image[testUnwindXDataRawOffset+5] = 3 | 3<<4
	report, err := InspectAMD64Unwind(image)
	if err != nil {
		t.Fatal(err)
	}
	if report.MaximumUnwindCodes != 1 {
		t.Fatalf("unwind report = %+v", report)
	}
}

func TestInspectAMD64UnwindAcceptsVersion2ShortEpilogDescriptor(t *testing.T) {
	t.Parallel()
	image := minimalAMD64PEWithUnwind()
	image[testUnwindXDataRawOffset] = 2
	image[testUnwindXDataRawOffset+2] = 1
	image[testUnwindXDataRawOffset+4] = 1
	image[testUnwindXDataRawOffset+5] = 6 | 1<<4
	report, err := InspectAMD64Unwind(image)
	if err != nil {
		t.Fatal(err)
	}
	if report.Version2Infos != 1 || report.MaximumUnwindCodes != 1 {
		t.Fatalf("unwind report = %+v", report)
	}
}

func TestInspectAMD64UnwindAcceptsVersion2LongEpilogDescriptor(t *testing.T) {
	t.Parallel()
	image := minimalAMD64PEWithUnwind()
	image[testUnwindXDataRawOffset] = 2
	image[testUnwindXDataRawOffset+2] = 2
	image[testUnwindXDataRawOffset+4] = 4
	image[testUnwindXDataRawOffset+5] = 6
	image[testUnwindXDataRawOffset+6] = 1
	report, err := InspectAMD64Unwind(image)
	if err != nil {
		t.Fatal(err)
	}
	if report.Version2Infos != 1 || report.MaximumUnwindCodes != 2 {
		t.Fatalf("unwind report = %+v", report)
	}
}

func TestInspectAMD64UnwindRejectsMalformedTables(t *testing.T) {
	t.Parallel()
	for _, testCase := range []struct {
		name string
		edit func([]byte)
		want string
	}{
		{name: "misaligned directory size", edit: func(image []byte) { binary.LittleEndian.PutUint32(image[testUnwindDirectoryOffset+4:], 13) }, want: "invalid RVA/size"},
		{name: "empty function", edit: func(image []byte) { binary.LittleEndian.PutUint32(image[testUnwindPDataRawOffset+4:], 0x1000) }, want: "invalid begin/end/info"},
		{name: "unmapped unwind info", edit: func(image []byte) { binary.LittleEndian.PutUint32(image[testUnwindPDataRawOffset+8:], 0x5000) }, want: "map UNWIND_INFO"},
		{name: "invalid version", edit: func(image []byte) { image[testUnwindXDataRawOffset] = 0 }, want: "invalid version/flags"},
		{name: "conflicting chain handler flags", edit: func(image []byte) { image[testUnwindXDataRawOffset] = 1 | 5<<3 }, want: "invalid version/flags"},
		{name: "truncated version-2 epilog", edit: func(image []byte) {
			image[testUnwindXDataRawOffset] = 2
			image[testUnwindXDataRawOffset+2] = 1
			image[testUnwindXDataRawOffset+4] = 4
			image[testUnwindXDataRawOffset+5] = 6
		}, want: "truncated version-2 epilog descriptor"},
		{name: "code slots exceed section", edit: func(image []byte) { image[testUnwindXDataRawOffset+2] = 255 }, want: "map complete UNWIND_INFO"},
	} {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			image := minimalAMD64PEWithUnwind()
			testCase.edit(image)
			if _, err := InspectAMD64Unwind(image); err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("InspectAMD64Unwind() error = %v", err)
			}
		})
	}
}

func minimalAMD64PEWithUnwind() []byte {
	image := make([]byte, 0xa00)
	copy(image, "MZ")
	binary.LittleEndian.PutUint32(image[0x3c:], testUnwindPEOffset)
	copy(image[testUnwindPEOffset:], "PE\x00\x00")
	coff := testUnwindPEOffset + 4
	binary.LittleEndian.PutUint16(image[coff:], machineAMD64)
	binary.LittleEndian.PutUint16(image[coff+2:], 3)
	binary.LittleEndian.PutUint16(image[coff+16:], 240)
	binary.LittleEndian.PutUint16(image[coff+18:], executableImage|largeAddressAware)

	optional := testUnwindOptionalOffset
	binary.LittleEndian.PutUint16(image[optional:], optionalHeader64Magic)
	binary.LittleEndian.PutUint32(image[optional+4:], 0x200)
	binary.LittleEndian.PutUint32(image[optional+8:], 0x400)
	binary.LittleEndian.PutUint32(image[optional+16:], 0x1000)
	binary.LittleEndian.PutUint32(image[optional+20:], 0x1000)
	binary.LittleEndian.PutUint64(image[optional+24:], 0x140000000)
	binary.LittleEndian.PutUint32(image[optional+32:], 0x1000)
	binary.LittleEndian.PutUint32(image[optional+36:], 0x200)
	binary.LittleEndian.PutUint16(image[optional+40:], 6)
	binary.LittleEndian.PutUint16(image[optional+48:], 6)
	binary.LittleEndian.PutUint32(image[optional+56:], 0x4000)
	binary.LittleEndian.PutUint32(image[optional+60:], 0x400)
	binary.LittleEndian.PutUint16(image[optional+68:], windowsConsoleSubsystem)
	binary.LittleEndian.PutUint64(image[optional+72:], 0x100000)
	binary.LittleEndian.PutUint64(image[optional+80:], 0x1000)
	binary.LittleEndian.PutUint64(image[optional+88:], 0x100000)
	binary.LittleEndian.PutUint64(image[optional+96:], 0x1000)
	binary.LittleEndian.PutUint32(image[optional+108:], 16)
	binary.LittleEndian.PutUint32(image[testUnwindDirectoryOffset:], 0x2000)
	binary.LittleEndian.PutUint32(image[testUnwindDirectoryOffset+4:], runtimeFunctionBytes)

	emitPESection(image, testUnwindSectionsOffset, ".text", 1, 0x1000, 0x200, testUnwindTextRawOffset, sectionContainsCode|sectionExecutable|sectionReadable)
	emitPESection(image, testUnwindSectionsOffset+40, ".pdata", runtimeFunctionBytes, 0x2000, 0x200, testUnwindPDataRawOffset, 0x40|sectionReadable)
	emitPESection(image, testUnwindSectionsOffset+80, ".rdata", unwindHeaderBytes, 0x3000, 0x200, testUnwindXDataRawOffset, 0x40|sectionReadable)
	image[testUnwindTextRawOffset] = 0xc3
	binary.LittleEndian.PutUint32(image[testUnwindPDataRawOffset:], 0x1000)
	binary.LittleEndian.PutUint32(image[testUnwindPDataRawOffset+4:], 0x1001)
	binary.LittleEndian.PutUint32(image[testUnwindPDataRawOffset+8:], 0x3000)
	image[testUnwindXDataRawOffset] = 1
	return image
}

func emitPESection(image []byte, offset int, name string, virtualSize, virtualAddress, rawSize, rawOffset, characteristics uint32) {
	copy(image[offset:offset+8], []byte(name))
	binary.LittleEndian.PutUint32(image[offset+8:], virtualSize)
	binary.LittleEndian.PutUint32(image[offset+12:], virtualAddress)
	binary.LittleEndian.PutUint32(image[offset+16:], rawSize)
	binary.LittleEndian.PutUint32(image[offset+20:], rawOffset)
	binary.LittleEndian.PutUint32(image[offset+36:], characteristics)
}
