package pecoff

import (
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"
)

const (
	coffArchiveHeaderBytes = 60
	maxArchiveMembers      = 4096
	maxArchiveSymbols      = 1 << 20
)

var coffArchiveSignature = [8]byte{'!', '<', 'a', 'r', 'c', 'h', '>', '\n'}

// ArchiveMember records one bounded AMD64 COFF object member and the symbols that the preferred
// second linker member indexes to it.
type ArchiveMember struct {
	Name         string
	HeaderOffset uint32
	DataOffset   uint32
	Size         uint32
	Symbols      []string
	Object       ObjectReport
}

// ArchiveSymbol records one preferred-index lookup result. Archive membership and final-link
// selection remain separate facts.
type ArchiveSymbol struct {
	Name               string
	Member             string
	MemberHeaderOffset uint32
}

// ArchiveReport is a bounded structural receipt for one Microsoft AMD64 COFF archive.
type ArchiveReport struct {
	Members []ArchiveMember
	Symbols []ArchiveSymbol
}

type rawArchiveMember struct {
	rawName      string
	headerOffset uint32
	dataOffset   uint32
	data         []byte
}

// InspectArchive validates the Microsoft COFF archive container, both linker members, the
// preferred second-linker-member index, and every contained AMD64 object.
func InspectArchive(archive []byte) (ArchiveReport, error) {
	if len(archive) < len(coffArchiveSignature)+2*coffArchiveHeaderBytes || len(archive) > maxExecutableBytes {
		return ArchiveReport{}, fmt.Errorf("COFF archive has invalid size %d", len(archive))
	}
	if string(archive[:len(coffArchiveSignature)]) != string(coffArchiveSignature[:]) {
		return ArchiveReport{}, fmt.Errorf("COFF archive has an invalid signature")
	}
	rawMembers, err := parseRawArchiveMembers(archive)
	if err != nil {
		return ArchiveReport{}, err
	}
	if len(rawMembers) < 3 || rawMembers[0].rawName != "/" || rawMembers[1].rawName != "/" {
		return ArchiveReport{}, fmt.Errorf("COFF archive does not begin with two linker members")
	}

	objectStart := 2
	var longnames []byte
	if rawMembers[objectStart].rawName == "//" {
		longnames = rawMembers[objectStart].data
		objectStart++
	}
	if objectStart >= len(rawMembers) {
		return ArchiveReport{}, fmt.Errorf("COFF archive has no object members")
	}

	objectsByOffset := make(map[uint32]int, len(rawMembers)-objectStart)
	report := ArchiveReport{Members: make([]ArchiveMember, 0, len(rawMembers)-objectStart)}
	for _, raw := range rawMembers[objectStart:] {
		name, err := resolveArchiveMemberName(raw.rawName, longnames)
		if err != nil {
			return ArchiveReport{}, err
		}
		object, err := InspectObject(raw.data)
		if err != nil {
			return ArchiveReport{}, fmt.Errorf("inspect COFF archive member %q: %w", name, err)
		}
		if _, duplicate := objectsByOffset[raw.headerOffset]; duplicate {
			return ArchiveReport{}, fmt.Errorf("COFF archive repeats member offset %d", raw.headerOffset)
		}
		objectsByOffset[raw.headerOffset] = len(report.Members)
		report.Members = append(report.Members, ArchiveMember{
			Name: name, HeaderOffset: raw.headerOffset, DataOffset: raw.dataOffset,
			Size: uint32(len(raw.data)), Object: object,
		})
	}

	if err := validateFirstArchiveLinkerMember(rawMembers[0].data, objectsByOffset, report.Members); err != nil {
		return ArchiveReport{}, err
	}
	index, memberOffsets, err := parseSecondArchiveLinkerMember(rawMembers[1].data, objectsByOffset)
	if err != nil {
		return ArchiveReport{}, err
	}
	if len(memberOffsets) != len(report.Members) {
		return ArchiveReport{}, fmt.Errorf("COFF archive second linker member indexes %d members, want %d", len(memberOffsets), len(report.Members))
	}
	for _, symbol := range index {
		memberIndex := objectsByOffset[symbol.MemberHeaderOffset]
		if !objectDefinesExternalSymbol(report.Members[memberIndex].Object, symbol.Name) {
			return ArchiveReport{}, fmt.Errorf("COFF archive second linker member symbol %q is not defined externally by its indexed member", symbol.Name)
		}
		symbol.Member = report.Members[memberIndex].Name
		report.Members[memberIndex].Symbols = append(report.Members[memberIndex].Symbols, symbol.Name)
		report.Symbols = append(report.Symbols, symbol)
	}
	return report, nil
}

