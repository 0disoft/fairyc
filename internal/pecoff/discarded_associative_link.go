package pecoff

import (
	"bytes"
	"crypto/sha256"
	"debug/pe"
	"encoding/binary"
	"fmt"
	"path/filepath"
	"strings"
)

// DiscardedAssociativeCOMDATSpec identifies two contributed ANY-COMDAT function candidates.
type DiscardedAssociativeCOMDATSpec struct {
	ArchiveName   string
	ArchiveMember string
	DirectObject  string
	SourceSymbol  string
}

// DiscardedAssociativeCOMDATReport records the selected owner and distinct unwind payloads.
type DiscardedAssociativeCOMDATReport struct {
	SelectedOwner           string
	DiscardedOwner          string
	SelectedBeginRVA        uint32
	SelectedEndRVA          uint32
	SelectedUnwindRVA       uint32
	SelectedUnwindDigest    [sha256.Size]byte
	DiscardedUnwindDigest   [sha256.Size]byte
	DiscardedPayloadMatches int
	DiscardedRawMatches     int
	OrphanPDataBytes        int
}

type associativeCOMDATCandidate struct {
	owner        string
	entry        amd64PDataEntryRelocations
	beginAddend  uint32
	endAddend    uint32
	unwindAddend uint32
	signature    []byte
}

type runtimeFunctionEntry struct {
	begin  uint32
	end    uint32
	unwind uint32
}

