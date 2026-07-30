package pecoff

import (
	"bytes"
	"debug/pe"
	"encoding/binary"
	"fmt"
	"path/filepath"
	"strings"
)

// AssociativeCOMDATUnwindSpec identifies one selected archive function and its owning archive.
type AssociativeCOMDATUnwindSpec struct {
	ArchiveName  string
	SourceMember string
	SourceSymbol string
}

// AssociativeCOMDATUnwindReport binds one function COMDAT to its exact .pdata and .xdata children
// and to the selected function's final runtime-function entry.
type AssociativeCOMDATUnwindReport struct {
	ParentSectionIndex int
	PDataSectionIndex  int
	XDataSectionIndex  int
	BeginTarget        string
	UnwindTarget       string
	BeginAddend        uint32
	EndAddend          uint32
	UnwindAddend       uint32
	FinalBeginRVA      uint32
	FinalEndRVA        uint32
	FinalUnwindRVA     uint32
	ExpectedEndRVA     uint32
	ExpectedUnwindRVA  uint32
}

// InspectAssociativeCOMDATUnwindLink proves that one map-owned selected function comes from a
// non-associative COMDAT parent whose exact .pdata entry and referenced .xdata section are direct
// associative children, then binds all three object fields to the final exception directory.
func InspectAssociativeCOMDATUnwindLink(object, image, linkMap []byte, spec AssociativeCOMDATUnwindSpec) (AssociativeCOMDATUnwindReport, error) {
	if err := validateAssociativeCOMDATUnwindSpec(spec); err != nil {
		return AssociativeCOMDATUnwindReport{}, err
	}
	sourceSymbol := spec.SourceSymbol
	if sourceSymbol == "" {
		return AssociativeCOMDATUnwindReport{}, fmt.Errorf("associative COMDAT unwind identity is invalid")
	}
	objectReport, err := InspectObject(object)
	if err != nil {
		return AssociativeCOMDATUnwindReport{}, fmt.Errorf("inspect associative COMDAT unwind object: %w", err)
	}
	source, err := exactObjectSymbol(objectReport.SymbolRecords, sourceSymbol)
	if err != nil {
		return AssociativeCOMDATUnwindReport{}, err
	}
	if err := requireDefinedExternalFunction(objectReport, sourceSymbol, "associative COMDAT parent"); err != nil {
		return AssociativeCOMDATUnwindReport{}, err
	}
	file, err := pe.NewFile(bytes.NewReader(object))
	if err != nil {
		return AssociativeCOMDATUnwindReport{}, fmt.Errorf("reparse associative COMDAT unwind object: %w", err)
	}
	defer file.Close()
	entry, err := exactAssociativeCOMDATUnwind(objectReport, file, source)
	if err != nil {
		return AssociativeCOMDATUnwindReport{}, err
	}
	beginOffset, err := objectRelocationRawOffset(object, entry.Section, entry.Begin, 4)
	if err != nil {
		return AssociativeCOMDATUnwindReport{}, fmt.Errorf("associative COMDAT .pdata begin: %w", err)
	}
	endOffset, err := objectRelocationRawOffset(object, entry.Section, entry.End, 4)
	if err != nil {
		return AssociativeCOMDATUnwindReport{}, fmt.Errorf("associative COMDAT .pdata end: %w", err)
	}
	unwindOffset, err := objectRelocationRawOffset(object, entry.Section, entry.Unwind, 4)
	if err != nil {
		return AssociativeCOMDATUnwindReport{}, fmt.Errorf("associative COMDAT .pdata unwind-info: %w", err)
	}
	beginAddend := binary.LittleEndian.Uint32(object[beginOffset:])
	endAddend := binary.LittleEndian.Uint32(object[endOffset:])
	unwindAddend := binary.LittleEndian.Uint32(object[unwindOffset:])

	imageReport, err := InspectExecutable(image)
	if err != nil {
		return AssociativeCOMDATUnwindReport{}, fmt.Errorf("inspect associative COMDAT unwind image: %w", err)
	}
	if _, err := InspectAMD64Unwind(image); err != nil {
		return AssociativeCOMDATUnwindReport{}, fmt.Errorf("inspect associative COMDAT final unwind table: %w", err)
	}
	mapSymbols, err := inspectMSVCMapSymbols(linkMap, sourceSymbol, entry.UnwindTarget.Name)
	if err != nil {
		return AssociativeCOMDATUnwindReport{}, err
	}
	sourceMap, found := mapSymbols[sourceSymbol]
	if !found {
		return AssociativeCOMDATUnwindReport{}, fmt.Errorf("MSVC link map has no associative COMDAT parent %q", sourceSymbol)
	}
	ownerSpec := ArchiveCOMDATLinkSpec{ArchiveName: spec.ArchiveName, SelectedMember: spec.SourceMember}
	if classifyMSVCMapOwner(sourceMap.Owner, ownerSpec) != "archive" {
		return AssociativeCOMDATUnwindReport{}, fmt.Errorf("MSVC link map associative COMDAT parent owner is %q, want %s:%s", sourceMap.Owner, spec.ArchiveName, spec.SourceMember)
	}
	sourceRVA, err := preferredAddressRVA(imageReport, sourceMap.PreferredAddress, "associative COMDAT parent")
	if err != nil {
		return AssociativeCOMDATUnwindReport{}, err
	}
	expectedBegin, err := addRVA(sourceRVA, beginAddend, "associative COMDAT .pdata begin")
	if err != nil {
		return AssociativeCOMDATUnwindReport{}, err
	}
	expectedEnd, err := addRVA(sourceRVA, endAddend, "associative COMDAT .pdata end")
	if err != nil {
		return AssociativeCOMDATUnwindReport{}, err
	}
	unwindMap, found := mapSymbols[entry.UnwindTarget.Name]
	if !found {
		return AssociativeCOMDATUnwindReport{}, fmt.Errorf("MSVC link map has no associative COMDAT unwind-info symbol %q", entry.UnwindTarget.Name)
	}
	if classifyMSVCMapOwner(unwindMap.Owner, ownerSpec) != "archive" {
		return AssociativeCOMDATUnwindReport{}, fmt.Errorf("MSVC link map associative COMDAT unwind-info owner is %q, want %s:%s", unwindMap.Owner, spec.ArchiveName, spec.SourceMember)
	}
	unwindBaseRVA, err := preferredAddressRVA(imageReport, unwindMap.PreferredAddress, "associative COMDAT unwind-info")
	if err != nil {
		return AssociativeCOMDATUnwindReport{}, err
	}
	expectedUnwind, err := addRVA(unwindBaseRVA, unwindAddend, "associative COMDAT .pdata unwind-info")
	if err != nil {
		return AssociativeCOMDATUnwindReport{}, err
	}
	beginRVA, endRVA, unwindRVA, err := exactRuntimeFunctionByBegin(image, imageReport, expectedBegin)
	if err != nil {
		return AssociativeCOMDATUnwindReport{}, err
	}
	if endRVA != expectedEnd {
		return AssociativeCOMDATUnwindReport{}, fmt.Errorf("associative COMDAT final end RVA is %#x, want %#x", endRVA, expectedEnd)
	}
	if unwindRVA != expectedUnwind {
		return AssociativeCOMDATUnwindReport{}, fmt.Errorf("associative COMDAT final unwind-info RVA is %#x, want %#x", unwindRVA, expectedUnwind)
	}
	if endRVA <= beginRVA || unwindRVA == 0 {
		return AssociativeCOMDATUnwindReport{}, fmt.Errorf("associative COMDAT final runtime-function entry is incomplete")
	}
	return AssociativeCOMDATUnwindReport{
		ParentSectionIndex: int(source.SectionNumber), PDataSectionIndex: entry.Begin.SectionIndex,
		XDataSectionIndex: int(entry.UnwindTarget.SectionNumber), BeginTarget: entry.BeginTarget.Name,
		UnwindTarget: entry.UnwindTarget.Name, BeginAddend: beginAddend, EndAddend: endAddend,
		UnwindAddend: unwindAddend, FinalBeginRVA: beginRVA, FinalEndRVA: endRVA,
		FinalUnwindRVA: unwindRVA, ExpectedEndRVA: expectedEnd, ExpectedUnwindRVA: expectedUnwind,
	}, nil
}

