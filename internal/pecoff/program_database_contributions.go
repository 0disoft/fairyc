package pecoff

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"
	"unicode/utf8"
)

const (
	pdbDBIHeaderBytes             = 64
	pdbSectionContribVersion60    = 0xf12eba2d
	pdbSectionContribVersion2     = 0xf13151e4
	pdbSectionContribRecordBytes  = 28
	pdbSectionContrib2RecordBytes = 32
	pdbModuleFixedBytes           = 64
)

// PDBModule identifies one DBI compiland. Names are reduced to their final
// path component so diagnostics cannot disclose temporary build directories.
type PDBModule struct {
	Index        uint16
	Name         string
	ObjectName   string
	SymbolStream uint16
	SymbolBytes  uint32
	C11Bytes     uint32
	C13Bytes     uint32
}

func inspectPDBModules(data []byte) ([]PDBModule, error) {
	modules := make([]PDBModule, 0)
	for offset := 0; offset < len(data); {
		if len(data)-offset < pdbModuleFixedBytes || len(modules) >= 1<<16 {
			return nil, fmt.Errorf("PDB module information is truncated or exceeds the module index range")
		}
		cursor := offset + pdbModuleFixedBytes
		readName := func() (string, error) {
			end := bytes.IndexByte(data[cursor:], 0)
			if end < 0 {
				return "", fmt.Errorf("PDB module name is unterminated")
			}
			if end == 0 {
				cursor++
				return "<unnamed>", nil
			}
			name := data[cursor : cursor+end]
			cursor += end + 1
			base := string(name)
			if separator := strings.LastIndexAny(base, `/\`); separator >= 0 {
				base = base[separator+1:]
			}
			if base == "" || base == "." || base == ".." || !utf8.ValidString(base) ||
				strings.IndexFunc(base, func(r rune) bool { return r < 0x20 || r == 0x7f }) >= 0 {
				return "", fmt.Errorf("PDB module name has an invalid final component")
			}
			return base, nil
		}
		name, err := readName()
		if err != nil {
			return nil, fmt.Errorf("PDB module %d at byte %d: %w", len(modules), offset, err)
		}
		objectName, err := readName()
		if err != nil {
			return nil, fmt.Errorf("PDB module %d at byte %d: %w", len(modules), offset, err)
		}
		next := (cursor + 3) &^ 3
		if next > len(data) {
			return nil, fmt.Errorf("PDB module record is truncated at alignment padding")
		}
		for _, padding := range data[cursor:next] {
			if padding != 0 {
				return nil, fmt.Errorf("PDB module record has nonzero alignment padding")
			}
		}
		modules = append(modules, PDBModule{
			Index: uint16(len(modules)), Name: name, ObjectName: objectName,
			SymbolStream: binary.LittleEndian.Uint16(data[offset+34:]),
			SymbolBytes:  binary.LittleEndian.Uint32(data[offset+36:]),
			C11Bytes:     binary.LittleEndian.Uint32(data[offset+40:]),
			C13Bytes:     binary.LittleEndian.Uint32(data[offset+44:]),
		})
		offset = next
	}
	return modules, nil
}

// PDBSectionContribution is one DBI record for a final-image section range.
// ModuleIndex is a zero-based compiland index, not a function identity.
type PDBSectionContribution struct {
	Section         uint16
	Offset          uint32
	Size            uint32
	Characteristics uint32
	ModuleIndex     uint16
}

// PDBSectionContributionReport keeps the DBI provenance needed for bounded
// final-image ownership analysis. It does not assign source function names.
type PDBSectionContributionReport struct {
	Version       uint32
	Modules       []PDBModule
	Contributions []PDBSectionContribution
}

// InspectPDBSectionContributions binds the PDB to its PE image, then reads the
// bounded DBI section-contribution substream. No paths or private names escape.
func InspectPDBSectionContributions(image, pdb []byte) (PDBSectionContributionReport, error) {
	identity, err := InspectProgramDatabaseAssociation(image, pdb)
	if err != nil {
		return PDBSectionContributionReport{}, err
	}
	pe, err := InspectExecutable(image)
	if err != nil {
		return PDBSectionContributionReport{}, err
	}
	dbi, _, err := readPDBStream(pdb, 3)
	if err != nil {
		return PDBSectionContributionReport{}, err
	}
	if len(dbi) < pdbDBIHeaderBytes || binary.LittleEndian.Uint32(dbi[8:]) != identity.Age ||
		binary.LittleEndian.Uint16(dbi[58:]) != pe.Machine {
		return PDBSectionContributionReport{}, fmt.Errorf("PDB DBI header does not match the PE/PDB identity")
	}
	sizes := []uint32{
		binary.LittleEndian.Uint32(dbi[24:]), binary.LittleEndian.Uint32(dbi[28:]),
		binary.LittleEndian.Uint32(dbi[32:]), binary.LittleEndian.Uint32(dbi[36:]),
		binary.LittleEndian.Uint32(dbi[40:]), binary.LittleEndian.Uint32(dbi[48:]),
		binary.LittleEndian.Uint32(dbi[52:]),
	}
	total := uint64(pdbDBIHeaderBytes)
	for _, size := range sizes {
		if size > uint32(len(dbi)) {
			return PDBSectionContributionReport{}, fmt.Errorf("PDB DBI substream exceeds the bounded stream")
		}
		total += uint64(size)
	}
	if total != uint64(len(dbi)) || sizes[1] < 4 {
		return PDBSectionContributionReport{}, fmt.Errorf("PDB DBI substream sizes are inconsistent")
	}
	begin := pdbDBIHeaderBytes + uint64(sizes[0])
	modules, err := inspectPDBModules(dbi[pdbDBIHeaderBytes:begin])
	if err != nil {
		return PDBSectionContributionReport{}, err
	}
	data := dbi[begin : begin+uint64(sizes[1])]
	version := binary.LittleEndian.Uint32(data)
	recordBytes := pdbSectionContribRecordBytes
	switch version {
	case pdbSectionContribVersion60:
	case pdbSectionContribVersion2:
		recordBytes = pdbSectionContrib2RecordBytes
	default:
		return PDBSectionContributionReport{}, fmt.Errorf("unsupported PDB section-contribution version %#x", version)
	}
	if (len(data)-4)%recordBytes != 0 {
		return PDBSectionContributionReport{}, fmt.Errorf("PDB section-contribution records are truncated")
	}
	report := PDBSectionContributionReport{
		Version: version, Modules: modules,
		Contributions: make([]PDBSectionContribution, 0, (len(data)-4)/recordBytes),
	}
	for offset := 4; offset < len(data); offset += recordBytes {
		record := data[offset : offset+recordBytes]
		section := binary.LittleEndian.Uint16(record)
		start := int32(binary.LittleEndian.Uint32(record[4:]))
		size := int32(binary.LittleEndian.Uint32(record[8:]))
		module := binary.LittleEndian.Uint16(record[16:])
		if section == 0 || int(section) > len(pe.Sections) || start < 0 || size <= 0 || int(module) >= len(modules) ||
			uint64(start)+uint64(size) > uint64(pe.Sections[section-1].VirtualSize) {
			return PDBSectionContributionReport{}, fmt.Errorf("PDB section contribution has an invalid image range or module index")
		}
		item := PDBSectionContribution{
			Section: section, Offset: uint32(start), Size: uint32(size),
			Characteristics: binary.LittleEndian.Uint32(record[12:]), ModuleIndex: module,
		}
		if len(report.Contributions) != 0 {
			previous := report.Contributions[len(report.Contributions)-1]
			if item.Section < previous.Section || (item.Section == previous.Section && item.Offset <= previous.Offset) {
				return PDBSectionContributionReport{}, fmt.Errorf("PDB section contributions are not strictly ordered")
			}
		}
		report.Contributions = append(report.Contributions, item)
	}
	return report, nil
}