// InspectDiscardedAssociativeCOMDATLink proves that both contributed objects define the same
// ANY-COMDAT function with direct associative unwind children, exactly one parent owns the final
// map row, its three .pdata fields reproduce the final runtime-function entry, and no final
// runtime-function references the discarded candidate's distinct plain unwind payload.
func InspectDiscardedAssociativeCOMDATLink(archiveObject, directObject, image, linkMap []byte, spec DiscardedAssociativeCOMDATSpec) (DiscardedAssociativeCOMDATReport, error) {
	if err := validateDiscardedAssociativeCOMDATSpec(spec); err != nil {
		return DiscardedAssociativeCOMDATReport{}, err
	}
	archive, err := inspectAssociativeCOMDATCandidate(archiveObject, spec.SourceSymbol, "archive")
	if err != nil {
		return DiscardedAssociativeCOMDATReport{}, fmt.Errorf("inspect archive associative COMDAT candidate: %w", err)
	}
	direct, err := inspectAssociativeCOMDATCandidate(directObject, spec.SourceSymbol, "direct")
	if err != nil {
		return DiscardedAssociativeCOMDATReport{}, fmt.Errorf("inspect direct associative COMDAT candidate: %w", err)
	}
	if bytes.Equal(archive.signature, direct.signature) {
		return DiscardedAssociativeCOMDATReport{}, fmt.Errorf("associative COMDAT candidates have indistinguishable unwind payloads")
	}

	ownerSpec := ArchiveCOMDATLinkSpec{
		ArchiveName: spec.ArchiveName, SelectedMember: spec.ArchiveMember, DirectObject: spec.DirectObject,
	}
	names := []string{spec.SourceSymbol, archive.entry.UnwindTarget.Name}
	if direct.entry.UnwindTarget.Name != archive.entry.UnwindTarget.Name {
		names = append(names, direct.entry.UnwindTarget.Name)
	}
	mapSymbols, err := inspectMSVCMapSymbols(linkMap, names...)
	if err != nil {
		return DiscardedAssociativeCOMDATReport{}, err
	}
	sourceMap, found := mapSymbols[spec.SourceSymbol]
	if !found {
		return DiscardedAssociativeCOMDATReport{}, fmt.Errorf("MSVC link map has no competing associative COMDAT parent %q", spec.SourceSymbol)
	}
	selectedOwner := classifyMSVCMapOwner(sourceMap.Owner, ownerSpec)
	var selected associativeCOMDATCandidate
	var discarded associativeCOMDATCandidate
	switch selectedOwner {
	case "archive":
		selected, discarded = archive, direct
	case "direct":
		selected, discarded = direct, archive
	default:
		return DiscardedAssociativeCOMDATReport{}, fmt.Errorf("MSVC link map competing associative COMDAT owner is %q, want a declared candidate", sourceMap.Owner)
	}
	discardedOwner := discarded.owner

	unwindMap, found := mapSymbols[selected.entry.UnwindTarget.Name]
	if !found {
		return DiscardedAssociativeCOMDATReport{}, fmt.Errorf("MSVC link map has no selected associative unwind symbol %q", selected.entry.UnwindTarget.Name)
	}
	if classifyMSVCMapOwner(unwindMap.Owner, ownerSpec) != selectedOwner {
		return DiscardedAssociativeCOMDATReport{}, fmt.Errorf("MSVC link map selected associative unwind owner is %q, want %s", unwindMap.Owner, selectedOwner)
	}
	if discarded.entry.UnwindTarget.Name != selected.entry.UnwindTarget.Name {
		if discardedMap, retained := mapSymbols[discarded.entry.UnwindTarget.Name]; retained {
			return DiscardedAssociativeCOMDATReport{}, fmt.Errorf("MSVC link map retained discarded associative unwind symbol %q from %q", discarded.entry.UnwindTarget.Name, discardedMap.Owner)
		}
	}

	imageReport, err := InspectExecutable(image)
	if err != nil {
		return DiscardedAssociativeCOMDATReport{}, fmt.Errorf("inspect discarded associative COMDAT image: %w", err)
	}
	if _, err := InspectAMD64Unwind(image); err != nil {
		return DiscardedAssociativeCOMDATReport{}, fmt.Errorf("inspect discarded associative COMDAT unwind table: %w", err)
	}
	orphanPDataBytes, err := countOrphanPDataBytes(image, imageReport)
	if err != nil {
		return DiscardedAssociativeCOMDATReport{}, err
	}
	if orphanPDataBytes != 0 {
		return DiscardedAssociativeCOMDATReport{}, fmt.Errorf("final image retains %d nonzero .pdata bytes outside the exception directory", orphanPDataBytes)
	}
	parentRVA, err := preferredAddressRVA(imageReport, sourceMap.PreferredAddress, "selected associative COMDAT parent")
	if err != nil {
		return DiscardedAssociativeCOMDATReport{}, err
	}
	expectedBegin, err := addRVA(parentRVA, selected.beginAddend, "selected associative COMDAT begin")
	if err != nil {
		return DiscardedAssociativeCOMDATReport{}, err
	}
	expectedEnd, err := addRVA(parentRVA, selected.endAddend, "selected associative COMDAT end")
	if err != nil {
		return DiscardedAssociativeCOMDATReport{}, err
	}
	unwindBaseRVA, err := preferredAddressRVA(imageReport, unwindMap.PreferredAddress, "selected associative COMDAT unwind")
	if err != nil {
		return DiscardedAssociativeCOMDATReport{}, err
	}
	expectedUnwind, err := addRVA(unwindBaseRVA, selected.unwindAddend, "selected associative COMDAT unwind")
	if err != nil {
		return DiscardedAssociativeCOMDATReport{}, err
	}
	beginRVA, endRVA, unwindRVA, err := exactRuntimeFunctionByBegin(image, imageReport, expectedBegin)
	if err != nil {
		return DiscardedAssociativeCOMDATReport{}, err
	}
	if endRVA != expectedEnd || unwindRVA != expectedUnwind {
		return DiscardedAssociativeCOMDATReport{}, fmt.Errorf("selected associative COMDAT final entry is %#x/%#x/%#x, want %#x/%#x/%#x", beginRVA, endRVA, unwindRVA, expectedBegin, expectedEnd, expectedUnwind)
	}
	selectedFinalSignature, err := plainImageUnwindSignature(image, imageReport, unwindRVA, endRVA-beginRVA)
	if err != nil {
		return DiscardedAssociativeCOMDATReport{}, err
	}
	if !bytes.Equal(selectedFinalSignature, selected.signature) {
		return DiscardedAssociativeCOMDATReport{}, fmt.Errorf("selected associative COMDAT final unwind payload does not match its object")
	}

	entries, err := runtimeFunctionEntries(image, imageReport)
	if err != nil {
		return DiscardedAssociativeCOMDATReport{}, err
	}
	discardedMatches := 0
	for _, entry := range entries {
		signature, err := plainImageUnwindSignature(image, imageReport, entry.unwind, entry.end-entry.begin)
		if err != nil {
			continue
		}
		if bytes.Equal(signature, discarded.signature) {
			discardedMatches++
		}
	}
	if discardedMatches != 0 {
		return DiscardedAssociativeCOMDATReport{}, fmt.Errorf("final image references the discarded associative unwind payload %d times", discardedMatches)
	}
	discardedRawMatches, err := countImageSectionPayload(image, imageReport, discarded.signature)
	if err != nil {
		return DiscardedAssociativeCOMDATReport{}, err
	}
	if discardedRawMatches != 0 {
		return DiscardedAssociativeCOMDATReport{}, fmt.Errorf("final image retains the discarded associative unwind payload %d times", discardedRawMatches)
	}
	return DiscardedAssociativeCOMDATReport{
		SelectedOwner: selectedOwner, DiscardedOwner: discardedOwner,
		SelectedBeginRVA: beginRVA, SelectedEndRVA: endRVA, SelectedUnwindRVA: unwindRVA,
		SelectedUnwindDigest: sha256.Sum256(selected.signature), DiscardedUnwindDigest: sha256.Sum256(discarded.signature),
		DiscardedPayloadMatches: discardedMatches, DiscardedRawMatches: discardedRawMatches, OrphanPDataBytes: orphanPDataBytes,
	}, nil
}

