package pecoff

import (
	"encoding/binary"
	"strings"
	"testing"
)

const (
	testPEOffset               = 0x80
	testOptionalHeaderOffset   = testPEOffset + 4 + 20
	testSectionHeaderOffset    = testOptionalHeaderOffset + 240
	testSectionCharacteristics = testSectionHeaderOffset + 36
)

func TestInspectExecutableAcceptsClosedAMD64PE32PlusImage(t *testing.T) {
	t.Parallel()
	report, err := InspectExecutable(minimalAMD64PE())
	if err != nil {
		t.Fatal(err)
	}
	if report.Machine != machineAMD64 || report.ImageBase != 0x140000000 || report.EntryPointRVA != 0x1000 || report.Subsystem != windowsConsoleSubsystem {
		t.Fatalf("PE report identity = %+v", report)
	}
	if len(report.Sections) != 1 || report.Sections[0].Name != ".text" || report.ImportDirectoryPresent || report.ExceptionDirectoryPresent {
		t.Fatalf("PE report sections = %+v", report)
	}
}

func TestInspectExecutableRejectsMalformedIdentityAndRanges(t *testing.T) {
	t.Parallel()
	for _, testCase := range []struct {
		name string
		edit func([]byte)
		want string
	}{
		{name: "missing MZ", edit: func(image []byte) { image[0] = 0 }, want: "MZ signature"},
		{name: "x86 machine", edit: func(image []byte) { binary.LittleEndian.PutUint16(image[testPEOffset+4:], 0x014c) }, want: "want AMD64"},
		{name: "GUI subsystem", edit: func(image []byte) { binary.LittleEndian.PutUint16(image[testOptionalHeaderOffset+68:], 2) }, want: "want Windows console"},
		{name: "unaligned image base", edit: func(image []byte) { binary.LittleEndian.PutUint64(image[testOptionalHeaderOffset+24:], 0x140000001) }, want: "64 KiB aligned"},
		{name: "entry not executable", edit: func(image []byte) {
			binary.LittleEndian.PutUint32(image[testSectionCharacteristics:], sectionContainsCode|sectionReadable)
		}, want: ".text section is not readable executable code"},
		{name: "misaligned raw size", edit: func(image []byte) {
			binary.LittleEndian.PutUint32(image[testSectionHeaderOffset+16:], 1)
		}, want: "invalid raw range"},
		{name: "raw range overlaps headers", edit: func(image []byte) {
			binary.LittleEndian.PutUint32(image[testSectionHeaderOffset+20:], 0)
		}, want: "invalid raw range"},
		{name: "raw range outside file", edit: func(image []byte) {
			binary.LittleEndian.PutUint32(image[testSectionHeaderOffset+20:], uint32(len(image)))
		}, want: "invalid raw range"},
	} {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			image := minimalAMD64PE()
			testCase.edit(image)
			if _, err := InspectExecutable(image); err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("InspectExecutable() error = %v", err)
			}
		})
	}
}

func TestInspectExecutableRejectsTruncatedImage(t *testing.T) {
	t.Parallel()
	if _, err := InspectExecutable([]byte("MZ")); err == nil || !strings.Contains(err.Error(), "invalid size") {
		t.Fatalf("InspectExecutable() error = %v", err)
	}
}

func minimalAMD64PE() []byte {
	image := make([]byte, 0x400)
	copy(image, "MZ")
	binary.LittleEndian.PutUint32(image[0x3c:], testPEOffset)
	copy(image[testPEOffset:], "PE\x00\x00")
	coff := testPEOffset + 4
	binary.LittleEndian.PutUint16(image[coff:], machineAMD64)
	binary.LittleEndian.PutUint16(image[coff+2:], 1)
	binary.LittleEndian.PutUint16(image[coff+16:], 240)
	binary.LittleEndian.PutUint16(image[coff+18:], executableImage|largeAddressAware)

	optional := testOptionalHeaderOffset
	binary.LittleEndian.PutUint16(image[optional:], optionalHeader64Magic)
	binary.LittleEndian.PutUint32(image[optional+4:], 0x200)
	binary.LittleEndian.PutUint32(image[optional+16:], 0x1000)
	binary.LittleEndian.PutUint32(image[optional+20:], 0x1000)
	binary.LittleEndian.PutUint64(image[optional+24:], 0x140000000)
	binary.LittleEndian.PutUint32(image[optional+32:], 0x1000)
	binary.LittleEndian.PutUint32(image[optional+36:], 0x200)
	binary.LittleEndian.PutUint16(image[optional+40:], 6)
	binary.LittleEndian.PutUint16(image[optional+48:], 6)
	binary.LittleEndian.PutUint32(image[optional+56:], 0x2000)
	binary.LittleEndian.PutUint32(image[optional+60:], 0x200)
	binary.LittleEndian.PutUint16(image[optional+68:], windowsConsoleSubsystem)
	binary.LittleEndian.PutUint64(image[optional+72:], 0x100000)
	binary.LittleEndian.PutUint64(image[optional+80:], 0x1000)
	binary.LittleEndian.PutUint64(image[optional+88:], 0x100000)
	binary.LittleEndian.PutUint64(image[optional+96:], 0x1000)
	binary.LittleEndian.PutUint32(image[optional+108:], 16)

	section := testSectionHeaderOffset
	copy(image[section:], ".text")
	binary.LittleEndian.PutUint32(image[section+8:], 1)
	binary.LittleEndian.PutUint32(image[section+12:], 0x1000)
	binary.LittleEndian.PutUint32(image[section+16:], 0x200)
	binary.LittleEndian.PutUint32(image[section+20:], 0x200)
	binary.LittleEndian.PutUint32(image[section+36:], sectionContainsCode|sectionExecutable|sectionReadable)
	image[0x200] = 0xc3
	return image
}
