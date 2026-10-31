package pecoff

import (
	"bytes"
	"debug/pe"
	"encoding/binary"
	"fmt"
	"math"
)

// GeneratedRelocationValue binds an actual generated object field to its owned
// linked site and target. Addend preserves the raw bits, including REL32 signs.
type GeneratedRelocationValue struct {
	SectionIndex                     int
	Object, Section, Target          string
	Type                             uint16
	ObjectOffset, SiteRVA, TargetRVA uint32
	Addend, Expected, Actual         uint64
}

type GeneratedRelocationFamily struct {
	Object                      string
	Type                        uint16
	Observed, Linked, Discarded int
}

type GeneratedRelocationCoverage struct {
	Families                    []GeneratedRelocationFamily
	Values                      []GeneratedRelocationValue
	Observed, Linked, Discarded int
}

// InspectGeneratedRelocations checks every relocation in the two named real
// production objects. Unknown kinds, unresolved owners, unaccounted discards and
// final-byte disagreement fail closed. Debug metadata is outside this profile.
func InspectGeneratedRelocations(unit, runtime, image, linkMap, linkLog []byte) (GeneratedRelocationCoverage, error) {
	var result GeneratedRelocationCoverage
	parents, err := InspectGeneratedCOMDATSelection(unit, runtime, image, linkMap, linkLog)
	if err != nil {
		return result, err
	}
	report, err := InspectExecutable(image)
	if err != nil {
		return result, err
	}
	if _, err := InspectAMD64Unwind(image); err != nil {
		return result, err
	}
	entries, err := runtimeFunctionEntries(image, report)
	if err != nil {
		return result, err
	}
	f, err := pe.NewFile(bytes.NewReader(image))
	if err != nil {
		return result, err
	}
	defer f.Close()
	directory := f.OptionalHeader.(*pe.OptionalHeader64).DataDirectory[exceptionDirectoryIndex].VirtualAddress
	entrySites := map[[2]uint32][]uint32{}
	for index, entry := range entries {
		key := [2]uint32{entry.begin, entry.end}
		entrySites[key] = append(entrySites[key], directory+uint32(index)*12)
	}
	for _, input := range []struct {
		name string
		raw  []byte
	}{{"unit.obj", unit}, {"runtime.obj", runtime}} {
		object, err := InspectObject(input.raw)
		if err != nil {
			return result, err
		}
		if len(object.RelocationRecords) > 65536 {
			return result, fmt.Errorf("generated relocation proof exceeds 65536 fields per object")
		}
		if len(object.RelocationRecords) == 0 {
			return result, fmt.Errorf("%s has no measured relocations", input.name)
		}
		symbols := map[uint32]ObjectSymbol{}
		var wanted []string
		for _, s := range object.SymbolRecords {
			symbols[s.TableIndex] = s
			wanted = append(wanted, s.Name)
		}
		mapped, err := inspectMSVCMapSymbols(linkMap, wanted...)
		if err != nil {
			return result, err
		}
		bases := map[int]uint32{}
		discarded := map[int]bool{}
		for _, p := range parents.Parents {
			if p.Object == input.name {
				discarded[p.SectionIndex] = !p.Retained
			}
		}
		for index, s := range object.Sections {
			if s.COMDATSelection == ObjectCOMDATAssociative {
				d, ok := discarded[s.AssociativeSection]
				if !ok {
					return result, fmt.Errorf("unmeasured associative parent")
				}
				discarded[index+1] = d
			}
		}
		for _, s := range object.SymbolRecords {
			m, ok := mapped[s.Name]
			if s.SectionNumber <= 0 || !ok || generatedObjectOwner(m.Owner) != input.name || discarded[int(s.SectionNumber)] {
				continue
			}
			rva, err := preferredAddressRVA(report, m.PreferredAddress, "generated symbol")
			if err != nil {
				return result, err
			}
			if rva < s.Value {
				return result, fmt.Errorf("generated section base underflows")
			}
			base := rva - s.Value
			if previous, ok := bases[int(s.SectionNumber)]; ok && previous != base {
				return result, fmt.Errorf("%s section %d has contradictory map positions", input.name, s.SectionNumber)
			}
			bases[int(s.SectionNumber)] = base
		}
		resolve := func(s ObjectSymbol) (uint32, error) {
			if s.SectionNumber > 0 {
				if discarded[int(s.SectionNumber)] {
					return 0, fmt.Errorf("reference to discarded section")
				}
				if b, ok := bases[int(s.SectionNumber)]; ok && uint64(b)+uint64(s.Value) < uint64(report.SizeOfImage) {
					return b + s.Value, nil
				}
			}
			if s.SectionNumber == 0 {
				if m, ok := mapped[s.Name]; ok {
					return preferredAddressRVA(report, m.PreferredAddress, "generated external target")
				}
			}
			return 0, fmt.Errorf("%s unresolved target %q in section %d", input.name, s.Name, s.SectionNumber)
		}
		familyIndexes := map[uint16]int{}
		relocationSites := map[[2]uint32]ObjectRelocation{}
		fieldBytes := map[[2]uint32]bool{}
		for _, rel := range object.RelocationRecords {
			width, err := generatedRelocationWidth(rel.Type)
			if err != nil {
				return result, err
			}
			for j := uint32(0); j < width; j++ {
				byteKey := [2]uint32{uint32(rel.SectionIndex), rel.VirtualAddress + j}
				if fieldBytes[byteKey] {
					return result, fmt.Errorf("overlapping generated relocation fields")
				}
				fieldBytes[byteKey] = true
			}
			key := [2]uint32{uint32(rel.SectionIndex), rel.VirtualAddress}
			if _, duplicate := relocationSites[key]; duplicate {
				return result, fmt.Errorf("duplicate generated relocation field")
			}
			relocationSites[key] = rel
		}
		for _, rel := range object.RelocationRecords {
			width, err := generatedRelocationWidth(rel.Type)
			if err != nil {
				return result, err
			}
			fi, ok := familyIndexes[rel.Type]
			if !ok {
				fi = len(result.Families)
				familyIndexes[rel.Type] = fi
				result.Families = append(result.Families, GeneratedRelocationFamily{Object: input.name, Type: rel.Type})
			}
			result.Observed++
			result.Families[fi].Observed++
			if discarded[rel.SectionIndex] {
				result.Discarded++
				result.Families[fi].Discarded++
				continue
			}
			section := object.Sections[rel.SectionIndex-1]
			if uint64(rel.VirtualAddress)+uint64(width) > uint64(section.RawSize) {
				return result, fmt.Errorf("generated relocation field exceeds section")
			}
			at := uint64(section.RawOffset) + uint64(rel.VirtualAddress)
			addend := uint64(binary.LittleEndian.Uint32(input.raw[at:]))
			if width == 8 {
				addend = binary.LittleEndian.Uint64(input.raw[at:])
			}
			target, err := resolve(symbols[rel.SymbolIndex])
			if err != nil {
				return result, err
			}
			var site uint32
			if section.Name == ".pdata" {
				if rel.Type != ObjectRelocationAMD64Addr32NB {
					return result, fmt.Errorf("generated .pdata requires ADDR32NB")
				}
				start := rel.VirtualAddress / 12 * 12
				var begin, end uint32
				foundBegin, foundEnd := false, false
				for _, offset := range []uint32{start, start + 4} {
					other, ok := relocationSites[[2]uint32{uint32(rel.SectionIndex), offset}]
					if !ok || other.Type != ObjectRelocationAMD64Addr32NB {
						return result, fmt.Errorf("generated .pdata lacks exact begin/end relocations")
					}
					rva, e := resolve(symbols[other.SymbolIndex])
					if e != nil {
						return result, e
					}
					off := uint64(section.RawOffset) + uint64(other.VirtualAddress)
					n := uint64(rva) + uint64(binary.LittleEndian.Uint32(input.raw[off:]))
					if n > math.MaxUint32 {
						return result, fmt.Errorf("pdata target overflows")
					}
					if other.VirtualAddress == start {
						begin = uint32(n)
						foundBegin = true
					} else {
						end = uint32(n)
						foundEnd = true
					}
				}
				matches := entrySites[[2]uint32{begin, end}]
				if !foundBegin || !foundEnd || len(matches) != 1 || rel.VirtualAddress%12%4 != 0 {
					return result, fmt.Errorf("%s .pdata field has no unique final entry", input.name)
				}
				site = matches[0] + rel.VirtualAddress%12
			} else {
				b, ok := bases[rel.SectionIndex]
				if !ok {
					return result, fmt.Errorf("%s section %d %s has no owned image position", input.name, rel.SectionIndex, section.Name)
				}
				if uint64(b)+uint64(rel.VirtualAddress) > math.MaxUint32 {
					return result, fmt.Errorf("site overflows")
				}
				site = b + rel.VirtualAddress
			}
			expected, err := generatedRelocationExpected(rel.Type, report.ImageBase, target, site, addend)
			if err != nil {
				return result, err
			}
			data, err := generatedImageBytes(image, report, site, width)
			if err != nil {
				return result, err
			}
			actual := uint64(binary.LittleEndian.Uint32(data))
			if width == 8 {
				actual = binary.LittleEndian.Uint64(data)
			}
			if actual != expected {
				return result, fmt.Errorf("%s %s relocation %#x -> %s at %#x: linked %#x, expected %#x", input.name, section.Name, rel.Type, rel.Symbol, site, actual, expected)
			}
			result.Linked++
			result.Families[fi].Linked++
			result.Values = append(result.Values, GeneratedRelocationValue{Object: input.name, Section: section.Name, SectionIndex: rel.SectionIndex, Target: rel.Symbol, Type: rel.Type, ObjectOffset: rel.VirtualAddress, SiteRVA: site, TargetRVA: target, Addend: addend, Expected: expected, Actual: actual})
		}
	}
	return result, nil
}