func countOrphanPDataBytes(image []byte, report Report) (int, error) {
	file, err := pe.NewFile(bytes.NewReader(image))
	if err != nil {
		return 0, fmt.Errorf("parse closed .pdata image: %w", err)
	}
	defer file.Close()
	optional, ok := file.OptionalHeader.(*pe.OptionalHeader64)
	if !ok || optional == nil {
		return 0, fmt.Errorf("closed .pdata image has no PE32+ header")
	}
	directory := optional.DataDirectory[exceptionDirectoryIndex]
	var pdata Section
	found := false
	for _, section := range report.Sections {
		if section.Name != ".pdata" {
			continue
		}
		if found {
			return 0, fmt.Errorf("final image repeats .pdata sections")
		}
		pdata = section
		found = true
	}
	if !found || pdata.RawOffset == 0 || pdata.RawSize == 0 || directory.VirtualAddress < pdata.VirtualAddress {
		return 0, fmt.Errorf("final image has no closed file-backed .pdata section")
	}
	directoryWithin := uint64(directory.VirtualAddress - pdata.VirtualAddress)
	directoryEnd := directoryWithin + uint64(directory.Size)
	if directoryEnd > uint64(pdata.RawSize) || uint64(pdata.RawOffset)+uint64(pdata.RawSize) > uint64(len(image)) {
		return 0, fmt.Errorf("final exception directory does not fit its .pdata raw section")
	}
	raw := image[pdata.RawOffset : uint64(pdata.RawOffset)+uint64(pdata.RawSize)]
	orphanBytes := 0
	for index, value := range raw {
		if uint64(index) >= directoryWithin && uint64(index) < directoryEnd {
			continue
		}
		if value != 0 {
			orphanBytes++
		}
	}
	return orphanBytes, nil
}

func countImageSectionPayload(image []byte, report Report, payload []byte) (int, error) {
	if len(payload) < 8 {
		return 0, fmt.Errorf("discarded associative payload is too short for an exact raw-image oracle")
	}
	matches := 0
	for _, section := range report.Sections {
		start := uint64(section.RawOffset)
		end := start + uint64(section.RawSize)
		if section.RawSize == 0 {
			continue
		}
		if start == 0 || end > uint64(len(image)) {
			return 0, fmt.Errorf("final image section %q has an invalid raw range", section.Name)
		}
		remaining := image[start:end]
		for {
			index := bytes.Index(remaining, payload)
			if index < 0 {
				break
			}
			matches++
			remaining = remaining[index+1:]
		}
	}
	return matches, nil
}

