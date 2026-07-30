package pecoff

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

const (
	coffBigObjectHeaderBytes = 56
	coffBigObjectSymbolBytes = 20
)

var coffBigObjectClassID = [16]byte{
	0xc7, 0xa1, 0xba, 0xd1, 0xee, 0xba, 0xa9, 0x4b,
	0xaf, 0x20, 0xfa, 0xf6, 0x6a, 0xa4, 0xdc, 0xb8,
}

type bigCOFFFileRange struct {
	kind       string
	start, end uint64
}

func isBigCOFFObject(object []byte) bool {
	return len(object) >= 4 && binary.LittleEndian.Uint16(object) == 0 && binary.LittleEndian.Uint16(object[2:]) == 0xffff
}

func inspectBigCOFFObject(object []byte) (ObjectReport, error) {
	if len(object) < coffBigObjectHeaderBytes || len(object) > maxExecutableBytes {
		return ObjectReport{}, fmt.Errorf("COFF bigobj has invalid size %d", len(object))
	}
	version := binary.LittleEndian.Uint16(object[4:])
	machine := binary.LittleEndian.Uint16(object[6:])
	if version != 2 {
		return ObjectReport{}, fmt.Errorf("COFF bigobj version is %d, want 2", version)
	}
	if machine != machineAMD64 {
		return ObjectReport{}, fmt.Errorf("COFF bigobj machine is %#x, want AMD64", machine)
	}
	if !bytes.Equal(object[12:28], coffBigObjectClassID[:]) {
		return ObjectReport{}, fmt.Errorf("COFF anonymous object has unsupported class ID")
	}
	if !bytes.Equal(object[28:44], make([]byte, 16)) {
		return ObjectReport{}, fmt.Errorf("COFF bigobj has nonzero reserved metadata fields")
	}
	sectionCount := uint64(binary.LittleEndian.Uint32(object[44:]))
	if sectionCount == 0 || sectionCount > maxObjectSections {
		return ObjectReport{}, fmt.Errorf("COFF bigobj has invalid section count %d", sectionCount)
	}
	headerEnd := uint64(coffBigObjectHeaderBytes) + sectionCount*coffSectionHeaderBytes
	if headerEnd > uint64(len(object)) {
		return ObjectReport{}, fmt.Errorf("COFF bigobj section table exceeds the file")
	}
	symbolOffset := uint64(binary.LittleEndian.Uint32(object[48:]))
	symbolCount := uint64(binary.LittleEndian.Uint32(object[52:]))
	if symbolCount == 0 || symbolCount > maxObjectSymbols {
		return ObjectReport{}, fmt.Errorf("COFF bigobj has invalid symbol count %d", symbolCount)
	}
	symbolEnd := symbolOffset + symbolCount*coffBigObjectSymbolBytes
	if symbolEnd < symbolOffset || symbolOffset < headerEnd || symbolEnd+4 > uint64(len(object)) {
		return ObjectReport{}, fmt.Errorf("COFF bigobj has an invalid symbol-table range")
	}
	stringBytes := uint64(binary.LittleEndian.Uint32(object[symbolEnd:]))
	if stringBytes < 4 || symbolEnd+stringBytes < symbolEnd || symbolEnd+stringBytes > uint64(len(object)) {
		return ObjectReport{}, fmt.Errorf("COFF bigobj has an invalid string table")
	}
	stringTable := object[symbolEnd : symbolEnd+stringBytes]

	report := ObjectReport{
		Machine: machine, BigObject: true, BigObjectVersion: version,
		Sections: make([]ObjectSection, 0, int(sectionCount)), Symbols: make([]string, 0, int(symbolCount)),
		SymbolRecords: make([]ObjectSymbol, 0, int(symbolCount)),
	}
	primaryByIndex := make(map[uint32]int, int(symbolCount))
	for index := uint64(0); index < symbolCount; {
		offset := symbolOffset + index*coffBigObjectSymbolBytes
		entry := object[offset : offset+coffBigObjectSymbolBytes]
		name, err := bigCOFFName(entry[:8], stringTable)
		if err != nil {
			return ObjectReport{}, fmt.Errorf("read COFF bigobj symbol %d name: %w", index, err)
		}
		if name == "" {
			return ObjectReport{}, fmt.Errorf("COFF bigobj symbol %d has no name", index)
		}
		auxiliary := entry[19]
		if index+1+uint64(auxiliary) > symbolCount {
			return ObjectReport{}, fmt.Errorf("COFF bigobj symbol %q auxiliary records exceed the symbol table", name)
		}
		if auxiliary != 0 {
			return ObjectReport{}, fmt.Errorf("COFF bigobj symbol %q uses unsupported auxiliary records", name)
		}
		sectionNumber := int32(binary.LittleEndian.Uint32(entry[12:]))
		if sectionNumber < -2 || sectionNumber > int32(sectionCount) {
			return ObjectReport{}, fmt.Errorf("COFF bigobj symbol %q has invalid section %d", name, sectionNumber)
		}
		record := ObjectSymbol{
			TableIndex: uint32(index), Name: name, Value: binary.LittleEndian.Uint32(entry[8:]),
			SectionNumber: sectionNumber, Type: binary.LittleEndian.Uint16(entry[16:]),
			StorageClass: entry[18], AuxiliaryRecords: auxiliary,
		}
		report.Symbols = append(report.Symbols, name)
		report.SymbolRecords = append(report.SymbolRecords, record)
		primaryByIndex[uint32(index)] = len(report.SymbolRecords) - 1
		index++
	}

	textSectionFound := false
	fileRanges := []bigCOFFFileRange{{kind: "symbol and string tables", start: symbolOffset, end: symbolEnd + stringBytes}}
	for index := 0; index < int(sectionCount); index++ {
		headerOffset := coffBigObjectHeaderBytes + index*coffSectionHeaderBytes
		header := object[headerOffset : headerOffset+coffSectionHeaderBytes]
		name, err := bigCOFFShortSectionName(header[:8])
		if err != nil {
			return ObjectReport{}, fmt.Errorf("read COFF bigobj section %d name: %w", index, err)
		}
		if name == "" {
			return ObjectReport{}, fmt.Errorf("COFF bigobj section %d has no name", index)
		}
		rawSize := binary.LittleEndian.Uint32(header[16:])
		rawOffset := binary.LittleEndian.Uint32(header[20:])
		characteristics := binary.LittleEndian.Uint32(header[36:])
		if characteristics&sectionLinkCOMDAT != 0 {
			return ObjectReport{}, fmt.Errorf("COFF bigobj section %q uses unsupported COMDAT metadata", name)
		}
		if rawSize > 0 {
			if rawOffset == 0 {
				if characteristics&sectionUninitializedData == 0 {
					return ObjectReport{}, fmt.Errorf("COFF bigobj section %q has an invalid zero raw pointer", name)
				}
			} else {
				start := uint64(rawOffset)
				end := start + uint64(rawSize)
				if start < headerEnd || end > uint64(len(object)) {
					return ObjectReport{}, fmt.Errorf("COFF bigobj section %q has an invalid raw range", name)
				}
				if err := addBigCOFFFileRange(&fileRanges, bigCOFFFileRange{kind: "section " + name + " raw data", start: start, end: end}); err != nil {
					return ObjectReport{}, err
				}
			}
		}
		relocations, err := inspectCOFFSectionRelocations(object, coffBigObjectHeaderBytes, headerEnd, index, name)
		if err != nil {
			return ObjectReport{}, err
		}
		if len(relocations) > maxObjectRelocations-report.Relocations {
			return ObjectReport{}, fmt.Errorf("COFF bigobj has more than %d relocations", maxObjectRelocations)
		}
		if len(relocations) > 0 {
			storedRecords := uint64(len(relocations))
			if characteristics&sectionRelocationOverflow != 0 {
				storedRecords++
			}
			start := uint64(binary.LittleEndian.Uint32(header[24:]))
			if err := addBigCOFFFileRange(&fileRanges, bigCOFFFileRange{
				kind: "section " + name + " relocations", start: start, end: start + storedRecords*coffRelocationBytes,
			}); err != nil {
				return ObjectReport{}, err
			}
		}
		for _, relocation := range relocations {
			if !validAMD64RelocationType(relocation.Type) {
				return ObjectReport{}, fmt.Errorf("COFF bigobj section %q has unknown AMD64 relocation type %#x", name, relocation.Type)
			}
			symbolPosition, found := primaryByIndex[relocation.SymbolTableIndex]
			if !found {
				if uint64(relocation.SymbolTableIndex) >= symbolCount {
					return ObjectReport{}, fmt.Errorf("COFF bigobj section %q relocation references missing symbol %d", name, relocation.SymbolTableIndex)
				}
				return ObjectReport{}, fmt.Errorf("COFF bigobj section %q relocation references auxiliary symbol %d", name, relocation.SymbolTableIndex)
			}
			symbol := report.SymbolRecords[symbolPosition]
			report.RelocationRecords = append(report.RelocationRecords, ObjectRelocation{
				SectionIndex: index + 1, Section: name, VirtualAddress: relocation.VirtualAddress,
				SymbolIndex: relocation.SymbolTableIndex, Symbol: symbol.Name, Type: relocation.Type,
			})
		}
		containsCode := characteristics&sectionContainsCode != 0
		executable := characteristics&sectionExecutable != 0
		readable := characteristics&sectionReadable != 0
		if isCOFFTextSection(name) {
			if !containsCode || !executable || !readable {
				return ObjectReport{}, fmt.Errorf("COFF bigobj .text section is not readable executable code")
			}
			textSectionFound = true
		}
		report.Relocations += len(relocations)
		report.Sections = append(report.Sections, ObjectSection{
			Name: name, RawOffset: rawOffset, RawSize: rawSize,
			Relocations: len(relocations), Characteristics: characteristics,
		})
	}
	if !textSectionFound {
		return ObjectReport{}, fmt.Errorf("COFF bigobj has no .text or .text$ code section")
	}
	return report, nil
}

