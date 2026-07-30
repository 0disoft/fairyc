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

const amd64AddressReceiptTargetOffset = amd64REL32ReceiptMagicBytes + 8

// AMD64AddressLinkSpec identifies the ADDR64 target pointer and the source function's
// .pdata ADDR32NB entry relocations in one closed MSVC object-to-image proof.
type AMD64AddressLinkSpec struct {
	SourceSymbol   string
	TargetSymbol   string
	ReceiptSymbol  string
	SourceObject   string
	TargetObject   string
	ReceiptSection string
	ReceiptMagic   [amd64REL32ReceiptMagicBytes]byte
}

// AMD64AddressLinkReport records independently linked absolute and image-relative values.
// It is internal validation evidence, not a stable serialized format.
type AMD64AddressLinkReport struct {
	ADDR64ObjectAddend       uint64
	ADDR64LinkedValue        uint64
	ADDR64ExpectedValue      uint64
	ADDR32NBObjectAddend     uint32
	ADDR32NBObjectTarget     string
	ADDR32NBLinkedBeginRVA   uint32
	ADDR32NBExpectedBeginRVA uint32
	ADDR32NBEndObjectAddend  uint32
	ADDR32NBEndObjectTarget  string
	RuntimeFunctionEndRVA    uint32
	ExpectedFunctionEndRVA   uint32
	ADDR32NBUnwindAddend     uint32
	ADDR32NBUnwindTarget     string
	UnwindInfoRVA            uint32
	ExpectedUnwindInfoRVA    uint32
}

type amd64PDataEntryRelocations struct {
	Section      *pe.Section
	Begin        ObjectRelocation
	BeginTarget  ObjectSymbol
	End          ObjectRelocation
	EndTarget    ObjectSymbol
	Unwind       ObjectRelocation
	UnwindTarget ObjectSymbol
}