func inspectAssociativeCOMDATCandidate(object []byte, sourceSymbol, owner string) (associativeCOMDATCandidate, error) {
	report, err := InspectObject(object)
	if err != nil {
		return associativeCOMDATCandidate{}, err
	}
	source, err := exactObjectSymbol(report.SymbolRecords, sourceSymbol)
	if err != nil {
		return associativeCOMDATCandidate{}, err
	}
	if err := requireDefinedExternalFunction(report, sourceSymbol, "competing associative COMDAT parent"); err != nil {
		return associativeCOMDATCandidate{}, err
	}
	file, err := pe.NewFile(bytes.NewReader(object))
	if err != nil {
		return associativeCOMDATCandidate{}, fmt.Errorf("reparse associative COMDAT candidate: %w", err)
	}
	defer file.Close()
	entry, err := exactAssociativeCOMDATUnwind(report, file, source)
	if err != nil {
		return associativeCOMDATCandidate{}, err
	}
	parent := report.Sections[source.SectionNumber-1]
	if parent.COMDATSelection != ObjectCOMDATAny {
		return associativeCOMDATCandidate{}, fmt.Errorf("competing function parent uses COMDAT selection %d, want ANY", parent.COMDATSelection)
	}
	beginAddend, err := objectRelocationUint32(object, entry.Section, entry.Begin, "begin")
	if err != nil {
		return associativeCOMDATCandidate{}, err
	}
	endAddend, err := objectRelocationUint32(object, entry.Section, entry.End, "end")
	if err != nil {
		return associativeCOMDATCandidate{}, err
	}
	unwindAddend, err := objectRelocationUint32(object, entry.Section, entry.Unwind, "unwind")
	if err != nil {
		return associativeCOMDATCandidate{}, err
	}
	if endAddend <= beginAddend {
		return associativeCOMDATCandidate{}, fmt.Errorf("associative COMDAT candidate has an invalid function span")
	}
	signature, err := plainObjectUnwindSignature(object, report, entry, unwindAddend, endAddend-beginAddend)
	if err != nil {
		return associativeCOMDATCandidate{}, err
	}
	return associativeCOMDATCandidate{
		owner: owner, entry: entry, beginAddend: beginAddend, endAddend: endAddend,
		unwindAddend: unwindAddend, signature: signature,
	}, nil
}

func objectRelocationUint32(object []byte, section *pe.Section, relocation ObjectRelocation, label string) (uint32, error) {
	offset, err := objectRelocationRawOffset(object, section, relocation, 4)
	if err != nil {
		return 0, fmt.Errorf("associative COMDAT %s relocation: %w", label, err)
	}
	return binary.LittleEndian.Uint32(object[offset:]), nil
}

func plainObjectUnwindSignature(object []byte, report ObjectReport, entry amd64PDataEntryRelocations, unwindAddend, functionSize uint32) ([]byte, error) {
	sectionIndex := int(entry.UnwindTarget.SectionNumber)
	if sectionIndex <= 0 || sectionIndex > len(report.Sections) {
		return nil, fmt.Errorf("associative COMDAT unwind target section is invalid")
	}
	section := report.Sections[sectionIndex-1]
	within := uint64(entry.UnwindTarget.Value) + uint64(unwindAddend)
	start := uint64(section.RawOffset) + within
	end := uint64(section.RawOffset) + uint64(section.RawSize)
	if section.RawOffset == 0 || start >= end || end > uint64(len(object)) {
		return nil, fmt.Errorf("associative COMDAT unwind payload range is invalid")
	}
	size, err := plainUnwindSignatureSize(object[start:end], functionSize)
	if err != nil {
		return nil, fmt.Errorf("inspect associative COMDAT object unwind payload: %w", err)
	}
	return bytes.Clone(object[start : start+uint64(size)]), nil
}

func plainImageUnwindSignature(image []byte, report Report, rva, functionSize uint32) ([]byte, error) {
	offset, _, err := mapImageRVA(report, rva, unwindHeaderBytes)
	if err != nil {
		return nil, fmt.Errorf("map plain final unwind payload: %w", err)
	}
	size, err := plainUnwindSignatureSize(image[offset:], functionSize)
	if err != nil {
		return nil, err
	}
	offset, _, err = mapImageRVA(report, rva, uint32(size))
	if err != nil {
		return nil, fmt.Errorf("map complete plain final unwind payload: %w", err)
	}
	return bytes.Clone(image[offset : offset+size]), nil
}