func generatedRelocationWidth(kind uint16) (uint32, error) {
	if kind == ObjectRelocationAMD64Addr64 {
		return 8, nil
	}
	if kind == ObjectRelocationAMD64Addr32NB || kind >= ObjectRelocationAMD64Rel32 && kind <= ObjectRelocationAMD64Rel32_5 {
		return 4, nil
	}
	return 0, fmt.Errorf("unmeasured generated relocation %#x", kind)
}

func generatedRelocationExpected(kind uint16, base uint64, target, site uint32, addend uint64) (uint64, error) {
	if kind >= ObjectRelocationAMD64Rel32 && kind <= ObjectRelocationAMD64Rel32_5 {
		n := int64(target) + int64(int32(uint32(addend))) - int64(site) - 4 - int64(kind-ObjectRelocationAMD64Rel32)
		if n < math.MinInt32 || n > math.MaxInt32 {
			return 0, fmt.Errorf("generated REL32 overflow")
		}
		return uint64(uint32(int32(n))), nil
	}
	if kind == ObjectRelocationAMD64Addr32NB {
		n := uint64(target) + addend
		if n > math.MaxUint32 {
			return 0, fmt.Errorf("generated ADDR32NB overflow")
		}
		return n, nil
	}
	if kind == ObjectRelocationAMD64Addr64 {
		if base > math.MaxUint64-uint64(target) || base+uint64(target) > math.MaxUint64-addend {
			return 0, fmt.Errorf("generated ADDR64 overflow")
		}
		return base + uint64(target) + addend, nil
	}
	return 0, fmt.Errorf("unmeasured generated relocation %#x", kind)
}
