package pecoff

import (
	"bytes"
	"debug/pe"
	"encoding/binary"
	"fmt"
	"math"
)

// CodeViewSectionOffsetPair records one adjacent SECREL/SECTION relocation pair in `.debug$S`.
// Resolved values are the object-level COFF formulas, not values observed in a PDB stream.
type CodeViewSectionOffsetPair struct {
	Symbol          string
	SymbolSection   uint16
	SymbolValue     uint32
	SecRelSite      uint32
	SectionSite     uint32
	SecRelAddend    uint32
	SectionAddend   uint16
	ResolvedOffset  uint32
	ResolvedSection uint16
}

// CodeViewRelocationReport records every closed `.debug$S` SECREL/SECTION pair in one object.
type CodeViewRelocationReport struct {
	Pairs []CodeViewSectionOffsetPair
}

// InspectCodeViewRelocations validates adjacent AMD64 SECREL/SECTION pairs and independently
// computes symbol-section-relative values from the raw object addends.
func InspectCodeViewRelocations(object []byte) (CodeViewRelocationReport, error) {
	objectReport, err := InspectObject(object)
	if err != nil {
		return CodeViewRelocationReport{}, fmt.Errorf("inspect CodeView COFF object: %w", err)
	}
	file, err := pe.NewFile(bytes.NewReader(object))
	if err != nil {
		return CodeViewRelocationReport{}, fmt.Errorf("reparse CodeView COFF object: %w", err)
	}
	defer file.Close()
	symbols := make(map[uint32]ObjectSymbol, len(objectReport.SymbolRecords))
	for _, symbol := range objectReport.SymbolRecords {
		symbols[symbol.TableIndex] = symbol
	}
	sectionRelocations := make(map[[3]uint32][]ObjectRelocation)
	secrelCount := 0
	sectionCount := 0
	for _, relocation := range objectReport.RelocationRecords {
		if relocation.Section != ".debug$S" {
			continue
		}
		switch relocation.Type {
		case ObjectRelocationAMD64SecRel:
			secrelCount++
		case ObjectRelocationAMD64Section:
			sectionCount++
			key := [3]uint32{uint32(relocation.SectionIndex), relocation.SymbolIndex, relocation.VirtualAddress}
			sectionRelocations[key] = append(sectionRelocations[key], relocation)
		}
	}
	if secrelCount == 0 || sectionCount == 0 {
		return CodeViewRelocationReport{}, fmt.Errorf("CodeView object has %d SECREL and %d SECTION relocations", secrelCount, sectionCount)
	}
	pairs := make([]CodeViewSectionOffsetPair, 0, secrelCount)
	usedSections := make(map[[3]uint32]struct{}, sectionCount)
	for _, relocation := range objectReport.RelocationRecords {
		if relocation.Section != ".debug$S" || relocation.Type != ObjectRelocationAMD64SecRel {
			continue
		}
		if relocation.VirtualAddress > math.MaxUint32-4 {
			return CodeViewRelocationReport{}, fmt.Errorf("CodeView SECREL for %q overflows its adjacent SECTION site", relocation.Symbol)
		}
		sectionKey := [3]uint32{uint32(relocation.SectionIndex), relocation.SymbolIndex, relocation.VirtualAddress + 4}
		matches := sectionRelocations[sectionKey]
		if len(matches) != 1 {
			return CodeViewRelocationReport{}, fmt.Errorf("CodeView SECREL for %q at %#x has %d adjacent SECTION relocations", relocation.Symbol, relocation.VirtualAddress, len(matches))
		}
		if _, duplicate := usedSections[sectionKey]; duplicate {
			return CodeViewRelocationReport{}, fmt.Errorf("CodeView SECTION for %q at %#x is reused", relocation.Symbol, relocation.VirtualAddress+4)
		}
		usedSections[sectionKey] = struct{}{}
		symbol, found := symbols[relocation.SymbolIndex]
		if !found || symbol.SectionNumber <= 0 || int(symbol.SectionNumber) > len(file.Sections) {
			return CodeViewRelocationReport{}, fmt.Errorf("CodeView pair target %q is not defined in an object section", relocation.Symbol)
		}
		debugSection := file.Sections[relocation.SectionIndex-1]
		secrelOffset, err := objectSectionFileOffset(debugSection, relocation.VirtualAddress, 4, uint64(len(object)))
		if err != nil {
			return CodeViewRelocationReport{}, fmt.Errorf("map CodeView SECREL for %q: %w", relocation.Symbol, err)
		}
		sectionOffset, err := objectSectionFileOffset(debugSection, relocation.VirtualAddress+4, 2, uint64(len(object)))
		if err != nil {
			return CodeViewRelocationReport{}, fmt.Errorf("map CodeView SECTION for %q: %w", relocation.Symbol, err)
		}
		secrelAddend := binary.LittleEndian.Uint32(object[secrelOffset:])
		sectionAddend := binary.LittleEndian.Uint16(object[sectionOffset:])
		resolvedOffset := uint64(symbol.Value) + uint64(secrelAddend)
		resolvedSection := uint64(uint16(symbol.SectionNumber)) + uint64(sectionAddend)
		if resolvedOffset > math.MaxUint32 || resolvedSection == 0 || resolvedSection > math.MaxUint16 {
			return CodeViewRelocationReport{}, fmt.Errorf("CodeView pair for %q overflows its resolved section:offset", relocation.Symbol)
		}
		if resolvedSection > uint64(len(file.Sections)) {
			return CodeViewRelocationReport{}, fmt.Errorf("CodeView pair for %q resolves to section %d outside the object", relocation.Symbol, resolvedSection)
		}
		resolvedTarget := file.Sections[resolvedSection-1]
		if resolvedOffset >= uint64(resolvedTarget.Size) {
			return CodeViewRelocationReport{}, fmt.Errorf("CodeView pair for %q resolves to offset %#x outside section %q", relocation.Symbol, resolvedOffset, resolvedTarget.Name)
		}
		pairs = append(pairs, CodeViewSectionOffsetPair{
			Symbol: symbol.Name, SymbolSection: uint16(symbol.SectionNumber), SymbolValue: symbol.Value,
			SecRelSite: relocation.VirtualAddress, SectionSite: relocation.VirtualAddress + 4,
			SecRelAddend: secrelAddend, SectionAddend: sectionAddend,
			ResolvedOffset: uint32(resolvedOffset), ResolvedSection: uint16(resolvedSection),
		})
	}
	if len(usedSections) != sectionCount {
		return CodeViewRelocationReport{}, fmt.Errorf("CodeView object has %d unpaired SECTION relocations", sectionCount-len(usedSections))
	}
	return CodeViewRelocationReport{Pairs: pairs}, nil
}

func objectSectionFileOffset(section *pe.Section, virtualAddress, width uint32, objectSize uint64) (int, error) {
	if virtualAddress < section.VirtualAddress {
		return 0, fmt.Errorf("relocation site precedes section virtual address")
	}
	sectionOffset := uint64(virtualAddress - section.VirtualAddress)
	if sectionOffset+uint64(width) > uint64(section.Size) {
		return 0, fmt.Errorf("relocation site exceeds section bytes")
	}
	fileOffset := uint64(section.Offset) + sectionOffset
	if fileOffset+uint64(width) > objectSize || fileOffset > math.MaxInt {
		return 0, fmt.Errorf("relocation site exceeds object bytes")
	}
	return int(fileOffset), nil
}
