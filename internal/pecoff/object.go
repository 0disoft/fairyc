package pecoff

import (
	"bytes"
	"debug/pe"
	"fmt"
	"strings"
)

const (
	coffFileHeaderBytes       = 20
	coffSectionHeaderBytes    = 40
	coffRelocationBytes       = 10
	maxObjectSections         = 4096
	sectionRelocationOverflow = 0x01000000
	sectionLinkCOMDAT         = 0x00001000
	coffStorageClassStatic    = 3
	coffCOMDATNoDuplicates    = 1
	coffCOMDATAny             = 2
	coffCOMDATSameSize        = 3
	coffCOMDATExactMatch      = 4
	coffCOMDATAssociative     = 5
	coffCOMDATLargest         = 6
)

const (
	ObjectRelocationAMD64Absolute uint16 = 0x0000
	ObjectRelocationAMD64Addr64   uint16 = 0x0001
	ObjectRelocationAMD64Addr32   uint16 = 0x0002
	ObjectRelocationAMD64Addr32NB uint16 = 0x0003
	ObjectRelocationAMD64Rel32    uint16 = 0x0004
	ObjectRelocationAMD64Rel32_1  uint16 = 0x0005
	ObjectRelocationAMD64Rel32_2  uint16 = 0x0006
	ObjectRelocationAMD64Rel32_3  uint16 = 0x0007
	ObjectRelocationAMD64Rel32_4  uint16 = 0x0008
	ObjectRelocationAMD64Rel32_5  uint16 = 0x0009
	ObjectRelocationAMD64Section  uint16 = 0x000a
	ObjectRelocationAMD64SecRel   uint16 = 0x000b
	ObjectRelocationAMD64SecRel7  uint16 = 0x000c
	ObjectRelocationAMD64Token    uint16 = 0x000d
	ObjectRelocationAMD64SRel32   uint16 = 0x000e
	ObjectRelocationAMD64Pair     uint16 = 0x000f
	ObjectRelocationAMD64SSpan32  uint16 = 0x0010
	ObjectCOMDATNoDuplicates      uint8  = coffCOMDATNoDuplicates
	ObjectCOMDATAny               uint8  = coffCOMDATAny
	ObjectCOMDATSameSize          uint8  = coffCOMDATSameSize
	ObjectCOMDATExactMatch        uint8  = coffCOMDATExactMatch
	ObjectCOMDATAssociative       uint8  = coffCOMDATAssociative
	ObjectCOMDATLargest           uint8  = coffCOMDATLargest
)

// ObjectSection is one bounded COFF object section receipt.
type ObjectSection struct {
	Name               string
	RawOffset          uint32
	RawSize            uint32
	Relocations        int
	Characteristics    uint32
	COMDATSelection    uint8
	AssociativeSection int
}

// ObjectSymbol is one primary COFF symbol receipt. Auxiliary records stay structural metadata and
// cannot be relocation targets.
type ObjectSymbol struct {
	TableIndex       uint32
	Name             string
	Value            uint32
	SectionNumber    int16
	Type             uint16
	StorageClass     uint8
	AuxiliaryRecords uint8
}

// ObjectRelocation identifies one object-section relocation and its primary symbol target.
type ObjectRelocation struct {
	SectionIndex   int
	Section        string
	VirtualAddress uint32
	SymbolIndex    uint32
	Symbol         string
	Type           uint16
}

// ObjectReport records the closed COFF object facts consumed by the Windows C driver.
type ObjectReport struct {
	Machine           uint16
	Characteristics   uint16
	Sections          []ObjectSection
	Symbols           []string
	SymbolRecords     []ObjectSymbol
	RelocationRecords []ObjectRelocation
	Relocations       int
}