// InspectAMD64AddressLink proves two existing MSVC relocation paths from the same caller object:
// the receipt's ADDR64 target pointer resolves to S + A, and the function's .pdata ADDR32NB begin
// field resolves to S + A - ImageBase. Exact MSVC map rows provide the independent final symbols.
func InspectAMD64AddressLink(object, image, linkMap []byte, spec AMD64AddressLinkSpec) (AMD64AddressLinkReport, error) {
	if err := validateAMD64AddressLinkSpec(spec); err != nil {
		return AMD64AddressLinkReport{}, err
	}
	objectReport, err := InspectObject(object)
	if err != nil {
		return AMD64AddressLinkReport{}, fmt.Errorf("inspect AMD64 address-relocation object: %w", err)
	}
	imageReport, err := InspectExecutable(image)
	if err != nil {
		return AMD64AddressLinkReport{}, fmt.Errorf("inspect AMD64 address-relocation image: %w", err)
	}
	if _, err := InspectAMD64Unwind(image); err != nil {
		return AMD64AddressLinkReport{}, fmt.Errorf("inspect linked AMD64 unwind table: %w", err)
	}

	source, err := exactObjectSymbol(objectReport.SymbolRecords, spec.SourceSymbol)
	if err != nil {
		return AMD64AddressLinkReport{}, err
	}
	if source.SectionNumber <= 0 || int(source.SectionNumber) > len(objectReport.Sections) {
		return AMD64AddressLinkReport{}, fmt.Errorf("AMD64 address source symbol %q is not defined", source.Name)
	}
	if err := requireREL32FunctionSymbol(source, "address source"); err != nil {
		return AMD64AddressLinkReport{}, err
	}
	target, err := exactObjectSymbol(objectReport.SymbolRecords, spec.TargetSymbol)
	if err != nil {
		return AMD64AddressLinkReport{}, err
	}
	if target.SectionNumber != 0 {
		return AMD64AddressLinkReport{}, fmt.Errorf("AMD64 ADDR64 target symbol %q is not an undefined cross-object reference", target.Name)
	}
	if err := requireREL32FunctionSymbol(target, "ADDR64 target"); err != nil {
		return AMD64AddressLinkReport{}, err
	}
	receipt, err := exactObjectSymbol(objectReport.SymbolRecords, spec.ReceiptSymbol)
	if err != nil {
		return AMD64AddressLinkReport{}, err
	}
	if receipt.SectionNumber <= 0 || int(receipt.SectionNumber) > len(objectReport.Sections) || receipt.Type != 0 ||
		receipt.StorageClass != coffStorageClassExternal || receipt.AuxiliaryRecords != 0 {
		return AMD64AddressLinkReport{}, fmt.Errorf("AMD64 ADDR64 receipt symbol %q is not exact external data", receipt.Name)
	}
	receiptSection := objectReport.Sections[receipt.SectionNumber-1]
	if receiptSection.Name != spec.ReceiptSection || receiptSection.Characteristics&sectionReadable == 0 ||
		receiptSection.Characteristics&(sectionContainsCode|sectionExecutable|sectionWritable) != 0 {
		return AMD64AddressLinkReport{}, fmt.Errorf("AMD64 ADDR64 receipt symbol %q is not in the declared read-only section", receipt.Name)
	}

	mapSymbols, err := inspectMSVCMapSymbols(linkMap, spec.SourceSymbol, spec.TargetSymbol)
	if err != nil {
		return AMD64AddressLinkReport{}, err
	}
	sourceMap, found := mapSymbols[spec.SourceSymbol]
	if !found {
		return AMD64AddressLinkReport{}, fmt.Errorf("MSVC link map has no address source symbol %q", spec.SourceSymbol)
	}
	if !msvcMapOwnerIsObject(sourceMap.Owner, spec.SourceObject) {
		return AMD64AddressLinkReport{}, fmt.Errorf("MSVC link map source owner is %q, want %q", sourceMap.Owner, spec.SourceObject)
	}
	targetMap, found := mapSymbols[spec.TargetSymbol]
	if !found {
		return AMD64AddressLinkReport{}, fmt.Errorf("MSVC link map has no ADDR64 target symbol %q", spec.TargetSymbol)
	}
	if !msvcMapOwnerIsObject(targetMap.Owner, spec.TargetObject) {
		return AMD64AddressLinkReport{}, fmt.Errorf("MSVC link map target owner is %q, want %q", targetMap.Owner, spec.TargetObject)
	}
	sourceRVA, err := preferredImageRVA(imageReport, sourceMap.PreferredAddress, "address source map")
	if err != nil {
		return AMD64AddressLinkReport{}, err
	}
	targetRVA, err := preferredImageRVA(imageReport, targetMap.PreferredAddress, "ADDR64 target map")
	if err != nil {
		return AMD64AddressLinkReport{}, err
	}
	if err := requireExecutableRVARange(imageReport, sourceRVA, sourceRVA+1); err != nil {
		return AMD64AddressLinkReport{}, fmt.Errorf("AMD64 address source map is not executable: %w", err)
	}
	if err := requireExecutableRVARange(imageReport, targetRVA, targetRVA+1); err != nil {
		return AMD64AddressLinkReport{}, fmt.Errorf("AMD64 ADDR64 target map is not executable: %w", err)
	}

	objectFile, err := pe.NewFile(bytes.NewReader(object))
	if err != nil {
		return AMD64AddressLinkReport{}, fmt.Errorf("reparse AMD64 address-relocation object: %w", err)
	}
	defer objectFile.Close()
	receiptObjectSection := objectFile.Sections[receipt.SectionNumber-1]
	addr64Site := uint64(receiptObjectSection.VirtualAddress) + uint64(receipt.Value) + amd64AddressReceiptTargetOffset
	addr64Relocation, err := exactAddressRelocation(objectReport.RelocationRecords, int(receipt.SectionNumber), receiptSection.Name, target.TableIndex, ObjectRelocationAMD64Addr64, addr64Site)
	if err != nil {
		return AMD64AddressLinkReport{}, fmt.Errorf("AMD64 ADDR64: %w", err)
	}
	addr64ObjectOffset, err := objectRelocationRawOffset(object, receiptObjectSection, addr64Relocation, 8)
	if err != nil {
		return AMD64AddressLinkReport{}, fmt.Errorf("AMD64 ADDR64: %w", err)
	}
	magicObjectOffset := uint64(receiptObjectSection.Offset) + uint64(receipt.Value)
	if magicObjectOffset+amd64REL32ReceiptMagicBytes > uint64(len(object)) ||
		!bytes.Equal(object[magicObjectOffset:magicObjectOffset+amd64REL32ReceiptMagicBytes], spec.ReceiptMagic[:]) {
		return AMD64AddressLinkReport{}, fmt.Errorf("AMD64 ADDR64 object receipt magic is missing at the declared symbol")
	}
	addr64Addend := binary.LittleEndian.Uint64(object[addr64ObjectOffset:])
	if targetMap.PreferredAddress > math.MaxUint64-addr64Addend {
		return AMD64AddressLinkReport{}, fmt.Errorf("AMD64 ADDR64 expected value overflows")
	}
	addr64Expected := targetMap.PreferredAddress + addr64Addend
	sourceReceiptVA, addr64Linked, err := readREL32Receipt(image, imageReport, AMD64REL32LinkSpec{
		SourceSymbol: spec.SourceSymbol, TargetSymbol: spec.TargetSymbol,
		ReceiptSection: spec.ReceiptSection, ReceiptMagic: spec.ReceiptMagic,
	})
	if err != nil {
		return AMD64AddressLinkReport{}, fmt.Errorf("read AMD64 address receipt: %w", err)
	}
	if sourceReceiptVA != sourceMap.PreferredAddress {
		return AMD64AddressLinkReport{}, fmt.Errorf("AMD64 address source receipt is %#x, want map address %#x", sourceReceiptVA, sourceMap.PreferredAddress)
	}
	if addr64Linked != addr64Expected {
		return AMD64AddressLinkReport{}, fmt.Errorf("AMD64 ADDR64 linked value is %#x, want %#x", addr64Linked, addr64Expected)
	}

	pdataEntry, err := exactPDataEntryRelocations(objectReport, objectFile, source)
	if err != nil {
		return AMD64AddressLinkReport{}, err
	}
	beginObjectOffset, err := objectRelocationRawOffset(object, pdataEntry.Section, pdataEntry.Begin, 4)
	if err != nil {
		return AMD64AddressLinkReport{}, fmt.Errorf("AMD64 ADDR32NB begin: %w", err)
	}
	endObjectOffset, err := objectRelocationRawOffset(object, pdataEntry.Section, pdataEntry.End, 4)
	if err != nil {
		return AMD64AddressLinkReport{}, fmt.Errorf("AMD64 ADDR32NB end: %w", err)
	}
	unwindObjectOffset, err := objectRelocationRawOffset(object, pdataEntry.Section, pdataEntry.Unwind, 4)
	if err != nil {
		return AMD64AddressLinkReport{}, fmt.Errorf("AMD64 ADDR32NB unwind-info: %w", err)
	}
	beginAddend := binary.LittleEndian.Uint32(object[beginObjectOffset:])
	endAddend := binary.LittleEndian.Uint32(object[endObjectOffset:])
	unwindAddend := binary.LittleEndian.Uint32(object[unwindObjectOffset:])
	expectedBegin, err := addRVA(sourceRVA, beginAddend, "AMD64 ADDR32NB begin")
	if err != nil {
		return AMD64AddressLinkReport{}, err
	}
	expectedEnd, err := addRVA(sourceRVA, endAddend, "AMD64 ADDR32NB end")
	if err != nil {
		return AMD64AddressLinkReport{}, err
	}
	unwindMapSymbols, err := inspectMSVCMapSymbols(linkMap, pdataEntry.UnwindTarget.Name)
	if err != nil {
		return AMD64AddressLinkReport{}, err
	}
	unwindMap, found := unwindMapSymbols[pdataEntry.UnwindTarget.Name]
	if !found {
		return AMD64AddressLinkReport{}, fmt.Errorf("MSVC link map has no unwind-info symbol %q", pdataEntry.UnwindTarget.Name)
	}
	if !msvcMapOwnerIsObject(unwindMap.Owner, spec.SourceObject) {
		return AMD64AddressLinkReport{}, fmt.Errorf("MSVC link map unwind-info owner is %q, want %q", unwindMap.Owner, spec.SourceObject)
	}
	unwindBaseRVA, err := preferredImageRVA(imageReport, unwindMap.PreferredAddress, "unwind-info map")
	if err != nil {
		return AMD64AddressLinkReport{}, err
	}
	expectedUnwind, err := addRVA(unwindBaseRVA, unwindAddend, "AMD64 ADDR32NB unwind-info")
	if err != nil {
		return AMD64AddressLinkReport{}, err
	}
	linkedBegin, linkedEnd, unwindRVA, err := exactRuntimeFunctionByBegin(image, imageReport, expectedBegin)
	if err != nil {
		return AMD64AddressLinkReport{}, err
	}
	if linkedEnd != expectedEnd {
		return AMD64AddressLinkReport{}, fmt.Errorf("AMD64 ADDR32NB linked end RVA is %#x, want %#x", linkedEnd, expectedEnd)
	}
	if unwindRVA != expectedUnwind {
		return AMD64AddressLinkReport{}, fmt.Errorf("AMD64 ADDR32NB linked unwind-info RVA is %#x, want %#x", unwindRVA, expectedUnwind)
	}

	return AMD64AddressLinkReport{
		ADDR64ObjectAddend: addr64Addend, ADDR64LinkedValue: addr64Linked, ADDR64ExpectedValue: addr64Expected,
		ADDR32NBObjectAddend: beginAddend, ADDR32NBObjectTarget: pdataEntry.BeginTarget.Name, ADDR32NBLinkedBeginRVA: linkedBegin,
		ADDR32NBExpectedBeginRVA: expectedBegin, ADDR32NBEndObjectAddend: endAddend, ADDR32NBEndObjectTarget: pdataEntry.EndTarget.Name,
		RuntimeFunctionEndRVA: linkedEnd, ExpectedFunctionEndRVA: expectedEnd,
		ADDR32NBUnwindAddend: unwindAddend, ADDR32NBUnwindTarget: pdataEntry.UnwindTarget.Name,
		UnwindInfoRVA: unwindRVA, ExpectedUnwindInfoRVA: expectedUnwind,
	}, nil
}

