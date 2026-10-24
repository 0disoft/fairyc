package pecoff

import (
	"bytes"
	"debug/pe"
	"encoding/binary"
	"fmt"
	"path/filepath"
	"strings"
)

// AMD64ChainLinkReport binds a closed non-COMDAT object's chain fields to exact
// owned linker-map anchors. It makes no archive/COMDAT/production coverage claim.
type AMD64ChainLinkReport struct {
	SourceRVA        uint32
	Records          []AMD64ChainLinkRecord
	RelocationFields int
}

type AMD64ChainLinkRecord struct {
	UnwindRVA                 uint32
	Addends, Expected, Linked [3]uint32
}

// InspectAMD64ChainLink proves S + A - ImageBase for each chain tail. The exact
// fixture identities avoid learning expected addresses from the linked fields.
func InspectAMD64ChainLink(object, image, linkMap []byte, objectName string) (AMD64ChainLinkReport, error) {
	var result AMD64ChainLinkReport
	if objectName == "" || filepath.Base(objectName) != objectName || !strings.EqualFold(filepath.Ext(objectName), ".obj") {
		return result, fmt.Errorf("chain link requires an exact object basename")
	}
	obj, err := InspectObject(object)
	if err != nil {
		return result, err
	}
	exe, err := InspectExecutable(image)
	if err != nil {
		return result, err
	}
	unwind, err := InspectAMD64Unwind(image)
	if err != nil {
		return result, err
	}
	file, err := pe.NewFile(bytes.NewReader(object))
	if err != nil {
		return result, err
	}
	defer file.Close()
	for _, section := range obj.Sections {
		if section.Characteristics&sectionLinkCOMDAT != 0 {
			return result, fmt.Errorf("chain link fixture does not admit COMDAT sections")
		}
	}
	maps, err := inspectMSVCMapSymbols(linkMap, "fy_chain", "fy_code", "fy_meta")
	if err != nil {
		return result, err
	}
	symbols := make(map[uint32]ObjectSymbol)
	bases := make(map[int32]uint32)
	for _, s := range obj.SymbolRecords {
		symbols[s.TableIndex] = s
	}
	for _, name := range []string{"fy_code", "fy_meta", "fy_chain"} {
		symbol, err := exactObjectSymbol(obj.SymbolRecords, name)
		if err != nil {
			return result, err
		}
		mapped, found := maps[name]
		if !found || !msvcMapOwnerIsObject(mapped.Owner, objectName) || symbol.SectionNumber <= 0 || int(symbol.SectionNumber) > len(obj.Sections) {
			return result, fmt.Errorf("chain anchor %s has no exact object/map owner", name)
		}
		rva, err := preferredAddressRVA(exe, mapped.PreferredAddress, "chain anchor")
		if err != nil || rva < symbol.Value {
			return result, fmt.Errorf("chain anchor %s has invalid RVA", name)
		}
		base := rva - symbol.Value
		if old, exists := bases[symbol.SectionNumber]; exists && old != base {
			return result, fmt.Errorf("chain anchor %s disagrees with section base", name)
		}
		bases[symbol.SectionNumber] = base
		if name == "fy_chain" {
			result.SourceRVA = rva
		}
		section := obj.Sections[symbol.SectionNumber-1]
		if name == "fy_meta" {
			if section.Name != ".xdata" || section.Characteristics&sectionReadable == 0 || section.Characteristics&(sectionExecutable|sectionWritable) != 0 {
				return result, fmt.Errorf("chain metadata anchor is not read-only .xdata")
			}
		} else if section.Characteristics&sectionExecutable == 0 {
			return result, fmt.Errorf("chain code anchor is not executable")
		}
	}
	if err := requireDefinedExternalFunction(obj, "fy_chain", "chain source"); err != nil {
		return result, err
	}
	resolve := func(sectionIndex int, site uint32, code bool) (uint32, uint32, error) {
		section := file.Sections[sectionIndex-1]
		reloc, symbol, err := exactPDataFieldRelocation(obj.RelocationRecords, symbols, sectionIndex, section.Name, uint64(section.VirtualAddress)+uint64(site))
		if err != nil {
			return 0, 0, fmt.Errorf("chain field: %w", err)
		}
		want := "fy_meta"
		if code {
			want = "fy_code"
		}
		if symbol.Name != want {
			return 0, 0, fmt.Errorf("chain field targets %q, want %q", symbol.Name, want)
		}
		at, err := objectRelocationRawOffset(object, section, reloc, 4)
		if err != nil {
			return 0, 0, err
		}
		addend := binary.LittleEndian.Uint32(object[at:])
		if uint64(symbol.Value)+uint64(addend) > uint64(obj.Sections[symbol.SectionNumber-1].RawSize) {
			return 0, 0, fmt.Errorf("chain addend leaves its owned section")
		}
		base, found := bases[symbol.SectionNumber]
		if !found {
			return 0, 0, fmt.Errorf("chain field has no mapped section")
		}
		value, err := addRVA(base, symbol.Value, "chain symbol")
		if err == nil {
			value, err = addRVA(value, addend, "chain addend")
		}
		return value, addend, err
	}
	pdataCount := 0
	entries := make(map[[3]uint32]bool)
	for sectionIndex, section := range file.Sections {
		if section.Name != ".pdata" {
			continue
		}
		if section.Size == 0 || section.Size%12 != 0 {
			return result, fmt.Errorf("chain .pdata has invalid size")
		}
		for entry := uint32(0); entry < section.Size; entry += 12 {
			var expected [3]uint32
			for field := range expected {
				expected[field], _, err = resolve(sectionIndex+1, entry+uint32(field)*4, field != 2)
				if err != nil {
					return result, err
				}
			}
			found := false
			for _, fn := range unwind.Functions {
				if fn.BeginRVA == expected[0] && fn.EndRVA == expected[1] && fn.UnwindInfoRVA == expected[2] {
					found = true
					break
				}
			}
			if !found {
				return result, fmt.Errorf("chain source .pdata does not match final function entry")
			}
			if entries[expected] {
				return result, fmt.Errorf("chain object repeats a .pdata entry")
			}
			entries[expected] = true
			pdataCount++
			meta, err := exactObjectSymbol(obj.SymbolRecords, "fy_meta")
			if err != nil {
				return result, err
			}
			xdata := file.Sections[meta.SectionNumber-1]
			base := bases[meta.SectionNumber]
			if expected[2] < base {
				return result, fmt.Errorf("chain unwind offset underflows")
			}
			position := expected[2] - base
			if uint64(position)+4 > uint64(xdata.Size) {
				return result, fmt.Errorf("chain object header is out of range")
			}
			raw := uint64(xdata.Offset) + uint64(position)
			prefix := uint32(4 + ((int(object[raw+2])+1)&^1)*2)
			length := prefix
			chained := object[raw]>>3&4 != 0
			if chained {
				length += 12
			}
			if uint64(position)+uint64(length) > uint64(xdata.Size) {
				return result, fmt.Errorf("chain object record is out of range")
			}
			linked, _, err := mapImageRVA(exe, expected[2], length)
			if err != nil {
				return result, err
			}
			if !bytes.Equal(object[raw:raw+uint64(prefix)], image[linked:linked+int(prefix)]) {
				return result, fmt.Errorf("chain unwind prefix drifted")
			}
			if !chained {
				continue
			}
			record := AMD64ChainLinkRecord{UnwindRVA: expected[2]}
			for field := range record.Expected {
				record.Expected[field], record.Addends[field], err = resolve(int(meta.SectionNumber), position+prefix+uint32(field)*4, field != 2)
				if err != nil {
					return result, err
				}
				record.Linked[field] = binary.LittleEndian.Uint32(image[linked+int(prefix)+field*4:])
				if record.Linked[field] != record.Expected[field] {
					return result, fmt.Errorf("chain tail field %d is %#x, want %#x", field, record.Linked[field], record.Expected[field])
				}
				result.RelocationFields++
			}
			result.Records = append(result.Records, record)
		}
	}
	if pdataCount < 2 || len(result.Records) == 0 {
		return result, fmt.Errorf("chain object has no complete chained .pdata records")
	}
	for _, record := range result.Records {
		if !entries[record.Expected] {
			return result, fmt.Errorf("chain parent is not an owned .pdata entry")
		}
	}
	return result, nil
}