// InspectObject parses and validates one bounded AMD64 COFF object file.
func InspectObject(object []byte) (ObjectReport, error) {
	if len(object) < coffFileHeaderBytes || len(object) > maxExecutableBytes {
		return ObjectReport{}, fmt.Errorf("COFF object has invalid size %d", len(object))
	}
	if len(object) >= 2 && object[0] == 'M' && object[1] == 'Z' {
		return ObjectReport{}, fmt.Errorf("COFF object must not be a linked PE image")
	}
	file, err := pe.NewFile(bytes.NewReader(object))
	if err != nil {
		return ObjectReport{}, fmt.Errorf("parse COFF object: %w", err)
	}
	defer file.Close()
	if file.FileHeader.Machine != machineAMD64 {
		return ObjectReport{}, fmt.Errorf("COFF object machine is %#x, want AMD64", file.FileHeader.Machine)
	}
	if file.FileHeader.SizeOfOptionalHeader != 0 || file.OptionalHeader != nil {
		return ObjectReport{}, fmt.Errorf("COFF object unexpectedly has a PE optional header")
	}
	if file.FileHeader.Characteristics&executableImage != 0 || file.FileHeader.Characteristics&dynamicLibrary != 0 {
		return ObjectReport{}, fmt.Errorf("COFF object has linked-image characteristics %#x", file.FileHeader.Characteristics)
	}
	if len(file.Sections) == 0 || len(file.Sections) > maxObjectSections {
		return ObjectReport{}, fmt.Errorf("COFF object has invalid section count %d", len(file.Sections))
	}
	headerEnd := uint64(coffFileHeaderBytes + len(file.Sections)*coffSectionHeaderBytes)
	if headerEnd > uint64(len(object)) {
		return ObjectReport{}, fmt.Errorf("COFF object section table exceeds the file")
	}
	report := ObjectReport{
		Machine:         file.FileHeader.Machine,
		Characteristics: file.FileHeader.Characteristics,
		Sections:        make([]ObjectSection, 0, len(file.Sections)),
		Symbols:         make([]string, 0, len(file.Symbols)),
		SymbolRecords:   make([]ObjectSymbol, 0, len(file.Symbols)),
	}
	primarySymbols, err := inspectPrimaryCOFFSymbols(file)
	if err != nil {
		return ObjectReport{}, err
	}
	report.SymbolRecords = primarySymbols.records
	for _, symbol := range report.SymbolRecords {
		report.Symbols = append(report.Symbols, symbol.Name)
	}
	comdatBySection, err := inspectCOFFCOMDAT(file, primarySymbols)
	if err != nil {
		return ObjectReport{}, err
	}
	textSectionFound := false
	rawRanges := make([][2]uint64, 0, len(file.Sections))
	for index, section := range file.Sections {
		if section.Name == "" {
			return ObjectReport{}, fmt.Errorf("COFF object section %d has no name", index)
		}
		if section.Characteristics&sectionRelocationOverflow != 0 {
			return ObjectReport{}, fmt.Errorf("COFF object section %q uses unsupported relocation overflow", section.Name)
		}
		if section.Size > 0 {
			if section.Offset == 0 {
				if section.Characteristics&sectionUninitializedData == 0 {
					return ObjectReport{}, fmt.Errorf("COFF object section %q has an invalid zero raw pointer", section.Name)
				}
			} else {
				start := uint64(section.Offset)
				end := start + uint64(section.Size)
				if start < headerEnd || end > uint64(len(object)) {
					return ObjectReport{}, fmt.Errorf("COFF object section %q has an invalid raw range", section.Name)
				}
				for _, existing := range rawRanges {
					if start < existing[1] && existing[0] < end {
						return ObjectReport{}, fmt.Errorf("COFF object section %q overlaps another raw range", section.Name)
					}
				}
				rawRanges = append(rawRanges, [2]uint64{start, end})
			}
		}
		if len(section.Relocs) > 0 {
			start := uint64(section.PointerToRelocations)
			end := start + uint64(len(section.Relocs))*coffRelocationBytes
			if start < headerEnd || end > uint64(len(object)) {
				return ObjectReport{}, fmt.Errorf("COFF object section %q has an invalid relocation range", section.Name)
			}
		}
		for _, relocation := range section.Relocs {
			if !validAMD64RelocationType(relocation.Type) {
				return ObjectReport{}, fmt.Errorf("COFF object section %q has unknown AMD64 relocation type %#x", section.Name, relocation.Type)
			}
			symbol, found := primarySymbols.byTableIndex[relocation.SymbolTableIndex]
			if !found {
				if uint64(relocation.SymbolTableIndex) >= uint64(len(file.COFFSymbols)) {
					return ObjectReport{}, fmt.Errorf("COFF object section %q relocation references missing symbol %d", section.Name, relocation.SymbolTableIndex)
				}
				return ObjectReport{}, fmt.Errorf("COFF object section %q relocation references auxiliary symbol %d", section.Name, relocation.SymbolTableIndex)
			}
			if symbol.Name == "" {
				return ObjectReport{}, fmt.Errorf("COFF object section %q relocation references missing symbol %d", section.Name, relocation.SymbolTableIndex)
			}
			report.RelocationRecords = append(report.RelocationRecords, ObjectRelocation{
				SectionIndex: index + 1, Section: section.Name, VirtualAddress: relocation.VirtualAddress,
				SymbolIndex: relocation.SymbolTableIndex, Symbol: symbol.Name, Type: relocation.Type,
			})
		}
		containsCode := section.Characteristics&sectionContainsCode != 0
		executable := section.Characteristics&sectionExecutable != 0
		readable := section.Characteristics&sectionReadable != 0
		if isCOFFTextSection(section.Name) {
			if !containsCode || !executable || !readable {
				return ObjectReport{}, fmt.Errorf("COFF object .text section is not readable executable code")
			}
			textSectionFound = true
		}
		sectionReceipt := ObjectSection{
			Name:            section.Name,
			RawOffset:       section.Offset,
			RawSize:         section.Size,
			Relocations:     len(section.Relocs),
			Characteristics: section.Characteristics,
		}
		if comdat, found := comdatBySection[index+1]; found {
			sectionReceipt.COMDATSelection = comdat.Selection
			sectionReceipt.AssociativeSection = int(comdat.SecNum)
		}
		report.Relocations += len(section.Relocs)
		report.Sections = append(report.Sections, sectionReceipt)
	}
	if !textSectionFound {
		return ObjectReport{}, fmt.Errorf("COFF object has no .text or .text$ code section")
	}
	return report, nil
}

