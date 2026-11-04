package pecoff

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"slices"
	"strings"
	"testing"
)

// Each generated function owns a code COMDAT and direct .xdata/.pdata children.
// The final image has sorted runtime-function entries and valid handler infos.
func generatedUnwindRangesFixture(functions int) (object, image, mapping []byte) {
	object, _, _ = generatedSelectionFixture(functions * 3)
	symbolStart := int(binary.LittleEndian.Uint32(object[8:]))
	align := func(n, alignment int) int { return (n + alignment - 1) &^ (alignment - 1) }
	textSize, pdataSize, xdataSize := align(functions*64, 0x200), align(functions*12, 0x200), align(functions*64, 0x200)
	pdataRaw, xdataRaw := 0x400+textSize, 0x400+textSize+pdataSize
	pdataRVA := uint32(0x1000 + align(textSize, 0x1000))
	xdataRVA := pdataRVA + uint32(align(pdataSize, 0x1000))
	image = make([]byte, xdataRaw+xdataSize)
	copy(image, minimalAMD64PEWithUnwind()[:0x400])
	emitPESection(image, testUnwindSectionsOffset, ".text", uint32(functions*64), 0x1000, uint32(textSize), 0x400, sectionContainsCode|sectionExecutable|sectionReadable)
	emitPESection(image, testUnwindSectionsOffset+40, ".pdata", uint32(functions*12), pdataRVA, uint32(pdataSize), uint32(pdataRaw), sectionInitializedData|sectionReadable)
	emitPESection(image, testUnwindSectionsOffset+80, ".xdata", uint32(functions*64), xdataRVA, uint32(xdataSize), uint32(xdataRaw), sectionInitializedData|sectionReadable)
	binary.LittleEndian.PutUint32(image[testUnwindDirectoryOffset:], pdataRVA)
	binary.LittleEndian.PutUint32(image[testUnwindDirectoryOffset+4:], uint32(functions*12))
	binary.LittleEndian.PutUint32(image[testUnwindOptionalOffset+56:], xdataRVA+uint32(align(xdataSize, 0x1000)))
	var linkMap bytes.Buffer
	for i := range functions {
		codeRVA, unwindRVA := uint32(0x1000+i*64), xdataRVA+uint32(i*64)
		for kind := range 3 {
			index := i*3 + kind
			header := object[coffFileHeaderBytes+index*coffSectionHeaderBytes:]
			raw := object[binary.LittleEndian.Uint32(header[20:]):][:64]
			relocations := object[binary.LittleEndian.Uint32(header[24:]):]
			definition := object[symbolStart+index*54:]
			rva := codeRVA
			if kind == 0 {
				copy(image[0x400+i*64:], raw)
			} else {
				clear(raw)
				name, count, size := ".xdata", 1, 64
				rva = unwindRVA
				if kind == 2 {
					name, count, size, rva = ".pdata", 3, 12, pdataRVA+uint32(i*12)
				}
				clear(header[:8])
				copy(header, name)
				binary.LittleEndian.PutUint32(header[16:], uint32(size))
				binary.LittleEndian.PutUint16(header[32:], uint16(count))
				binary.LittleEndian.PutUint32(header[36:], sectionInitializedData|sectionReadable|sectionLinkCOMDAT)
				clear(definition[:8])
				copy(definition, name)
				binary.LittleEndian.PutUint32(definition[18:], uint32(size))
				binary.LittleEndian.PutUint16(definition[22:], uint16(count))
				binary.LittleEndian.PutUint16(definition[30:], uint16(i*3+1))
				definition[32] = ObjectCOMDATAssociative
				for j := range count {
					rel := relocations[j*coffRelocationBytes:]
					at, target := uint32(j*4), uint32(i*9+2)
					if kind == 1 {
						at = 4
					} else if j == 2 {
						target += 3
					}
					binary.LittleEndian.PutUint32(rel, at)
					binary.LittleEndian.PutUint32(rel[4:], target)
					binary.LittleEndian.PutUint16(rel[8:], ObjectRelocationAMD64Addr32NB)
				}
				if kind == 1 {
					raw[0] = 1 | unwindFlagExceptionHandler<<3
					copy(image[xdataRaw+i*64:], raw)
					binary.LittleEndian.PutUint32(image[xdataRaw+i*64+4:], codeRVA)
				} else {
					binary.LittleEndian.PutUint32(raw[4:], 1)
					entry := image[pdataRaw+i*12:]
					binary.LittleEndian.PutUint32(entry, codeRVA)
					binary.LittleEndian.PutUint32(entry[4:], codeRVA+1)
					binary.LittleEndian.PutUint32(entry[8:], unwindRVA)
				}
			}
			fmt.Fprintf(&linkMap, "0001:%08x s%07d %016x f unit.obj\n", index*64, index, 0x140000000+uint64(rva))
		}
	}
	return object, image, linkMap.Bytes()
}

func TestGeneratedUnwindSectionRanges(t *testing.T) {
	for _, count := range []int{1, 3, 128} {
		object, image, mapping := generatedUnwindRangesFixture(count)
		beforeObject, beforeImage := slices.Clone(object), slices.Clone(image)
		report, err := InspectGeneratedUnwindSelection(object, minimalAMD64COFF(), image, mapping, []byte("link ok"))
		if err != nil || report.Retained != 2*count || report.RelocationFields != 4*count || report.MatchedPDataEntries != count || report.MatchedXDataBytes != 64*count {
			t.Fatalf("functions=%d: report=%+v error=%v", count, report, err)
		}
		for i, child := range report.Children {
			if child.ParentSectionIndex != i/2*3+1 || child.SectionIndex != i/2*3+i%2+2 || child.Relocations != 1+i%2*2 {
				t.Fatalf("child %d: %+v", i, child)
			}
		}
		if !bytes.Equal(object, beforeObject) || !bytes.Equal(image, beforeImage) {
			t.Fatal("unwind relocation patching modified caller-owned bytes")
		}
	}
}

