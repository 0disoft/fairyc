package pecoff

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"
	"testing"
)

func TestInspectArchiveAcceptsTwoIndexedAMD64Objects(t *testing.T) {
	archive := syntheticCOFFArchive()
	report, err := InspectArchive(archive)
	if err != nil {
		t.Fatalf("InspectArchive() error = %v", err)
	}
	if len(report.Members) != 2 || report.Members[0].Name != "winner.obj" || report.Members[1].Name != "unused.obj" {
		t.Fatalf("archive members = %+v", report.Members)
	}
	if len(report.Symbols) != 2 || report.Symbols[0] != (ArchiveSymbol{
		Name: "unused", Member: "unused.obj", MemberHeaderOffset: report.Members[1].HeaderOffset,
	}) || report.Symbols[1] != (ArchiveSymbol{
		Name: "winner", Member: "winner.obj", MemberHeaderOffset: report.Members[0].HeaderOffset,
	}) {
		t.Fatalf("archive symbols = %+v", report.Symbols)
	}
	if len(report.Members[0].Symbols) != 1 || report.Members[0].Symbols[0] != "winner" ||
		len(report.Members[1].Symbols) != 1 || report.Members[1].Symbols[0] != "unused" {
		t.Fatalf("member symbol indexes = %+v", report.Members)
	}
	for _, member := range report.Members {
		if member.Object.Machine != machineAMD64 || len(member.Object.SymbolRecords) != 1 {
			t.Fatalf("member object receipt = %+v", member)
		}
	}
}

func TestInspectArchiveRejectsMalformedContainerAndIndexes(t *testing.T) {
	baseline := syntheticCOFFArchive()
	baselineReport, err := InspectArchive(baseline)
	if err != nil {
		t.Fatal(err)
	}
	winnerData := int(baselineReport.Members[0].DataOffset)
	secondHeader := nextSyntheticArchiveMemberOffset(len(coffArchiveSignature), syntheticFirstLinkerSize())
	secondData := secondHeader + coffArchiveHeaderBytes
	memberCount := int(binary.LittleEndian.Uint32(baseline[secondData:]))
	symbolCountOffset := secondData + 4 + memberCount*4
	indicesOffset := symbolCountOffset + 4

	tests := []struct {
		name string
		edit func([]byte)
		want string
	}{
		{name: "signature", edit: func(value []byte) { value[0] = '?' }, want: "invalid signature"},
		{name: "header terminator", edit: func(value []byte) { value[8+58] = '?' }, want: "header terminator"},
		{name: "unknown second member offset", edit: func(value []byte) {
			binary.LittleEndian.PutUint32(value[secondData+4:], 1)
		}, want: "unknown member offset"},
		{name: "zero second symbol index", edit: func(value []byte) {
			binary.LittleEndian.PutUint16(value[indicesOffset:], 0)
		}, want: "invalid member index"},
		{name: "first index symbol missing from member", edit: func(value []byte) {
			copy(value[len(coffArchiveSignature)+coffArchiveHeaderBytes+12:], "zzzzzz")
		}, want: "not defined externally"},
		{name: "indexed symbol section is outside object", edit: func(value []byte) {
			binary.LittleEndian.PutUint16(value[winnerData+testCOFFSymbolOffset+12:], 2)
		}, want: "not defined externally"},
		{name: "nonlexical second symbols", edit: func(value []byte) {
			names := indicesOffset + 2*int(binary.LittleEndian.Uint32(value[symbolCountOffset:]))
			copy(value[names:], "zinner\x00ausedx\x00")
		}, want: "not lexically ordered"},
		{name: "trailing linker bytes", edit: func(value []byte) {
			binary.LittleEndian.PutUint32(value[symbolCountOffset:], 1)
		}, want: "trailing symbol data"},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			archive := bytes.Clone(baseline)
			testCase.edit(archive)
			if _, err := InspectArchive(archive); err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("InspectArchive() error = %v, want %q", err, testCase.want)
			}
		})
	}
}

func syntheticCOFFArchive() []byte {
	winner := namedMinimalAMD64COFF("winner")
	unused := namedMinimalAMD64COFF("unused")
	firstSize := syntheticFirstLinkerSize()
	secondSize := syntheticSecondLinkerSize()
	firstHeader := len(coffArchiveSignature)
	secondHeader := nextSyntheticArchiveMemberOffset(firstHeader, firstSize)
	winnerHeader := nextSyntheticArchiveMemberOffset(secondHeader, secondSize)
	unusedHeader := nextSyntheticArchiveMemberOffset(winnerHeader, len(winner))

	first := make([]byte, firstSize)
	binary.BigEndian.PutUint32(first, 2)
	binary.BigEndian.PutUint32(first[4:], uint32(winnerHeader))
	binary.BigEndian.PutUint32(first[8:], uint32(unusedHeader))
	copy(first[12:], "winner\x00unused\x00")

	second := make([]byte, secondSize)
	binary.LittleEndian.PutUint32(second, 2)
	binary.LittleEndian.PutUint32(second[4:], uint32(winnerHeader))
	binary.LittleEndian.PutUint32(second[8:], uint32(unusedHeader))
	binary.LittleEndian.PutUint32(second[12:], 2)
	binary.LittleEndian.PutUint16(second[16:], 2)
	binary.LittleEndian.PutUint16(second[18:], 1)
	copy(second[20:], "unused\x00winner\x00")

	var archive bytes.Buffer
	archive.Write(coffArchiveSignature[:])
	writeSyntheticArchiveMember(&archive, "/", first)
	writeSyntheticArchiveMember(&archive, "/", second)
	writeSyntheticArchiveMember(&archive, "winner.obj/", winner)
	writeSyntheticArchiveMember(&archive, "unused.obj/", unused)
	return archive.Bytes()
}

func namedMinimalAMD64COFF(name string) []byte {
	if len(name) > 8 {
		panic("synthetic COFF symbol exceeds the inline name")
	}
	object := minimalAMD64COFF()
	for index := 0; index < 8; index++ {
		object[testCOFFSymbolOffset+index] = 0
	}
	copy(object[testCOFFSymbolOffset:], name)
	return object
}

func syntheticFirstLinkerSize() int {
	return 4 + 2*4 + len("winner") + 1 + len("unused") + 1
}

func syntheticSecondLinkerSize() int {
	return 4 + 2*4 + 4 + 2*2 + len("unused") + 1 + len("winner") + 1
}

func nextSyntheticArchiveMemberOffset(header, size int) int {
	next := header + coffArchiveHeaderBytes + size
	if next&1 != 0 {
		next++
	}
	return next
}

func writeSyntheticArchiveMember(archive *bytes.Buffer, name string, data []byte) {
	header := bytes.Repeat([]byte{' '}, coffArchiveHeaderBytes)
	copy(header[:16], name)
	copy(header[16:28], "0")
	copy(header[28:34], "0")
	copy(header[34:40], "0")
	copy(header[40:48], "0")
	copy(header[48:58], fmt.Sprintf("%d", len(data)))
	header[58] = '`'
	header[59] = '\n'
	archive.Write(header)
	archive.Write(data)
	if len(data)&1 != 0 {
		archive.WriteByte('\n')
	}
}
