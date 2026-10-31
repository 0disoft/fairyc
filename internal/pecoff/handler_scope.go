package pecoff

import (
	"bytes"
	"debug/pe"
	"encoding/binary"
	"fmt"
	"math"
	"strings"
)

type AMD64ScopeRecord struct{ Begin, End, Handler, Jump uint32 }
type AMD64ScopeReport struct {
	Link             AMD64HandlerLinkReport
	ObjectDataOffset uint32
	DataRVA          uint32
	Records          []AMD64ScopeRecord
	RelocationFields int
	GSWord           *uint32
}

// InspectAMD64ScopeTable binds the closed C-specific scope layout to actual
// ADDR32NB object fields and final bytes. GS admission is limited to the simple
// unaligned four-byte trailer, not C++ EH tables or arbitrary handler formats.
func InspectAMD64ScopeTable(object, image, linkMap []byte, spec AMD64HandlerLinkSpec) (AMD64ScopeReport, error) {
	var result AMD64ScopeReport
	if spec.HandlerSymbol != "__C_specific_handler" && spec.HandlerSymbol != "__GSHandlerCheck_SEH" {
		return result, fmt.Errorf("unsupported scope handler")
	}
	link, err := InspectAMD64HandlerLink(object, image, linkMap, spec)
	if err != nil {
		return result, err
	}
	result.Link = link
	r, err := InspectObject(object)
	if err != nil {
		return result, err
	}
	source, err := exactObjectSymbol(r.SymbolRecords, spec.SourceSymbol)
	if err != nil {
		return result, err
	}
	f, err := pe.NewFile(bytes.NewReader(object))
	if err != nil {
		return result, err
	}
	defer f.Close()
	pdata, err := exactPDataEntryRelocations(r, f, source)
	if err != nil {
		return result, err
	}
	pos, err := objectRelocationRawOffset(object, pdata.Section, pdata.Unwind, 4)
	if err != nil {
		return result, err
	}
	section := r.Sections[pdata.UnwindTarget.SectionNumber-1]
	offset := uint64(pdata.UnwindTarget.Value) + uint64(binary.LittleEndian.Uint32(object[pos:])) + uint64(link.UnwindPrefixBytes) + 4
	if offset+4 > uint64(section.RawSize) {
		return result, fmt.Errorf("scope count exceeds object section")
	}
	rawStart := uint64(section.RawOffset) + offset
	count := binary.LittleEndian.Uint32(object[rawStart:])
	if count == 0 || count > 128 {
		return result, fmt.Errorf("scope count outside 1..128")
	}
	size := uint64(4) + uint64(count)*16
	gs := spec.HandlerSymbol == "__GSHandlerCheck_SEH"
	if gs {
		size += 4
	}
	if offset+size > uint64(section.RawSize) || rawStart+size > uint64(len(object)) {
		return result, fmt.Errorf("scope table exceeds object section")
	}
	expected := bytes.Clone(object[rawStart : rawStart+size])
	result.ObjectDataOffset = uint32(rawStart)
	if uint64(link.UnwindInfoRVA)+uint64(link.UnwindPrefixBytes)+4 > math.MaxUint32 {
		return result, fmt.Errorf("scope data RVA overflow")
	}
	result.DataRVA = link.UnwindInfoRVA + uint32(link.UnwindPrefixBytes) + 4
	imageReport, err := InspectExecutable(image)
	if err != nil {
		return result, err
	}
	actual, err := generatedImageBytes(image, imageReport, result.DataRVA, uint32(size))
	if err != nil {
		return result, err
	}
	if binary.LittleEndian.Uint32(actual) != count {
		return result, fmt.Errorf("scope count drifted")
	}
	var wanted []string
	symbols := map[uint32]ObjectSymbol{}
	for _, s := range r.SymbolRecords {
		symbols[s.TableIndex] = s
		wanted = append(wanted, s.Name)
	}
	mapped, err := inspectMSVCMapSymbols(linkMap, wanted...)
	if err != nil {
		return result, err
	}
	if link.SourceRVA < source.Value {
		return result, fmt.Errorf("scope source section base underflow")
	}
	bases := map[int32]uint32{source.SectionNumber: link.SourceRVA - source.Value}
	for _, s := range r.SymbolRecords {
		if s.SectionNumber <= 0 || strings.HasPrefix(s.Name, ".") {
			continue
		}
		m, ok := mapped[s.Name]
		if !ok || !msvcMapOwnerIsObject(m.Owner, spec.SourceObject) {
			continue
		}
		rva, err := preferredAddressRVA(imageReport, m.PreferredAddress, s.Name)
		if err != nil {
			return result, err
		}
		if rva < s.Value {
			return result, fmt.Errorf("scope section base underflow")
		}
		base := rva - s.Value
		if previous, ok := bases[s.SectionNumber]; ok && previous != base {
			return result, fmt.Errorf("ambiguous scope section base")
		}
		bases[s.SectionNumber] = base
	}
	fields := map[uint32]ObjectRelocation{}
	for _, rel := range r.RelocationRecords {
		if rel.SectionIndex != int(pdata.UnwindTarget.SectionNumber) || uint64(rel.VirtualAddress) < offset || uint64(rel.VirtualAddress) >= offset+size {
			continue
		}
		at := uint32(uint64(rel.VirtualAddress) - offset)
		if _, ok := fields[at]; ok || at < 4 || at >= 4+count*16 || at%4 != 0 || rel.Type != ObjectRelocationAMD64Addr32NB {
			return result, fmt.Errorf("invalid scope field relocation")
		}
		fields[at] = rel
	}
	for i := uint32(0); i < count; i++ {
		for j := uint32(0); j < 4; j++ {
			at := 4 + i*16 + j*4
			addend := binary.LittleEndian.Uint32(expected[at:])
			rel, ok := fields[at]
			if !ok {
				if !(j == 2 && addend == 1 || j == 3 && addend == 0) {
					return result, fmt.Errorf("scope address lacks relocation")
				}
				continue
			}
			target, ok := symbols[rel.SymbolIndex]
			if !ok {
				return result, fmt.Errorf("scope relocation target missing")
			}
			var value uint64
			if target.SectionNumber > 0 {
				base, ok := bases[target.SectionNumber]
				if !ok {
					return result, fmt.Errorf("scope target %s lacks owned section base", target.Name)
				}
				value = uint64(base) + uint64(target.Value)
			} else if target.SectionNumber == 0 && target.StorageClass == coffStorageClassExternal {
				m, ok := mapped[target.Name]
				if !ok {
					return result, fmt.Errorf("scope target lacks map address")
				}
				rva, err := preferredAddressRVA(imageReport, m.PreferredAddress, target.Name)
				if err != nil {
					return result, err
				}
				value = uint64(rva)
			} else {
				return result, fmt.Errorf("unadmitted scope target kind")
			}
			value += uint64(addend)
			if value > math.MaxUint32 {
				return result, fmt.Errorf("scope field overflow")
			}
			binary.LittleEndian.PutUint32(expected[at:], uint32(value))
			result.RelocationFields++
		}
		at := 4 + i*16
		record := AMD64ScopeRecord{binary.LittleEndian.Uint32(expected[at:]), binary.LittleEndian.Uint32(expected[at+4:]), binary.LittleEndian.Uint32(expected[at+8:]), binary.LittleEndian.Uint32(expected[at+12:])}
		if record.Begin >= record.End || requireExecutableRVARange(imageReport, record.Begin, record.End) != nil {
			return result, fmt.Errorf("invalid executable scope range")
		}
		if record.Handler != 1 && requireExecutableRVARange(imageReport, record.Handler, record.Handler+1) != nil || record.Jump != 0 && requireExecutableRVARange(imageReport, record.Jump, record.Jump+1) != nil || record.Jump == 0 && record.Handler <= 1 {
			return result, fmt.Errorf("invalid scope handler/jump")
		}
		result.Records = append(result.Records, record)
	}
	if !bytes.Equal(expected, actual) {
		return result, fmt.Errorf("scope payload differs from relocated object")
	}
	if gs {
		word := binary.LittleEndian.Uint32(expected[len(expected)-4:])
		if word&4 != 0 {
			return result, fmt.Errorf("aligned GS trailer is outside scope proof")
		}
		if word&3 != uint32(link.ObjectFlags)&3 {
			return result, fmt.Errorf("GS handler flags disagree with unwind flags")
		}
		result.GSWord = &word
	}
	return result, nil
}
