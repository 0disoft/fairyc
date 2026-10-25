package pecoff

import (
	"cmp"
	"debug/pe"
	"fmt"
	"slices"
)

// Validate storage before interpreting symbols or relocations. Sort only a
// private index list, preserving section/symbol numbering and report order.
func verifyCOFFRawRanges(sections []*pe.Section, headerEnd, objectBytes uint64) error {
	indices := make([]int, 0, len(sections))
	for index, section := range sections {
		if section.Size == 0 {
			continue
		}
		if section.Offset == 0 {
			if section.Characteristics&sectionUninitializedData == 0 {
				return fmt.Errorf("COFF object section %q has an invalid zero raw pointer", section.Name)
			}
			continue
		}
		start := uint64(section.Offset)
		end := start + uint64(section.Size)
		if start < headerEnd || end > objectBytes {
			return fmt.Errorf("COFF object section %q has an invalid raw range", section.Name)
		}
		indices = append(indices, index)
	}
	slices.SortFunc(indices, func(left, right int) int {
		return cmp.Compare(sections[left].Offset, sections[right].Offset)
	})
	for position := 1; position < len(indices); position++ {
		left, right := indices[position-1], indices[position]
		end := uint64(sections[left].Offset) + uint64(sections[left].Size)
		if uint64(sections[right].Offset) < end {
			// Diagnose the later input section, as the incremental check did.
			return fmt.Errorf("COFF object section %q overlaps another raw range", sections[max(left, right)].Name)
		}
	}
	return nil
}
