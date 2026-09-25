package pecoff

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"
	"unicode/utf8"
)

const pdbProcedureFixedBytes = 39

// PDBProcedureRecord describes a CodeView procedure. A record alone does not
// prove that its code survived linking into the associated executable.
type PDBProcedureRecord struct {
	Name    string
	Section uint16
	Offset  uint32
	Size    uint32
}

// InspectPDBModuleProcedureRecords reads only one DBI-selected module stream.
// Its result is raw symbol evidence, not a final-image code-size account.
func InspectPDBModuleProcedureRecords(image, pdb []byte, moduleIndex uint16) ([]PDBProcedureRecord, error) {
	report, err := InspectPDBSectionContributions(image, pdb)
	if err != nil {
		return nil, err
	}
	if int(moduleIndex) >= len(report.Modules) {
		return nil, fmt.Errorf("PDB module index %d is missing", moduleIndex)
	}
	module := report.Modules[moduleIndex]
	if module.SymbolStream == 0xffff {
		if module.SymbolBytes != 0 || module.C11Bytes != 0 || module.C13Bytes != 0 {
			return nil, fmt.Errorf("PDB module %d has symbol bytes without a stream", moduleIndex)
		}
		return nil, nil
	}
	if module.SymbolBytes < 4 {
		return nil, fmt.Errorf("PDB module %d has an invalid symbol-byte size", moduleIndex)
	}
	stream, _, err := readPDBStream(pdb, uint32(module.SymbolStream))
	if err != nil {
		return nil, fmt.Errorf("PDB module %d symbol stream: %w", moduleIndex, err)
	}
	if uint64(module.SymbolBytes)+uint64(module.C11Bytes)+uint64(module.C13Bytes) > uint64(len(stream)) {
		return nil, fmt.Errorf("PDB module %d substreams exceed the bounded stream", moduleIndex)
	}
	if binary.LittleEndian.Uint32(stream) != 4 {
		return nil, fmt.Errorf("PDB module %d has an unsupported symbol signature", moduleIndex)
	}
	return inspectPDBProcedureRecords(stream[4:module.SymbolBytes])
}

func inspectPDBProcedureRecords(symbols []byte) ([]PDBProcedureRecord, error) {
	var procedures []PDBProcedureRecord
	for offset := 0; offset < len(symbols); {
		if len(symbols)-offset < 4 {
			return nil, fmt.Errorf("PDB symbol record header is truncated at byte %d", offset)
		}
		length := int(binary.LittleEndian.Uint16(symbols[offset:])) + 2
		if length < 4 || length > len(symbols)-offset {
			return nil, fmt.Errorf("PDB symbol record exceeds the symbol substream at byte %d", offset)
		}
		record := symbols[offset : offset+length]
		switch binary.LittleEndian.Uint16(record[2:]) {
		case 0x110f, 0x1110, 0x1146, 0x1147, 0x1155, 0x1156:
			if len(record) < pdbProcedureFixedBytes+1 {
				return nil, fmt.Errorf("PDB procedure record is truncated at byte %d", offset)
			}
			end := bytes.IndexByte(record[pdbProcedureFixedBytes:], 0)
			if end <= 0 {
				return nil, fmt.Errorf("PDB procedure name is missing or unterminated at byte %d", offset)
			}
			name := string(record[pdbProcedureFixedBytes : pdbProcedureFixedBytes+end])
			if slash := strings.LastIndexAny(name, `/\`); slash >= 0 {
				name = name[slash+1:]
			}
			if name == "" || name == "." || name == ".." || !utf8.ValidString(name) ||
				strings.IndexFunc(name, func(r rune) bool { return r < 0x20 || r == 0x7f }) >= 0 {
				return nil, fmt.Errorf("PDB procedure name has an invalid final component at byte %d", offset)
			}
			section := binary.LittleEndian.Uint16(record[36:])
			size := binary.LittleEndian.Uint32(record[16:])
			start := binary.LittleEndian.Uint32(record[32:])
			if section == 0 || size == 0 || uint64(start)+uint64(size) > 1<<32 {
				return nil, fmt.Errorf("PDB procedure record has an invalid code range at byte %d", offset)
			}
			procedures = append(procedures, PDBProcedureRecord{
				Name: name, Section: section, Offset: start, Size: size,
			})
		}
		offset += length
	}
	return procedures, nil
}
