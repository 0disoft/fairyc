//go:build windows

package compiler_test

import (
	"testing"

	"github.com/0disoft/fairyc/internal/pecoff"
)

type scannerPDBUnwindOwnership struct {
	ByModule       map[uint16]uint64
	TotalBytes     uint64
	Attributed     uint64
	Unmatched      uint64
	Ambiguous      uint64
	MatchedEntries int
}

// A function is assigned only when one module's contribution contains the
// complete validated unwind interval. Partial and conflicting ranges stay out.
func scannerPDBUnwindOwners(image pecoff.Report, functions []pecoff.RuntimeFunction, contributions []pecoff.PDBSectionContribution) scannerPDBUnwindOwnership {
	result := scannerPDBUnwindOwnership{ByModule: make(map[uint16]uint64)}
	for _, function := range functions {
		width := uint64(function.EndRVA - function.BeginRVA)
		result.TotalBytes += width
		var sectionNumber uint16
		var begin, end uint64
		for index, section := range image.Sections {
			base := uint64(section.VirtualAddress)
			if uint64(function.BeginRVA) >= base && uint64(function.EndRVA) <= base+uint64(section.VirtualSize) {
				sectionNumber = uint16(index + 1)
				begin = uint64(function.BeginRVA) - base
				end = uint64(function.EndRVA) - base
				break
			}
		}
		if sectionNumber == 0 {
			result.Unmatched += width
			continue
		}
		modules := make(map[uint16]struct{})
		for _, contribution := range contributions {
			if contribution.Section != sectionNumber || contribution.Characteristics&0x20 == 0 {
				continue
			}
			if uint64(contribution.Offset) <= begin && uint64(contribution.Offset)+uint64(contribution.Size) >= end {
				modules[contribution.ModuleIndex] = struct{}{}
			}
		}
		switch len(modules) {
		case 0:
			result.Unmatched += width
		case 1:
			for module := range modules {
				result.ByModule[module] += width
			}
			result.Attributed += width
			result.MatchedEntries++
		default:
			result.Ambiguous += width
		}
	}
	return result
}

func TestScannerPDBUnwindOwnershipRejectsPartialAndConflictingRanges(t *testing.T) {
	image := pecoff.Report{Sections: []pecoff.Section{{VirtualAddress: 0x1000, VirtualSize: 0x100}}}
	functions := []pecoff.RuntimeFunction{
		{BeginRVA: 0x1000, EndRVA: 0x1010},
		{BeginRVA: 0x1010, EndRVA: 0x1020},
		{BeginRVA: 0x1020, EndRVA: 0x1030},
		{BeginRVA: 0x1030, EndRVA: 0x1045},
	}
	contributions := []pecoff.PDBSectionContribution{
		{Section: 1, Offset: 0, Size: 0x10, Characteristics: 0x20, ModuleIndex: 1},
		{Section: 1, Offset: 0x10, Size: 0x30, Characteristics: 0x20, ModuleIndex: 2},
		{Section: 1, Offset: 0x20, Size: 0x10, Characteristics: 0x20, ModuleIndex: 3},
	}
	result := scannerPDBUnwindOwners(image, functions, contributions)
	if result.TotalBytes != 69 || result.Attributed != 32 || result.Ambiguous != 16 ||
		result.Unmatched != 21 || result.MatchedEntries != 2 ||
		result.ByModule[1] != 16 || result.ByModule[2] != 16 || len(result.ByModule) != 2 {
		t.Fatalf("PDB unwind ownership = %+v", result)
	}
}
