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

// AMD64REL32VariantsSpec identifies one generated six-variant object and its linked target.
type AMD64REL32VariantsSpec struct {
	SourceSymbol string
	TargetSymbol string
	SourceObject string
	TargetObject string
}

// AMD64REL32VariantValue records one independently recomputed linked displacement.
type AMD64REL32VariantValue struct {
	Type                 uint16
	TrailingBytes        uint32
	ObjectVirtualAddress uint32
	ObjectAddend         int32
	LinkedSiteRVA        uint32
	LinkedDisplacement   int32
	ExpectedDisplacement int32
}

// AMD64REL32VariantsReport records all six REL32 through REL32_5 formulas.
type AMD64REL32VariantsReport struct {
	SourceRVA uint32
	TargetRVA uint32
	Variants  []AMD64REL32VariantValue
}

// InspectAMD64REL32VariantsLink proves that one exact object contributes one relocation of every
// REL32 variant and that LINK applies S + A - (P + 4 + trailingBytes) at each final site.
func InspectAMD64REL32VariantsLink(object, image, linkMap []byte, spec AMD64REL32VariantsSpec) (AMD64REL32VariantsReport, error) {
	if err := validateAMD64REL32VariantsSpec(spec); err != nil {
		return AMD64REL32VariantsReport{}, err
	}
	objectReport, err := InspectObject(object)
	if err != nil {
		return AMD64REL32VariantsReport{}, fmt.Errorf("inspect AMD64 REL32 variants object: %w", err)
	}
	source, err := exactObjectSymbol(objectReport.SymbolRecords, spec.SourceSymbol)
	if err != nil {
		return AMD64REL32VariantsReport{}, err
	}
	if err := requireDefinedExternalFunction(objectReport, spec.SourceSymbol, "REL32 variants source"); err != nil {
		return AMD64REL32VariantsReport{}, err
	}
	target, err := exactObjectSymbol(objectReport.SymbolRecords, spec.TargetSymbol)
	if err != nil {
		return AMD64REL32VariantsReport{}, err
	}
	if target.SectionNumber != 0 {
		return AMD64REL32VariantsReport{}, fmt.Errorf("AMD64 REL32 variants target %q is not undefined", target.Name)
	}
	if err := requireREL32FunctionSymbol(target, "variants target"); err != nil {
		return AMD64REL32VariantsReport{}, err
	}

	mapSymbols, err := inspectMSVCMapSymbols(linkMap, spec.SourceSymbol, spec.TargetSymbol)
	if err != nil {
		return AMD64REL32VariantsReport{}, err
	}
	sourceMap, found := mapSymbols[spec.SourceSymbol]
	if !found || !msvcMapOwnerIsObject(sourceMap.Owner, spec.SourceObject) {
		return AMD64REL32VariantsReport{}, fmt.Errorf("MSVC link map REL32 variants source owner is %q, want %q", sourceMap.Owner, spec.SourceObject)
	}
	targetMap, found := mapSymbols[spec.TargetSymbol]
	if !found || !msvcMapOwnerIsObject(targetMap.Owner, spec.TargetObject) {
		return AMD64REL32VariantsReport{}, fmt.Errorf("MSVC link map REL32 variants target owner is %q, want %q", targetMap.Owner, spec.TargetObject)
	}
	imageReport, err := InspectExecutable(image)
	if err != nil {
		return AMD64REL32VariantsReport{}, fmt.Errorf("inspect AMD64 REL32 variants image: %w", err)
	}
	sourceRVA, err := preferredAddressRVA(imageReport, sourceMap.PreferredAddress, "REL32 variants source")
	if err != nil {
		return AMD64REL32VariantsReport{}, err
	}
	targetRVA, err := preferredAddressRVA(imageReport, targetMap.PreferredAddress, "REL32 variants target")
	if err != nil {
		return AMD64REL32VariantsReport{}, err
	}
	if err := requireExecutableRVARange(imageReport, sourceRVA, sourceRVA+1); err != nil {
		return AMD64REL32VariantsReport{}, fmt.Errorf("REL32 variants source is not executable: %w", err)
	}
	if err := requireExecutableRVARange(imageReport, targetRVA, targetRVA+1); err != nil {
		return AMD64REL32VariantsReport{}, fmt.Errorf("REL32 variants target is not executable: %w", err)
	}

	file, err := pe.NewFile(bytes.NewReader(object))
	if err != nil {
		return AMD64REL32VariantsReport{}, fmt.Errorf("reparse AMD64 REL32 variants object: %w", err)
	}
	defer file.Close()
	section := file.Sections[source.SectionNumber-1]
	sectionReceipt := objectReport.Sections[source.SectionNumber-1]
	functionStart := uint64(section.VirtualAddress) + uint64(source.Value)
	functionEnd := uint64(section.VirtualAddress) + uint64(section.Size)
	if functionStart >= functionEnd || functionEnd > math.MaxUint32 {
		return AMD64REL32VariantsReport{}, fmt.Errorf("AMD64 REL32 variants source range is invalid")
	}
	variants := make([]AMD64REL32VariantValue, 0, amd64REL32VariantCount)
	for trailing := uint32(0); trailing < amd64REL32VariantCount; trailing++ {
		relocationType := ObjectRelocationAMD64Rel32 + uint16(trailing)
		relocation, err := exactREL32VariantRelocation(
			objectReport.RelocationRecords, int(source.SectionNumber), sectionReceipt.Name, target.TableIndex,
			relocationType, uint32(functionStart), uint32(functionEnd),
		)
		if err != nil {
			return AMD64REL32VariantsReport{}, err
		}
		if relocation.VirtualAddress < section.VirtualAddress {
			return AMD64REL32VariantsReport{}, fmt.Errorf("AMD64 REL32_%d variant precedes its section", trailing)
		}
		sectionOffset := uint64(relocation.VirtualAddress - section.VirtualAddress)
		if sectionOffset+4+uint64(trailing) > uint64(section.Size) {
			return AMD64REL32VariantsReport{}, fmt.Errorf("AMD64 REL32_%d variant exceeds its source bytes", trailing)
		}
		objectOffset := uint64(section.Offset) + sectionOffset
		if objectOffset+4+uint64(trailing) > uint64(len(object)) {
			return AMD64REL32VariantsReport{}, fmt.Errorf("AMD64 REL32_%d object bytes are out of range", trailing)
		}
		objectAddend := int32(binary.LittleEndian.Uint32(object[objectOffset:]))
		sourceRelativeSite := sectionOffset - uint64(source.Value)
		linkedSite := uint64(sourceRVA) + sourceRelativeSite
		if linkedSite == 0 || linkedSite+4+uint64(trailing) > math.MaxUint32 {
			return AMD64REL32VariantsReport{}, fmt.Errorf("AMD64 REL32_%d linked site overflows", trailing)
		}
		linkedSiteRVA := uint32(linkedSite)
		linkedOffset, linkedSection, err := mapImageRVA(imageReport, linkedSiteRVA, 4+trailing)
		if err != nil {
			return AMD64REL32VariantsReport{}, fmt.Errorf("map AMD64 REL32_%d linked bytes: %w", trailing, err)
		}
		if linkedSection.Characteristics&sectionContainsCode == 0 || linkedSection.Characteristics&sectionExecutable == 0 {
			return AMD64REL32VariantsReport{}, fmt.Errorf("AMD64 REL32_%d linked site is not executable", trailing)
		}
		if trailing > 0 && !bytes.Equal(object[objectOffset+4:objectOffset+4+uint64(trailing)], image[linkedOffset+4:linkedOffset+4+int(trailing)]) {
			return AMD64REL32VariantsReport{}, fmt.Errorf("AMD64 REL32_%d trailing bytes drifted", trailing)
		}
		linkedDisplacement := int32(binary.LittleEndian.Uint32(image[linkedOffset:]))
		expected, err := amd64REL32VariantDisplacement(targetRVA, objectAddend, linkedSiteRVA, trailing)
		if err != nil {
			return AMD64REL32VariantsReport{}, err
		}
		if linkedDisplacement != expected {
			return AMD64REL32VariantsReport{}, fmt.Errorf("AMD64 REL32_%d linked displacement is %d, want %d", trailing, linkedDisplacement, expected)
		}
		variants = append(variants, AMD64REL32VariantValue{
			Type: relocationType, TrailingBytes: trailing, ObjectVirtualAddress: relocation.VirtualAddress,
			ObjectAddend: objectAddend, LinkedSiteRVA: linkedSiteRVA,
			LinkedDisplacement: linkedDisplacement, ExpectedDisplacement: expected,
		})
	}
	return AMD64REL32VariantsReport{SourceRVA: sourceRVA, TargetRVA: targetRVA, Variants: variants}, nil
}