func validAMD64RelocationType(value uint16) bool {
	return value >= ObjectRelocationAMD64Absolute && value <= ObjectRelocationAMD64SSpan32
}

type inspectedCOFFSymbols struct {
	records      []ObjectSymbol
	byTableIndex map[uint32]ObjectSymbol
}

func inspectPrimaryCOFFSymbols(file *pe.File) (inspectedCOFFSymbols, error) {
	if len(file.Symbols) == 0 || len(file.COFFSymbols) == 0 {
		return inspectedCOFFSymbols{}, fmt.Errorf("COFF object has no symbol table")
	}
	result := inspectedCOFFSymbols{
		records:      make([]ObjectSymbol, 0, len(file.Symbols)),
		byTableIndex: make(map[uint32]ObjectSymbol, len(file.Symbols)),
	}
	for index := 0; index < len(file.COFFSymbols); {
		symbol := &file.COFFSymbols[index]
		name, err := symbol.FullName(file.StringTable)
		if err != nil {
			return inspectedCOFFSymbols{}, fmt.Errorf("read COFF object symbol %d name: %w", index, err)
		}
		if name == "" {
			return inspectedCOFFSymbols{}, fmt.Errorf("COFF object symbol %d has no name", index)
		}
		auxiliaryEnd := index + 1 + int(symbol.NumberOfAuxSymbols)
		if auxiliaryEnd > len(file.COFFSymbols) {
			return inspectedCOFFSymbols{}, fmt.Errorf("COFF object symbol %q auxiliary records exceed the symbol table", name)
		}
		record := ObjectSymbol{
			TableIndex: uint32(index), Name: name, Value: symbol.Value,
			SectionNumber: symbol.SectionNumber, Type: symbol.Type,
			StorageClass: symbol.StorageClass, AuxiliaryRecords: symbol.NumberOfAuxSymbols,
		}
		result.records = append(result.records, record)
		result.byTableIndex[uint32(index)] = record
		index = auxiliaryEnd
	}
	return result, nil
}

