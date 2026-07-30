package pecoff

import (
	"bytes"
	"debug/pe"
	"encoding/binary"
	"fmt"
	"math"
	"path/filepath"
	"strings"
)

// AMD64RelocationOverflowSpec identifies the generated extended-relocation object in one image.
type AMD64RelocationOverflowSpec struct {
	SourceSymbol string
	SourceObject string
}

// AMD64RelocationOverflowReport records the bounded object-to-map-to-image proof.
type AMD64RelocationOverflowReport struct {
	Relocations          int
	SourceRVA            uint32
	LinkedDisplacement   int32
	ExpectedDisplacement int32
}

// InspectAMD64RelocationOverflowLink proves that LINK consumed the exact extended-relocation
// object and preserved its leaf function in executable image bytes.
func InspectAMD64RelocationOverflowLink(object, image, linkMap []byte, spec AMD64RelocationOverflowSpec) (AMD64RelocationOverflowReport, error) {
	if !validShortCOFFSymbol(spec.SourceSymbol) || filepath.Base(spec.SourceObject) != spec.SourceObject ||
		!strings.EqualFold(filepath.Ext(spec.SourceObject), ".obj") {
		return AMD64RelocationOverflowReport{}, fmt.Errorf("AMD64 relocation-overflow spec has invalid identities")
	}
	objectReport, err := InspectObject(object)
	if err != nil {
		return AMD64RelocationOverflowReport{}, fmt.Errorf("inspect AMD64 relocation-overflow object: %w", err)
	}
	if objectReport.Relocations != amd64RelocationOverflowProbeCount {
		return AMD64RelocationOverflowReport{}, fmt.Errorf("AMD64 relocation-overflow object has %d relocations, want %d", objectReport.Relocations, amd64RelocationOverflowProbeCount)
	}
	source, err := exactObjectSymbol(objectReport.SymbolRecords, spec.SourceSymbol)
	if err != nil {
		return AMD64RelocationOverflowReport{}, err
	}
	if err := requireDefinedExternalFunction(objectReport, spec.SourceSymbol, "relocation-overflow source"); err != nil {
		return AMD64RelocationOverflowReport{}, err
	}
	mapSymbols, err := inspectMSVCMapSymbols(linkMap, spec.SourceSymbol)
	if err != nil {
		return AMD64RelocationOverflowReport{}, err
	}
	mapSource, found := mapSymbols[spec.SourceSymbol]
	if !found || !msvcMapOwnerIsObject(mapSource.Owner, spec.SourceObject) {
		return AMD64RelocationOverflowReport{}, fmt.Errorf("MSVC link map relocation-overflow source owner is %q, want %q", mapSource.Owner, spec.SourceObject)
	}
	imageReport, err := InspectExecutable(image)
	if err != nil {
		return AMD64RelocationOverflowReport{}, fmt.Errorf("inspect AMD64 relocation-overflow image: %w", err)
	}
	sourceRVA, err := preferredAddressRVA(imageReport, mapSource.PreferredAddress, "relocation-overflow source")
	if err != nil {
		return AMD64RelocationOverflowReport{}, err
	}
	file, err := pe.NewFile(bytes.NewReader(object))
	if err != nil {
		return AMD64RelocationOverflowReport{}, fmt.Errorf("reparse AMD64 relocation-overflow object: %w", err)
	}
	defer file.Close()
	section := file.Sections[source.SectionNumber-1]
	if source.Value != 0 || section.Size != 11 || uint64(section.Offset)+uint64(section.Size) > uint64(len(object)) {
		return AMD64RelocationOverflowReport{}, fmt.Errorf("AMD64 relocation-overflow source bytes are out of range")
	}
	data, err := exactObjectSymbol(objectReport.SymbolRecords, "fy_ovdat")
	if err != nil || data.SectionNumber != source.SectionNumber || data.Value != 7 || data.StorageClass != coffStorageClassStatic {
		return AMD64RelocationOverflowReport{}, fmt.Errorf("AMD64 relocation-overflow data symbol is invalid")
	}
	absCount := 0
	var finalRelocation ObjectRelocation
	for index, relocation := range objectReport.RelocationRecords {
		if relocation.Type == ObjectRelocationAMD64Absolute {
			absCount++
			continue
		}
		if relocation.Type != ObjectRelocationAMD64Rel32 || relocation.SectionIndex != int(source.SectionNumber) ||
			relocation.VirtualAddress != 2 || relocation.SymbolIndex != data.TableIndex || relocation.Symbol != data.Name || finalRelocation.Symbol != "" {
			return AMD64RelocationOverflowReport{}, fmt.Errorf("AMD64 relocation-overflow final semantic record is invalid")
		}
		if index != len(objectReport.RelocationRecords)-1 {
			return AMD64RelocationOverflowReport{}, fmt.Errorf("AMD64 relocation-overflow REL32 is record %d, want final record %d", index, len(objectReport.RelocationRecords)-1)
		}
		finalRelocation = relocation
	}
	if absCount != amd64RelocationOverflowProbeCount-1 || finalRelocation.Symbol == "" {
		return AMD64RelocationOverflowReport{}, fmt.Errorf("AMD64 relocation-overflow record distribution is invalid")
	}
	sourceBytes := object[section.Offset : section.Offset+section.Size]
	if !bytes.Equal(sourceBytes[:2], []byte{0x8b, 0x05}) || !bytes.Equal(sourceBytes[6:], []byte{0xc3, 42, 0, 0, 0}) {
		return AMD64RelocationOverflowReport{}, fmt.Errorf("AMD64 relocation-overflow source encoding is invalid")
	}
	objectAddend := int32(binary.LittleEndian.Uint32(sourceBytes[2:]))
	imageOffset, imageSection, err := mapImageRVA(imageReport, sourceRVA, uint32(len(sourceBytes)))
	if err != nil {
		return AMD64RelocationOverflowReport{}, fmt.Errorf("map AMD64 relocation-overflow source bytes: %w", err)
	}
	if imageSection.Characteristics&sectionContainsCode == 0 || imageSection.Characteristics&sectionExecutable == 0 {
		return AMD64RelocationOverflowReport{}, fmt.Errorf("AMD64 relocation-overflow source is not executable")
	}
	linkedBytes := image[imageOffset : imageOffset+len(sourceBytes)]
	if !bytes.Equal(linkedBytes[:2], sourceBytes[:2]) || !bytes.Equal(linkedBytes[6:], sourceBytes[6:]) {
		return AMD64RelocationOverflowReport{}, fmt.Errorf("AMD64 relocation-overflow source bytes drifted in the linked image")
	}
	linkedDisplacement := int32(binary.LittleEndian.Uint32(linkedBytes[2:]))
	expected := int64(sourceRVA) + int64(data.Value) + int64(objectAddend) - (int64(sourceRVA) + 2 + 4)
	if expected < math.MinInt32 || expected > math.MaxInt32 || linkedDisplacement != int32(expected) {
		return AMD64RelocationOverflowReport{}, fmt.Errorf("AMD64 relocation-overflow linked displacement is %d, want %d", linkedDisplacement, expected)
	}
	return AMD64RelocationOverflowReport{
		Relocations: objectReport.Relocations, SourceRVA: sourceRVA,
		LinkedDisplacement: linkedDisplacement, ExpectedDisplacement: int32(expected),
	}, nil
}
