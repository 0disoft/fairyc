package pecoff

import (
	"bytes"
	"debug/pe"
	"encoding/binary"
	"strings"
	"testing"
)

func TestInspectAMD64ChainLink(t *testing.T) {
	object, image, linkMap := testChainLinkInputs(t)
	report, err := InspectAMD64ChainLink(object, image, linkMap, "chain.obj")
	if err != nil {
		t.Fatal(err)
	}
	if report.SourceRVA != 0x1008 || len(report.Records) != 2 || report.RelocationFields != 6 {
		t.Fatalf("report=%+v", report)
	}
	for _, record := range report.Records {
		if record.Expected != record.Linked {
			t.Fatalf("unbound chain: %+v", record)
		}
		for _, addend := range record.Addends {
			if addend == 0 {
				t.Fatal("fixture lost nonzero addend coverage")
			}
		}
	}
}

func TestInspectAMD64ChainLinkRejectsWrongBindings(t *testing.T) {
	for _, tc := range []struct {
		name string
		edit func([]byte, []byte, *[]byte)
	}{
		{"object begin addend", func(o, _ []byte, _ *[]byte) { o[202+36]++ }},
		{"object end addend", func(o, _ []byte, _ *[]byte) { o[202+40]-- }},
		{"object unwind addend", func(o, _ []byte, _ *[]byte) { o[202+44] += 4 }},
		{"relocation type", func(o, _ []byte, _ *[]byte) { binary.LittleEndian.PutUint16(o[340+8:], ObjectRelocationAMD64Addr64) }},
		{"relocation target", func(o, _ []byte, _ *[]byte) { binary.LittleEndian.PutUint32(o[340+4:], 1) }},
		{"missing relocation", func(o, _ []byte, _ *[]byte) { binary.LittleEndian.PutUint32(o[340:], 0) }},
		{"duplicate relocation", func(o, _ []byte, _ *[]byte) { binary.LittleEndian.PutUint32(o[350:], 20) }},
		{"overflow addend", func(o, _ []byte, _ *[]byte) { binary.LittleEndian.PutUint32(o[202+20:], 0xffffffff) }},
		{"wrong map owner", func(_, _ []byte, m *[]byte) { *m = bytes.ReplaceAll(*m, []byte("chain.obj"), []byte("other.obj")) }},
		{"wrong map address", func(_, _ []byte, m *[]byte) { *m = bytes.Replace(*m, []byte("140003000"), []byte("140003004"), 1) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			object, image, linkMap := testChainLinkInputs(t)
			tc.edit(object, image, &linkMap)
			if _, err := InspectAMD64ChainLink(object, image, linkMap, "chain.obj"); err == nil {
				t.Fatal("wrong binding accepted")
			}
		})
	}
}

// These are still structurally valid chains. Only exact object/map/image
// binding detects that the linked parent tuple names the wrong code or record.
func TestInspectAMD64ChainLinkRejectsStructurallyValidWrongTail(t *testing.T) {
	for field, value := range []uint32{0x100d, 0x100f, 0x3008} {
		object, image, linkMap := testChainLinkInputs(t)
		binary.LittleEndian.PutUint32(image[testUnwindXDataRawOffset+36+field*4:], value)
		if _, err := InspectAMD64Unwind(image); err != nil {
			t.Fatalf("field %d structural precondition: %v", field, err)
		}
		if _, err := InspectAMD64ChainLink(object, image, linkMap, "chain.obj"); err == nil || !strings.Contains(err.Error(), "chain tail field") {
			t.Fatalf("field %d: %v", field, err)
		}
	}
}

func testChainLinkInputs(t *testing.T) ([]byte, []byte, []byte) {
	t.Helper()
	object, err := BuildAMD64ChainLinkProbe()
	if err != nil {
		t.Fatal(err)
	}
	file, err := pe.NewFile(bytes.NewReader(object))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	image := minimalAMD64PEWithUnwind()
	for i, section := range file.Sections {
		data, err := section.Data()
		if err != nil {
			t.Fatal(err)
		}
		raw := []int{testUnwindTextRawOffset, testUnwindPDataRawOffset, testUnwindXDataRawOffset}[i]
		copy(image[raw:], data)
		binary.LittleEndian.PutUint32(image[testUnwindSectionsOffset+i*40+8:], uint32(len(data)))
		for _, reloc := range section.Relocs {
			base := uint32(0x1000)
			if reloc.SymbolTableIndex == 2 {
				base = 0x3000
			}
			at := raw + int(reloc.VirtualAddress)
			binary.LittleEndian.PutUint32(image[at:], base+binary.LittleEndian.Uint32(image[at:]))
		}
	}
	binary.LittleEndian.PutUint32(image[testUnwindDirectoryOffset+4:], 36)
	linkMap := []byte("0001:00000000 fy_code 0000000140001000 chain.obj\n0001:00000008 fy_chain 0000000140001008 f chain.obj\n0003:00000000 fy_meta 0000000140003000 chain.obj\n")
	return object, image, linkMap
}