func addRVA(base, addend uint32, label string) (uint32, error) {
	value := uint64(base) + uint64(addend)
	if value > math.MaxUint32 {
		return 0, fmt.Errorf("%s expected RVA overflows", label)
	}
	return uint32(value), nil
}

func validateAMD64AddressLinkSpec(spec AMD64AddressLinkSpec) error {
	if spec.SourceSymbol == "" || spec.TargetSymbol == "" || spec.ReceiptSymbol == "" ||
		spec.SourceSymbol == spec.TargetSymbol || spec.SourceSymbol == spec.ReceiptSymbol || spec.TargetSymbol == spec.ReceiptSymbol ||
		spec.ReceiptSection == "" || len(spec.ReceiptSection) > 8 || strings.ContainsAny(spec.SourceSymbol+spec.TargetSymbol+spec.ReceiptSymbol, " \t\r\n") ||
		filepath.Base(spec.SourceObject) != spec.SourceObject || filepath.Ext(spec.SourceObject) != ".obj" ||
		filepath.Base(spec.TargetObject) != spec.TargetObject || filepath.Ext(spec.TargetObject) != ".obj" ||
		strings.EqualFold(spec.SourceObject, spec.TargetObject) {
		return fmt.Errorf("AMD64 address-link spec has invalid symbol, object, or section identity")
	}
	if spec.ReceiptMagic == ([amd64REL32ReceiptMagicBytes]byte{}) {
		return fmt.Errorf("AMD64 address-link spec has an empty receipt magic")
	}
	return nil
}

