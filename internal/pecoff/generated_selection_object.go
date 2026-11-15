package pecoff

import (
	"fmt"
	"strings"
)

// All backing records are read-only after construction. Inspection accumulates
// receipts locally and never masks caller-owned object or executable bytes.
type generatedCOMDATObject struct {
	name   string
	raw    []byte
	report ObjectReport
	owners map[int32][]string
	mapped map[string]msvcMapSymbol
}

func readGeneratedCOMDATObject(name string, raw, linkMap []byte) (generatedCOMDATObject, error) {
	report, err := InspectObject(raw)
	if err != nil {
		return generatedCOMDATObject{}, fmt.Errorf("%s: %w", name, err)
	}
	object := generatedCOMDATObject{name: name, raw: raw, report: report, owners: make(map[int32][]string)}
	var wanted []string
	for _, symbol := range report.SymbolRecords {
		if symbol.SectionNumber <= 0 || symbol.Value != 0 ||
			(symbol.StorageClass != coffStorageClassExternal && symbol.StorageClass != coffStorageClassStatic) || strings.HasPrefix(symbol.Name, ".") {
			continue
		}
		selection := report.Sections[symbol.SectionNumber-1].COMDATSelection
		if selection == 0 || selection == ObjectCOMDATAssociative {
			continue
		}
		object.owners[symbol.SectionNumber] = append(object.owners[symbol.SectionNumber], symbol.Name)
		wanted = append(wanted, symbol.Name)
	}
	object.mapped, err = inspectMSVCMapSymbols(linkMap, wanted...)
	return object, err
}

func (object generatedCOMDATObject) inspect(image []byte, peReport Report, discards map[string]struct{}) (GeneratedCOMDATSelection, error) {
	var result GeneratedCOMDATSelection
	relocationOffset := 0
	for index, section := range object.report.Sections {
		// Every section advances the contiguous relocation cursor, including
		// non-COMDAT, associative and discarded sections.
		relocations := object.report.RelocationRecords[relocationOffset : relocationOffset+section.Relocations]
		relocationOffset += section.Relocations
		if section.COMDATSelection == 0 {
			continue
		}
		if section.COMDATSelection == ObjectCOMDATAssociative {
			result.AssociativeSectionInventory++
			continue
		}
		parent, address, err := object.parent(index, section, discards)
		if err != nil {
			return result, err
		}
		if !parent.Retained {
			result.Discarded++
			result.Parents = append(result.Parents, parent)
			continue
		}
		parent, err = object.inspectContent(parent, section, relocations, image, peReport, address)
		if err != nil {
			return result, err
		}
		result.Retained++
		result.Parents = append(result.Parents, parent)
	}
	return result, nil
}

func (object generatedCOMDATObject) parent(index int, section ObjectSection, discards map[string]struct{}) (GeneratedCOMDATParent, uint64, error) {
	names := object.owners[int32(index+1)]
	if len(names) != 1 {
		return GeneratedCOMDATParent{}, 0, fmt.Errorf("%s COMDAT section %d has %d possible owners", object.name, index+1, len(names))
	}
	name := names[0]
	selected, found := object.mapped[name]
	owned := found && generatedObjectOwner(selected.Owner) == object.name
	_, discarded := discards[object.name+":"+name]
	if owned == discarded {
		return GeneratedCOMDATParent{}, 0, fmt.Errorf("%s:%s requires exactly one owned map or discard record", object.name, name)
	}
	parent := GeneratedCOMDATParent{Object: object.name, Symbol: name, SectionIndex: index + 1,
		Selection: section.COMDATSelection, SourceBytes: section.RawSize, Retained: owned}
	return parent, selected.PreferredAddress, nil
}

func (object generatedCOMDATObject) inspectContent(parent GeneratedCOMDATParent, section ObjectSection, relocations []ObjectRelocation, image []byte, peReport Report, address uint64) (GeneratedCOMDATParent, error) {
	rva, err := preferredAddressRVA(peReport, address, "generated COMDAT")
	if err != nil {
		return parent, err
	}
	actual, err := generatedImageView(image, peReport, rva, section.RawSize)
	if err != nil {
		return parent, err
	}
	expected := object.raw[section.RawOffset : uint64(section.RawOffset)+uint64(section.RawSize)]
	fields, err := generatedCOMDATRelocationFields(relocations, len(expected))
	if err != nil {
		return parent, fmt.Errorf("%s:%s %w", object.name, parent.Symbol, err)
	}
	compared, relocated, digest, err := compareGeneratedCOMDATContent(expected, actual, fields)
	if err != nil {
		return parent, fmt.Errorf("%s:%s %w", object.name, parent.Symbol, err)
	}
	parent.ImageRVA, parent.ComparedBytes, parent.RelocatedBytes, parent.NonRelocatedSHA256 = rva, compared, relocated, digest
	return parent, nil
}
