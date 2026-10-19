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

func TestCandidatePEOracle(t *testing.T) {
	path := os.Getenv("FAIRYC_PE_ORACLE")
	if path == "" {
		t.Skip("explicit authored PE oracle only")
	}
	type sample struct {
		name string
		data []byte
	}
	cases := []sample{{"minimal", minimalAMD64PE()}, {"unwind", minimalAMD64PEWithUnwind()}, {"handler", testHandlerImage()}}
	add := func(name string, data []byte) { cases = append(cases, sample{name, data}) }
	for _, seed := range append([]sample(nil), cases...) {
		for cut := 0; cut < len(seed.data); cut += 13 {
			add(fmt.Sprintf("%s/cut/%d", seed.name, cut), bytes.Clone(seed.data[:cut]))
		}
		for at := 0; at < testSectionHeaderOffset+40; at += 2 {
			for _, value := range []uint32{0, 1, 2, 16, 96, 97, 112, 120, 128, 240, 248, 512, 4096, 65536, 0xffffffff} {
				b := bytes.Clone(seed.data)
				binary.LittleEndian.PutUint32(b[at:], value)
				add(fmt.Sprintf("%s/word/%d/%d", seed.name, at, value), b)
			}
		}
	}
	// All operation/info pairs under versions 1 and 2, with enough code slots
	// for three-slot operations. Invalid ordering and flags are independently mutated.
	for version := byte(1); version <= 2; version++ {
		for op := byte(0); op < 16; op++ {
			for info := byte(0); info < 16; info++ {
				b := minimalAMD64PEWithUnwind()
				b[testUnwindXDataRawOffset] = version
				b[testUnwindXDataRawOffset+1] = 1
				b[testUnwindXDataRawOffset+2] = 3
				b[testUnwindXDataRawOffset+4] = 1
				b[testUnwindXDataRawOffset+5] = op | info<<4
				add(fmt.Sprintf("operation/%d/%d/%d", version, op, info), b)
			}
		}
	}
	for at := testUnwindXDataRawOffset; at < testUnwindXDataRawOffset+20; at++ {
		for _, value := range []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 15, 16, 31, 63, 127, 255} {
			b := minimalAMD64PEWithUnwind()
			b[at] = value
			add(fmt.Sprintf("info/%d/%d", at, value), b)
		}
	}
	for _, field := range []int{testUnwindDirectoryOffset, testUnwindDirectoryOffset + 4, testUnwindPDataRawOffset, testUnwindPDataRawOffset + 4, testUnwindPDataRawOffset + 8} {
		for _, value := range []uint32{0, 1, 4, 12, 13, 24, 0x1000, 0x1001, 0x2000, 0x3000, 0xfffffff0, 0xffffffff} {
			b := minimalAMD64PEWithUnwind()
			binary.LittleEndian.PutUint32(b[field:], value)
			add(fmt.Sprintf("unwind-field/%d/%d", field, value), b)
		}
	}
	// Optional directory count can be smaller or larger than the usual 16.
	for _, n := range []uint16{0, 1, 2, 3, 4, 15, 16, 17, 20} {
		b := minimalAMD64PE()
		old := testSectionHeaderOffset
		fresh := testOptionalHeaderOffset + 112 + int(n)*8
		section := bytes.Clone(b[old : old+40])
		clear(b[testOptionalHeaderOffset+112 : 0x200])
		copy(b[fresh:], section)
		binary.LittleEndian.PutUint16(b[testPEOffset+20:], 112+n*8)
		binary.LittleEndian.PutUint32(b[testOptionalHeaderOffset+108:], uint32(n))
		add(fmt.Sprintf("directories/%d", n), b)
	}
	// Shared info is counted once; chained records follow the same bounded Go
	// structural contract (no recursive chain traversal).
	for _, shared := range []bool{false, true} {
		b := minimalAMD64PEWithUnwind()
		binary.LittleEndian.PutUint32(b[testUnwindDirectoryOffset+4:], 24)
		binary.LittleEndian.PutUint32(b[testUnwindPDataRawOffset+12:], 0x1001)
		binary.LittleEndian.PutUint32(b[testUnwindPDataRawOffset+16:], 0x1002)
		rva := uint32(0x3004)
		if shared {
			rva = 0x3000
		}
		binary.LittleEndian.PutUint32(b[testUnwindPDataRawOffset+20:], rva)
		b[testUnwindXDataRawOffset+4] = 1
		add(fmt.Sprintf("shared/%t", shared), b)
	}
	b := minimalAMD64PEWithUnwind()
	b[testUnwindXDataRawOffset] = 1 | 4<<3
	binary.LittleEndian.PutUint32(b[testUnwindXDataRawOffset+4:], 0x1000)
	binary.LittleEndian.PutUint32(b[testUnwindXDataRawOffset+8:], 0x1001)
	binary.LittleEndian.PutUint32(b[testUnwindXDataRawOffset+12:], 0x3000)
	add("chained", b)
	// Genuine MSVC artifacts are explicit caller inputs; none is executed here.
	if directory := os.Getenv("FAIRYC_PE_IMAGES"); directory != "" {
		paths, err := filepath.Glob(filepath.Join(directory, "*.exe"))
		if err != nil || len(paths) < 4 {
			t.Fatal("four real PE inputs required", err)
		}
		for _, p := range paths {
			raw, err := os.ReadFile(p)
			if err != nil {
				t.Fatal(err)
			}
			add("real/"+filepath.Base(p), raw)
		}
	}
	if compiler := os.Getenv("FAIRYC_PE_COMPILER"); compiler != "" {
		raw, err := os.ReadFile(compiler)
		if err != nil {
			t.Fatal(err)
		}
		add("real/compiler", raw)
	}
	var out bytes.Buffer
	out.WriteString("FYCPEVERIFY1\n")
	put := func(n uint32) {
		if err := binary.Write(&out, binary.LittleEndian, n); err != nil {
			t.Fatal(err)
		}
	}
	part := func(b []byte) { put(uint32(len(b))); out.Write(b) }
	put(uint32(len(cases)))
	var peAccepted, unwindAccepted int
	for _, c := range cases {
		part([]byte(c.name))
		part(c.data)
		var text strings.Builder
		if r, err := InspectExecutable(c.data); err == nil {
			peAccepted++
			fmt.Fprintf(&text, "%d/%d/%d/%d/%d/%d/%d/%d/%d/%d/%t/%t\n", r.Machine, r.Characteristics, r.ImageBase, r.EntryPointRVA, r.Subsystem, r.SectionAlignment, r.FileAlignment, r.SizeOfImage, r.SizeOfHeaders, len(r.Sections), r.ImportDirectoryPresent, r.ExceptionDirectoryPresent)
			for _, s := range r.Sections {
				fmt.Fprintf(&text, "%s/%d/%d/%d/%d/%d\n", s.Name, s.VirtualAddress, s.VirtualSize, s.RawOffset, s.RawSize, s.Characteristics)
			}
		}
		part([]byte(text.String()))
		text.Reset()
		if r, err := InspectAMD64Unwind(c.data); err == nil {
			unwindAccepted++
			fmt.Fprintf(&text, "%d/%d/%d/%d/%d/%d\n", r.RuntimeFunctions, r.UniqueUnwindInfos, r.HandlerInfos, r.ChainedInfos, r.Version2Infos, r.MaximumUnwindCodes)
			for _, f := range r.Functions {
				fmt.Fprintf(&text, "%d/%d/%d\n", f.BeginRVA, f.EndRVA, f.UnwindInfoRVA)
			}
		}
		part([]byte(text.String()))
	}
	if err := os.WriteFile(path, out.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	t.Logf("exported %d PE cases, accepted PE=%d unwind=%d, bytes=%d", len(cases), peAccepted, unwindAccepted, out.Len())
}