func exactREL32VariantRelocation(relocations []ObjectRelocation, sectionIndex int, section string, targetIndex uint32, relocationType uint16, functionStart, functionEnd uint32) (ObjectRelocation, error) {
	var result ObjectRelocation
	matches := 0
	for _, relocation := range relocations {
		if relocation.SectionIndex != sectionIndex || relocation.Section != section || relocation.SymbolIndex != targetIndex ||
			relocation.Type != relocationType || relocation.VirtualAddress < functionStart || relocation.VirtualAddress >= functionEnd {
			continue
		}
		result = relocation
		matches++
	}
	if matches != 1 {
		return ObjectRelocation{}, fmt.Errorf("AMD64 REL32 variant type %#x has %d exact source-to-target relocations, want one", relocationType, matches)
	}
	return result, nil
}

func amd64REL32VariantDisplacement(targetRVA uint32, addend int32, siteRVA, trailing uint32) (int32, error) {
	if trailing >= amd64REL32VariantCount {
		return 0, fmt.Errorf("AMD64 REL32 variant trailing-byte count %d is invalid", trailing)
	}
	expected := int64(targetRVA) + int64(addend) - int64(siteRVA) - 4 - int64(trailing)
	if expected < math.MinInt32 || expected > math.MaxInt32 {
		return 0, fmt.Errorf("AMD64 REL32_%d expected displacement is out of range", trailing)
	}
	return int32(expected), nil
}

func validateAMD64REL32VariantsSpec(spec AMD64REL32VariantsSpec) error {
	if spec.SourceSymbol == "" || spec.TargetSymbol == "" || spec.SourceSymbol == spec.TargetSymbol ||
		strings.ContainsAny(spec.SourceSymbol+spec.TargetSymbol, " \t\r\n") ||
		filepath.Base(spec.SourceObject) != spec.SourceObject || !strings.EqualFold(filepath.Ext(spec.SourceObject), ".obj") ||
		filepath.Base(spec.TargetObject) != spec.TargetObject || !strings.EqualFold(filepath.Ext(spec.TargetObject), ".obj") ||
		strings.EqualFold(spec.SourceObject, spec.TargetObject) {
		return fmt.Errorf("AMD64 REL32 variants spec has invalid identities")
	}
	return nil
}
