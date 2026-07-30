package pecoff

import (
	"encoding/binary"
	"fmt"
	"strings"
)

const amd64REL32VariantCount = 6

// BuildAMD64REL32VariantProbeObject creates one closed leaf-function COFF object whose skipped
// data table contains exactly one relocation of each REL32 through REL32_5 type.
func BuildAMD64REL32VariantProbeObject(sourceSymbol, targetSymbol string) ([]byte, error) {
	if !validShortCOFFSymbol(sourceSymbol) || !validShortCOFFSymbol(targetSymbol) || sourceSymbol == targetSymbol {
		return nil, fmt.Errorf("AMD64 REL32 variant object has invalid short symbol identities")
	}
	const (
		rawOffset   = coffFileHeaderBytes + coffSectionHeaderBytes
		rawSize     = 47
		relocOffset = rawOffset + rawSize
		symbols     = relocOffset + amd64REL32VariantCount*coffRelocationBytes
	)
	object := make([]byte, symbols+2*18+4)
	binary.LittleEndian.PutUint16(object, machineAMD64)
	binary.LittleEndian.PutUint16(object[2:], 1)
	binary.LittleEndian.PutUint32(object[8:], symbols)
	binary.LittleEndian.PutUint32(object[12:], 2)

	section := coffFileHeaderBytes
	copy(object[section:], ".text")
	binary.LittleEndian.PutUint32(object[section+16:], rawSize)
	binary.LittleEndian.PutUint32(object[section+20:], rawOffset)
	binary.LittleEndian.PutUint32(object[section+24:], relocOffset)
	binary.LittleEndian.PutUint16(object[section+32:], amd64REL32VariantCount)
	binary.LittleEndian.PutUint32(object[section+36:], sectionContainsCode|sectionExecutable|sectionReadable)

	// Jump over all relocation slots. The table is never executed; the leaf body returns 42.
	object[rawOffset] = 0xeb
	object[rawOffset+1] = 39
	site := uint32(2)
	for trailing := uint32(0); trailing < amd64REL32VariantCount; trailing++ {
		addend := int32(trailing*3) - 7
		binary.LittleEndian.PutUint32(object[rawOffset+int(site):], uint32(addend))
		for marker := uint32(0); marker < trailing; marker++ {
			object[rawOffset+int(site+4+marker)] = byte(0xa0 + trailing*8 + marker)
		}
		relocation := relocOffset + int(trailing)*coffRelocationBytes
		binary.LittleEndian.PutUint32(object[relocation:], site)
		binary.LittleEndian.PutUint32(object[relocation+4:], 1)
		binary.LittleEndian.PutUint16(object[relocation+8:], ObjectRelocationAMD64Rel32+uint16(trailing))
		site += 4 + trailing
	}
	if site != 41 {
		return nil, fmt.Errorf("AMD64 REL32 variant object layout drifted to %d", site)
	}
	copy(object[rawOffset+int(site):], []byte{0xb8, 42, 0, 0, 0, 0xc3})

	copy(object[symbols:], sourceSymbol)
	binary.LittleEndian.PutUint16(object[symbols+12:], 1)
	binary.LittleEndian.PutUint16(object[symbols+14:], coffSymbolTypeFunction)
	object[symbols+16] = coffStorageClassExternal
	target := symbols + 18
	copy(object[target:], targetSymbol)
	binary.LittleEndian.PutUint16(object[target+14:], coffSymbolTypeFunction)
	object[target+16] = coffStorageClassExternal
	binary.LittleEndian.PutUint32(object[target+18:], 4)
	if _, err := InspectObject(object); err != nil {
		return nil, fmt.Errorf("validate generated AMD64 REL32 variant object: %w", err)
	}
	return object, nil
}

func validShortCOFFSymbol(name string) bool {
	return name != "" && len(name) <= 8 && !strings.ContainsAny(name, "\x00 \t\r\n")
}
