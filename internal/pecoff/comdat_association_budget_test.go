package pecoff

import (
	"encoding/binary"
	"fmt"
	"strings"
	"testing"
)

func TestInspectObjectAcceptsCOMDATAssociationChain(t *testing.T) {
	for _, count := range []int{1, 128, maxObjectSections} {
		report, err := InspectObject(chainedCOMDATObject(count))
		if err != nil || len(report.Sections) != count || report.Sections[count-1].AssociativeSection != count-1 {
			t.Fatalf("COMDAT chain inspection: sections=%d err=%v", len(report.Sections), err)
		}
	}
}

func TestCOMDATAssociationSharedTailStillRejectsDisconnectedCycle(t *testing.T) {
	const count = 128
	object := chainedCOMDATObject(count)
	symbolStart := int(binary.LittleEndian.Uint32(object[8:]))
	// Most sections share a valid terminal. A disconnected 126 <-> 128
	// cycle must still be rejected, regardless of map traversal order.
	for index := 1; index < count; index++ {
		binary.LittleEndian.PutUint16(object[symbolStart+index*54+18+12:], 1)
	}
	binary.LittleEndian.PutUint16(object[symbolStart+125*54+18+12:], 128)
	binary.LittleEndian.PutUint16(object[symbolStart+127*54+18+12:], 126)
	if _, err := InspectObject(object); err == nil || !strings.Contains(err.Error(), "contains a cycle") {
		t.Fatalf("disconnected COMDAT cycle accepted: %v", err)
	}
}

func TestCOMDATAssociationSharedTailPreservesParents(t *testing.T) {
	const count = 128
	object := chainedCOMDATObject(count)
	symbolStart := int(binary.LittleEndian.Uint32(object[8:]))
	// Sections 65 onward converge on section 64, whose own chain reaches 1.
	for index := 64; index < count; index++ {
		binary.LittleEndian.PutUint16(object[symbolStart+index*54+18+12:], 64)
	}
	report, err := InspectObject(object)
	if err != nil {
		t.Fatal(err)
	}
	for index := 64; index < count; index++ {
		if report.Sections[index].AssociativeSection != 64 {
			t.Fatalf("section %d lost its direct parent", index+1)
		}
	}
}

func BenchmarkInspectObjectCOMDATAssociationChain(b *testing.B) {
	for _, count := range []int{2048, 4096} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			object := chainedCOMDATObject(count)
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				report, err := InspectObject(object)
				if err != nil || len(report.Sections) != count {
					b.Fatalf("COMDAT chain inspection: %v", err)
				}
			}
		})
	}
}

func chainedCOMDATObject(count int) []byte {
	rawStart := coffFileHeaderBytes + count*coffSectionHeaderBytes
	symbolStart := rawStart + count
	symbolEnd := symbolStart + count*54 // Definition, auxiliary, external key.
	object := make([]byte, symbolEnd+4)
	binary.LittleEndian.PutUint16(object, machineAMD64)
	binary.LittleEndian.PutUint16(object[2:], uint16(count))
	binary.LittleEndian.PutUint32(object[8:], uint32(symbolStart))
	binary.LittleEndian.PutUint32(object[12:], uint32(count*3))
	binary.LittleEndian.PutUint32(object[symbolEnd:], 4)
	for index := range count {
		name := ".rdata$x"
		flags := uint32(sectionInitializedData | sectionReadable | sectionLinkCOMDAT)
		selection, parent := uint8(coffCOMDATAssociative), uint16(index)
		if index == 0 {
			name, flags = ".text$mn", sectionContainsCode|sectionExecutable|sectionReadable|sectionLinkCOMDAT
			selection, parent = coffCOMDATAny, 0
		}
		header := object[coffFileHeaderBytes+index*coffSectionHeaderBytes:]
		copy(header, name)
		binary.LittleEndian.PutUint32(header[16:], 1)
		binary.LittleEndian.PutUint32(header[20:], uint32(rawStart+index))
		binary.LittleEndian.PutUint32(header[36:], flags)
		object[rawStart+index] = 0xc3
		definition := object[symbolStart+index*54:]
		copy(definition, name)
		binary.LittleEndian.PutUint16(definition[12:], uint16(index+1))
		definition[16], definition[17] = coffStorageClassStatic, 1
		binary.LittleEndian.PutUint32(definition[18:], 1)
		binary.LittleEndian.PutUint16(definition[18+12:], parent)
		definition[18+14] = selection
		copy(definition[36:], fmt.Sprintf("s%07d", index))
		binary.LittleEndian.PutUint16(definition[36+12:], uint16(index+1))
		definition[36+16] = coffStorageClassExternal
	}
	return object
}