func TestGeneratedUnwindRangesSkipDiscardedAndEmptyChildren(t *testing.T) {
	object, image, mapping := generatedUnwindRangesFixture(3)
	var kept []string
	log := ""
	for _, line := range strings.Split(string(mapping), "\n") {
		fields := strings.Fields(line)
		if len(fields) > 1 && fields[1] <= "s0000002" {
			log += "Discarded " + fields[1] + " from unit.obj\n"
		} else {
			kept = append(kept, line)
		}
	}
	report, err := InspectGeneratedUnwindSelection(object, minimalAMD64COFF(), image, []byte(strings.Join(kept, "\n")), []byte(log))
	if err != nil || report.Discarded != 2 || report.Retained != 4 || report.RelocationFields != 8 || report.MatchedPDataEntries != 2 {
		t.Fatalf("discarded children: %+v / %v", report, err)
	}
	// A valid header without a handler needs no relocations. Later .pdata and
	// function groups must still consume their own ranges, including all code.
	object, image, mapping = generatedUnwindRangesFixture(3)
	header := object[coffFileHeaderBytes+coffSectionHeaderBytes:]
	binary.LittleEndian.PutUint16(header[32:], 0)
	symbolStart := binary.LittleEndian.Uint32(object[8:])
	binary.LittleEndian.PutUint16(object[symbolStart+54+22:], 0)
	object[binary.LittleEndian.Uint32(header[20:])] = 1
	xdataRaw := binary.LittleEndian.Uint32(image[testUnwindSectionsOffset+80+20:])
	image[xdataRaw] = 1
	clear(image[xdataRaw+4 : xdataRaw+8])
	report, err = InspectGeneratedUnwindSelection(object, minimalAMD64COFF(), image, mapping, []byte("link ok"))
	if err != nil || report.Retained != 6 || report.RelocationFields != 11 || report.Children[0].Relocations != 0 {
		t.Fatalf("empty relocation range: %+v / %v", report, err)
	}
}

func TestGeneratedUnwindRangesPreserveFailures(t *testing.T) {
	for _, test := range []struct {
		name string
		edit func(object, image, header, rel []byte)
		want string
	}{
		{"unaligned", func(_, _, _, rel []byte) {
			binary.LittleEndian.PutUint32(rel[2*coffRelocationBytes:], 9)
		}, "unit.obj:s0000002 has an invalid or unmeasured unwind relocation"},
		{"out of range", func(_, _, _, rel []byte) {
			binary.LittleEndian.PutUint32(rel[2*coffRelocationBytes:], 12)
		}, "unit.obj:s0000002 has an invalid or unmeasured unwind relocation"},
		{"duplicate", func(_, _, _, rel []byte) {
			binary.LittleEndian.PutUint32(rel[coffRelocationBytes:], 0)
		}, "unit.obj:s0000002 has an invalid or unmeasured unwind relocation"},
		{"missing field", func(object, _, header, _ []byte) {
			binary.LittleEndian.PutUint16(header[32:], 2)
			start := binary.LittleEndian.Uint32(object[8:])
			binary.LittleEndian.PutUint16(object[start+2*54+22:], 2)
		}, "generated .pdata lacks three independent RVA relocations"},
		{"overflow", func(object, _, header, _ []byte) {
			at := binary.LittleEndian.Uint32(header[20:])
			binary.LittleEndian.PutUint32(object[at+4:], 0xffffffff)
		}, "generated unwind RVA overflows"},
		{"wrong parent", func(_, _, _, rel []byte) {
			binary.LittleEndian.PutUint32(rel[4:], 5)
		}, "unit.obj:s0000002 code relocation does not name its associated parent"},
		{"earlier child wins", func(object, image, _, _ []byte) {
			xdataRaw := binary.LittleEndian.Uint32(image[testUnwindSectionsOffset+80+20:])
			image[xdataRaw+63] = 1
			header := object[coffFileHeaderBytes+4*coffSectionHeaderBytes:]
			offset := binary.LittleEndian.Uint32(header[24:])
			binary.LittleEndian.PutUint16(object[offset+8:], ObjectRelocationAMD64SRel32)
		}, "unit.obj:s0000001 final unwind bytes differ"},
	} {
		t.Run(test.name, func(t *testing.T) {
			object, image, mapping := generatedUnwindRangesFixture(3)
			header := object[coffFileHeaderBytes+2*coffSectionHeaderBytes:]
			rel := object[binary.LittleEndian.Uint32(header[24:]):]
			test.edit(object, image, header, rel)
			_, err := InspectGeneratedUnwindSelection(object, minimalAMD64COFF(), image, mapping, []byte("link ok"))
			if err == nil || err.Error() != test.want {
				t.Fatalf("error=%v, want %q", err, test.want)
			}
		})
	}
}

func BenchmarkGeneratedUnwindSectionRanges(b *testing.B) {
	for _, count := range []int{256, 4096} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			object, image, mapping := generatedUnwindRangesFixture(count)
			runtime, log := minimalAMD64COFF(), []byte("link ok")
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				report, err := InspectGeneratedUnwindSelection(object, runtime, image, mapping, log)
				if err != nil || report.MatchedPDataEntries != count {
					b.Fatalf("unwind inspection: %v", err)
				}
			}
		})
	}
}
