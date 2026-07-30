package pecoff

import (
	"bytes"
	"debug/pe"
	"encoding/binary"
	"fmt"
)

const amd64REL32ReceiptMagicBytes = 16

const (
	coffStorageClassExternal = 2
	coffSymbolTypeFunction   = 0x20
)

// AMD64REL32LinkSpec identifies one closed object-to-image REL32 proof. The receipt is a
// project-owned image section containing the exact magic followed by preferred-image virtual
// addresses for the source and target functions.
type AMD64REL32LinkSpec struct {
	SourceSymbol   string
	TargetSymbol   string
	ReceiptSection string
	ReceiptMagic   [amd64REL32ReceiptMagicBytes]byte
}

// AMD64REL32LinkReport records the independently recomputed object addend and linked displacement.
// It is internal validation evidence, not a stable serialized format.
type AMD64REL32LinkReport struct {
	ObjectSection        string
	ObjectVirtualAddress uint32
	ObjectAddend         int32
	SourceRVA            uint32
	TargetRVA            uint32
	LinkedSiteRVA        uint32
	LinkedDisplacement   int32
	ExpectedDisplacement int32
}

// InspectAMD64REL32Link proves that one direct x64 call relocation in object was resolved to the
// receipt-named target in image according to the Microsoft REL32 formula S + A - (P + 4).
func InspectAMD64REL32Link(object, image []byte, spec AMD64REL32LinkSpec) (AMD64REL32LinkReport, error) {
	if spec.SourceSymbol == "" || spec.TargetSymbol == "" || spec.SourceSymbol == spec.TargetSymbol || spec.ReceiptSection == "" {
		return AMD64REL32LinkReport{}, fmt.Errorf("AMD64 REL32 link spec has invalid symbol or section identity")
	}
	if spec.ReceiptMagic == ([amd64REL32ReceiptMagicBytes]byte{}) {
		return AMD64REL32LinkReport{}, fmt.Errorf("AMD64 REL32 link spec has an empty receipt magic")
	}
	objectReport, err := InspectObject(object)
	if err != nil {
		return AMD64REL32LinkReport{}, fmt.Errorf("inspect AMD64 REL32 object: %w", err)
	}
	imageReport, err := InspectExecutable(image)
	if err != nil {
		return AMD64REL32LinkReport{}, fmt.Errorf("inspect AMD64 REL32 image: %w", err)
	}

	source, err := exactObjectSymbol(objectReport.SymbolRecords, spec.SourceSymbol)
	if err != nil {
		return AMD64REL32LinkReport{}, err
	}
	if source.SectionNumber <= 0 || int(source.SectionNumber) > len(objectReport.Sections) {
		return AMD64REL32LinkReport{}, fmt.Errorf("AMD64 REL32 source symbol %q is not defined in an object section", source.Name)
	}
	if err := requireREL32FunctionSymbol(source, "source"); err != nil {
		return AMD64REL32LinkReport{}, err
	}
	target, err := exactObjectSymbol(objectReport.SymbolRecords, spec.TargetSymbol)
	if err != nil {
		return AMD64REL32LinkReport{}, err
	}
	if target.SectionNumber != 0 {
		return AMD64REL32LinkReport{}, fmt.Errorf("AMD64 REL32 target symbol %q is not an undefined cross-object reference", target.Name)
	}
	if err := requireREL32FunctionSymbol(target, "target"); err != nil {
		return AMD64REL32LinkReport{}, err
	}
	sourceSectionReceipt := objectReport.Sections[source.SectionNumber-1]
	if err := requireREL32SourceSection(sourceSectionReceipt); err != nil {
		return AMD64REL32LinkReport{}, err
	}
	sourceSectionName := sourceSectionReceipt.Name

	objectFile, err := pe.NewFile(bytes.NewReader(object))
	if err != nil {
		return AMD64REL32LinkReport{}, fmt.Errorf("reparse AMD64 REL32 object: %w", err)
	}
	defer objectFile.Close()
	objectSection := objectFile.Sections[source.SectionNumber-1]
	functionStart := uint64(objectSection.VirtualAddress) + uint64(source.Value)
	functionEnd := uint64(objectSection.VirtualAddress) + uint64(objectSection.Size)
	for _, symbol := range objectReport.SymbolRecords {
		if symbol.SectionNumber == source.SectionNumber && symbol.Value > source.Value {
			candidate := uint64(objectSection.VirtualAddress) + uint64(symbol.Value)
			if candidate < functionEnd {
				functionEnd = candidate
			}
		}
	}
	if functionStart >= functionEnd {
		return AMD64REL32LinkReport{}, fmt.Errorf("AMD64 REL32 source symbol %q has an empty object range", source.Name)
	}
	if functionEnd > uint64(^uint32(0)) {
		return AMD64REL32LinkReport{}, fmt.Errorf("AMD64 REL32 source symbol %q range overflows COFF virtual addresses", source.Name)
	}
	relocation, err := exactREL32Relocation(
		objectReport.RelocationRecords, sourceSectionName, target.TableIndex,
		uint32(functionStart), uint32(functionEnd),
	)
	if err != nil {
		return AMD64REL32LinkReport{}, err
	}
	if relocation.VirtualAddress < objectSection.VirtualAddress {
		return AMD64REL32LinkReport{}, fmt.Errorf("AMD64 REL32 relocation precedes its object section")
	}
	sectionOffset := uint64(relocation.VirtualAddress - objectSection.VirtualAddress)
	if sectionOffset < uint64(source.Value)+1 || sectionOffset+4 > uint64(objectSection.Size) {
		return AMD64REL32LinkReport{}, fmt.Errorf("AMD64 REL32 relocation is outside the source function bytes")
	}
	objectSite := uint64(objectSection.Offset) + sectionOffset
	if objectSite < 1 || objectSite+4 > uint64(len(object)) {
		return AMD64REL32LinkReport{}, fmt.Errorf("AMD64 REL32 object relocation bytes are out of range")
	}
	if object[objectSite-1] != 0xe8 {
		return AMD64REL32LinkReport{}, fmt.Errorf("AMD64 REL32 object relocation is not a direct CALL rel32 operand")
	}
	objectAddend := int32(binary.LittleEndian.Uint32(object[objectSite : objectSite+4]))

	sourceVA, targetVA, err := readREL32Receipt(image, imageReport, spec)
	if err != nil {
		return AMD64REL32LinkReport{}, err
	}
	sourceRVA, err := preferredImageRVA(imageReport, sourceVA, "source")
	if err != nil {
		return AMD64REL32LinkReport{}, err
	}
	targetRVA, err := preferredImageRVA(imageReport, targetVA, "target")
	if err != nil {
		return AMD64REL32LinkReport{}, err
	}
	if err := requireExecutableRVARange(imageReport, sourceRVA, sourceRVA+1); err != nil {
		return AMD64REL32LinkReport{}, fmt.Errorf("AMD64 REL32 source receipt is not executable: %w", err)
	}
	if err := requireExecutableRVARange(imageReport, targetRVA, targetRVA+1); err != nil {
		return AMD64REL32LinkReport{}, fmt.Errorf("AMD64 REL32 target receipt is not executable: %w", err)
	}

	sourceRelativeSite := sectionOffset - uint64(source.Value)
	linkedSite := uint64(sourceRVA) + sourceRelativeSite
	if linkedSite == 0 || linkedSite+4 > uint64(^uint32(0)) {
		return AMD64REL32LinkReport{}, fmt.Errorf("AMD64 REL32 linked relocation site overflows its RVA")
	}
	linkedSiteRVA := uint32(linkedSite)
	linkedOffset, linkedSection, err := mapImageRVA(imageReport, linkedSiteRVA-1, 5)
	if err != nil {
		return AMD64REL32LinkReport{}, fmt.Errorf("map AMD64 REL32 linked instruction: %w", err)
	}
	if linkedSection.Characteristics&sectionContainsCode == 0 || linkedSection.Characteristics&sectionExecutable == 0 {
		return AMD64REL32LinkReport{}, fmt.Errorf("AMD64 REL32 linked instruction is not executable code")
	}
	if image[linkedOffset] != 0xe8 {
		return AMD64REL32LinkReport{}, fmt.Errorf("AMD64 REL32 linked relocation is not a direct CALL rel32 operand")
	}
	linkedDisplacement := int32(binary.LittleEndian.Uint32(image[linkedOffset+1 : linkedOffset+5]))
	expectedDisplacement, err := amd64REL32Displacement(targetRVA, objectAddend, linkedSiteRVA)
	if err != nil {
		return AMD64REL32LinkReport{}, err
	}
	if linkedDisplacement != expectedDisplacement {
		return AMD64REL32LinkReport{}, fmt.Errorf("AMD64 REL32 linked displacement is %d, want %d", linkedDisplacement, expectedDisplacement)
	}
	return AMD64REL32LinkReport{
		ObjectSection: sourceSectionName, ObjectVirtualAddress: relocation.VirtualAddress,
		ObjectAddend: objectAddend, SourceRVA: sourceRVA, TargetRVA: targetRVA,
		LinkedSiteRVA: linkedSiteRVA, LinkedDisplacement: linkedDisplacement,
		ExpectedDisplacement: expectedDisplacement,
	}, nil
}

