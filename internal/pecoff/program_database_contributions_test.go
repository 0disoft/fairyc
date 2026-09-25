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
	if len(report.Modules) != 1 || report.Modules[0] != (PDBModule{Index: 0, Name: "unit.obj", ObjectName: "unit.obj"}) {
		t.Fatalf("PDB modules = %+v", report.Modules)
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
		{"unsupported record version", func(value []byte) { binary.LittleEndian.PutUint32(value[5*512+148:], 0) }, "unsupported"},
		{"invalid PE section", func(value []byte) { binary.LittleEndian.PutUint16(value[5*512+152:], 99) }, "invalid image range"},
		{"negative width", func(value []byte) { binary.LittleEndian.PutUint32(value[5*512+160:], 0xffffffff) }, "invalid image range"},
		{"missing module", func(value []byte) { binary.LittleEndian.PutUint16(value[5*512+168:], 0xffff) }, "module index"},
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

func TestInspectPDBSectionContributionsModulesBoundsAndPrivatePaths(t *testing.T) {
	record := testPDBModule(`C:\scratch\unit.obj`, `D:\toolchain\runtime.lib`)
	modules, err := inspectPDBModules(record)
	if err != nil || len(modules) != 1 || modules[0].Name != "unit.obj" || modules[0].ObjectName != "runtime.lib" {
		t.Fatalf("module names = %+v, %v", modules, err)
	}
	second := testPDBModule("runtime.obj", "runtime.obj")
	modules, err = inspectPDBModules(append(record, second...))
	if err != nil || len(modules) != 2 || modules[1].Index != 1 || modules[1].Name != "runtime.obj" {
		t.Fatalf("two module records = %+v, %v", modules, err)
	}
	unnamed, err := inspectPDBModules(testPDBModule("", ""))
	if err != nil || len(unnamed) != 1 || unnamed[0].Name != "<unnamed>" || unnamed[0].ObjectName != "<unnamed>" {
		t.Fatalf("unnamed PDB module = %+v, %v", unnamed, err)
	}
	for _, testCase := range []struct {
		name string
		data []byte
	}{
		{"short fixed header", record[:pdbModuleFixedBytes-1]},
		{"missing first terminator", append(make([]byte, pdbModuleFixedBytes), 'a')},
		{"missing second terminator", append(append(make([]byte, pdbModuleFixedBytes), 'a', 0), 'b')},
		{"truncated padding", record[:len(record)-1]},
		{"nonzero padding", func() []byte { changed := append([]byte(nil), record...); changed[len(changed)-1] = 1; return changed }()},
		{"empty final component", testPDBModule(`C:\scratch\`, "unit.obj")},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			if _, err := inspectPDBModules(testCase.data); err == nil {
				t.Fatal("malformed module record was accepted")
			}
		})
	}
}

func testPDBModule(name, objectName string) []byte {
	record := make([]byte, pdbModuleFixedBytes)
	record = append(record, name...)
	record = append(record, 0)
	record = append(record, objectName...)
	record = append(record, 0)
	for len(record)%4 != 0 {
		record = append(record, 0)
	}
	return record
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
	binary.LittleEndian.PutUint32(pdb[directory+16:], 180)
	binary.LittleEndian.PutUint32(pdb[directory+20:], 4)
	binary.LittleEndian.PutUint32(pdb[directory+24:], 5)
	dbi := 5 * blockSize
	binary.LittleEndian.PutUint32(pdb[dbi:], 0xffffffff)
	binary.LittleEndian.PutUint32(pdb[dbi+4:], 19990903)
	binary.LittleEndian.PutUint32(pdb[dbi+8:], 7)
	binary.LittleEndian.PutUint32(pdb[dbi+24:], 84)
	binary.LittleEndian.PutUint32(pdb[dbi+28:], 32)
	binary.LittleEndian.PutUint16(pdb[dbi+58:], machineAMD64)
	copy(pdb[dbi+64:], testPDBModule("unit.obj", "unit.obj"))
	binary.LittleEndian.PutUint32(pdb[dbi+148:], pdbSectionContribVersion60)
	binary.LittleEndian.PutUint16(pdb[dbi+152:], 1)
	binary.LittleEndian.PutUint32(pdb[dbi+160:], 1)
	binary.LittleEndian.PutUint32(pdb[dbi+164:], sectionContainsCode|sectionExecutable|sectionReadable)
	return pdb
}
