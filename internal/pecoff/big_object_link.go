package pecoff

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"path/filepath"
	"strings"
)

// AMD64BigObjectSpec identifies one generated bigobj in a final MSVC image.
type AMD64BigObjectSpec struct {
	SourceSymbol string
	SourceObject string
}

// AMD64BigObjectReport records the exact 32-bit-symbol object-to-map-to-image proof.
type AMD64BigObjectReport struct {
	Version              uint16
	SourceRVA            uint32
	LinkedDisplacement   int32
	ExpectedDisplacement int32
}

// InspectAMD64BigObjectLink proves one exact version-2 bigobj symbol and relocation path through
// the MSVC map and final image.
func InspectAMD64BigObjectLink(object, image, linkMap []byte, spec AMD64BigObjectSpec) (AMD64BigObjectReport, error) {
	if !validShortCOFFSymbol(spec.SourceSymbol) || filepath.Base(spec.SourceObject) != spec.SourceObject ||
		!strings.EqualFold(filepath.Ext(spec.SourceObject), ".obj") {
		return AMD64BigObjectReport{}, fmt.Errorf("AMD64 bigobj spec has invalid identities")
	}
	objectReport, err := InspectObject(object)
	if err != nil {
		return AMD64BigObjectReport{}, fmt.Errorf("inspect AMD64 bigobj: %w", err)
	}
	if !objectReport.BigObject || objectReport.BigObjectVersion != 2 || len(objectReport.Sections) != 1 || objectReport.Relocations != 1 {
		return AMD64BigObjectReport{}, fmt.Errorf("AMD64 bigobj receipt is incomplete")
	}
	source, err := exactObjectSymbol(objectReport.SymbolRecords, spec.SourceSymbol)
	if err != nil {
		return AMD64BigObjectReport{}, err
	}
	if source.SectionNumber != 1 || source.Value != 0 || source.Type != coffSymbolTypeFunction || source.StorageClass != coffStorageClassExternal {
		return AMD64BigObjectReport{}, fmt.Errorf("AMD64 bigobj source symbol is invalid")
	}
	data, err := exactObjectSymbol(objectReport.SymbolRecords, "fy_bgdat")
	if err != nil {
		return AMD64BigObjectReport{}, err
	}
	if data.SectionNumber != 1 || data.Value != 7 || data.StorageClass != coffStorageClassStatic {
		return AMD64BigObjectReport{}, fmt.Errorf("AMD64 bigobj data symbol is invalid")
	}
	relocation := objectReport.RelocationRecords[0]
	if relocation.SectionIndex != 1 || relocation.Section != objectReport.Sections[0].Name || relocation.VirtualAddress != 2 ||
		relocation.SymbolIndex != data.TableIndex || relocation.Symbol != data.Name || relocation.Type != ObjectRelocationAMD64Rel32 {
		return AMD64BigObjectReport{}, fmt.Errorf("AMD64 bigobj relocation is invalid")
	}
	mapSymbols, err := inspectMSVCMapSymbols(linkMap, spec.SourceSymbol)
	if err != nil {
		return AMD64BigObjectReport{}, err
	}
	mapSource, found := mapSymbols[spec.SourceSymbol]
	if !found || !msvcMapOwnerIsObject(mapSource.Owner, spec.SourceObject) {
		return AMD64BigObjectReport{}, fmt.Errorf("MSVC link map bigobj source owner is %q, want %q", mapSource.Owner, spec.SourceObject)
	}
	imageReport, err := InspectExecutable(image)
	if err != nil {
		return AMD64BigObjectReport{}, fmt.Errorf("inspect AMD64 bigobj image: %w", err)
	}
	sourceRVA, err := preferredAddressRVA(imageReport, mapSource.PreferredAddress, "bigobj source")
	if err != nil {
		return AMD64BigObjectReport{}, err
	}
	section := objectReport.Sections[0]
	if section.RawSize != amd64BigObjectRawSize || uint64(section.RawOffset)+uint64(section.RawSize) > uint64(len(object)) {
		return AMD64BigObjectReport{}, fmt.Errorf("AMD64 bigobj source bytes are out of range")
	}
	sourceBytes := object[section.RawOffset : section.RawOffset+section.RawSize]
	if !bytes.Equal(sourceBytes[:2], []byte{0x8b, 0x05}) || !bytes.Equal(sourceBytes[6:], []byte{0xc3, 42, 0, 0, 0}) {
		return AMD64BigObjectReport{}, fmt.Errorf("AMD64 bigobj source encoding is invalid")
	}
	objectAddend := int32(binary.LittleEndian.Uint32(sourceBytes[2:]))
	imageOffset, imageSection, err := mapImageRVA(imageReport, sourceRVA, section.RawSize)
	if err != nil {
		return AMD64BigObjectReport{}, fmt.Errorf("map AMD64 bigobj source bytes: %w", err)
	}
	if imageSection.Characteristics&sectionContainsCode == 0 || imageSection.Characteristics&sectionExecutable == 0 {
		return AMD64BigObjectReport{}, fmt.Errorf("AMD64 bigobj source is not executable")
	}
	linkedBytes := image[imageOffset : imageOffset+int(section.RawSize)]
	if !bytes.Equal(linkedBytes[:2], sourceBytes[:2]) || !bytes.Equal(linkedBytes[6:], sourceBytes[6:]) {
		return AMD64BigObjectReport{}, fmt.Errorf("AMD64 bigobj source bytes drifted in the linked image")
	}
	linkedDisplacement := int32(binary.LittleEndian.Uint32(linkedBytes[2:]))
	expected := int64(sourceRVA) + int64(data.Value) + int64(objectAddend) - (int64(sourceRVA) + int64(relocation.VirtualAddress) + 4)
	if expected < math.MinInt32 || expected > math.MaxInt32 || linkedDisplacement != int32(expected) {
		return AMD64BigObjectReport{}, fmt.Errorf("AMD64 bigobj linked displacement is %d, want %d", linkedDisplacement, expected)
	}
	return AMD64BigObjectReport{
		Version: objectReport.BigObjectVersion, SourceRVA: sourceRVA,
		LinkedDisplacement: linkedDisplacement, ExpectedDisplacement: int32(expected),
	}, nil
}
