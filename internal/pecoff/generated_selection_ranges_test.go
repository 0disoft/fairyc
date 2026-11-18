package pecoff

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"slices"
	"strings"
	"testing"
)

// Model function-level COMDATs coalesced into one PE code section. Relocated
// fields intentionally differ from the object; all other bytes must agree.
func generatedSelectionFixture(count int) (object, image, linkMap []byte) {
	const size, relocations = 64, 8
	rawStart := coffFileHeaderBytes + count*coffSectionHeaderBytes
	relStart := rawStart + count*size
	symbolStart := relStart + count*relocations*coffRelocationBytes
	object = make([]byte, symbolStart+count*54+4)
	binary.LittleEndian.PutUint16(object, machineAMD64)
	binary.LittleEndian.PutUint16(object[2:], uint16(count))
	binary.LittleEndian.PutUint32(object[8:], uint32(symbolStart))
	binary.LittleEndian.PutUint32(object[12:], uint32(count*3))
	binary.LittleEndian.PutUint32(object[len(object)-4:], 4)
	rawSize := (count*size + 0x1ff) &^ 0x1ff
	image = make([]byte, 0x200+rawSize)
	copy(image, minimalAMD64PE())
	binary.LittleEndian.PutUint32(image[testSectionHeaderOffset+8:], uint32(count*size))
	binary.LittleEndian.PutUint32(image[testSectionHeaderOffset+16:], uint32(rawSize))
	binary.LittleEndian.PutUint32(image[testOptionalHeaderOffset+56:], uint32((0x1000+count*size+0xfff)&^0xfff))
	var mapping bytes.Buffer
	for i := range count {
		header := object[coffFileHeaderBytes+i*coffSectionHeaderBytes:]
		copy(header, ".text$mn")
		binary.LittleEndian.PutUint32(header[16:], size)
		binary.LittleEndian.PutUint32(header[20:], uint32(rawStart+i*size))
		binary.LittleEndian.PutUint32(header[24:], uint32(relStart+i*relocations*coffRelocationBytes))
		binary.LittleEndian.PutUint16(header[32:], relocations)
		binary.LittleEndian.PutUint32(header[36:], sectionContainsCode|sectionExecutable|sectionReadable|sectionLinkCOMDAT)
		definition := object[symbolStart+i*54:]
		copy(definition, ".text$mn")
		binary.LittleEndian.PutUint16(definition[12:], uint16(i+1))
		definition[16], definition[17] = coffStorageClassStatic, 1
		binary.LittleEndian.PutUint32(definition[18:], size)
		binary.LittleEndian.PutUint16(definition[22:], relocations)
		definition[32] = ObjectCOMDATAny
		name := fmt.Sprintf("s%07d", i)
		copy(definition[36:], name)
		binary.LittleEndian.PutUint16(definition[48:], uint16(i+1))
		definition[52] = coffStorageClassExternal
		object[rawStart+i*size] = 0xc3
		image[0x200+i*size] = 0xc3
		for j := range relocations {
			rel := object[relStart+(i*relocations+j)*coffRelocationBytes:]
			binary.LittleEndian.PutUint32(rel, uint32(4+j*4))
			binary.LittleEndian.PutUint32(rel[4:], uint32(i*3+2))
			binary.LittleEndian.PutUint16(rel[8:], ObjectRelocationAMD64Rel32)
			binary.LittleEndian.PutUint32(image[0x200+i*size+4+j*4:], 0x12345678)
		}
		fmt.Fprintf(&mapping, "0001:%08x %s %016x f unit.obj\n", i*size, name, 0x140001000+uint64(i*size))
	}
	return object, image, mapping.Bytes()
}

func TestGeneratedSelectionSectionRelocationRanges(t *testing.T) {
	for _, count := range []int{1, 3, 128} {
		object, image, mapping := generatedSelectionFixture(count)
		beforeObject, beforeImage := slices.Clone(object), slices.Clone(image)
		report, err := InspectGeneratedCOMDATSelection(object, minimalAMD64COFF(), image, mapping, []byte("link ok"))
		if err != nil || report.Retained != count || len(report.Parents) != count {
			t.Fatalf("%d sections: report=%+v error=%v", count, report, err)
		}
		for i, parent := range report.Parents {
			if parent.SectionIndex != i+1 || parent.RelocatedBytes != 32 || parent.ComparedBytes != 32 {
				t.Fatalf("section %d: %+v", i+1, parent)
			}
		}
		if !bytes.Equal(object, beforeObject) || !bytes.Equal(image, beforeImage) {
			t.Fatal("relocation masking modified caller-owned bytes")
		}
	}
}