func inspectCOFFCOMDAT(file *pe.File, symbols inspectedCOFFSymbols) (map[int]pe.COFFSymbolAuxFormat5, error) {
	result := make(map[int]pe.COFFSymbolAuxFormat5)
	firstPrimaryBySection := make(map[int]int, len(file.Sections))
	primaryCountBySection := make(map[int]int, len(file.Sections))
	for position, symbol := range symbols.records {
		if symbol.SectionNumber <= 0 || int(symbol.SectionNumber) > len(file.Sections) {
			continue
		}
		sectionNumber := int(symbol.SectionNumber)
		primaryCountBySection[sectionNumber]++
		if _, found := firstPrimaryBySection[sectionNumber]; !found {
			firstPrimaryBySection[sectionNumber] = position
		}
		if symbol.StorageClass != coffStorageClassStatic || symbol.Name != file.Sections[sectionNumber-1].Name ||
			symbol.Value != 0 || symbol.Type != 0 {
			continue
		}
		if symbol.AuxiliaryRecords != 1 {
			return nil, fmt.Errorf("COFF object section %q definition has %d auxiliary records, want exactly 1", symbol.Name, symbol.AuxiliaryRecords)
		}
		section := file.Sections[symbol.SectionNumber-1]
		auxiliary, err := file.COFFSymbolReadSectionDefAux(int(symbol.TableIndex))
		if err != nil {
			return nil, fmt.Errorf("read COFF object section %q definition: %w", section.Name, err)
		}
		if _, duplicate := result[sectionNumber]; duplicate {
			return nil, fmt.Errorf("COFF object section %q repeats its section definition", section.Name)
		}
		if auxiliary.Size != section.Size || int(auxiliary.NumRelocs) != len(section.Relocs) ||
			auxiliary.NumLineNumbers != section.NumberOfLineNumbers {
			return nil, fmt.Errorf("COFF object section %q definition does not match its header", section.Name)
		}
		isCOMDAT := section.Characteristics&sectionLinkCOMDAT != 0
		if !isCOMDAT {
			if auxiliary.Selection != 0 || auxiliary.SecNum != 0 {
				return nil, fmt.Errorf("COFF object non-COMDAT section %q has selection %d and associative parent %d", section.Name, auxiliary.Selection, auxiliary.SecNum)
			}
			continue
		}
		if auxiliary.Selection < coffCOMDATNoDuplicates || auxiliary.Selection > coffCOMDATLargest {
			return nil, fmt.Errorf("COFF object COMDAT section %q has invalid selection %d", section.Name, auxiliary.Selection)
		}
		if auxiliary.Selection == coffCOMDATAssociative {
			if auxiliary.SecNum == 0 || int(auxiliary.SecNum) > len(file.Sections) || auxiliary.SecNum == uint16(symbol.SectionNumber) {
				return nil, fmt.Errorf("COFF object associative COMDAT section %q has invalid parent %d", section.Name, auxiliary.SecNum)
			}
		} else if auxiliary.SecNum != 0 {
			return nil, fmt.Errorf("COFF object COMDAT section %q has unexpected associative parent %d", section.Name, auxiliary.SecNum)
		}
		result[sectionNumber] = *auxiliary
	}
	for index, section := range file.Sections {
		if section.Characteristics&sectionLinkCOMDAT == 0 {
			continue
		}
		sectionNumber := index + 1
		if _, found := result[sectionNumber]; !found {
			return nil, fmt.Errorf("COFF object COMDAT section %q has no exact section definition", section.Name)
		}
		position, found := firstPrimaryBySection[sectionNumber]
		if !found || symbols.records[position].Name != section.Name ||
			symbols.records[position].StorageClass != coffStorageClassStatic ||
			symbols.records[position].Value != 0 || symbols.records[position].Type != 0 ||
			symbols.records[position].AuxiliaryRecords != 1 {
			return nil, fmt.Errorf("COFF object COMDAT section %q definition is not its first primary symbol", section.Name)
		}
		if primaryCountBySection[sectionNumber] < 2 {
			return nil, fmt.Errorf("COFF object COMDAT section %q has no following COMDAT symbol", section.Name)
		}
	}
	for sectionNumber := range result {
		if err := validateCOMDATAssociation(file, result, sectionNumber); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func validateCOMDATAssociation(file *pe.File, definitions map[int]pe.COFFSymbolAuxFormat5, start int) error {
	seen := make(map[int]struct{})
	current := start
	for {
		if _, duplicate := seen[current]; duplicate {
			return fmt.Errorf("COFF object COMDAT association from section %q contains a cycle", file.Sections[start-1].Name)
		}
		seen[current] = struct{}{}
		definition := definitions[current]
		if definition.Selection != coffCOMDATAssociative {
			return nil
		}
		parent := int(definition.SecNum)
		if parent <= 0 || parent > len(file.Sections) || parent == current {
			return fmt.Errorf("COFF object associative COMDAT section %q has invalid parent %d", file.Sections[current-1].Name, parent)
		}
		if file.Sections[parent-1].Characteristics&sectionLinkCOMDAT == 0 {
			return fmt.Errorf("COFF object associative COMDAT section %q parent %q is not COMDAT", file.Sections[current-1].Name, file.Sections[parent-1].Name)
		}
		if _, found := definitions[parent]; !found {
			return fmt.Errorf("COFF object associative COMDAT section %q parent %q has no exact section definition", file.Sections[current-1].Name, file.Sections[parent-1].Name)
		}
		current = parent
	}
}

func isCOFFTextSection(name string) bool {
	return name == ".text" || strings.HasPrefix(name, ".text$")
}
