package pecoff

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestArchiveAcceptsIndividuallyBoundedObjectNames(t *testing.T) {
	archive := sharedObjectNameArchive(8, 513, 64<<10)
	raw, err := parseRawArchiveMembers(archive)
	if err != nil {
		t.Fatal(err)
	}
	for _, member := range raw[2:] {
		if err := VerifyObjectParserBudget(member.data); err != nil {
			t.Fatalf("individual member exceeds its existing budget: %v", err)
		}
	}
}

func TestInspectArchiveRejectsCumulativeObjectNames(t *testing.T) {
	if _, err := InspectArchive(sharedObjectNameArchive(8, 513, 64<<10)); !errors.Is(err, ErrNameBudgetExceeded) {
		t.Fatalf("archive bypassed cumulative name budget: %v", err)
	}
}

func TestArchiveObjectNameBudgetExactBoundary(t *testing.T) {
	archive := sharedObjectNameArchive(2, 513, 64<<10)
	raw, err := parseRawArchiveMembers(archive)
	if err != nil {
		t.Fatal(err)
	}
	last := raw[len(raw)-1].data
	for _, skip := range []uint32{25, 26, 27} {
		// Two sets of 512 long names use exactly 64 MiB. Inline symbol
		// and section names add 26 bytes; trim the last name to compensate.
		binary.LittleEndian.PutUint32(last[testCOFFSymbolOffset+512*18+4:], 4+skip)
		err := verifyArchiveObjectNameBudget(raw[2:])
		if (err == nil) != (skip >= 26) {
			t.Fatalf("boundary skip %d: %v", skip, err)
		}
	}
}

func TestInspectArchiveNameBudgetAcceptsMixedFormats(t *testing.T) {
	archive := indexedObjectNameArchive([][]byte{
		ordinaryObjectWithSharedNames(4, "ordinary_long_symbol"), testBigObject(),
	})
	report, err := InspectArchive(archive)
	if err != nil || len(report.Members) != 2 || !report.Members[1].Object.BigObject {
		t.Fatalf("mixed-format archive: members=%d err=%v", len(report.Members), err)
	}
}

func TestArchiveObjectNameBudgetRejectsShortMembers(t *testing.T) {
	for _, size := range []int{0, 1, 19} {
		if err := verifyArchiveObjectNameBudget([]rawArchiveMember{{data: make([]byte, size)}}); err == nil {
			t.Fatalf("short object of %d bytes admitted", size)
		}
	}
}

func BenchmarkInspectArchiveExpandedObjectNames(b *testing.B) {
	archive := sharedObjectNameArchive(8, 513, 64<<10)
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		report, err := InspectArchive(archive)
		// Keep one fixture for the admitted baseline and the bounded rejection.
		if err != nil && !errors.Is(err, ErrNameBudgetExceeded) {
			b.Fatal(err)
		}
		if err == nil && len(report.Members) != 8 {
			b.Fatalf("members=%d", len(report.Members))
		}
	}
}

func sharedObjectNameArchive(members, symbols, nameBytes int) []byte {
	objects := make([][]byte, members)
	for index := range objects {
		objects[index] = ordinaryObjectWithSharedNames(symbols, strings.Repeat("n", nameBytes))
	}
	return indexedObjectNameArchive(objects)
}

func indexedObjectNameArchive(objects [][]byte) []byte {
	var names bytes.Buffer
	for index, object := range objects {
		name := fmt.Sprintf("s%07d", index)
		offset := int(binary.LittleEndian.Uint32(object[8:]))
		if isBigCOFFObject(object) {
			offset = int(binary.LittleEndian.Uint32(object[48:]))
		}
		copy(object[offset:offset+8], name)
		names.WriteString(name)
		names.WriteByte(0)
	}
	members := len(objects)
	first := make([]byte, 4+4*members+names.Len())
	second := make([]byte, 8+6*members+names.Len())
	secondHeader := nextSyntheticArchiveMemberOffset(len(coffArchiveSignature), len(first))
	objectHeader := nextSyntheticArchiveMemberOffset(secondHeader, len(second))
	binary.BigEndian.PutUint32(first, uint32(members))
	binary.LittleEndian.PutUint32(second, uint32(members))
	binary.LittleEndian.PutUint32(second[4+4*members:], uint32(members))
	for index, object := range objects {
		binary.BigEndian.PutUint32(first[4+4*index:], uint32(objectHeader))
		binary.LittleEndian.PutUint32(second[4+4*index:], uint32(objectHeader))
		binary.LittleEndian.PutUint16(second[8+4*members+2*index:], uint16(index+1))
		objectHeader = nextSyntheticArchiveMemberOffset(objectHeader, len(object))
	}
	copy(first[4+4*members:], names.Bytes())
	copy(second[8+6*members:], names.Bytes())
	var archive bytes.Buffer
	archive.Write(coffArchiveSignature[:])
	writeSyntheticArchiveMember(&archive, "/", first)
	writeSyntheticArchiveMember(&archive, "/", second)
	for index, object := range objects {
		writeSyntheticArchiveMember(&archive, fmt.Sprintf("m%d.obj/", index), object)
	}
	return archive.Bytes()
}
