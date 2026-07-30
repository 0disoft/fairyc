package pecoff

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"
)

const maxMSVCMapBytes = 8 * 1024 * 1024

// ArchiveCOMDATLinkSpec identifies one closed archive-extraction and pick-any COMDAT proof.
type ArchiveCOMDATLinkSpec struct {
	ArchiveName      string
	SelectedMember   string
	UnusedMember     string
	DirectObject     string
	DemandSymbol     string
	UnusedSymbol     string
	COMDATSymbol     string
	ReceiptSection   string
	ReceiptMagic     [16]byte
	ArchiveCandidate int32
	DirectCandidate  int32
}

// ArchiveCOMDATLinkReport binds the preferred archive index, link-map owners, and final PE value.
// It is internal validation evidence, not a stable serialized format.
type ArchiveCOMDATLinkReport struct {
	ArchiveMembers      int
	IndexedSymbols      int
	ExtractedMember     string
	DemandOwner         string
	COMDATOwner         string
	TargetRVA           uint32
	COMDATRVA           uint32
	SelectedCOMDATValue int32
}

type msvcMapSymbol struct {
	Name             string
	PreferredAddress uint64
	Owner            string
}

// InspectArchiveCOMDATLink proves that the preferred archive index points the unresolved demand to
// one member, that the final map attributes the demanded function and pick-any COMDAT to concrete
// inputs, and that the final PE addresses and candidate value agree with that selection trace.
func InspectArchiveCOMDATLink(archive, directObject, image, linkMap []byte, spec ArchiveCOMDATLinkSpec) (ArchiveCOMDATLinkReport, error) {
	if err := validateArchiveCOMDATLinkSpec(spec); err != nil {
		return ArchiveCOMDATLinkReport{}, err
	}
	archiveReport, err := InspectArchive(archive)
	if err != nil {
		return ArchiveCOMDATLinkReport{}, fmt.Errorf("inspect archive selection input: %w", err)
	}
	imageReport, err := InspectExecutable(image)
	if err != nil {
		return ArchiveCOMDATLinkReport{}, fmt.Errorf("inspect archive selection image: %w", err)
	}
	directReport, err := InspectObject(directObject)
	if err != nil {
		return ArchiveCOMDATLinkReport{}, fmt.Errorf("inspect direct COMDAT object: %w", err)
	}

	selected, err := exactArchiveMember(archiveReport.Members, spec.SelectedMember)
	if err != nil {
		return ArchiveCOMDATLinkReport{}, err
	}
	unused, err := exactArchiveMember(archiveReport.Members, spec.UnusedMember)
	if err != nil {
		return ArchiveCOMDATLinkReport{}, err
	}
	if selected.HeaderOffset == unused.HeaderOffset {
		return ArchiveCOMDATLinkReport{}, fmt.Errorf("archive selection members are not distinct")
	}
	demandIndex, err := exactArchiveSymbol(archiveReport.Symbols, spec.DemandSymbol)
	if err != nil {
		return ArchiveCOMDATLinkReport{}, err
	}
	if demandIndex.Member != selected.Name {
		return ArchiveCOMDATLinkReport{}, fmt.Errorf("archive demand symbol %q indexes member %q, want %q", spec.DemandSymbol, demandIndex.Member, selected.Name)
	}
	unusedIndex, err := exactArchiveSymbol(archiveReport.Symbols, spec.UnusedSymbol)
	if err != nil {
		return ArchiveCOMDATLinkReport{}, err
	}
	if unusedIndex.Member != unused.Name {
		return ArchiveCOMDATLinkReport{}, fmt.Errorf("archive unused symbol %q indexes member %q, want %q", spec.UnusedSymbol, unusedIndex.Member, unused.Name)
	}
	if err := requireDefinedExternalFunction(selected.Object, spec.DemandSymbol, "archive demand"); err != nil {
		return ArchiveCOMDATLinkReport{}, err
	}
	if err := requireDefinedExternalNonCOMDATFunction(unused.Object, spec.UnusedSymbol, "archive unused"); err != nil {
		return ArchiveCOMDATLinkReport{}, err
	}
	selectedBytes := archive[int(selected.DataOffset):int(selected.DataOffset+selected.Size)]
	if err := requirePickAnyReadOnlyCOMDAT(selectedBytes, selected.Object, spec.COMDATSymbol, "archive", spec.ArchiveCandidate); err != nil {
		return ArchiveCOMDATLinkReport{}, err
	}
	if err := requirePickAnyReadOnlyCOMDAT(directObject, directReport, spec.COMDATSymbol, "direct", spec.DirectCandidate); err != nil {
		return ArchiveCOMDATLinkReport{}, err
	}
	mapSymbols, err := inspectMSVCMapSymbols(linkMap, spec.DemandSymbol, spec.COMDATSymbol, spec.UnusedSymbol)
	if err != nil {
		return ArchiveCOMDATLinkReport{}, err
	}
	demandMap, found := mapSymbols[spec.DemandSymbol]
	if !found {
		return ArchiveCOMDATLinkReport{}, fmt.Errorf("MSVC link map has no demanded symbol %q", spec.DemandSymbol)
	}
	demandOwner := classifyMSVCMapOwner(demandMap.Owner, spec)
	if demandOwner != "archive" {
		return ArchiveCOMDATLinkReport{}, fmt.Errorf("MSVC link map demand owner is %q, want selected archive member", demandOwner)
	}
	comdatMap, found := mapSymbols[spec.COMDATSymbol]
	if !found {
		return ArchiveCOMDATLinkReport{}, fmt.Errorf("MSVC link map has no COMDAT symbol %q", spec.COMDATSymbol)
	}
	comdatOwner := classifyMSVCMapOwner(comdatMap.Owner, spec)
	if comdatOwner != "archive" && comdatOwner != "direct" {
		return ArchiveCOMDATLinkReport{}, fmt.Errorf("MSVC link map COMDAT owner %q is not a declared candidate", comdatMap.Owner)
	}
	if _, found := mapSymbols[spec.UnusedSymbol]; found {
		return ArchiveCOMDATLinkReport{}, fmt.Errorf("MSVC link map retained unused archive symbol %q", spec.UnusedSymbol)
	}

	targetVA, comdatVA, err := readArchiveSelectionReceipt(image, imageReport, spec)
	if err != nil {
		return ArchiveCOMDATLinkReport{}, err
	}
	if demandMap.PreferredAddress != targetVA {
		return ArchiveCOMDATLinkReport{}, fmt.Errorf("MSVC link map demand address %#x does not match receipt %#x", demandMap.PreferredAddress, targetVA)
	}
	if comdatMap.PreferredAddress != comdatVA {
		return ArchiveCOMDATLinkReport{}, fmt.Errorf("MSVC link map COMDAT address %#x does not match receipt %#x", comdatMap.PreferredAddress, comdatVA)
	}
	targetRVA, err := preferredAddressRVA(imageReport, targetVA, "archive target")
	if err != nil {
		return ArchiveCOMDATLinkReport{}, err
	}
	if err := requireExecutableRVARange(imageReport, targetRVA, targetRVA+1); err != nil {
		return ArchiveCOMDATLinkReport{}, fmt.Errorf("archive-selected target is not executable: %w", err)
	}
	comdatRVA, err := preferredAddressRVA(imageReport, comdatVA, "selected COMDAT")
	if err != nil {
		return ArchiveCOMDATLinkReport{}, err
	}
	comdatOffset, comdatSection, err := mapImageRVA(imageReport, comdatRVA, 4)
	if err != nil {
		return ArchiveCOMDATLinkReport{}, fmt.Errorf("map selected COMDAT data: %w", err)
	}
	if comdatSection.Characteristics&sectionReadable == 0 ||
		comdatSection.Characteristics&(sectionContainsCode|sectionExecutable|sectionWritable) != 0 {
		return ArchiveCOMDATLinkReport{}, fmt.Errorf("selected COMDAT is not read-only data")
	}
	actualValue := int32(binary.LittleEndian.Uint32(image[comdatOffset : comdatOffset+4]))
	expectedValue := spec.ArchiveCandidate
	if comdatOwner == "direct" {
		expectedValue = spec.DirectCandidate
	}
	if actualValue != expectedValue {
		return ArchiveCOMDATLinkReport{}, fmt.Errorf("selected COMDAT value is %d, want %d for %s owner", actualValue, expectedValue, comdatOwner)
	}

	return ArchiveCOMDATLinkReport{
		ArchiveMembers: len(archiveReport.Members), IndexedSymbols: len(archiveReport.Symbols),
		ExtractedMember: selected.Name, DemandOwner: demandOwner, COMDATOwner: comdatOwner, TargetRVA: targetRVA,
		COMDATRVA: comdatRVA, SelectedCOMDATValue: actualValue,
	}, nil
}

