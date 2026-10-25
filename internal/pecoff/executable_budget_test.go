package pecoff

import (
	"encoding/binary"
	"fmt"
	"strings"
	"testing"
)

func TestInspectExecutableSectionBudgetPrecedesParserAllocation(t *testing.T) {
	for _, count := range []uint16{0, maxSections + 1, 65535} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			image := minimalAMD64PE()[:testOptionalHeaderOffset]
			binary.LittleEndian.PutUint16(image[testPEOffset+6:], count)
			if _, err := InspectExecutable(image); err == nil || !strings.Contains(err.Error(), fmt.Sprintf("invalid section count %d", count)) {
				t.Fatalf("section budget did not precede parsing: %v", err)
			}
		})
	}
}

func TestInspectExecutableSectionBudgetPreservesHeaderValidation(t *testing.T) {
	for _, offset := range []uint32{0, 63, 0xffffffff} {
		t.Run(fmt.Sprint(offset), func(t *testing.T) {
			image := minimalAMD64PE()
			binary.LittleEndian.PutUint32(image[0x3c:], offset)
			if _, err := InspectExecutable(image); err == nil || !strings.Contains(err.Error(), "parse PE executable") {
				t.Fatalf("malformed header did not reach parser validation: %v", err)
			}
		})
	}
	image := minimalAMD64PE()[:testOptionalHeaderOffset]
	binary.LittleEndian.PutUint16(image[testPEOffset+6:], maxSections)
	if _, err := InspectExecutable(image); err == nil || !strings.Contains(err.Error(), "parse PE executable") {
		t.Fatalf("section count at the limit was rejected by preflight: %v", err)
	}
}

func BenchmarkInspectExecutableOverSectionBudget(b *testing.B) {
	image := executableWithAliasedRelocations(256, 16384)
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if _, err := InspectExecutable(image); err == nil || !strings.Contains(err.Error(), "invalid section count 256") {
			b.Fatal(err)
		}
	}
}

func executableWithAliasedRelocations(sections, relocations int) []byte {
	headerEnd := testSectionHeaderOffset + sections*coffSectionHeaderBytes
	image := make([]byte, headerEnd+relocations*coffRelocationBytes)
	copy(image, minimalAMD64PE()[:testSectionHeaderOffset])
	binary.LittleEndian.PutUint16(image[testPEOffset+6:], uint16(sections))
	for index := range sections {
		header := image[testSectionHeaderOffset+index*coffSectionHeaderBytes:]
		copy(header, ".text")
		binary.LittleEndian.PutUint32(header[24:], uint32(headerEnd))
		binary.LittleEndian.PutUint16(header[32:], uint16(relocations))
	}
	return image
}
