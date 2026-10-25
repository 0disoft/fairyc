package pecoff

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"
	"testing"
)

func TestArchiveLongNamesRemainIndependentOfInput(t *testing.T) {
	first, second := "first_long_member_name.obj", "second_long_member_name.obj"
	archive := sharedLongNameArchive(4, []byte(first+"\x00"+second+"\x00"))
	raw, err := parseRawArchiveMembers(archive)
	if err != nil {
		t.Fatal(err)
	}
	copy(archive[raw[4].headerOffset:], "/1")
	copy(archive[raw[5].headerOffset:], fmt.Sprintf("/%d", len(first)+1))
	report, err := InspectArchive(archive)
	if err != nil {
		t.Fatal(err)
	}
	for i := range raw[2].data {
		raw[2].data[i] = 'x'
	}
	for i, want := range []string{first, first[1:], second, first} {
		if report.Members[i].Name != want || report.Symbols[i].Member != want {
			t.Fatalf("member %d no longer owns its name: %q / %q", i, report.Members[i].Name, report.Symbols[i].Member)
		}
	}
}

func TestArchiveRejectsInvalidLongNameReferences(t *testing.T) {
	for _, test := range []struct {
		name, reference, table, want string
	}{
		{"empty table", "/0", "", "invalid long-name reference"},
		{"empty name", "/0", "\x00", "not terminated"},
		{"unterminated", "/0", "name", "not terminated"},
		{"out of range", "/9", "name\x00", "invalid long-name reference"},
		{"invalid offset", "/abc", "name\x00", "invalid long-name reference"},
	} {
		t.Run(test.name, func(t *testing.T) {
			archive := sharedLongNameArchive(1, []byte(test.table))
			raw, err := parseRawArchiveMembers(archive)
			if err != nil {
				t.Fatal(err)
			}
			copy(archive[raw[3].headerOffset:], test.reference)
			if _, err := InspectArchive(archive); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("invalid long name: %v, want %q", err, test.want)
			}
		})
	}
}

func BenchmarkInspectArchiveSharedLongName(b *testing.B) {
	for _, members := range []int{32, 256} {
		b.Run(fmt.Sprint(members), func(b *testing.B) {
			name := strings.Repeat("a", 128<<10)
			archive := sharedLongNameArchive(members, []byte(name+"\x00"))
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				report, err := InspectArchive(archive)
				if err != nil || len(report.Members) != members || report.Members[members-1].Name != name {
					b.Fatalf("shared long-name archive: %v", err)
				}
			}
		})
	}
}

func sharedLongNameArchive(members int, longnames []byte) []byte {
	var names bytes.Buffer
	objects := make([][]byte, members)
	for i := range objects {
		name := fmt.Sprintf("s%07d", i)
		names.WriteString(name)
		names.WriteByte(0)
		objects[i] = namedMinimalAMD64COFF(name)
	}
	first := make([]byte, 4+4*members+names.Len())
	second := make([]byte, 8+6*members+names.Len())
	secondHeader := nextSyntheticArchiveMemberOffset(len(coffArchiveSignature), len(first))
	longHeader := nextSyntheticArchiveMemberOffset(secondHeader, len(second))
	objectHeader := nextSyntheticArchiveMemberOffset(longHeader, len(longnames))
	binary.BigEndian.PutUint32(first, uint32(members))
	binary.LittleEndian.PutUint32(second, uint32(members))
	binary.LittleEndian.PutUint32(second[4+4*members:], uint32(members))
	for i, object := range objects {
		binary.BigEndian.PutUint32(first[4+4*i:], uint32(objectHeader))
		binary.LittleEndian.PutUint32(second[4+4*i:], uint32(objectHeader))
		binary.LittleEndian.PutUint16(second[8+4*members+2*i:], uint16(i+1))
		objectHeader = nextSyntheticArchiveMemberOffset(objectHeader, len(object))
	}
	copy(first[4+4*members:], names.Bytes())
	copy(second[8+6*members:], names.Bytes())
	var archive bytes.Buffer
	archive.Write(coffArchiveSignature[:])
	writeSyntheticArchiveMember(&archive, "/", first)
	writeSyntheticArchiveMember(&archive, "/", second)
	writeSyntheticArchiveMember(&archive, "//", longnames)
	for _, object := range objects {
		writeSyntheticArchiveMember(&archive, "/0", object)
	}
	return archive.Bytes()
}