func exactAddressRelocation(relocations []ObjectRelocation, sectionIndex int, section string, symbolIndex uint32, relocationType uint16, virtualAddress uint64) (ObjectRelocation, error) {
	if virtualAddress > math.MaxUint32 {
		return ObjectRelocation{}, fmt.Errorf("relocation virtual address overflows")
	}
	var result ObjectRelocation
	found := false
	for _, relocation := range relocations {
		if relocation.SectionIndex != sectionIndex || relocation.Section != section || relocation.SymbolIndex != symbolIndex || relocation.Type != relocationType || relocation.VirtualAddress != uint32(virtualAddress) {
			continue
		}
		if found {
			return ObjectRelocation{}, fmt.Errorf("object repeats the exact relocation")
		}
		result = relocation
		found = true
	}
	if !found {
		return ObjectRelocation{}, fmt.Errorf("object has no exact relocation")
	}
	return result, nil
}

func objectRelocationRawOffset(object []byte, section *pe.Section, relocation ObjectRelocation, size uint64) (uint64, error) {
	if relocation.VirtualAddress < section.VirtualAddress {
		return 0, fmt.Errorf("relocation precedes its object section")
	}
	sectionOffset := uint64(relocation.VirtualAddress - section.VirtualAddress)
	if sectionOffset+size > uint64(section.Size) || section.Offset == 0 {
		return 0, fmt.Errorf("relocation bytes are outside the object section")
	}
	objectOffset := uint64(section.Offset) + sectionOffset
	if objectOffset+size > uint64(len(object)) {
		return 0, fmt.Errorf("relocation bytes exceed the object")
	}
	return objectOffset, nil
}

