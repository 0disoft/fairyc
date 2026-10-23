package pecoff

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"slices"
	"strings"
)

// GeneratedUnwindChild binds one source .pdata/.xdata section to its parent.
// ImageRVA names mapped .xdata; .pdata is matched against the exception directory.
type GeneratedUnwindChild struct {
	Object, Symbol, Kind             string
	SectionIndex, ParentSectionIndex int
	Retained                         bool
	SourceBytes, ImageRVA            uint32
	Relocations, MatchedPDataEntries int
	PatchedSHA256                    [32]byte
}

type GeneratedUnwindSelection struct {
	Children                                                                      []GeneratedUnwindChild
	Retained, Discarded, RelocationFields, MatchedPDataEntries, MatchedXDataBytes int
}

// InspectGeneratedUnwindSelection checks all direct associative unwind children
// of the measured generated-unit/runtime parents. Retained ADDR32NB fields are
// resolved against the owned section or exact external map symbol, then compared
// with final .xdata bytes or exception-directory entries. This is link evidence,
// not execution of the platform unwinder through every represented frame.
func InspectGeneratedUnwindSelection(unit, runtime, image, linkMap, linkLog []byte) (GeneratedUnwindSelection, error) {
	var result GeneratedUnwindSelection
	parents, err := InspectGeneratedCOMDATSelection(unit, runtime, image, linkMap, linkLog)
	if err != nil {
		return result, err
	}
	peReport, err := InspectExecutable(image)
	if err != nil {
		return result, err
	}
	if _, err = InspectAMD64Unwind(image); err != nil {
		return result, err
	}
	entries, err := runtimeFunctionEntries(image, peReport)
	if err != nil {
		return result, err
	}
	if orphaned, err := countOrphanPDataBytes(image, peReport); err != nil || orphaned != 0 {
		return result, fmt.Errorf("generated unwind image has orphaned .pdata bytes: %d (%v)", orphaned, err)
	}
	entryCounts := make(map[runtimeFunctionEntry]int, len(entries))
	for _, entry := range entries {
		entryCounts[entry]++
	}
	discards := generatedDiscardRecords(linkLog)
	for _, input := range []struct {
		name string
		raw  []byte
	}{{"unit.obj", unit}, {"runtime.obj", runtime}} {
		object, err := InspectObject(input.raw)
		if err != nil {
			return result, err
		}
		parentBySection := make(map[int]GeneratedCOMDATParent)
		bases := make(map[int]uint32)
		for _, parent := range parents.Parents {
			if parent.Object == input.name {
				parentBySection[parent.SectionIndex] = parent
				if parent.Retained {
					bases[parent.SectionIndex] = parent.ImageRVA
				}
			}
		}
		names := make(map[int][]string)
		symbols := make(map[uint32]ObjectSymbol)
		var wanted []string
		for _, symbol := range object.SymbolRecords {
			symbols[symbol.TableIndex] = symbol
			if strings.HasPrefix(symbol.Name, ".") {
				continue
			}
			wanted = append(wanted, symbol.Name)
			if symbol.SectionNumber > 0 && symbol.Value == 0 && (symbol.StorageClass == 2 || symbol.StorageClass == 3) {
				names[int(symbol.SectionNumber)] = append(names[int(symbol.SectionNumber)], symbol.Name)
			}
		}
		mapped, err := inspectMSVCMapSymbols(linkMap, wanted...)
		if err != nil {
			return result, err
		}
		for index, section := range object.Sections {
			if section.COMDATSelection != ObjectCOMDATAssociative {
				continue
			}
			parent, ok := parentBySection[section.AssociativeSection]
			if !ok {
				return result, fmt.Errorf("%s unwind child %d has no measured direct parent", input.name, index+1)
			}
			if section.Name != ".pdata" && section.Name != ".xdata" {
				return result, fmt.Errorf("%s has unmeasured associative section %s", input.name, section.Name)
			}
			if len(names[index+1]) != 1 {
				return result, fmt.Errorf("%s unwind child %d has ambiguous identity", input.name, index+1)
			}
			name := names[index+1][0]
			entry, found := mapped[name]
			owned := found && generatedObjectOwner(entry.Owner) == input.name
			_, discarded := discards[input.name+":"+name]
			if !parent.Retained {
				if owned || !discarded {
					return result, fmt.Errorf("%s:%s does not follow its discarded parent", input.name, name)
				}
			} else {
				if discarded {
					return result, fmt.Errorf("%s:%s was discarded with a retained parent", input.name, name)
				}
				if section.Name == ".xdata" {
					if !owned {
						return result, fmt.Errorf("%s:%s has no owned unwind map entry", input.name, name)
					}
					rva, err := preferredAddressRVA(peReport, entry.PreferredAddress, "generated unwind")
					if err != nil {
						return result, err
					}
					bases[index+1] = rva
				}
			}
		}
		resolve := func(symbol ObjectSymbol) (uint32, error) {
			if symbol.SectionNumber > 0 {
				if symbol.Value > object.Sections[symbol.SectionNumber-1].RawSize {
					return 0, fmt.Errorf("%s unwind target %s exceeds its section", input.name, symbol.Name)
				}
				base, ok := bases[int(symbol.SectionNumber)]
				if ok && uint64(base)+uint64(symbol.Value) < uint64(peReport.SizeOfImage) {
					return base + symbol.Value, nil
				}
				if entry, ok := mapped[symbol.Name]; ok && generatedObjectOwner(entry.Owner) == input.name {
					return preferredAddressRVA(peReport, entry.PreferredAddress, "generated local unwind relocation")
				}
			} else if symbol.SectionNumber == 0 {
				if entry, ok := mapped[symbol.Name]; ok {
					return preferredAddressRVA(peReport, entry.PreferredAddress, "generated external unwind relocation")
				}
			}
			return 0, fmt.Errorf("%s unwind target %s is unresolved", input.name, symbol.Name)
		}
		for index, section := range object.Sections {
			if section.COMDATSelection != ObjectCOMDATAssociative {
				continue
			}
			parent := parentBySection[section.AssociativeSection]
			child := GeneratedUnwindChild{Object: input.name, Symbol: names[index+1][0], Kind: section.Name, SectionIndex: index + 1, ParentSectionIndex: section.AssociativeSection, Retained: parent.Retained, SourceBytes: section.RawSize}
			if !parent.Retained {
				result.Discarded++
				result.Children = append(result.Children, child)
				continue
			}
			expected := slices.Clone(input.raw[section.RawOffset : uint64(section.RawOffset)+uint64(section.RawSize)])
			positions := make(map[uint32]bool)
			for _, relocation := range object.RelocationRecords {
				if relocation.SectionIndex != index+1 {
					continue
				}
				at := relocation.VirtualAddress
				if relocation.Type != ObjectRelocationAMD64Addr32NB || at%4 != 0 || uint64(at)+4 > uint64(len(expected)) || positions[at] {
					return result, fmt.Errorf("%s:%s has an invalid or unmeasured unwind relocation", input.name, child.Symbol)
				}
				positions[at] = true
				symbol, ok := symbols[relocation.SymbolIndex]
				if !ok {
					return result, fmt.Errorf("missing unwind relocation symbol")
				}
				if section.Name == ".pdata" {
					if at%12 != 8 {
						if int(symbol.SectionNumber) != parent.SectionIndex {
							return result, fmt.Errorf("%s:%s code relocation does not name its associated parent", input.name, child.Symbol)
						}
					} else {
						if symbol.SectionNumber <= 0 {
							return result, fmt.Errorf("generated .pdata references foreign unwind storage")
						}
						unwind := object.Sections[symbol.SectionNumber-1]
						if unwind.Name != ".xdata" || unwind.COMDATSelection != ObjectCOMDATAssociative || unwind.AssociativeSection != parent.SectionIndex {
							return result, fmt.Errorf("%s:%s unwind relocation does not name an associated sibling", input.name, child.Symbol)
						}
					}
				}
				rva, err := resolve(symbol)
				if err != nil {
					return result, err
				}
				patched := uint64(rva) + uint64(binary.LittleEndian.Uint32(expected[at:]))
				if patched > 0xffffffff {
					return result, fmt.Errorf("generated unwind RVA overflows")
				}
				binary.LittleEndian.PutUint32(expected[at:], uint32(patched))
				child.Relocations++
			}
			if section.Name == ".xdata" {
				child.ImageRVA = bases[index+1]
				actual, err := generatedImageBytes(image, peReport, child.ImageRVA, section.RawSize)
				if err != nil {
					return result, err
				}
				if len(expected) == 0 || !bytes.Equal(expected, actual) {
					return result, fmt.Errorf("%s:%s final unwind bytes differ", input.name, child.Symbol)
				}
				result.MatchedXDataBytes += len(expected)
			} else {
				if len(expected) == 0 || len(expected)%12 != 0 {
					return result, fmt.Errorf("generated .pdata has incomplete entries")
				}
				for at := 0; at < len(expected); at += 12 {
					if !positions[uint32(at)] || !positions[uint32(at+4)] || !positions[uint32(at+8)] {
						return result, fmt.Errorf("generated .pdata lacks three independent RVA relocations")
					}
					entry := runtimeFunctionEntry{binary.LittleEndian.Uint32(expected[at:]), binary.LittleEndian.Uint32(expected[at+4:]), binary.LittleEndian.Uint32(expected[at+8:])}
					if entry.begin < parent.ImageRVA || entry.end <= entry.begin || uint64(entry.end) > uint64(parent.ImageRVA)+uint64(parent.SourceBytes) || entryCounts[entry] != 1 {
						return result, fmt.Errorf("%s:%s has no exact parent-bounded runtime-function entry", input.name, child.Symbol)
					}
					child.MatchedPDataEntries++
				}
				result.MatchedPDataEntries += child.MatchedPDataEntries
			}
			child.PatchedSHA256 = sha256.Sum256(expected)
			result.RelocationFields += child.Relocations
			result.Retained++
			result.Children = append(result.Children, child)
		}
	}
	return result, nil
}
