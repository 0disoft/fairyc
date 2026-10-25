package pecoff

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// debug/pe eagerly materializes every section's relocation array. Bound that
// work before parsing, including sections which alias the same on-disk table.
// Full range, overflow-marker and symbol validation stays in InspectObject.
func verifyCOFFRelocationBudget(object []byte) error {
	sections := int(binary.LittleEndian.Uint16(object[2:4]))
	if sections == 0 || sections > maxObjectSections {
		return fmt.Errorf("COFF object has invalid section count %d", sections)
	}
	table := coffFileHeaderBytes + int(binary.LittleEndian.Uint16(object[16:18]))
	if table+sections*coffSectionHeaderBytes > len(object) {
		// debug/pe rejects truncated section headers before reading relocations.
		return nil
	}
	remaining := uint64(maxObjectRelocations)
	for index := range sections {
		header := object[table+index*coffSectionHeaderBytes:]
		count := uint64(binary.LittleEndian.Uint16(header[32:34]))
		if count == 0xffff && binary.LittleEndian.Uint32(header[36:40])&sectionRelocationOverflow != 0 {
			start := uint64(binary.LittleEndian.Uint32(header[24:28]))
			if start+coffRelocationBytes <= uint64(len(object)) {
				stored := uint64(binary.LittleEndian.Uint32(object[start:]))
				if stored > 0 {
					// Bound both debug/pe's 16-bit count and our extended count.
					count = max(count, stored-1)
				}
			}
		}
		if count > maxObjectRelocations {
			name := bytes.TrimRight(header[:8], "\x00")
			return fmt.Errorf("COFF object section %q has %d relocations, limit is %d", name, count, maxObjectRelocations)
		}
		if count > remaining {
			return fmt.Errorf("COFF object has more than %d relocations", maxObjectRelocations)
		}
		remaining -= count
	}
	return nil
}