func validateArchiveCOMDATLinkSpec(spec ArchiveCOMDATLinkSpec) error {
	names := []string{
		spec.ArchiveName, spec.SelectedMember, spec.UnusedMember, spec.DirectObject,
		spec.DemandSymbol, spec.UnusedSymbol, spec.COMDATSymbol, spec.ReceiptSection,
	}
	for _, name := range names {
		if name == "" {
			return fmt.Errorf("archive COMDAT link spec has an empty identity")
		}
	}
	if spec.SelectedMember == spec.UnusedMember || spec.DemandSymbol == spec.UnusedSymbol ||
		spec.ArchiveCandidate == spec.DirectCandidate || spec.ReceiptMagic == ([16]byte{}) {
		return fmt.Errorf("archive COMDAT link spec has ambiguous members, symbols, candidates, or magic")
	}
	if strings.ContainsAny(spec.ArchiveName+spec.SelectedMember+spec.UnusedMember+spec.DirectObject, "\\/") ||
		!strings.HasSuffix(strings.ToLower(spec.ArchiveName), ".lib") ||
		!strings.HasSuffix(strings.ToLower(spec.SelectedMember), ".obj") ||
		!strings.HasSuffix(strings.ToLower(spec.UnusedMember), ".obj") ||
		!strings.HasSuffix(strings.ToLower(spec.DirectObject), ".obj") ||
		len(spec.ReceiptSection) > 8 || !strings.HasPrefix(spec.ReceiptSection, ".") ||
		strings.ContainsAny(spec.DemandSymbol+spec.UnusedSymbol+spec.COMDATSymbol, " \t\r\n") {
		return fmt.Errorf("archive COMDAT link spec has invalid file, section, or symbol spelling")
	}
	return nil
}

