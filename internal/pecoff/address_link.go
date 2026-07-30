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
// .pdata ADDR32NB begin relocation in one closed MSVC object-to-image proof.
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
	RuntimeFunctionEndRVA    uint32
	UnwindInfoRVA            uint32
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

	pdataRelocation, pdataSection, pdataTarget, err := exactPDataBeginRelocation(objectReport, objectFile, source)
	if err != nil {
		return AMD64AddressLinkReport{}, err
	}
	pdataObjectOffset, err := objectRelocationRawOffset(object, pdataSection, pdataRelocation, 4)
	if err != nil {
		return AMD64AddressLinkReport{}, fmt.Errorf("AMD64 ADDR32NB: %w", err)
	}
	addr32Addend := binary.LittleEndian.Uint32(object[pdataObjectOffset:])
	addr32Expected64 := uint64(sourceRVA) + uint64(addr32Addend)
	if addr32Expected64 > math.MaxUint32 {
		return AMD64AddressLinkReport{}, fmt.Errorf("AMD64 ADDR32NB expected RVA overflows")
	}
	addr32Expected := uint32(addr32Expected64)
	linkedBegin, linkedEnd, unwindRVA, err := exactRuntimeFunctionByBegin(image, imageReport, addr32Expected)
	if err != nil {
		return AMD64AddressLinkReport{}, err
	}

	return AMD64AddressLinkReport{
		ADDR64ObjectAddend: addr64Addend, ADDR64LinkedValue: addr64Linked, ADDR64ExpectedValue: addr64Expected,
		ADDR32NBObjectAddend: addr32Addend, ADDR32NBObjectTarget: pdataTarget.Name, ADDR32NBLinkedBeginRVA: linkedBegin,
		ADDR32NBExpectedBeginRVA: addr32Expected, RuntimeFunctionEndRVA: linkedEnd, UnwindInfoRVA: unwindRVA,
	}, nil
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

func exactPDataBeginRelocation(report ObjectReport, file *pe.File, source ObjectSymbol) (ObjectRelocation, *pe.Section, ObjectSymbol, error) {
	var result ObjectRelocation
	var resultSection *pe.Section
	var resultTarget ObjectSymbol
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
			if found {
				return ObjectRelocation{}, nil, ObjectSymbol{}, fmt.Errorf("AMD64 ADDR32NB object repeats the source .pdata begin relocation")
			}
			result = relocation
			resultSection = objectSection
			resultTarget = target
			found = true
		}
	}
	if !found {
		return ObjectRelocation{}, nil, ObjectSymbol{}, fmt.Errorf("AMD64 ADDR32NB object has no exact source .pdata begin relocation; candidates %v", candidates)
	}
	return result, resultSection, resultTarget, nil
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
