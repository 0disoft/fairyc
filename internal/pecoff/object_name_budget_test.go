package pecoff

import (
	"encoding/binary"
	"errors"
	"strings"
	"testing"
)

func TestCOFFNameBudgetExactBoundary(t *testing.T) {
	object := ordinaryObjectWithSharedNames(1024, strings.Repeat("n", 64<<10))
	last := testCOFFSymbolOffset + 1023*18
	for _, skip := range []uint32{5, 4, 6} {
		binary.LittleEndian.PutUint32(object[last+4:], 4+skip)
		err := verifyCOFFNameBudget(object, 0)
		// The section name consumes five bytes. The final symbol can
		// reference a suffix, bringing the aggregate to exactly 64 MiB.
		if (err == nil) != (skip >= 5) {
			t.Fatalf("boundary skip %d: %v", skip, err)
		}
	}
}

func TestCOFFNameBudgetSkipsAuxiliaryRecords(t *testing.T) {
	object := ordinaryObjectWithSharedNames(2048, strings.Repeat("n", 64<<10))
	for index := 0; index < 2048; index += 256 {
		object[testCOFFSymbolOffset+index*18+17] = 255
	}
	report, err := InspectObject(object)
	if err != nil || len(report.SymbolRecords) != 8 {
		t.Fatalf("auxiliary records charged as names: symbols=%d err=%v", len(report.SymbolRecords), err)
	}
}

func TestCOFFNameBudgetCoversSectionsAndSharedBudget(t *testing.T) {
	for _, counts := range [][2]int{{1025, 0}, {512, 512}} {
		for _, reference := range []string{"/4", "/+4"} {
			object := sharedCOFFNameBudgetFixture(counts[0], counts[1], strings.Repeat("n", (64<<10)+1))
			for index := 0; index < counts[0]; index++ {
				name := object[coffFileHeaderBytes+index*coffSectionHeaderBytes:][:8]
				clear(name)
				copy(name, reference)
			}
			if _, err := InspectObject(object); !errors.Is(err, ErrNameBudgetExceeded) {
				t.Fatalf("sections=%d symbols=%d reference=%q: %v", counts[0], counts[1], reference, err)
			}
		}
	}
}

func TestCOFFNameBudgetChecksCompleteHeadersBeforeTruncatedOnes(t *testing.T) {
	object := sharedCOFFNameBudgetFixture(1025, 0, strings.Repeat("n", 64<<10))
	binary.LittleEndian.PutUint16(object[2:], maxObjectSections)
	if _, err := InspectObject(object); !errors.Is(err, ErrNameBudgetExceeded) {
		t.Fatalf("truncated headers bypassed prefix name budget: %v", err)
	}
}

func TestInspectExecutableRejectsExpandedSymbolNames(t *testing.T) {
	image := minimalAMD64PE()
	object := ordinaryObjectWithSharedNames(1024, strings.Repeat("n", 64<<10))
	binary.LittleEndian.PutUint32(image[testPEOffset+4+8:], uint32(len(image)))
	binary.LittleEndian.PutUint32(image[testPEOffset+4+12:], 1024)
	image = append(image, object[testCOFFSymbolOffset:]...)
	if _, err := InspectExecutable(image); !errors.Is(err, ErrNameBudgetExceeded) {
		t.Fatalf("PE symbol names bypassed the shared budget: %v", err)
	}
}

func sharedCOFFNameBudgetFixture(sections, symbols int, name string) []byte {
	symbolOffset := coffFileHeaderBytes + sections*coffSectionHeaderBytes
	stringOffset := symbolOffset + symbols*18
	object := make([]byte, stringOffset+4+len(name)+1)
	binary.LittleEndian.PutUint16(object, machineAMD64)
	binary.LittleEndian.PutUint16(object[2:], uint16(sections))
	binary.LittleEndian.PutUint32(object[8:], uint32(symbolOffset))
	binary.LittleEndian.PutUint32(object[12:], uint32(symbols))
	for index := range sections {
		copy(object[coffFileHeaderBytes+index*coffSectionHeaderBytes:], "/4")
	}
	for index := range symbols {
		binary.LittleEndian.PutUint32(object[symbolOffset+index*18+4:], 4)
	}
	binary.LittleEndian.PutUint32(object[stringOffset:], uint32(4+len(name)+1))
	copy(object[stringOffset+4:], name)
	return object
}

func TestInspectObjectRejectsExpandedSymbolNames(t *testing.T) {
	object := ordinaryObjectWithSharedNames(1024, strings.Repeat("n", 64<<10))
	// Symbols alone use 64 MiB, before counting the .text section name.
	if _, err := InspectObject(object); err == nil || !strings.Contains(err.Error(), "name byte budget") {
		t.Fatalf("expanded names were not rejected before materialization: %v", err)
	}
}

func BenchmarkInspectObjectExpandedNameBudget(b *testing.B) {
	object := ordinaryObjectWithSharedNames(2048, strings.Repeat("n", 64<<10))
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_, err := InspectObject(object)
		// Both versions run so the same fixture measures admission before
		// the fix and bounded rejection afterward. The regression test gates rejection.
		if err != nil && !strings.Contains(err.Error(), "name byte budget") {
			b.Fatal(err)
		}
	}
}
