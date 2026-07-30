package pecoff

import (
	"encoding/binary"
	"fmt"
)

const (
	amd64BigObjectRawOffset    = coffBigObjectHeaderBytes + coffSectionHeaderBytes
	amd64BigObjectRawSize      = 11
	amd64BigObjectRelocOffset  = amd64BigObjectRawOffset + amd64BigObjectRawSize
	amd64BigObjectSymbolOffset = amd64BigObjectRelocOffset + coffRelocationBytes
)

// BuildAMD64BigObjectProbe creates one exact version-2 bigobj whose internal REL32 controls a
// RIP-relative load of the value 42.
func BuildAMD64BigObjectProbe(sourceSymbol string) ([]byte, error) {
	if !validShortCOFFSymbol(sourceSymbol) {
		return nil, fmt.Errorf("AMD64 bigobj has invalid short symbol identity")
	}
	const symbolCount = 2
	object := make([]byte, amd64BigObjectSymbolOffset+symbolCount*coffBigObjectSymbolBytes+4)
	binary.LittleEndian.PutUint16(object[2:], 0xffff)
	binary.LittleEndian.PutUint16(object[4:], 2)
	binary.LittleEndian.PutUint16(object[6:], machineAMD64)
	copy(object[12:28], coffBigObjectClassID[:])
	binary.LittleEndian.PutUint32(object[44:], 1)
	binary.LittleEndian.PutUint32(object[48:], amd64BigObjectSymbolOffset)
	binary.LittleEndian.PutUint32(object[52:], symbolCount)

	section := coffBigObjectHeaderBytes
	copy(object[section:], ".text")
	binary.LittleEndian.PutUint32(object[section+16:], amd64BigObjectRawSize)
	binary.LittleEndian.PutUint32(object[section+20:], amd64BigObjectRawOffset)
	binary.LittleEndian.PutUint32(object[section+24:], amd64BigObjectRelocOffset)
	binary.LittleEndian.PutUint16(object[section+32:], 1)
	binary.LittleEndian.PutUint32(object[section+36:], sectionContainsCode|sectionExecutable|sectionReadable)

	copy(object[amd64BigObjectRawOffset:], []byte{0x8b, 0x05, 0, 0, 0, 0, 0xc3, 42, 0, 0, 0})
	binary.LittleEndian.PutUint32(object[amd64BigObjectRelocOffset:], 2)
	binary.LittleEndian.PutUint32(object[amd64BigObjectRelocOffset+4:], 1)
	binary.LittleEndian.PutUint16(object[amd64BigObjectRelocOffset+8:], ObjectRelocationAMD64Rel32)

	copy(object[amd64BigObjectSymbolOffset:], sourceSymbol)
	binary.LittleEndian.PutUint32(object[amd64BigObjectSymbolOffset+12:], 1)
	binary.LittleEndian.PutUint16(object[amd64BigObjectSymbolOffset+16:], coffSymbolTypeFunction)
	object[amd64BigObjectSymbolOffset+18] = coffStorageClassExternal
	dataSymbol := amd64BigObjectSymbolOffset + coffBigObjectSymbolBytes
	copy(object[dataSymbol:], "fy_bgdat")
	binary.LittleEndian.PutUint32(object[dataSymbol+8:], 7)
	binary.LittleEndian.PutUint32(object[dataSymbol+12:], 1)
	object[dataSymbol+18] = coffStorageClassStatic
	binary.LittleEndian.PutUint32(object[dataSymbol+coffBigObjectSymbolBytes:], 4)
	if _, err := InspectObject(object); err != nil {
		return nil, fmt.Errorf("validate generated AMD64 bigobj: %w", err)
	}
	return object, nil
}