func exactArchiveMember(members []ArchiveMember, name string) (ArchiveMember, error) {
	var result ArchiveMember
	found := false
	for _, member := range members {
		if !strings.EqualFold(member.Name, name) {
			continue
		}
		if found {
			return ArchiveMember{}, fmt.Errorf("COFF archive repeats member %q", name)
		}
		result = member
		found = true
	}
	if !found {
		return ArchiveMember{}, fmt.Errorf("COFF archive has no member %q", name)
	}
	return result, nil
}

func exactArchiveSymbol(symbols []ArchiveSymbol, name string) (ArchiveSymbol, error) {
	var result ArchiveSymbol
	found := false
	for _, symbol := range symbols {
		if symbol.Name != name {
			continue
		}
		if found {
			return ArchiveSymbol{}, fmt.Errorf("COFF archive index repeats symbol %q", name)
		}
		result = symbol
		found = true
	}
	if !found {
		return ArchiveSymbol{}, fmt.Errorf("COFF archive index has no symbol %q", name)
	}
	return result, nil
}

func requireDefinedExternalFunction(report ObjectReport, name, role string) error {
	symbol, err := exactObjectSymbol(report.SymbolRecords, name)
	if err != nil {
		return err
	}
	if symbol.SectionNumber <= 0 || int(symbol.SectionNumber) > len(report.Sections) ||
		symbol.Type != coffSymbolTypeFunction || symbol.StorageClass != coffStorageClassExternal || symbol.AuxiliaryRecords != 0 {
		return fmt.Errorf("%s symbol %q is not an exact defined external function", role, name)
	}
	if err := requireREL32SourceSection(report.Sections[symbol.SectionNumber-1]); err != nil {
		return fmt.Errorf("%s symbol %q is not defined in executable code: %w", role, name, err)
	}
	return nil
}

