package pecoff

import (
	"encoding/binary"
	"fmt"
)

const (
	amd64RelocationOverflowProbeCount = 0xffff
	amd64RelocationOverflowRawSize    = 11
)

// BuildAMD64RelocationOverflowProbeObject creates one leaf-function COFF object with the
// smallest extended relocation table emitted by LLVM-compatible COFF producers: 65,535
// semantic relocations preceded by one count sentinel.
func BuildAMD64RelocationOverflowProbeObject(sourceSymbol string) ([]byte, error) {
	if !validShortCOFFSymbol(sourceSymbol) {
		return nil, fmt.Errorf("AMD64 relocation-overflow object has invalid short symbol identity")
	}
	const (
		rawOffset     = coffFileHeaderBytes + coffSectionHeaderBytes
		rawSize       = amd64RelocationOverflowRawSize
		relocOffset   = rawOffset + rawSize
		storedRecords = amd64RelocationOverflowProbeCount + 1
		symbolOffset  = relocOffset + storedRecords*coffRelocationBytes
	)
	object := make([]byte, symbolOffset+2*18+4)
	binary.LittleEndian.PutUint16(object, machineAMD64)
	binary.LittleEndian.PutUint16(object[2:], 1)
	binary.LittleEndian.PutUint32(object[8:], symbolOffset)
	binary.LittleEndian.PutUint32(object[12:], 2)

	section := coffFileHeaderBytes
	copy(object[section:], ".text")
	binary.LittleEndian.PutUint32(object[section+16:], rawSize)
	binary.LittleEndian.PutUint32(object[section+20:], rawOffset)
	binary.LittleEndian.PutUint32(object[section+24:], relocOffset)
	binary.LittleEndian.PutUint16(object[section+32:], 0xffff)
	binary.LittleEndian.PutUint32(object[section+36:], sectionContainsCode|sectionExecutable|sectionReadable|sectionRelocationOverflow)

	// The final semantic relocation binds the RIP-relative load to the adjacent constant. If LINK
	// truncates the overflow table at the 16-bit marker, the function reads the wrong four bytes.
	copy(object[rawOffset:], []byte{0x8b, 0x05, 0, 0, 0, 0, 0xc3, 42, 0, 0, 0})
	binary.LittleEndian.PutUint32(object[relocOffset:], storedRecords)
	for index := 0; index < amd64RelocationOverflowProbeCount; index++ {
		relocation := relocOffset + (index+1)*coffRelocationBytes
		binary.LittleEndian.PutUint32(object[relocation+4:], 0)
		binary.LittleEndian.PutUint16(object[relocation+8:], ObjectRelocationAMD64Absolute)
	}
	lastRelocation := relocOffset + amd64RelocationOverflowProbeCount*coffRelocationBytes
	binary.LittleEndian.PutUint32(object[lastRelocation:], 2)
	binary.LittleEndian.PutUint32(object[lastRelocation+4:], 1)
	binary.LittleEndian.PutUint16(object[lastRelocation+8:], ObjectRelocationAMD64Rel32)

	copy(object[symbolOffset:], sourceSymbol)
	binary.LittleEndian.PutUint16(object[symbolOffset+12:], 1)
	binary.LittleEndian.PutUint16(object[symbolOffset+14:], coffSymbolTypeFunction)
	object[symbolOffset+16] = coffStorageClassExternal
	dataSymbol := symbolOffset + 18
	copy(object[dataSymbol:], "fy_ovdat")
	binary.LittleEndian.PutUint32(object[dataSymbol+8:], 7)
	binary.LittleEndian.PutUint16(object[dataSymbol+12:], 1)
	object[dataSymbol+16] = coffStorageClassStatic
	binary.LittleEndian.PutUint32(object[symbolOffset+2*18:], 4)
	if _, err := InspectObject(object); err != nil {
		return nil, fmt.Errorf("validate generated AMD64 relocation-overflow object: %w", err)
	}
	return object, nil
}