func parseRawArchiveMembers(archive []byte) ([]rawArchiveMember, error) {
	members := make([]rawArchiveMember, 0)
	position := len(coffArchiveSignature)
	for position < len(archive) {
		if len(members) >= maxArchiveMembers+3 {
			return nil, fmt.Errorf("COFF archive exceeds the member budget")
		}
		if position+coffArchiveHeaderBytes > len(archive) {
			return nil, fmt.Errorf("COFF archive has a truncated member header")
		}
		header := archive[position : position+coffArchiveHeaderBytes]
		if header[58] != '`' || header[59] != '\n' {
			return nil, fmt.Errorf("COFF archive member at %d has an invalid header terminator", position)
		}
		rawName := strings.TrimSpace(string(header[:16]))
		if rawName == "" || strings.IndexByte(rawName, 0) >= 0 {
			return nil, fmt.Errorf("COFF archive member at %d has an invalid name", position)
		}
		sizeText := strings.TrimSpace(string(header[48:58]))
		size, err := strconv.ParseUint(sizeText, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("COFF archive member %q has an invalid size", rawName)
		}
		dataOffset := position + coffArchiveHeaderBytes
		end := uint64(dataOffset) + size
		if end > uint64(len(archive)) {
			return nil, fmt.Errorf("COFF archive member %q exceeds the file", rawName)
		}
		members = append(members, rawArchiveMember{
			rawName: rawName, headerOffset: uint32(position), dataOffset: uint32(dataOffset),
			data: archive[dataOffset:int(end)],
		})
		position = int(end)
		if position&1 != 0 {
			if position >= len(archive) || archive[position] != '\n' {
				return nil, fmt.Errorf("COFF archive member %q has invalid alignment padding", rawName)
			}
			position++
		}
	}
	if position != len(archive) {
		return nil, fmt.Errorf("COFF archive member layout does not consume the file")
	}
	return members, nil
}

func resolveArchiveMemberName(raw string, longnames []byte) (string, error) {
	if raw == "/" || raw == "//" {
		return "", fmt.Errorf("COFF archive has a misplaced special member")
	}
	if strings.HasPrefix(raw, "/") {
		offset, err := strconv.ParseUint(strings.TrimPrefix(raw, "/"), 10, 32)
		if err != nil || offset >= uint64(len(longnames)) {
			return "", fmt.Errorf("COFF archive member has invalid long-name reference %q", raw)
		}
		end := int(offset)
		for end < len(longnames) && longnames[end] != 0 {
			end++
		}
		if end == len(longnames) || end == int(offset) {
			return "", fmt.Errorf("COFF archive member long name %q is not terminated", raw)
		}
		return string(longnames[offset:uint64(end)]), nil
	}
	name := strings.TrimSuffix(raw, "/")
	if name == "" || strings.ContainsAny(name, "\\/") {
		return "", fmt.Errorf("COFF archive member has invalid short name %q", raw)
	}
	return name, nil
}

func validateFirstArchiveLinkerMember(data []byte, objectsByOffset map[uint32]int, members []ArchiveMember) error {
	if len(data) < 4 {
		return fmt.Errorf("COFF archive first linker member is truncated")
	}
	count := uint64(binary.BigEndian.Uint32(data))
	if count == 0 || count > maxArchiveSymbols {
		return fmt.Errorf("COFF archive first linker member has invalid symbol count %d", count)
	}
	offsetsEnd := uint64(4) + count*4
	if offsetsEnd > uint64(len(data)) {
		return fmt.Errorf("COFF archive first linker member offsets are truncated")
	}
	offsets := make([]uint32, 0, count)
	for index := uint64(0); index < count; index++ {
		offset := binary.BigEndian.Uint32(data[4+index*4:])
		if _, found := objectsByOffset[offset]; !found {
			return fmt.Errorf("COFF archive first linker member references unknown member offset %d", offset)
		}
		offsets = append(offsets, offset)
	}
	names, err := parseArchiveSymbolNames(data[offsetsEnd:], int(count), "first")
	if err != nil {
		return err
	}
	for index, name := range names {
		member := members[objectsByOffset[offsets[index]]]
		if !objectDefinesExternalSymbol(member.Object, name) {
			return fmt.Errorf("COFF archive first linker member symbol %q is not defined externally by its indexed member", name)
		}
	}
	return nil
}