func requireDefinedExternalNonCOMDATFunction(report ObjectReport, name, role string) error {
	if err := requireDefinedExternalFunction(report, name, role); err != nil {
		return err
	}
	symbol, err := exactObjectSymbol(report.SymbolRecords, name)
	if err != nil {
		return err
	}
	section := report.Sections[symbol.SectionNumber-1]
	if section.COMDATSelection != 0 || section.Characteristics&sectionLinkCOMDAT != 0 {
		return fmt.Errorf("%s symbol %q must be in a non-COMDAT section", role, name)
	}
	return nil
}

func requirePickAnyReadOnlyCOMDAT(object []byte, report ObjectReport, name, role string, expected int32) error {
	symbol, err := exactObjectSymbol(report.SymbolRecords, name)
	if err != nil {
		return err
	}
	if symbol.SectionNumber <= 0 || int(symbol.SectionNumber) > len(report.Sections) || symbol.Type != 0 ||
		symbol.StorageClass != coffStorageClassExternal || symbol.AuxiliaryRecords != 0 {
		return fmt.Errorf("%s COMDAT symbol %q is not exact external data", role, name)
	}
	section := report.Sections[symbol.SectionNumber-1]
	if section.COMDATSelection != ObjectCOMDATAny || section.Characteristics&sectionReadable == 0 ||
		section.Characteristics&(sectionContainsCode|sectionExecutable|sectionWritable) != 0 {
		return fmt.Errorf("%s COMDAT symbol %q is not in a pick-any read-only section", role, name)
	}
	if uint64(symbol.Value)+4 > uint64(section.RawSize) || section.RawOffset == 0 ||
		uint64(section.RawOffset)+uint64(symbol.Value)+4 > uint64(len(object)) {
		return fmt.Errorf("%s COMDAT symbol %q has an invalid raw value range", role, name)
	}
	actual := int32(binary.LittleEndian.Uint32(object[uint64(section.RawOffset)+uint64(symbol.Value):]))
	if actual != expected {
		return fmt.Errorf("%s COMDAT object value is %d, want %d", role, actual, expected)
	}
	return nil
}

func inspectMSVCMapSymbols(data []byte, names ...string) (map[string]msvcMapSymbol, error) {
	if len(data) == 0 || len(data) > maxMSVCMapBytes || !utf8.Valid(data) || bytes.IndexByte(data, 0) >= 0 {
		return nil, fmt.Errorf("MSVC link map has invalid text size or encoding")
	}
	wanted := make(map[string]struct{}, len(names))
	for _, name := range names {
		wanted[name] = struct{}{}
	}
	result := make(map[string]msvcMapSymbol, len(names))
	for _, line := range strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n") {
		if len(line) > 64*1024 {
			return nil, fmt.Errorf("MSVC link map line exceeds the size budget")
		}
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		name := fields[1]
		if _, needed := wanted[name]; !needed || !validMSVCMapSectionOffset(fields[0]) {
			continue
		}
		address, err := strconv.ParseUint(strings.TrimPrefix(fields[2], "0x"), 16, 64)
		if err != nil || address == 0 {
			return nil, fmt.Errorf("MSVC link map symbol %q has an invalid preferred address", name)
		}
		if _, duplicate := result[name]; duplicate {
			return nil, fmt.Errorf("MSVC link map repeats symbol %q", name)
		}
		result[name] = msvcMapSymbol{Name: name, PreferredAddress: address, Owner: fields[len(fields)-1]}
	}
	return result, nil
}

