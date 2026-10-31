package pecoff

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// InspectIndexedArchiveObject checks a selected ordinary object against both
// Microsoft archive indexes. Other bodies remain opaque (including metadata and
// import members); this does not relax or replace InspectArchive's full checks.
func InspectIndexedArchiveObject(archive []byte, symbol string) (ArchiveMember, error) {
	var result ArchiveMember
	if len(archive) < 8+2*coffArchiveHeaderBytes || len(archive) > maxExecutableBytes || !bytes.Equal(archive[:8], coffArchiveSignature[:]) || symbol == "" {
		return result, fmt.Errorf("invalid selected archive input")
	}
	members, err := parseRawArchiveMembers(archive)
	if err != nil {
		return result, err
	}
	if len(members) < 3 || members[0].rawName != "/" || members[1].rawName != "/" {
		return result, fmt.Errorf("selected archive requires both linker members")
	}
	start := 2
	var longnames string
	if members[start].rawName == "//" {
		longnames = string(members[start].data)
		start++
	}
	if start >= len(members) {
		return result, fmt.Errorf("archive has no object bodies")
	}
	offsets := map[uint32]int{}
	for i, m := range members[start:] {
		offsets[m.headerOffset] = i + start
	}
	second, indexedOffsets, err := parseSecondArchiveLinkerMember(members[1].data, offsets)
	if err != nil {
		return result, err
	}
	if len(indexedOffsets) != len(offsets) {
		return result, fmt.Errorf("archive index omits object bodies")
	}
	first, err := selectedFirstArchiveIndex(members[0].data, offsets, symbol)
	if err != nil {
		return result, err
	}
	var selected uint32
	matches := 0
	for _, entry := range second {
		if entry.Name == symbol {
			selected = entry.MemberHeaderOffset
			matches++
		}
	}
	if matches != 1 || first != selected {
		return result, fmt.Errorf("archive indexes disagree or ambiguously select %q", symbol)
	}
	member := members[offsets[selected]]
	name, err := resolveArchiveMemberName(member.rawName, longnames, indexArchiveNameSpans(members[start:], longnames))
	if err != nil {
		return result, err
	}
	object, err := InspectObject(member.data)
	if err != nil {
		return result, fmt.Errorf("selected member %s: %w", name, err)
	}
	if !indexObjectExternalSymbols(object)[symbol] {
		return result, fmt.Errorf("selected archive member does not uniquely define %q", symbol)
	}
	result = ArchiveMember{Name: name, HeaderOffset: member.headerOffset, DataOffset: member.dataOffset, Size: uint32(len(member.data)), Symbols: []string{symbol}, Object: object}
	return result, nil
}

func selectedFirstArchiveIndex(data []byte, offsets map[uint32]int, symbol string) (uint32, error) {
	if len(data) < 4 {
		return 0, fmt.Errorf("truncated first archive index")
	}
	count := uint64(binary.BigEndian.Uint32(data))
	end := 4 + count*4
	if count == 0 || count > maxArchiveSymbols || end > uint64(len(data)) {
		return 0, fmt.Errorf("invalid first archive index count")
	}
	names, err := parseArchiveSymbolNames(data[end:], int(count), "first")
	if err != nil {
		return 0, err
	}
	var selected uint32
	matches := 0
	for i, name := range names {
		offset := binary.BigEndian.Uint32(data[4+i*4:])
		if _, ok := offsets[offset]; !ok {
			return 0, fmt.Errorf("first archive index points outside object bodies")
		}
		if name == symbol {
			selected = offset
			matches++
		}
	}
	if matches != 1 {
		return 0, fmt.Errorf("first archive index has no unique demand %q", symbol)
	}
	return selected, nil
}