func plainUnwindSignatureSize(data []byte, functionSize uint32) (int, error) {
	if len(data) < unwindHeaderBytes {
		return 0, fmt.Errorf("plain unwind payload is truncated")
	}
	version := data[0] & 0x07
	flags := data[0] >> 3
	prologSize := data[1]
	codeCount := int(data[2])
	frame := data[3]
	if (version != 1 && version != 2) || flags != 0 || uint32(prologSize) > functionSize || frame&0x0f == 0 && frame>>4 != 0 {
		return 0, fmt.Errorf("unwind payload is not plain version-1/2 metadata: version=%d flags=%#x prolog=%d function=%d frame=%#x", version, flags, prologSize, functionSize, frame)
	}
	alignedSlots := codeCount
	if alignedSlots%2 != 0 {
		alignedSlots++
	}
	size := unwindHeaderBytes + alignedSlots*2
	if size > len(data) {
		return 0, fmt.Errorf("plain unwind payload exceeds its data")
	}
	previousCodeOffset := 256
	epilogSeen := false
	for slot := 0; slot < codeCount; {
		codeOffset := int(data[unwindHeaderBytes+slot*2])
		opAndInfo := data[unwindHeaderBytes+slot*2+1]
		op := opAndInfo & 0x0f
		opInfo := opAndInfo >> 4
		if version == 2 && op == 6 {
			consumed := 1
			if !epilogSeen && opInfo&1 == 0 {
				consumed = 2
			}
			if slot+consumed > codeCount {
				return 0, fmt.Errorf("plain unwind payload has a truncated version-2 epilog descriptor")
			}
			epilogSeen = true
			slot += consumed
			continue
		}
		if version == 2 && op == 7 {
			slot++
			continue
		}
		if codeOffset > int(prologSize) || codeOffset > previousCodeOffset {
			return 0, fmt.Errorf("plain unwind payload has unsorted operations")
		}
		consumed, err := unwindOperationSlots(op, opInfo)
		if err != nil || slot+consumed > codeCount {
			return 0, fmt.Errorf("plain unwind payload has an invalid operation")
		}
		previousCodeOffset = codeOffset
		slot += consumed
	}
	return size, nil
}

func runtimeFunctionEntries(image []byte, report Report) ([]runtimeFunctionEntry, error) {
	file, err := pe.NewFile(bytes.NewReader(image))
	if err != nil {
		return nil, fmt.Errorf("parse runtime-function image: %w", err)
	}
	defer file.Close()
	optional, ok := file.OptionalHeader.(*pe.OptionalHeader64)
	if !ok || optional == nil {
		return nil, fmt.Errorf("runtime-function image has no PE32+ header")
	}
	directory := optional.DataDirectory[exceptionDirectoryIndex]
	offset, _, err := mapImageRVA(report, directory.VirtualAddress, directory.Size)
	if err != nil {
		return nil, fmt.Errorf("map runtime-function directory: %w", err)
	}
	entries := make([]runtimeFunctionEntry, 0, directory.Size/runtimeFunctionBytes)
	for position := uint32(0); position < directory.Size; position += runtimeFunctionBytes {
		entryOffset := offset + int(position)
		entries = append(entries, runtimeFunctionEntry{
			begin:  binary.LittleEndian.Uint32(image[entryOffset:]),
			end:    binary.LittleEndian.Uint32(image[entryOffset+4:]),
			unwind: binary.LittleEndian.Uint32(image[entryOffset+8:]),
		})
	}
	return entries, nil
}

func validateDiscardedAssociativeCOMDATSpec(spec DiscardedAssociativeCOMDATSpec) error {
	if spec.ArchiveName == "" || spec.ArchiveMember == "" || spec.DirectObject == "" || spec.SourceSymbol == "" ||
		filepath.Base(spec.ArchiveName) != spec.ArchiveName || !strings.EqualFold(filepath.Ext(spec.ArchiveName), ".lib") ||
		filepath.Base(spec.ArchiveMember) != spec.ArchiveMember || !strings.EqualFold(filepath.Ext(spec.ArchiveMember), ".obj") ||
		filepath.Base(spec.DirectObject) != spec.DirectObject || !strings.EqualFold(filepath.Ext(spec.DirectObject), ".obj") ||
		strings.EqualFold(spec.ArchiveMember, spec.DirectObject) || strings.ContainsAny(spec.SourceSymbol, " \t\r\n") {
		return fmt.Errorf("discarded associative COMDAT spec has invalid identities")
	}
	return nil
}