func validMSVCMapSectionOffset(value string) bool {
	section, offset, found := strings.Cut(value, ":")
	if !found || section == "" || offset == "" {
		return false
	}
	if _, err := strconv.ParseUint(section, 16, 16); err != nil {
		return false
	}
	_, err := strconv.ParseUint(offset, 16, 32)
	return err == nil
}

func classifyMSVCMapOwner(owner string, spec ArchiveCOMDATLinkSpec) string {
	normalized := strings.ToLower(strings.ReplaceAll(owner, "\\", "/"))
	if separator := strings.LastIndexByte(normalized, '/'); separator >= 0 {
		normalized = normalized[separator+1:]
	}
	archive := strings.ToLower(spec.ArchiveName)
	archiveStem := strings.TrimSuffix(archive, ".lib")
	member := strings.ToLower(spec.SelectedMember)
	if normalized == archive+":"+member || normalized == archiveStem+":"+member ||
		normalized == archive+"("+member+")" || normalized == archiveStem+"("+member+")" {
		return "archive"
	}
	direct := strings.ToLower(spec.DirectObject)
	if normalized == direct {
		return "direct"
	}
	return owner
}

func readArchiveSelectionReceipt(image []byte, report Report, spec ArchiveCOMDATLinkSpec) (uint64, uint64, error) {
	var receipt Section
	found := false
	for _, section := range report.Sections {
		if section.Name != spec.ReceiptSection {
			continue
		}
		if found {
			return 0, 0, fmt.Errorf("archive selection image repeats receipt section %q", spec.ReceiptSection)
		}
		receipt = section
		found = true
	}
	if !found {
		return 0, 0, fmt.Errorf("archive selection image has no receipt section %q", spec.ReceiptSection)
	}
	if receipt.Characteristics&sectionReadable == 0 || receipt.Characteristics&(sectionContainsCode|sectionExecutable|sectionWritable) != 0 {
		return 0, 0, fmt.Errorf("archive selection receipt section %q is not read-only data", spec.ReceiptSection)
	}
	logicalSize := min(receipt.VirtualSize, receipt.RawSize)
	if logicalSize < 32 {
		return 0, 0, fmt.Errorf("archive selection receipt has no complete file-backed logical value")
	}
	start := uint64(receipt.RawOffset)
	end := start + uint64(logicalSize)
	if end > uint64(len(image)) {
		return 0, 0, fmt.Errorf("archive selection receipt section exceeds the image")
	}
	sectionBytes := image[start:end]
	magic := spec.ReceiptMagic[:]
	offset := bytes.Index(sectionBytes, magic)
	if offset < 0 {
		return 0, 0, fmt.Errorf("archive selection receipt magic is missing")
	}
	if bytes.Index(sectionBytes[offset+1:], magic) >= 0 {
		return 0, 0, fmt.Errorf("archive selection receipt magic is ambiguous")
	}
	pointers := offset + len(magic)
	if pointers+16 > len(sectionBytes) {
		return 0, 0, fmt.Errorf("archive selection receipt is truncated")
	}
	return binary.LittleEndian.Uint64(sectionBytes[pointers:]), binary.LittleEndian.Uint64(sectionBytes[pointers+8:]), nil
}

func preferredAddressRVA(report Report, address uint64, label string) (uint32, error) {
	if address < report.ImageBase {
		return 0, fmt.Errorf("%s preferred address %#x precedes image base", label, address)
	}
	rva := address - report.ImageBase
	if rva == 0 || rva >= uint64(report.SizeOfImage) || rva > uint64(^uint32(0)) {
		return 0, fmt.Errorf("%s preferred address %#x is outside the image", label, address)
	}
	return uint32(rva), nil
}