func parseSecondArchiveLinkerMember(data []byte, objectsByOffset map[uint32]int) ([]ArchiveSymbol, []uint32, error) {
	if len(data) < 8 {
		return nil, nil, fmt.Errorf("COFF archive second linker member is truncated")
	}
	memberCount := uint64(binary.LittleEndian.Uint32(data))
	if memberCount == 0 || memberCount > maxArchiveMembers {
		return nil, nil, fmt.Errorf("COFF archive second linker member has invalid member count %d", memberCount)
	}
	offsetsEnd := uint64(4) + memberCount*4
	if offsetsEnd+4 > uint64(len(data)) {
		return nil, nil, fmt.Errorf("COFF archive second linker member offsets are truncated")
	}
	memberOffsets := make([]uint32, 0, memberCount)
	previous := uint32(0)
	for index := uint64(0); index < memberCount; index++ {
		offset := binary.LittleEndian.Uint32(data[4+index*4:])
		if _, found := objectsByOffset[offset]; !found {
			return nil, nil, fmt.Errorf("COFF archive second linker member references unknown member offset %d", offset)
		}
		if index > 0 && offset <= previous {
			return nil, nil, fmt.Errorf("COFF archive second linker member offsets are not strictly ordered")
		}
		previous = offset
		memberOffsets = append(memberOffsets, offset)
	}
	symbolCount := uint64(binary.LittleEndian.Uint32(data[offsetsEnd:]))
	if symbolCount == 0 || symbolCount > maxArchiveSymbols {
		return nil, nil, fmt.Errorf("COFF archive second linker member has invalid symbol count %d", symbolCount)
	}
	indicesStart := offsetsEnd + 4
	namesStart := indicesStart + symbolCount*2
	if namesStart > uint64(len(data)) {
		return nil, nil, fmt.Errorf("COFF archive second linker member indices are truncated")
	}
	names, err := parseArchiveSymbolNames(data[namesStart:], int(symbolCount), "second")
	if err != nil {
		return nil, nil, err
	}
	result := make([]ArchiveSymbol, 0, symbolCount)
	previousName := ""
	for index, name := range names {
		memberIndex := binary.LittleEndian.Uint16(data[indicesStart+uint64(index)*2:])
		if memberIndex == 0 || uint64(memberIndex) > memberCount {
			return nil, nil, fmt.Errorf("COFF archive second linker member symbol %q has invalid member index %d", name, memberIndex)
		}
		if index > 0 && name < previousName {
			return nil, nil, fmt.Errorf("COFF archive second linker member symbols are not lexically ordered")
		}
		previousName = name
		result = append(result, ArchiveSymbol{Name: name, MemberHeaderOffset: memberOffsets[memberIndex-1]})
	}
	return result, memberOffsets, nil
}

func parseArchiveSymbolNames(data []byte, count int, linker string) ([]string, error) {
	names := make([]string, 0, count)
	position := 0
	for len(names) < count {
		end := position
		for end < len(data) && data[end] != 0 {
			end++
		}
		if end == len(data) || end == position {
			return nil, fmt.Errorf("COFF archive %s linker member symbol names are truncated", linker)
		}
		name := string(data[position:end])
		if strings.IndexByte(name, '\n') >= 0 || strings.IndexByte(name, '\r') >= 0 {
			return nil, fmt.Errorf("COFF archive %s linker member has an invalid symbol name", linker)
		}
		names = append(names, name)
		position = end + 1
	}
	if position != len(data) {
		return nil, fmt.Errorf("COFF archive %s linker member has trailing symbol data", linker)
	}
	return names, nil
}

func objectDefinesExternalSymbol(report ObjectReport, name string) bool {
	found := false
	for _, symbol := range report.SymbolRecords {
		if symbol.Name != name || symbol.SectionNumber <= 0 || int(symbol.SectionNumber) > len(report.Sections) ||
			symbol.StorageClass != coffStorageClassExternal {
			continue
		}
		section := report.Sections[symbol.SectionNumber-1]
		if section.RawSize == 0 || symbol.Value >= section.RawSize {
			continue
		}
		if found {
			return false
		}
		found = true
	}
	return found
}
