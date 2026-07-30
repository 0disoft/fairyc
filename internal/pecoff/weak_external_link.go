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

// AMD64WeakExternalSpec identifies one fallback path and one strong-override path.
type AMD64WeakExternalSpec struct {
	ObjectName       string
	TargetObject     string
	FallbackEntry    string
	OverrideEntry    string
	MissingWeak      string
	OverrideWeak     string
	MissingFallback  string
	OverrideFallback string
}

// AMD64WeakExternalValue records one exact weak relocation and its selected final target.
type AMD64WeakExternalValue struct {
	EntrySymbol          string
	WeakSymbol           string
	FallbackSymbol       string
	SelectedSymbol       string
	UsedFallback         bool
	ObjectAddend         int32
	LinkedSiteRVA        uint32
	SelectedTargetRVA    uint32
	LinkedDisplacement   int32
	ExpectedDisplacement int32
}

// AMD64WeakExternalReport records both required weak-resolution outcomes.
type AMD64WeakExternalReport struct {
	Fallback AMD64WeakExternalValue
	Override AMD64WeakExternalValue
}

// InspectAMD64WeakExternalLink proves one missing weak alias selects its local fallback while one
// present strong definition overrides a different local fallback in the same real LINK image.
func InspectAMD64WeakExternalLink(object, image, linkMap []byte, spec AMD64WeakExternalSpec) (AMD64WeakExternalReport, error) {
	if err := validateAMD64WeakExternalSpec(spec); err != nil {
		return AMD64WeakExternalReport{}, err
	}
	objectReport, err := InspectObject(object)
	if err != nil {
		return AMD64WeakExternalReport{}, fmt.Errorf("inspect AMD64 weak-external object: %w", err)
	}
	imageReport, err := InspectExecutable(image)
	if err != nil {
		return AMD64WeakExternalReport{}, fmt.Errorf("inspect AMD64 weak-external image: %w", err)
	}
	objectFile, err := pe.NewFile(bytes.NewReader(object))
	if err != nil {
		return AMD64WeakExternalReport{}, fmt.Errorf("reparse AMD64 weak-external object: %w", err)
	}
	defer objectFile.Close()

	mapSymbols, err := inspectMSVCMapSymbols(linkMap, spec.FallbackEntry, spec.OverrideEntry, spec.MissingFallback, spec.OverrideWeak)
	if err != nil {
		return AMD64WeakExternalReport{}, err
	}
	fallback, err := inspectAMD64WeakExternalPath(object, objectReport, objectFile, image, imageReport, mapSymbols, spec.ObjectName,
		spec.FallbackEntry, spec.MissingWeak, spec.MissingFallback, spec.MissingFallback, coffWeakSearchAlias,
		func(owner string) bool { return msvcMapOwnerIsObject(owner, spec.ObjectName) }, spec.ObjectName, true)
	if err != nil {
		return AMD64WeakExternalReport{}, fmt.Errorf("AMD64 weak fallback path: %w", err)
	}
	override, err := inspectAMD64WeakExternalPath(object, objectReport, objectFile, image, imageReport, mapSymbols, spec.ObjectName,
		spec.OverrideEntry, spec.OverrideWeak, spec.OverrideFallback, spec.OverrideWeak, coffWeakSearchAlias,
		func(owner string) bool { return msvcMapOwnerIsObject(owner, spec.TargetObject) }, spec.TargetObject, false)
	if err != nil {
		return AMD64WeakExternalReport{}, fmt.Errorf("AMD64 weak override path: %w", err)
	}
	return AMD64WeakExternalReport{Fallback: fallback, Override: override}, nil
}

