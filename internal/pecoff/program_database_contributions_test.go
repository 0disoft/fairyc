package pecoff

import (
	"encoding/binary"
	"strings"
	"testing"
)

func TestInspectPDBSectionContributionsBindsFinalImage(t *testing.T) {
	image := testPDBImage()
	pdb := testPDBSectionContribution()
	report, err := InspectPDBSectionContributions(image, pdb)
	if err != nil {
		t.Fatal(err)
	}
	if report.Version != pdbSectionContribVersion60 || len(report.Contributions) != 1 {
		t.Fatalf("section contribution report = %+v", report)
	}
	item := report.Contributions[0]
	if item.Section != 1 || item.Offset != 0 || item.Size != 1 || item.ModuleIndex != 0 ||
		item.Characteristics != sectionContainsCode|sectionExecutable|sectionReadable {
		t.Fatalf("section contribution = %+v", item)
	}
	tests := []struct {
		name string
		edit func([]byte)
		want string
	}{
		{"DBI age", func(value []byte) { binary.LittleEndian.PutUint32(value[5*512+8:], 8) }, "does not match"},
		{"unsupported record version", func(value []byte) { binary.LittleEndian.PutUint32(value[5*512+64:], 0) }, "unsupported"},
		{"invalid PE section", func(value []byte) { binary.LittleEndian.PutUint16(value[5*512+68:], 99) }, "invalid image range"},
		{"negative width", func(value []byte) { binary.LittleEndian.PutUint32(value[5*512+76:], 0xffffffff) }, "invalid image range"},
		{"missing module", func(value []byte) { binary.LittleEndian.PutUint16(value[5*512+84:], 0xffff) }, "module index"},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			changed := append([]byte(nil), pdb...)
			testCase.edit(changed)
			if _, err := InspectPDBSectionContributions(image, changed); err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("section contribution error = %v, want %q", err, testCase.want)
			}
		})
	}
}

func testPDBSectionContribution() []byte {
	const blockSize = 512
	pdb := make([]byte, 6*blockSize)
	copy(pdb, testProgramDatabase())
	binary.LittleEndian.PutUint32(pdb[40:], 6)
	binary.LittleEndian.PutUint32(pdb[44:], 28)
	directory := 3 * blockSize
	binary.LittleEndian.PutUint32(pdb[directory:], 4)
	binary.LittleEndian.PutUint32(pdb[directory+12:], pdbNilStreamSize)
	binary.LittleEndian.PutUint32(pdb[directory+16:], 96)
	binary.LittleEndian.PutUint32(pdb[directory+20:], 4)
	binary.LittleEndian.PutUint32(pdb[directory+24:], 5)
	dbi := 5 * blockSize
	binary.LittleEndian.PutUint32(pdb[dbi:], 0xffffffff)
	binary.LittleEndian.PutUint32(pdb[dbi+4:], 19990903)
	binary.LittleEndian.PutUint32(pdb[dbi+8:], 7)
	binary.LittleEndian.PutUint32(pdb[dbi+28:], 32)
	binary.LittleEndian.PutUint16(pdb[dbi+58:], machineAMD64)
	binary.LittleEndian.PutUint32(pdb[dbi+64:], pdbSectionContribVersion60)
	binary.LittleEndian.PutUint16(pdb[dbi+68:], 1)
	binary.LittleEndian.PutUint32(pdb[dbi+76:], 1)
	binary.LittleEndian.PutUint32(pdb[dbi+80:], sectionContainsCode|sectionExecutable|sectionReadable)
	return pdb
}
