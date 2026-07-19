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
	sectionRelocationOverflow = 0x01000000
)

// ObjectSection is one bounded COFF object section receipt.
type ObjectSection struct {
	Name            string
	RawOffset       uint32
	RawSize         uint32
	Relocations     int
	Characteristics uint32
}

// ObjectReport records the closed COFF object facts consumed by the Windows C driver.
type ObjectReport struct {
	Machine         uint16
	Characteristics uint16
	Sections        []ObjectSection
	Symbols         []string
	Relocations     int
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
	if len(file.Sections) == 0 || len(file.Sections) > maxSections {
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
			if uint64(relocation.SymbolTableIndex) >= uint64(len(file.COFFSymbols)) {
				return ObjectReport{}, fmt.Errorf("COFF object section %q relocation references missing symbol %d", section.Name, relocation.SymbolTableIndex)
			}
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
		report.Relocations += len(section.Relocs)
		report.Sections = append(report.Sections, ObjectSection{
			Name:            section.Name,
			RawOffset:       section.Offset,
			RawSize:         section.Size,
			Relocations:     len(section.Relocs),
			Characteristics: section.Characteristics,
		})
	}
	if !textSectionFound {
		return ObjectReport{}, fmt.Errorf("COFF object has no .text or .text$ code section")
	}
	if len(file.Symbols) == 0 || len(file.COFFSymbols) == 0 {
		return ObjectReport{}, fmt.Errorf("COFF object has no symbol table")
	}
	for index, symbol := range file.Symbols {
		if symbol == nil || symbol.Name == "" {
			return ObjectReport{}, fmt.Errorf("COFF object symbol %d has no name", index)
		}
		report.Symbols = append(report.Symbols, symbol.Name)
	}
	return report, nil
}

func isCOFFTextSection(name string) bool {
	return name == ".text" || strings.HasPrefix(name, ".text$")
}