func inspectAMD64WeakExternalPath(object []byte, report ObjectReport, objectFile *pe.File, image []byte, imageReport Report,
	mapSymbols map[string]msvcMapSymbol, objectName, entryName, weakName, fallbackName, selectedName string, weakSearch uint32,
	selectedOwner func(string) bool, selectedOwnerDescription string, usedFallback bool,
) (AMD64WeakExternalValue, error) {
	entry, err := exactObjectSymbol(report.SymbolRecords, entryName)
	if err != nil {
		return AMD64WeakExternalValue{}, err
	}
	if err := requireDefinedExternalFunction(report, entryName, "weak entry"); err != nil {
		return AMD64WeakExternalValue{}, err
	}
	weak, err := exactObjectSymbol(report.SymbolRecords, weakName)
	if err != nil {
		return AMD64WeakExternalValue{}, err
	}
	if weak.SectionNumber != 0 || weak.Value != 0 || weak.Type != coffSymbolTypeFunction || weak.StorageClass != coffStorageClassWeakExternal ||
		weak.AuxiliaryRecords != 1 || weak.WeakFallback != fallbackName || weak.WeakSearch != weakSearch {
		return AMD64WeakExternalValue{}, fmt.Errorf("weak symbol %q has invalid search metadata", weakName)
	}
	fallback, err := exactObjectSymbol(report.SymbolRecords, fallbackName)
	if err != nil {
		return AMD64WeakExternalValue{}, err
	}
	if err := requireDefinedExternalFunction(report, fallbackName, "weak fallback"); err != nil {
		return AMD64WeakExternalValue{}, err
	}
	section := objectFile.Sections[entry.SectionNumber-1]
	entryEnd := uint64(entry.Value) + 5
	if entryEnd > uint64(section.Size) {
		return AMD64WeakExternalValue{}, fmt.Errorf("weak entry %q exceeds its object section", entryName)
	}
	siteValue := uint64(section.VirtualAddress) + uint64(entry.Value) + 1
	if siteValue > math.MaxUint32 {
		return AMD64WeakExternalValue{}, fmt.Errorf("weak entry %q relocation site overflows", entryName)
	}
	site := uint32(siteValue)
	relocation, err := exactPDataSiteRelocation(report.RelocationRecords, int(entry.SectionNumber), report.Sections[entry.SectionNumber-1].Name, uint64(site))
	if err != nil {
		return AMD64WeakExternalValue{}, fmt.Errorf("weak entry relocation: %w", err)
	}
	if relocation.Type != ObjectRelocationAMD64Rel32 || relocation.SymbolIndex != weak.TableIndex {
		return AMD64WeakExternalValue{}, fmt.Errorf("weak entry relocation targets %q type %#x, want %q REL32", relocation.Symbol, relocation.Type, weakName)
	}
	objectOffset, err := objectRelocationRawOffset(object, section, relocation, 4)
	if err != nil {
		return AMD64WeakExternalValue{}, err
	}
	if objectOffset == 0 || object[objectOffset-1] != 0xe9 {
		return AMD64WeakExternalValue{}, fmt.Errorf("weak entry %q is not a direct tail jump", entryName)
	}
	entryMap, found := mapSymbols[entryName]
	if !found || !msvcMapOwnerIsObject(entryMap.Owner, objectName) {
		return AMD64WeakExternalValue{}, fmt.Errorf("MSVC weak entry owner is %q, want %q", entryMap.Owner, objectName)
	}
	selectedMap, found := mapSymbols[selectedName]
	if !found || selectedOwner == nil || !selectedOwner(selectedMap.Owner) {
		return AMD64WeakExternalValue{}, fmt.Errorf("MSVC weak selected target owner is %q, want %q", selectedMap.Owner, selectedOwnerDescription)
	}
	entryRVA, err := preferredAddressRVA(imageReport, entryMap.PreferredAddress, "weak entry")
	if err != nil {
		return AMD64WeakExternalValue{}, err
	}
	selectedRVA, err := preferredAddressRVA(imageReport, selectedMap.PreferredAddress, "weak selected target")
	if err != nil {
		return AMD64WeakExternalValue{}, err
	}
	if entryRVA == math.MaxUint32 || selectedRVA == math.MaxUint32 {
		return AMD64WeakExternalValue{}, fmt.Errorf("weak entry or selected target RVA overflows")
	}
	if err := requireExecutableRVARange(imageReport, entryRVA, entryRVA+1); err != nil {
		return AMD64WeakExternalValue{}, fmt.Errorf("weak entry is not executable: %w", err)
	}
	if err := requireExecutableRVARange(imageReport, selectedRVA, selectedRVA+1); err != nil {
		return AMD64WeakExternalValue{}, fmt.Errorf("weak selected target is not executable: %w", err)
	}
	linkedSite := entryRVA + 1
	linkedOffset, linkedSection, err := mapImageRVA(imageReport, linkedSite, 4)
	if err != nil {
		return AMD64WeakExternalValue{}, err
	}
	if linkedSection.Characteristics&sectionExecutable == 0 || linkedOffset == 0 || image[linkedOffset-1] != 0xe9 {
		return AMD64WeakExternalValue{}, fmt.Errorf("linked weak entry %q is not executable direct jump code", entryName)
	}
	addend := int32(binary.LittleEndian.Uint32(object[objectOffset:]))
	linked := int32(binary.LittleEndian.Uint32(image[linkedOffset:]))
	expected, err := amd64REL32Displacement(selectedRVA, addend, linkedSite)
	if err != nil {
		return AMD64WeakExternalValue{}, err
	}
	if linked != expected {
		return AMD64WeakExternalValue{}, fmt.Errorf("linked weak displacement is %d, want %d", linked, expected)
	}
	return AMD64WeakExternalValue{
		EntrySymbol: entryName, WeakSymbol: weakName, FallbackSymbol: fallback.Name,
		SelectedSymbol: selectedName, UsedFallback: usedFallback, ObjectAddend: addend,
		LinkedSiteRVA: linkedSite, SelectedTargetRVA: selectedRVA,
		LinkedDisplacement: linked, ExpectedDisplacement: expected,
	}, nil
}

func validateAMD64WeakExternalSpec(spec AMD64WeakExternalSpec) error {
	names := []string{spec.FallbackEntry, spec.OverrideEntry, spec.MissingWeak, spec.OverrideWeak, spec.MissingFallback, spec.OverrideFallback}
	seen := make(map[string]struct{}, len(names))
	for _, name := range names {
		if name == "" || strings.ContainsAny(name, " \t\r\n") {
			return fmt.Errorf("AMD64 weak-external spec has invalid identities")
		}
		if _, duplicate := seen[name]; duplicate {
			return fmt.Errorf("AMD64 weak-external spec repeats symbol %q", name)
		}
		seen[name] = struct{}{}
	}
	for _, name := range []string{spec.ObjectName, spec.TargetObject} {
		if filepath.Base(name) != name || !strings.EqualFold(filepath.Ext(name), ".obj") {
			return fmt.Errorf("AMD64 weak-external spec has invalid object identity")
		}
	}
	if strings.EqualFold(spec.ObjectName, spec.TargetObject) {
		return fmt.Errorf("AMD64 weak-external spec repeats object identity")
	}
	return nil
}
