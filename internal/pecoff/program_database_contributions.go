package pecoff

import (
	"encoding/binary"
	"fmt"
)

const (
	pdbDBIHeaderBytes             = 64
	pdbSectionContribVersion60    = 0xf12eba2d
	pdbSectionContribVersion2     = 0xf13151e4
	pdbSectionContribRecordBytes  = 28
	pdbSectionContrib2RecordBytes = 32
)

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
		Version: version, Contributions: make([]PDBSectionContribution, 0, (len(data)-4)/recordBytes),
	}
	for offset := 4; offset < len(data); offset += recordBytes {
		record := data[offset : offset+recordBytes]
		section := binary.LittleEndian.Uint16(record)
		start := int32(binary.LittleEndian.Uint32(record[4:]))
		size := int32(binary.LittleEndian.Uint32(record[8:]))
		module := binary.LittleEndian.Uint16(record[16:])
		if section == 0 || int(section) > len(pe.Sections) || start < 0 || size <= 0 || module == 0xffff ||
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
