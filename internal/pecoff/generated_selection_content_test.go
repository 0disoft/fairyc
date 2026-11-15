package pecoff

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"reflect"
	"slices"
	"testing"
)

func TestGeneratedSelectionContentDigestAndIsolation(t *testing.T) {
	for _, size := range []int{64, 1 << 20} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			object, image, mapping := generatedSelectionLargeFixture(size)
			rawStart := coffFileHeaderBytes + coffSectionHeaderBytes
			for offset := 1; offset < size; offset++ {
				value := byte(offset*37 + 17)
				object[rawStart+offset], image[0x200+offset] = value, value
			}
			for offset := 4; offset < 36; offset++ {
				image[0x200+offset] ^= 0xff
			}
			runtime, log := minimalAMD64COFF(), []byte("link ok")
			inputs := [][]byte{object, runtime, image, mapping, log}
			before := make([][]byte, len(inputs))
			for i := range inputs {
				before[i] = slices.Clone(inputs[i])
			}
			expected := slices.Clone(object[coffFileHeaderBytes+coffSectionHeaderBytes : coffFileHeaderBytes+coffSectionHeaderBytes+size])
			clear(expected[4:36])
			wantHash := sha256.Sum256(expected)
			var previous GeneratedCOMDATSelection
			for attempt := range 2 {
				report, err := InspectGeneratedCOMDATSelection(object, runtime, image, mapping, log)
				if err != nil || report.Retained != 1 || len(report.Parents) != 1 {
					t.Fatalf("report=%+v error=%v", report, err)
				}
				parent := report.Parents[0]
				if parent.ComparedBytes != size-32 || parent.RelocatedBytes != 32 || parent.NonRelocatedSHA256 != wantHash {
					t.Fatalf("content receipt = %+v", parent)
				}
				if attempt > 0 && !reflect.DeepEqual(report, previous) {
					t.Fatal("repeated inspection changed the receipt")
				}
				previous = report
			}
			for i := range inputs {
				if !bytes.Equal(inputs[i], before[i]) {
					t.Fatalf("caller input %d was mutated", i)
				}
			}
		})
	}
}

func TestGeneratedSelectionContentPreservesFailureOrderAndPartialReceipt(t *testing.T) {
	for _, tc := range []struct {
		name string
		edit func([]byte, []byte)
		want string
		kept int
	}{
		{"overlap before differing bytes", func(object, image []byte) {
			offset := binary.LittleEndian.Uint32(object[coffFileHeaderBytes+24:])
			binary.LittleEndian.PutUint32(object[offset+coffRelocationBytes:], 4)
			image[0x200] ^= 1
		}, "unit.obj:s0000000 has overlapping relocation fields", 0},
		{"unsupported type before overlap", func(object, _ []byte) {
			offset := binary.LittleEndian.Uint32(object[coffFileHeaderBytes+24:])
			binary.LittleEndian.PutUint32(object[offset+coffRelocationBytes:], 4)
			binary.LittleEndian.PutUint16(object[offset+7*coffRelocationBytes+8:], ObjectRelocationAMD64SRel32)
		}, "unit.obj:s0000000 has unmeasured relocation 0xe", 0},
		{"all bytes relocated", func(object, _ []byte) {
			offset := binary.LittleEndian.Uint32(object[coffFileHeaderBytes+24:])
			for i := range 8 {
				binary.LittleEndian.PutUint32(object[offset+uint32(i*coffRelocationBytes):], uint32(i*8))
				binary.LittleEndian.PutUint16(object[offset+uint32(i*coffRelocationBytes)+8:], ObjectRelocationAMD64Addr64)
			}
		}, "unit.obj:s0000000 non-relocated COMDAT bytes differ or are empty", 0},
		{"later failure keeps earlier receipt", func(_ []byte, image []byte) {
			image[0x200+64+63] ^= 1
		}, "unit.obj:s0000001 non-relocated COMDAT bytes differ or are empty", 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			object, image, mapping := generatedSelectionFixture(3)
			tc.edit(object, image)
			beforeObject, beforeImage := slices.Clone(object), slices.Clone(image)
			report, err := InspectGeneratedCOMDATSelection(object, minimalAMD64COFF(), image, mapping, []byte("link ok"))
			if err == nil || err.Error() != tc.want || report.Retained != tc.kept || len(report.Parents) != tc.kept {
				t.Fatalf("report=%+v error=%v, want retained=%d error=%q", report, err, tc.kept, tc.want)
			}
			if !bytes.Equal(object, beforeObject) || !bytes.Equal(image, beforeImage) {
				t.Fatal("failed inspection changed its inputs")
			}
		})
	}
}

func TestGeneratedSelectionContentKeepsUnitReceiptOnRuntimeFailure(t *testing.T) {
	object, image, mapping := generatedSelectionFixture(3)
	beforeObject, beforeImage := slices.Clone(object), slices.Clone(image)
	// Reuse the same immutable object as both inputs. Its map entries belong to
	// unit.obj, so runtime ownership fails after all unit receipts are collected.
	report, err := InspectGeneratedCOMDATSelection(object, object, image, mapping, []byte("link ok"))
	want := "runtime.obj:s0000000 requires exactly one owned map or discard record"
	if err == nil || err.Error() != want || report.Retained != 3 || len(report.Parents) != 3 {
		t.Fatalf("report=%+v error=%v, want retained=3 error=%q", report, err, want)
	}
	if !bytes.Equal(object, beforeObject) || !bytes.Equal(image, beforeImage) {
		t.Fatal("failed runtime inspection changed aliased inputs")
	}
}

func BenchmarkGeneratedSelectionLargeSection(b *testing.B) {
	for _, size := range []int{64, 1 << 20} {
		b.Run(fmt.Sprint(size), func(b *testing.B) {
			object, image, mapping := generatedSelectionLargeFixture(size)
			runtime, log := minimalAMD64COFF(), []byte("link ok")
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				report, err := InspectGeneratedCOMDATSelection(object, runtime, image, mapping, log)
				if err != nil || report.Retained != 1 {
					b.Fatalf("selection: %+v / %v", report, err)
				}
			}
		})
	}
}

func generatedSelectionLargeFixture(size int) (object, image, mapping []byte) {
	object, image, mapping = generatedSelectionFixture(1)
	if size == 64 {
		return
	}
	delta := size - 64
	relocationStart := int(binary.LittleEndian.Uint32(object[coffFileHeaderBytes+24:]))
	symbolStart := int(binary.LittleEndian.Uint32(object[8:]))
	grown := make([]byte, len(object)+delta)
	copy(grown, object[:relocationStart])
	copy(grown[relocationStart+delta:], object[relocationStart:])
	object = grown
	binary.LittleEndian.PutUint32(object[8:], uint32(symbolStart+delta))
	binary.LittleEndian.PutUint32(object[coffFileHeaderBytes+16:], uint32(size))
	binary.LittleEndian.PutUint32(object[coffFileHeaderBytes+24:], uint32(relocationStart+delta))
	binary.LittleEndian.PutUint32(object[symbolStart+delta+18:], uint32(size))
	rawSize := (size + 0x1ff) &^ 0x1ff
	grown = make([]byte, 0x200+rawSize)
	copy(grown, image)
	image = grown
	binary.LittleEndian.PutUint32(image[testSectionHeaderOffset+8:], uint32(size))
	binary.LittleEndian.PutUint32(image[testSectionHeaderOffset+16:], uint32(rawSize))
	binary.LittleEndian.PutUint32(image[testOptionalHeaderOffset+56:], uint32((0x1000+size+0xfff)&^0xfff))
	return
}