func addBigCOFFFileRange(ranges *[]bigCOFFFileRange, candidate bigCOFFFileRange) error {
	if candidate.start >= candidate.end {
		return fmt.Errorf("COFF bigobj %s has an empty or reversed range", candidate.kind)
	}
	for _, existing := range *ranges {
		if candidate.start < existing.end && existing.start < candidate.end {
			return fmt.Errorf("COFF bigobj %s overlaps %s", candidate.kind, existing.kind)
		}
	}
	*ranges = append(*ranges, candidate)
	return nil
}

func bigCOFFShortSectionName(field []byte) (string, error) {
	if len(field) != 8 {
		return "", fmt.Errorf("invalid name field size")
	}
	if field[0] == '/' || binary.LittleEndian.Uint32(field) == 0 {
		return "", fmt.Errorf("unsupported long name")
	}
	end := bytes.IndexByte(field, 0)
	if end < 0 {
		return string(field), nil
	}
	if !bytes.Equal(field[end:], make([]byte, len(field)-end)) {
		return "", fmt.Errorf("short name has nonzero padding")
	}
	return string(field[:end]), nil
}

func bigCOFFName(field, stringTable []byte) (string, error) {
	if len(field) != 8 {
		return "", fmt.Errorf("invalid name field size")
	}
	if binary.LittleEndian.Uint32(field) == 0 {
		offset := uint64(binary.LittleEndian.Uint32(field[4:]))
		if offset < 4 || offset >= uint64(len(stringTable)) {
			return "", fmt.Errorf("string-table offset %d is out of range", offset)
		}
		remaining := stringTable[offset:]
		end := bytes.IndexByte(remaining, 0)
		if end < 0 {
			return "", fmt.Errorf("string-table name at offset %d is unterminated", offset)
		}
		return string(remaining[:end]), nil
	}
	end := bytes.IndexByte(field, 0)
	if end < 0 {
		end = len(field)
	} else if !bytes.Equal(field[end:], make([]byte, len(field)-end)) {
		return "", fmt.Errorf("short name has nonzero padding")
	}
	return string(field[:end]), nil
}