func validateAssociativeCOMDATUnwindSpec(spec AssociativeCOMDATUnwindSpec) error {
	if spec.ArchiveName == "" || spec.SourceMember == "" || spec.SourceSymbol == "" ||
		filepath.Base(spec.ArchiveName) != spec.ArchiveName || filepath.Ext(spec.ArchiveName) != ".lib" ||
		filepath.Base(spec.SourceMember) != spec.SourceMember || filepath.Ext(spec.SourceMember) != ".obj" ||
		strings.ContainsAny(spec.ArchiveName+spec.SourceMember, "\\/") || strings.ContainsAny(spec.SourceSymbol, " \t\r\n") {
		return fmt.Errorf("associative COMDAT unwind identity is invalid")
	}
	return nil
}

func exactAssociativeCOMDATUnwind(report ObjectReport, file *pe.File, source ObjectSymbol) (amd64PDataEntryRelocations, error) {
	if source.SectionNumber <= 0 || int(source.SectionNumber) > len(report.Sections) {
		return amd64PDataEntryRelocations{}, fmt.Errorf("associative COMDAT parent is undefined")
	}
	parent := report.Sections[source.SectionNumber-1]
	if parent.COMDATSelection == 0 || parent.COMDATSelection == ObjectCOMDATAssociative ||
		parent.Characteristics&sectionLinkCOMDAT == 0 ||
		parent.Characteristics&(sectionContainsCode|sectionExecutable|sectionReadable) != sectionContainsCode|sectionExecutable|sectionReadable {
		return amd64PDataEntryRelocations{}, fmt.Errorf("associative COMDAT unwind parent is not an executable root COMDAT")
	}
	entry, err := exactPDataEntryRelocations(report, file, source)
	if err != nil {
		return amd64PDataEntryRelocations{}, err
	}
	pdata := report.Sections[entry.Begin.SectionIndex-1]
	if pdata.COMDATSelection != ObjectCOMDATAssociative || pdata.AssociativeSection != int(source.SectionNumber) ||
		pdata.Characteristics&sectionLinkCOMDAT == 0 {
		return amd64PDataEntryRelocations{}, fmt.Errorf("source .pdata is not a direct associative child of its function COMDAT")
	}
	xdata := report.Sections[entry.UnwindTarget.SectionNumber-1]
	if xdata.COMDATSelection != ObjectCOMDATAssociative || xdata.AssociativeSection != int(source.SectionNumber) ||
		xdata.Characteristics&sectionLinkCOMDAT == 0 {
		return amd64PDataEntryRelocations{}, fmt.Errorf("source .xdata is not a direct associative child of its function COMDAT")
	}
	if entry.Begin.SectionIndex == int(entry.UnwindTarget.SectionNumber) || entry.Begin.SectionIndex == int(source.SectionNumber) ||
		entry.UnwindTarget.SectionNumber == source.SectionNumber {
		return amd64PDataEntryRelocations{}, fmt.Errorf("associative COMDAT unwind parent and children are not distinct sections")
	}
	return entry, nil
}
