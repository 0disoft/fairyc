package pecoff

import (
	"fmt"
	"strings"
	"testing"
)

func TestArchiveNameScanOrderAndShortNames(t *testing.T) {
	const first, second = "first_long_member.obj", "second_long_member.obj"
	table := first + "\x00" + second + "\x00"
	offsets := []int{len(first) + 2, 1, 0, len(first) + 1, 0}
	archive := sharedLongNameArchive(len(offsets)+1, []byte(table))
	raw, err := parseRawArchiveMembers(archive)
	if err != nil {
		t.Fatal(err)
	}
	for index, offset := range offsets {
		setArchiveNameReference(archive, raw[index+3], fmt.Sprintf("/%d", offset))
	}
	setArchiveNameReference(archive, raw[len(raw)-1], "short.obj/")
	report, err := InspectArchive(archive)
	if err != nil {
		t.Fatal(err)
	}
	clear(archive)
	for index, offset := range offsets {
		want := table[offset : offset+strings.IndexByte(table[offset:], 0)]
		if report.Members[index].Name != want || report.Symbols[index].Member != want {
			t.Fatalf("member %d: got %q / %q, want %q", index, report.Members[index].Name, report.Symbols[index].Member, want)
		}
	}
	if report.Members[len(offsets)].Name != "short.obj" {
		t.Fatalf("short member name changed: %q", report.Members[len(offsets)].Name)
	}
}

func TestArchiveNameScanSharesOneSpanAtMemberLimit(t *testing.T) {
	const nameBytes = 1 << 20
	table := strings.Repeat("a", nameBytes) + "\x00"
	archive := sharedLongNameArchive(maxArchiveMembers, []byte(table))
	raw, err := parseRawArchiveMembers(archive)
	if err != nil {
		t.Fatal(err)
	}
	for index := range raw[3:] {
		raw[index+3].rawName = fmt.Sprintf("/%d", (index*7919)%maxArchiveMembers)
	}
	spans := indexArchiveNameSpans(raw[3:], table)
	if len(spans) != 1 || spans[0] != (coffNameSpan{start: 0, end: nameBytes}) {
		t.Fatalf("suffix references must reuse one scan, got %v", spans)
	}
}

func TestArchiveNameScanPreservesFirstInvalidReference(t *testing.T) {
	for _, references := range [][2]string{{"/bad", "/4"}, {"/4", "/bad"}, {"/0", "/bad"}} {
		t.Run(references[0], func(t *testing.T) {
			archive := sharedLongNameArchive(2, []byte("\x00ok\x00unterminated"))
			raw, err := parseRawArchiveMembers(archive)
			if err != nil {
				t.Fatal(err)
			}
			for index, reference := range references {
				setArchiveNameReference(archive, raw[index+3], reference)
			}
			if _, err := InspectArchive(archive); err == nil || !strings.Contains(err.Error(), fmt.Sprintf("%q", references[0])) {
				t.Fatalf("first malformed member name not preserved: %v", err)
			}
		})
	}
}

func setArchiveNameReference(archive []byte, member rawArchiveMember, reference string) {
	copy(archive[member.headerOffset:member.headerOffset+16], fmt.Sprintf("%-16s", reference))
}

func BenchmarkInspectArchiveOverlappingNameScans(b *testing.B) {
	for _, members := range []int{1024, maxArchiveMembers} {
		b.Run(fmt.Sprint(members), func(b *testing.B) {
			const nameBytes = 4 << 20
			archive := sharedLongNameArchive(members, []byte(strings.Repeat("a", nameBytes)+"\x00"))
			raw, err := parseRawArchiveMembers(archive)
			if err != nil {
				b.Fatal(err)
			}
			for index, member := range raw[3:] {
				setArchiveNameReference(archive, member, fmt.Sprintf("/%d", (index*7919)%members))
			}
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				report, err := InspectArchive(archive)
				if err != nil || len(report.Members) != members || len(report.Members[0].Name) != nameBytes {
					b.Fatalf("overlapping archive names: %v", err)
				}
			}
		})
	}
}