func exactPDataEntryRelocations(report ObjectReport, file *pe.File, source ObjectSymbol) (amd64PDataEntryRelocations, error) {
	var result amd64PDataEntryRelocations
	found := false
	candidates := make([]string, 0, 8)
	symbols := make(map[uint32]ObjectSymbol, len(report.SymbolRecords))
	for _, symbol := range report.SymbolRecords {
		symbols[symbol.TableIndex] = symbol
	}
	for index, section := range report.Sections {
		if section.Name != ".pdata" && !strings.HasPrefix(section.Name, ".pdata$") {
			continue
		}
		objectSection := file.Sections[index]
		for _, relocation := range report.RelocationRecords {
			target, targetFound := symbols[relocation.SymbolIndex]
			if relocation.SectionIndex == index+1 && relocation.Section == section.Name && relocation.Type == ObjectRelocationAMD64Addr32NB && len(candidates) < 8 {
				candidates = append(candidates, fmt.Sprintf("%s@%#x(section=%d,value=%#x,aux=%d)", relocation.Symbol, relocation.VirtualAddress, target.SectionNumber, target.Value, target.AuxiliaryRecords))
			}
			if relocation.SectionIndex != index+1 || relocation.Section != section.Name || relocation.Type != ObjectRelocationAMD64Addr32NB || !targetFound ||
				target.SectionNumber != source.SectionNumber || target.Value != source.Value || target.AuxiliaryRecords != 0 ||
				relocation.VirtualAddress < objectSection.VirtualAddress {
				continue
			}
			entryOffset := relocation.VirtualAddress - objectSection.VirtualAddress
			if entryOffset%runtimeFunctionBytes != 0 || uint64(entryOffset)+runtimeFunctionBytes > uint64(objectSection.Size) {
				continue
			}
			if _, err := exactPDataSiteRelocation(report.RelocationRecords, index+1, section.Name, uint64(relocation.VirtualAddress)); err != nil {
				return amd64PDataEntryRelocations{}, fmt.Errorf("AMD64 ADDR32NB source .pdata begin: %w", err)
			}
			end, endTarget, err := exactPDataFieldRelocation(report.RelocationRecords, symbols, index+1, section.Name, uint64(relocation.VirtualAddress)+4)
			if err != nil {
				return amd64PDataEntryRelocations{}, fmt.Errorf("AMD64 ADDR32NB source .pdata end: %w", err)
			}
			if endTarget.SectionNumber != source.SectionNumber || endTarget.Value != source.Value || endTarget.AuxiliaryRecords != 0 {
				return amd64PDataEntryRelocations{}, fmt.Errorf("AMD64 ADDR32NB source .pdata end does not target the exact function alias")
			}
			unwind, unwindTarget, err := exactPDataFieldRelocation(report.RelocationRecords, symbols, index+1, section.Name, uint64(relocation.VirtualAddress)+8)
			if err != nil {
				return amd64PDataEntryRelocations{}, fmt.Errorf("AMD64 ADDR32NB source .pdata unwind-info: %w", err)
			}
			if unwindTarget.SectionNumber <= 0 || int(unwindTarget.SectionNumber) > len(report.Sections) || unwindTarget.AuxiliaryRecords != 0 {
				return amd64PDataEntryRelocations{}, fmt.Errorf("AMD64 ADDR32NB source .pdata unwind-info target %q is not exact .xdata", unwindTarget.Name)
			}
			unwindSection := report.Sections[unwindTarget.SectionNumber-1]
			if (unwindSection.Name != ".xdata" && !strings.HasPrefix(unwindSection.Name, ".xdata$")) ||
				unwindSection.RawOffset == 0 || unwindSection.Characteristics&sectionInitializedData == 0 ||
				unwindSection.Characteristics&sectionReadable == 0 ||
				unwindSection.Characteristics&(sectionContainsCode|sectionUninitializedData|sectionExecutable|sectionWritable) != 0 ||
				unwindTarget.Value >= unwindSection.RawSize {
				return amd64PDataEntryRelocations{}, fmt.Errorf("AMD64 ADDR32NB source .pdata unwind-info target %q is not bounded read-only .xdata", unwindTarget.Name)
			}
			if found {
				return amd64PDataEntryRelocations{}, fmt.Errorf("AMD64 ADDR32NB object repeats the source .pdata entry")
			}
			result = amd64PDataEntryRelocations{
				Section: objectSection, Begin: relocation, BeginTarget: target,
				End: end, EndTarget: endTarget, Unwind: unwind, UnwindTarget: unwindTarget,
			}
			found = true
		}
	}
	if !found {
		return amd64PDataEntryRelocations{}, fmt.Errorf("AMD64 ADDR32NB object has no exact source .pdata begin relocation; candidates %v", candidates)
	}
	return result, nil
}