func requireREL32FunctionSymbol(symbol ObjectSymbol, role string) error {
	if symbol.Type != coffSymbolTypeFunction || symbol.StorageClass != coffStorageClassExternal || symbol.AuxiliaryRecords != 0 {
		return fmt.Errorf("AMD64 REL32 %s symbol %q is not an exact external function", role, symbol.Name)
	}
	return nil
}

func requireREL32SourceSection(section ObjectSection) error {
	if !isCOFFTextSection(section.Name) || section.Characteristics&sectionContainsCode == 0 ||
		section.Characteristics&sectionExecutable == 0 || section.Characteristics&sectionReadable == 0 {
		return fmt.Errorf("AMD64 REL32 source section %q is not readable executable .text code", section.Name)
	}
	return nil
}

func amd64REL32Displacement(targetRVA uint32, addend int32, siteRVA uint32) (int32, error) {
	expected := int64(targetRVA) + int64(addend) - int64(siteRVA) - 4
	if expected < -1<<31 || expected > 1<<31-1 {
		return 0, fmt.Errorf("AMD64 REL32 expected displacement is out of range")
	}
	return int32(expected), nil
}

func exactObjectSymbol(symbols []ObjectSymbol, name string) (ObjectSymbol, error) {
	var result ObjectSymbol
	found := false
	for _, symbol := range symbols {
		if symbol.Name != name {
			continue
		}
		if found {
			return ObjectSymbol{}, fmt.Errorf("AMD64 REL32 object repeats symbol %q", name)
		}
		result = symbol
		found = true
	}
	if !found {
		return ObjectSymbol{}, fmt.Errorf("AMD64 REL32 object has no symbol %q", name)
	}
	return result, nil
}

