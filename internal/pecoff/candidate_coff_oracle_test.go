package pecoff

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCandidateCOFFOracle(t *testing.T) {
	path := os.Getenv("FAIRYC_COFF_ORACLE")
	if path == "" {
		t.Skip("explicit authored COFF oracle only")
	}
	type sample struct {
		name string
		data []byte
	}
	cases := []sample{}
	add := func(name string, data []byte) { cases = append(cases, sample{name, data}) }
	seeds := []sample{{"minimal", minimalAMD64COFF()}, {"comdat", comdatAMD64COFF()}, {"associative", associativeCOMDATAMD64COFF()}, {"weak", testWeakObject()}, {"big", testBigObject()}, {"bss", minimalAMD64COFFWithBSS()}}
	for _, seed := range seeds {
		add(seed.name, seed.data)
		for cut := 0; cut < len(seed.data); cut += 7 {
			add(fmt.Sprintf("%s/cut/%d", seed.name, cut), bytes.Clone(seed.data[:cut]))
		}
		for at := 0; at+4 <= len(seed.data); at += 4 {
			for _, value := range []uint32{0, 1, 2, 3, 5, 16, 96, 97, 16384, 65535, 65536, 0xffffffff} {
				b := bytes.Clone(seed.data)
				binary.LittleEndian.PutUint32(b[at:], value)
				add(fmt.Sprintf("%s/word/%d/%d", seed.name, at, value), b)
			}
		}
	}
	for _, count := range []int{96, 154, 4096, 4135, 16384, 16385} {
		add(fmt.Sprintf("fanout/%d", count), manySectionAMD64COFF(count))
	}
	for selection := byte(0); selection < 9; selection++ {
		for _, parent := range []uint16{0, 1, 2, 3, 65535} {
			b := associativeCOMDATAMD64COFF()
			aux := associativeCOFFSymbolOffset + 4*18
			b[aux+14] = selection
			binary.LittleEndian.PutUint16(b[aux+12:], parent)
			add(fmt.Sprintf("selection/%d/%d", selection, parent), b)
		}
	}
	for kind := uint16(0); kind < 20; kind++ {
		b := minimalAMD64COFF()
		binary.LittleEndian.PutUint16(b[testCOFFRelocOffset+8:], kind)
		add(fmt.Sprintf("relocation/%d", kind), b)
	}
	cycle := associativeCOMDATAMD64COFF()
	cycle[associativeCOFFSymbolOffset+18+14] = coffCOMDATAssociative
	binary.LittleEndian.PutUint16(cycle[associativeCOFFSymbolOffset+18+12:], 2)
	add("associative/two-section-cycle", cycle)
	parent := associativeCOMDATAMD64COFF()
	binary.LittleEndian.PutUint32(parent[coffFileHeaderBytes+36:], sectionContainsCode|sectionExecutable|sectionReadable)
	parent[associativeCOFFSymbolOffset+18+14] = 0
	add("associative/non-comdat-parent", parent)
	overflow, err := BuildAMD64RelocationOverflowProbeObject("probe")
	if err != nil {
		t.Fatal(err)
	}
	add("overflow", overflow)
	for _, edit := range []struct {
		at    int
		value uint32
	}{{coffFileHeaderBytes + 32, 0}, {coffFileHeaderBytes + 24, 0}, {71, 65535}, {71, 65537}, {75, 1}, {79, 4}} {
		b := bytes.Clone(overflow)
		binary.LittleEndian.PutUint32(b[edit.at:], edit.value)
		add(fmt.Sprintf("overflow/%d/%d", edit.at, edit.value), b)
	}
	if directory := os.Getenv("FAIRYC_COFF_OBJECTS"); directory != "" {
		paths, err := filepath.Glob(filepath.Join(directory, "*.obj"))
		if err != nil || len(paths) != 12 {
			t.Fatal("12 real object inputs required", err)
		}
		for _, p := range paths {
			b, err := os.ReadFile(p)
			if err != nil {
				t.Fatal(err)
			}
			add("real/"+filepath.Base(p), b)
		}
	}
	var out bytes.Buffer
	out.WriteString("FYCCOFFVERIFY1\n")
	put := func(n uint32) {
		if err := binary.Write(&out, binary.LittleEndian, n); err != nil {
			t.Fatal(err)
		}
	}
	part := func(b []byte) { put(uint32(len(b))); out.Write(b) }
	put(uint32(len(cases)))
	accepted := 0
	for _, c := range cases {
		part([]byte(c.name))
		part(c.data)
		var text strings.Builder
		if r, err := InspectObject(c.data); err == nil {
			accepted++
			fmt.Fprintf(&text, "%d/%d/%t/%d/%d/%d/%d\n", r.Machine, r.Characteristics, r.BigObject, r.BigObjectVersion, len(r.Sections), len(r.SymbolRecords), r.Relocations)
			for _, s := range r.Sections {
				fmt.Fprintf(&text, "%s/%d/%d/%d/%d/%d/%d\n", s.Name, s.RawOffset, s.RawSize, s.Relocations, s.Characteristics, s.COMDATSelection, s.AssociativeSection)
			}
			for _, s := range r.SymbolRecords {
				fmt.Fprintf(&text, "%d/%s/%d/%d/%d/%d/%d/%d/%s/%d\n", s.TableIndex, s.Name, s.Value, s.SectionNumber, s.Type, s.StorageClass, s.AuxiliaryRecords, s.WeakFallbackIndex, s.WeakFallback, s.WeakSearch)
			}
			for _, r := range r.RelocationRecords {
				fmt.Fprintf(&text, "%d/%s/%d/%d/%s/%d\n", r.SectionIndex, r.Section, r.VirtualAddress, r.SymbolIndex, r.Symbol, r.Type)
			}
		}
		part([]byte(text.String()))
		put(0)
	}
	if err := os.WriteFile(path, out.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	t.Logf("exported %d COFF cases, %d strict accepted, %d bytes", len(cases), accepted, out.Len())
}
