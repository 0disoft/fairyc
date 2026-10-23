package pecoff

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"slices"
	"strings"
)

// GeneratedCOMDATParent identifies one generated-unit/runtime COMDAT definition.
// Retained bytes exclude relocation fields, whose values have separate probes.
type GeneratedCOMDATParent struct {
	Object, Symbol                string
	SectionIndex                  int
	Selection                     uint8
	SourceBytes                   uint32
	Retained                      bool
	ImageRVA                      uint32
	ComparedBytes, RelocatedBytes int
	NonRelocatedSHA256            [32]byte
}

type GeneratedCOMDATSelection struct {
	Parents                     []GeneratedCOMDATParent
	Retained, Discarded         int
	AssociativeSectionInventory int
}

// InspectGeneratedCOMDATSelection binds the production unit.obj/runtime.obj
// parents to their owned map entries and final image bytes, or to an exact
// linker discard record plus absence from the owned map. Associative children
// are inventoried only; this does not validate their unwind relocations.
func InspectGeneratedCOMDATSelection(unit, runtime, image, linkMap, linkLog []byte) (GeneratedCOMDATSelection, error) {
	var result GeneratedCOMDATSelection
	if len(linkLog) == 0 || len(linkLog) > 8*1024*1024 || bytes.IndexByte(linkLog, 0) >= 0 {
		return result, fmt.Errorf("generated COMDAT linker log has invalid size or encoding")
	}
	peReport, err := InspectExecutable(image)
	if err != nil {
		return result, err
	}
	discards := generatedDiscardRecords(linkLog)
	inputs := []struct {
		name string
		raw  []byte
	}{{"unit.obj", unit}, {"runtime.obj", runtime}}
	for _, input := range inputs {
		object, err := InspectObject(input.raw)
		if err != nil {
			return result, fmt.Errorf("%s: %w", input.name, err)
		}
		owners := make(map[int32][]string)
		var wanted []string
		for _, symbol := range object.SymbolRecords {
			if symbol.SectionNumber <= 0 || symbol.Value != 0 ||
				(symbol.StorageClass != 2 && symbol.StorageClass != 3) || strings.HasPrefix(symbol.Name, ".") {
				continue
			}
			selection := object.Sections[symbol.SectionNumber-1].COMDATSelection
			if selection != 0 && selection != ObjectCOMDATAssociative {
				owners[symbol.SectionNumber] = append(owners[symbol.SectionNumber], symbol.Name)
				wanted = append(wanted, symbol.Name)
			}
		}
		mapped, err := inspectMSVCMapSymbols(linkMap, wanted...)
		if err != nil {
			return result, err
		}
		for index, section := range object.Sections {
			if section.COMDATSelection == 0 {
				continue
			}
			if section.COMDATSelection == ObjectCOMDATAssociative {
				result.AssociativeSectionInventory++
				continue
			}
			names := owners[int32(index+1)]
			if len(names) != 1 {
				return result, fmt.Errorf("%s COMDAT section %d has %d possible owners", input.name, index+1, len(names))
			}
			name := names[0]
			selected, found := mapped[name]
			owned := found && generatedObjectOwner(selected.Owner) == input.name
			_, discarded := discards[input.name+":"+name]
			if owned == discarded {
				return result, fmt.Errorf("%s:%s requires exactly one owned map or discard record", input.name, name)
			}
			parent := GeneratedCOMDATParent{Object: input.name, Symbol: name, SectionIndex: index + 1,
				Selection: section.COMDATSelection, SourceBytes: section.RawSize, Retained: owned}
			if owned {
				rva, err := preferredAddressRVA(peReport, selected.PreferredAddress, "generated COMDAT")
				if err != nil {
					return result, err
				}
				actual, err := generatedImageBytes(image, peReport, rva, section.RawSize)
				if err != nil {
					return result, err
				}
				expected := slices.Clone(input.raw[section.RawOffset : uint64(section.RawOffset)+uint64(section.RawSize)])
				type interval struct{ start, end uint32 }
				var masked []interval
				for _, relocation := range object.RelocationRecords {
					if relocation.SectionIndex != index+1 || relocation.Type == ObjectRelocationAMD64Absolute {
						continue
					}
					width := uint32(4)
					switch relocation.Type {
					case ObjectRelocationAMD64Addr64:
						width = 8
					case ObjectRelocationAMD64Section:
						width = 2
					case ObjectRelocationAMD64Addr32, ObjectRelocationAMD64Addr32NB, ObjectRelocationAMD64Rel32,
						ObjectRelocationAMD64Rel32_1, ObjectRelocationAMD64Rel32_2, ObjectRelocationAMD64Rel32_3,
						ObjectRelocationAMD64Rel32_4, ObjectRelocationAMD64Rel32_5, ObjectRelocationAMD64SecRel:
					default:
						return result, fmt.Errorf("%s:%s has unmeasured relocation %#x", input.name, name, relocation.Type)
					}
					start := relocation.VirtualAddress
					if uint64(start)+uint64(width) > uint64(len(expected)) {
						return result, fmt.Errorf("%s:%s relocation exceeds COMDAT bytes", input.name, name)
					}
					masked = append(masked, interval{start, start + width})
				}
				slices.SortFunc(masked, func(a, b interval) int {
					if a.start < b.start {
						return -1
					}
					if a.start > b.start {
						return 1
					}
					return 0
				})
				var previous uint32
				for _, field := range masked {
					if field.start < previous {
						return result, fmt.Errorf("%s:%s has overlapping relocation fields", input.name, name)
					}
					clear(expected[field.start:field.end])
					clear(actual[field.start:field.end])
					parent.RelocatedBytes += int(field.end - field.start)
					previous = field.end
				}
				parent.ComparedBytes = len(expected) - parent.RelocatedBytes
				if parent.ComparedBytes == 0 || !bytes.Equal(expected, actual) {
					return result, fmt.Errorf("%s:%s non-relocated COMDAT bytes differ or are empty", input.name, name)
				}
				parent.ImageRVA = rva
				parent.NonRelocatedSHA256 = sha256.Sum256(expected)
				result.Retained++
			} else {
				result.Discarded++
			}
			result.Parents = append(result.Parents, parent)
		}
	}
	return result, nil
}

func generatedObjectOwner(owner string) string {
	owner = strings.ToLower(strings.ReplaceAll(owner, "\\", "/"))
	return owner[strings.LastIndexByte(owner, '/')+1:]
}

func generatedDiscardRecords(log []byte) map[string]struct{} {
	result := make(map[string]struct{})
	for _, line := range strings.Split(string(log), "\n") {
		value, ok := strings.CutPrefix(strings.TrimSpace(line), "Discarded ")
		if !ok {
			continue
		}
		for _, object := range []string{"unit.obj", "runtime.obj"} {
			if symbol, ok := strings.CutSuffix(value, " from "+object); ok {
				result[object+":"+symbol] = struct{}{}
			}
		}
	}
	return result
}

func generatedImageBytes(image []byte, report Report, rva, count uint32) ([]byte, error) {
	for _, section := range report.Sections {
		if rva < section.VirtualAddress {
			continue
		}
		offset := uint64(rva - section.VirtualAddress)
		if offset+uint64(count) > uint64(min(section.RawSize, section.VirtualSize)) {
			continue
		}
		start := uint64(section.RawOffset) + offset
		return slices.Clone(image[start : start+uint64(count)]), nil
	}
	return nil, fmt.Errorf("generated COMDAT range %#x+%d is not file-backed", rva, count)
}