func exactPDataFieldRelocation(relocations []ObjectRelocation, symbols map[uint32]ObjectSymbol, sectionIndex int, section string, address uint64) (ObjectRelocation, ObjectSymbol, error) {
	result, err := exactPDataSiteRelocation(relocations, sectionIndex, section, address)
	if err != nil {
		return ObjectRelocation{}, ObjectSymbol{}, err
	}
	if result.Type != ObjectRelocationAMD64Addr32NB {
		return ObjectRelocation{}, ObjectSymbol{}, fmt.Errorf("relocation at %#x is not ADDR32NB", result.VirtualAddress)
	}
	target, ok := symbols[result.SymbolIndex]
	if !ok {
		return ObjectRelocation{}, ObjectSymbol{}, fmt.Errorf("relocation at %#x targets a missing symbol", result.VirtualAddress)
	}
	return result, target, nil
}

func exactPDataSiteRelocation(relocations []ObjectRelocation, sectionIndex int, section string, address uint64) (ObjectRelocation, error) {
	if address > math.MaxUint32 {
		return ObjectRelocation{}, fmt.Errorf("relocation address overflows")
	}
	wantAddress := uint32(address)
	var result ObjectRelocation
	count := 0
	for _, relocation := range relocations {
		if relocation.SectionIndex != sectionIndex || relocation.Section != section || relocation.VirtualAddress != wantAddress {
			continue
		}
		result = relocation
		count++
	}
	if count != 1 {
		return ObjectRelocation{}, fmt.Errorf("object has %d relocations at %#x, want exactly one", count, wantAddress)
	}
	return result, nil
}

func exactRuntimeFunctionByBegin(image []byte, report Report, beginRVA uint32) (uint32, uint32, uint32, error) {
	file, err := pe.NewFile(bytes.NewReader(image))
	if err != nil {
		return 0, 0, 0, fmt.Errorf("parse AMD64 ADDR32NB image: %w", err)
	}
	defer file.Close()
	optional, ok := file.OptionalHeader.(*pe.OptionalHeader64)
	if !ok || optional == nil {
		return 0, 0, 0, fmt.Errorf("AMD64 ADDR32NB image has no PE32+ header")
	}
	directory := optional.DataDirectory[exceptionDirectoryIndex]
	offset, _, err := mapImageRVA(report, directory.VirtualAddress, directory.Size)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("map AMD64 ADDR32NB exception directory: %w", err)
	}
	var endRVA uint32
	var unwindRVA uint32
	matches := 0
	for entry := uint32(0); entry < directory.Size; entry += runtimeFunctionBytes {
		entryOffset := offset + int(entry)
		if binary.LittleEndian.Uint32(image[entryOffset:]) != beginRVA {
			continue
		}
		matches++
		endRVA = binary.LittleEndian.Uint32(image[entryOffset+4:])
		unwindRVA = binary.LittleEndian.Uint32(image[entryOffset+8:])
	}
	if matches != 1 {
		return 0, 0, 0, fmt.Errorf("AMD64 ADDR32NB final unwind table has %d entries beginning at %#x", matches, beginRVA)
	}
	return beginRVA, endRVA, unwindRVA, nil
}

func msvcMapOwnerIsObject(owner, object string) bool {
	normalized := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(owner), "\\", "/"))
	want := strings.ToLower(object)
	return normalized == want || strings.HasSuffix(normalized, "/"+want)
}
