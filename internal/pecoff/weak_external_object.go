package pecoff

import (
	"encoding/binary"
	"fmt"
)

// BuildAMD64WeakExternalProbeObject creates two leaf tail-jump paths: one unresolved weak alias
// that selects its local fallback and one weak alias intended to be replaced by a strong definition.
func BuildAMD64WeakExternalProbeObject(fallbackEntry, overrideEntry, missingWeak, overrideWeak, missingFallback, overrideFallback string) ([]byte, error) {
	return buildAMD64WeakExternalProbeObject(
		fallbackEntry, overrideEntry, missingWeak, overrideWeak, missingFallback, overrideFallback,
		coffWeakSearchAlias, coffWeakSearchAlias,
	)
}

// BuildAMD64WeakSearchProbeObject creates two leaf tail-jump paths that differ only in whether
// LINK may search an archive for the weak symbol before selecting the local fallback.
func BuildAMD64WeakSearchProbeObject(noLibraryEntry, libraryEntry, noLibraryWeak, libraryWeak, noLibraryFallback, libraryFallback string) ([]byte, error) {
	return buildAMD64WeakExternalProbeObject(
		noLibraryEntry, libraryEntry, noLibraryWeak, libraryWeak, noLibraryFallback, libraryFallback,
		coffWeakSearchNoLibrary, coffWeakSearchLibrary,
	)
}

func buildAMD64WeakExternalProbeObject(firstEntry, secondEntry, firstWeak, secondWeak, firstFallback, secondFallback string, firstSearch, secondSearch uint32) ([]byte, error) {
	names := []string{firstEntry, secondEntry, firstWeak, secondWeak, firstFallback, secondFallback}
	seen := make(map[string]struct{}, len(names))
	for _, name := range names {
		if !validShortCOFFSymbol(name) {
			return nil, fmt.Errorf("AMD64 weak-external object has invalid short symbol identity")
		}
		if _, duplicate := seen[name]; duplicate {
			return nil, fmt.Errorf("AMD64 weak-external object repeats symbol %q", name)
		}
		seen[name] = struct{}{}
	}
	const (
		rawOffset   = coffFileHeaderBytes + coffSectionHeaderBytes
		rawSize     = 22
		relocOffset = rawOffset + rawSize
		symbols     = relocOffset + 2*coffRelocationBytes
		symbolCount = 8
	)
	object := make([]byte, symbols+symbolCount*18+4)
	binary.LittleEndian.PutUint16(object, machineAMD64)
	binary.LittleEndian.PutUint16(object[2:], 1)
	binary.LittleEndian.PutUint32(object[8:], symbols)
	binary.LittleEndian.PutUint32(object[12:], symbolCount)

	section := coffFileHeaderBytes
	copy(object[section:], ".text")
	binary.LittleEndian.PutUint32(object[section+16:], rawSize)
	binary.LittleEndian.PutUint32(object[section+20:], rawOffset)
	binary.LittleEndian.PutUint32(object[section+24:], relocOffset)
	binary.LittleEndian.PutUint16(object[section+32:], 2)
	binary.LittleEndian.PutUint32(object[section+36:], sectionContainsCode|sectionExecutable|sectionReadable)

	// Both entry points are leaf tail jumps, so they need no unwind record. Their local fallbacks
	// return distinct values; a separate strong definition supplies the override value at link time.
	copy(object[rawOffset:], []byte{0xe9, 0, 0, 0, 0, 0xe9, 0, 0, 0, 0, 0xb8, 17, 0, 0, 0, 0xc3, 0xb8, 19, 0, 0, 0, 0xc3})
	for index, site := range []uint32{1, 6} {
		relocation := relocOffset + index*coffRelocationBytes
		binary.LittleEndian.PutUint32(object[relocation:], site)
		targetIndex := uint32(2)
		if index == 1 {
			targetIndex = 4
		}
		binary.LittleEndian.PutUint32(object[relocation+4:], targetIndex)
		binary.LittleEndian.PutUint16(object[relocation+8:], ObjectRelocationAMD64Rel32)
	}

	emitWeakProbeSymbol(object[symbols:], firstEntry, 0, 1, coffSymbolTypeFunction, coffStorageClassExternal, 0)
	emitWeakProbeSymbol(object[symbols+18:], secondEntry, 5, 1, coffSymbolTypeFunction, coffStorageClassExternal, 0)
	emitWeakProbeSymbol(object[symbols+2*18:], firstWeak, 0, 0, coffSymbolTypeFunction, coffStorageClassWeakExternal, 1)
	binary.LittleEndian.PutUint32(object[symbols+3*18:], 6)
	binary.LittleEndian.PutUint32(object[symbols+3*18+4:], firstSearch)
	emitWeakProbeSymbol(object[symbols+4*18:], secondWeak, 0, 0, coffSymbolTypeFunction, coffStorageClassWeakExternal, 1)
	binary.LittleEndian.PutUint32(object[symbols+5*18:], 7)
	binary.LittleEndian.PutUint32(object[symbols+5*18+4:], secondSearch)
	emitWeakProbeSymbol(object[symbols+6*18:], firstFallback, 10, 1, coffSymbolTypeFunction, coffStorageClassExternal, 0)
	emitWeakProbeSymbol(object[symbols+7*18:], secondFallback, 16, 1, coffSymbolTypeFunction, coffStorageClassExternal, 0)
	binary.LittleEndian.PutUint32(object[symbols+symbolCount*18:], 4)
	if _, err := InspectObject(object); err != nil {
		return nil, fmt.Errorf("validate generated AMD64 weak-external object: %w", err)
	}
	return object, nil
}

func emitWeakProbeSymbol(destination []byte, name string, value uint32, section int16, symbolType uint16, storageClass, auxiliary uint8) {
	copy(destination, name)
	binary.LittleEndian.PutUint32(destination[8:], value)
	binary.LittleEndian.PutUint16(destination[12:], uint16(section))
	binary.LittleEndian.PutUint16(destination[14:], symbolType)
	destination[16] = storageClass
	destination[17] = auxiliary
}