func exactREL32Relocation(relocations []ObjectRelocation, section string, targetIndex, functionStart, functionEnd uint32) (ObjectRelocation, error) {
	var result ObjectRelocation
	found := false
	for _, relocation := range relocations {
		if relocation.Section != section || relocation.SymbolIndex != targetIndex || relocation.Type != ObjectRelocationAMD64Rel32 ||
			relocation.VirtualAddress < functionStart || relocation.VirtualAddress >= functionEnd {
			continue
		}
		if found {
			return ObjectRelocation{}, fmt.Errorf("AMD64 REL32 object repeats the source-to-target relocation")
		}
		result = relocation
		found = true
	}
	if !found {
		return ObjectRelocation{}, fmt.Errorf("AMD64 REL32 object has no exact source-to-target relocation")
	}
	return result, nil
}

func readREL32Receipt(image []byte, report Report, spec AMD64REL32LinkSpec) (uint64, uint64, error) {
	var receiptSection Section
	found := false
	for _, section := range report.Sections {
		if section.Name != spec.ReceiptSection {
			continue
		}
		if found {
			return 0, 0, fmt.Errorf("AMD64 REL32 image repeats receipt section %q", spec.ReceiptSection)
		}
		receiptSection = section
		found = true
	}
	if !found {
		return 0, 0, fmt.Errorf("AMD64 REL32 image has no receipt section %q", spec.ReceiptSection)
	}
	if receiptSection.Characteristics&sectionReadable == 0 ||
		receiptSection.Characteristics&(sectionContainsCode|sectionExecutable|sectionWritable) != 0 {
		return 0, 0, fmt.Errorf("AMD64 REL32 receipt section %q is not read-only data", spec.ReceiptSection)
	}
	start := uint64(receiptSection.RawOffset)
	logicalSize := min(receiptSection.VirtualSize, receiptSection.RawSize)
	if logicalSize < amd64REL32ReceiptMagicBytes+16 {
		return 0, 0, fmt.Errorf("AMD64 REL32 receipt section has no complete file-backed logical receipt")
	}
	end := start + uint64(logicalSize)
	if end > uint64(len(image)) {
		return 0, 0, fmt.Errorf("AMD64 REL32 receipt section exceeds the image")
	}
	sectionBytes := image[start:end]
	magic := spec.ReceiptMagic[:]
	receiptOffset := bytes.Index(sectionBytes, magic)
	if receiptOffset < 0 {
		return 0, 0, fmt.Errorf("AMD64 REL32 image receipt magic is missing")
	}
	if bytes.Index(sectionBytes[receiptOffset+1:], magic) >= 0 {
		return 0, 0, fmt.Errorf("AMD64 REL32 image receipt magic is ambiguous")
	}
	pointers := receiptOffset + len(magic)
	if pointers+16 > len(sectionBytes) {
		return 0, 0, fmt.Errorf("AMD64 REL32 image receipt is truncated")
	}
	return binary.LittleEndian.Uint64(sectionBytes[pointers:]), binary.LittleEndian.Uint64(sectionBytes[pointers+8:]), nil
}

func preferredImageRVA(report Report, address uint64, name string) (uint32, error) {
	if address < report.ImageBase {
		return 0, fmt.Errorf("AMD64 REL32 %s receipt address %#x precedes image base", name, address)
	}
	rva := address - report.ImageBase
	if rva == 0 || rva >= uint64(report.SizeOfImage) || rva > uint64(^uint32(0)) {
		return 0, fmt.Errorf("AMD64 REL32 %s receipt address %#x is outside the image", name, address)
	}
	return uint32(rva), nil
}