func TestGeneratedSelectionRangesAdvancePastSkippedSections(t *testing.T) {
	object, image, mapping := generatedSelectionFixture(5)
	symbolStart := int(binary.LittleEndian.Uint32(object[8:]))
	// A non-COMDAT, an associative child and a discarded parent precede the
	// retained parents. All have relocations which still advance the cursor.
	flags := object[coffFileHeaderBytes+36:]
	binary.LittleEndian.PutUint32(flags, binary.LittleEndian.Uint32(flags)&^sectionLinkCOMDAT)
	object[symbolStart+32] = 0
	object[symbolStart+54+32] = ObjectCOMDATAssociative
	binary.LittleEndian.PutUint16(object[symbolStart+54+30:], 3)
	var kept []string
	for _, line := range strings.Split(string(mapping), "\n") {
		if !strings.Contains(line, "s0000002") {
			kept = append(kept, line)
		}
	}
	// The next retained section has no relocations; the last still has eight.
	binary.LittleEndian.PutUint16(object[coffFileHeaderBytes+3*coffSectionHeaderBytes+32:], 0)
	binary.LittleEndian.PutUint16(object[symbolStart+3*54+22:], 0)
	clear(image[0x200+3*64+4 : 0x200+3*64+36])
	ownedMap := []byte(strings.Join(kept, "\n"))
	report, err := InspectGeneratedCOMDATSelection(object, minimalAMD64COFF(), image,
		ownedMap, []byte("ICF note: \xff\x00\nDiscarded s0000002 from unit.obj\n"))
	if err != nil || report.Retained != 2 || report.Discarded != 1 || report.AssociativeSectionInventory != 1 {
		t.Fatalf("skipped sections: %+v / %v", report, err)
	}
	if report.Parents[1].RelocatedBytes != 0 || report.Parents[2].RelocatedBytes != 32 {
		t.Fatalf("wrong relocation ranges after skipped/empty sections: %+v", report.Parents)
	}
	for _, invalid := range []string{"\x00", "\xff"} {
		log := []byte("Discarded s0000002" + invalid + " from unit.obj\n")
		if _, err := InspectGeneratedCOMDATSelection(object, minimalAMD64COFF(), image, ownedMap, log); err == nil || err.Error() != "invalid generated COMDAT discard symbol" {
			t.Errorf("invalid discard symbol byte %q: %v", invalid, err)
		}
	}
}

func TestGeneratedSelectionRangesPreserveFailures(t *testing.T) {
	for _, test := range []struct {
		name string
		edit func(object, image []byte)
		want string
	}{
		{"overlap", func(object, _ []byte) {
			offset := binary.LittleEndian.Uint32(object[coffFileHeaderBytes+24:])
			binary.LittleEndian.PutUint32(object[offset+coffRelocationBytes:], 4)
		}, "unit.obj:s0000000 has overlapping relocation fields"},
		{"range", func(object, _ []byte) {
			offset := binary.LittleEndian.Uint32(object[coffFileHeaderBytes+24:])
			binary.LittleEndian.PutUint32(object[offset:], 61)
		}, "unit.obj:s0000000 relocation exceeds COMDAT bytes"},
		{"type", func(object, _ []byte) {
			offset := binary.LittleEndian.Uint32(object[coffFileHeaderBytes+24:])
			binary.LittleEndian.PutUint16(object[offset+8:], ObjectRelocationAMD64SRel32)
		}, "unit.obj:s0000000 has unmeasured relocation 0xe"},
		{"earlier section wins", func(object, image []byte) {
			image[0x200+63] = 1
			offset := binary.LittleEndian.Uint32(object[coffFileHeaderBytes+coffSectionHeaderBytes+24:])
			binary.LittleEndian.PutUint16(object[offset+8:], ObjectRelocationAMD64SRel32)
		}, "unit.obj:s0000000 non-relocated COMDAT bytes differ or are empty"},
	} {
		t.Run(test.name, func(t *testing.T) {
			object, image, mapping := generatedSelectionFixture(3)
			test.edit(object, image)
			_, err := InspectGeneratedCOMDATSelection(object, minimalAMD64COFF(), image, mapping, []byte("link ok"))
			if err == nil || err.Error() != test.want {
				t.Fatalf("error=%v, want %q", err, test.want)
			}
		})
	}
}

func TestGeneratedSelectionRangesMaskWidthsAtSectionEnd(t *testing.T) {
	for _, kind := range []uint16{ObjectRelocationAMD64Section, ObjectRelocationAMD64Rel32, ObjectRelocationAMD64Addr64} {
		object, image, mapping := generatedSelectionFixture(3)
		width := map[uint16]int{ObjectRelocationAMD64Section: 2, ObjectRelocationAMD64Rel32: 4, ObjectRelocationAMD64Addr64: 8}[kind]
		for i := range 3 {
			offset := binary.LittleEndian.Uint32(object[coffFileHeaderBytes+i*coffSectionHeaderBytes+24:]) + 7*coffRelocationBytes
			binary.LittleEndian.PutUint32(object[offset:], uint32(64-width))
			binary.LittleEndian.PutUint16(object[offset+8:], kind)
			clear(image[0x200+i*64+32 : 0x200+i*64+36])
			for j := range width {
				image[0x200+(i+1)*64-width+j] = 0xff
			}
		}
		report, err := InspectGeneratedCOMDATSelection(object, minimalAMD64COFF(), image, mapping, []byte("link ok"))
		if err != nil || report.Retained != 3 {
			t.Fatalf("width %d: %+v / %v", width, report, err)
		}
		for _, parent := range report.Parents {
			if parent.RelocatedBytes != 28+width {
				t.Fatalf("width %d: %+v", width, parent)
			}
		}
	}
}

func BenchmarkGeneratedSelectionSectionRanges(b *testing.B) {
	for _, count := range []int{1024, 4096} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			object, image, mapping := generatedSelectionFixture(count)
			runtime, log := minimalAMD64COFF(), []byte("link ok")
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				report, err := InspectGeneratedCOMDATSelection(object, runtime, image, mapping, log)
				if err != nil || report.Retained != count {
					b.Fatalf("selection: retained=%d error=%v", report.Retained, err)
				}
			}
		})
	}
}
