package pecoff

import (
	"crypto/sha256"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"
)

const maxGeneratedCOMDATLinkLogBytes = 8 << 20

// GeneratedCOMDATParent identifies one generated-unit/runtime COMDAT definition.
// Retained bytes exclude relocation fields, whose values have separate probes.
type GeneratedCOMDATParent struct {
	Object, Symbol                string
	SectionIndex                  int
	Selection                     uint8
	SourceBytes                   uint32
	Retained                      bool
	ImageRVA                      uint32
	ComparedBytes, RelocatedBytes int
	NonRelocatedSHA256            [sha256.Size]byte
}

type GeneratedCOMDATSelection struct {
	Parents                     []GeneratedCOMDATParent
	Retained, Discarded         int
	AssociativeSectionInventory int
}

// InspectGeneratedCOMDATSelection binds the production unit.obj/runtime.obj
// parents to their owned map entries and final image bytes, or to an exact
// linker discard record plus absence from the owned map. Associative children
// are inventoried only; this does not validate their unwind relocations.
func InspectGeneratedCOMDATSelection(unit, runtime, image, linkMap, linkLog []byte) (GeneratedCOMDATSelection, error) {
	var result GeneratedCOMDATSelection
	if len(linkLog) == 0 || len(linkLog) > maxGeneratedCOMDATLinkLogBytes {
		return result, fmt.Errorf("generated COMDAT linker log has invalid size")
	}
	peReport, err := InspectExecutable(image)
	if err != nil {
		return result, err
	}
	discards, err := generatedDiscardRecords(linkLog)
	if err != nil {
		return result, err
	}
	inputs := []struct {
		name string
		raw  []byte
	}{{"unit.obj", unit}, {"runtime.obj", runtime}}
	for _, input := range inputs {
		object, err := readGeneratedCOMDATObject(input.name, input.raw, linkMap)
		if err != nil {
			return result, err
		}
		selected, err := object.inspect(image, peReport, discards)
		if result.Parents == nil {
			result.Parents = selected.Parents
		} else {
			result.Parents = append(result.Parents, selected.Parents...)
		}
		result.Retained += selected.Retained
		result.Discarded += selected.Discarded
		result.AssociativeSectionInventory += selected.AssociativeSectionInventory
		if err != nil {
			return result, err
		}
	}
	return result, nil
}

func generatedObjectOwner(owner string) string {
	owner = strings.ToLower(strings.ReplaceAll(owner, "\\", "/"))
	return owner[strings.LastIndexByte(owner, '/')+1:]
}

func generatedDiscardRecords(log []byte) (map[string]struct{}, error) {
	result := make(map[string]struct{})
	for _, line := range strings.Split(string(log), "\n") {
		value, ok := strings.CutPrefix(strings.TrimSpace(line), "Discarded ")
		if !ok {
			continue
		}
		for _, object := range []string{"unit.obj", "runtime.obj"} {
			if symbol, ok := strings.CutSuffix(value, " from "+object); ok {
				if !utf8.ValidString(symbol) || strings.IndexByte(symbol, 0) >= 0 {
					return nil, fmt.Errorf("invalid generated COMDAT discard symbol")
				}
				result[object+":"+symbol] = struct{}{}
			}
		}
	}
	return result, nil
}

func generatedImageBytes(image []byte, report Report, rva, count uint32) ([]byte, error) {
	data, err := generatedImageView(image, report, rva, count)
	if err != nil {
		return nil, err
	}
	return slices.Clone(data), nil
}

// generatedImageView borrows a validated range for synchronous, read-only inspection.
// Consumers that mask their bytes must keep using generatedImageBytes for an owned copy.
func generatedImageView(image []byte, report Report, rva, count uint32) ([]byte, error) {
	for _, section := range report.Sections {
		if rva < section.VirtualAddress {
			continue
		}
		offset := uint64(rva - section.VirtualAddress)
		if offset+uint64(count) > uint64(min(section.RawSize, section.VirtualSize)) {
			continue
		}
		start := uint64(section.RawOffset) + offset
		return image[start : start+uint64(count)], nil
	}
	return nil, fmt.Errorf("generated COMDAT range %#x+%d is not file-backed", rva, count)
}
