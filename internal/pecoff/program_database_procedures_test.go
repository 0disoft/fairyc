package pecoff

import (
	"encoding/binary"
	"strings"
	"testing"
)

func TestInspectPDBSectionContributionsProcedureRecords(t *testing.T) {
	local := testPDBProcedureRecord(0x110f, `C:\private\local`, 1, 1)
	global := testPDBProcedureRecord(0x1110, "global", 3, 1)
	binary.LittleEndian.PutUint32(global[32:], 5)
	unknown := []byte{2, 0, 0x55, 0x55}
	symbols := append(append(local, unknown...), global...)
	records, err := inspectPDBProcedureRecords(symbols)
	if err != nil || len(records) != 2 || records[0] != (PDBProcedureRecord{Name: "local", Section: 1, Offset: 2, Size: 1}) ||
		records[1] != (PDBProcedureRecord{Name: "global", Section: 1, Offset: 5, Size: 3}) {
		t.Fatalf("procedure records = %+v, %v", records, err)
	}

	pdb := testPDBProcedureDatabase(symbols)
	records, err = InspectPDBModuleProcedureRecords(testPDBImage(), pdb, 0)
	if err != nil || len(records) != 2 || records[0].Name != "local" {
		t.Fatalf("module procedures = %+v, %v", records, err)
	}
	if _, err := InspectPDBModuleProcedureRecords(testPDBImage(), pdb, 1); err == nil || !strings.Contains(err.Error(), "module index") {
		t.Fatalf("missing module error = %v", err)
	}

	for _, testCase := range []struct {
		name string
		edit func([]byte)
		want string
	}{
		{"missing module stream", func(data []byte) { binary.LittleEndian.PutUint16(data[5*512+64+34:], 99) }, "is missing"},
		{"symbol bytes past stream", func(data []byte) { binary.LittleEndian.PutUint32(data[5*512+64+36:], 513) }, "substreams exceed"},
		{"C13 bytes past stream", func(data []byte) { binary.LittleEndian.PutUint32(data[5*512+64+44:], 512) }, "substreams exceed"},
		{"bad signature", func(data []byte) { data[6*512] = 3 }, "unsupported symbol signature"},
		{"short symbol size", func(data []byte) { binary.LittleEndian.PutUint32(data[5*512+64+36:], 3) }, "invalid symbol-byte size"},
		{"unterminated last record", func(data []byte) { data[6*512+4+len(symbols)-1] = 'x' }, "unterminated"},
		{"misaligned symbol end", func(data []byte) {
			binary.LittleEndian.PutUint32(data[5*512+64+36:], uint32(5+len(symbols)))
			binary.LittleEndian.PutUint32(data[3*512+20:], uint32(5+len(symbols)))
		}, "truncated"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			changed := append([]byte(nil), pdb...)
			testCase.edit(changed)
			if _, err := InspectPDBModuleProcedureRecords(testPDBImage(), changed, 0); err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("procedure error = %v, want %q", err, testCase.want)
			}
		})
	}
}

func TestInspectPDBSectionContributionsProcedureRecordBounds(t *testing.T) {
	valid := testPDBProcedureRecord(0x110f, "function", 1, 1)
	for _, testCase := range []struct {
		name string
		data []byte
		want string
	}{
		{"truncated header", valid[:3], "header is truncated"},
		{"record outside symbols", func() []byte {
			value := append([]byte(nil), valid...)
			binary.LittleEndian.PutUint16(value, uint16(len(value)))
			return value
		}(), "exceeds"},
		{"truncated procedure", valid[:pdbProcedureFixedBytes], "exceeds"},
		{"missing terminator", func() []byte { value := append([]byte(nil), valid...); value[len(value)-1] = 'x'; return value }(), "unterminated"},
		{"missing name", testPDBProcedureRecord(0x1110, "", 1, 1), "missing"},
		{"zero section", testPDBProcedureRecord(0x1110, "name", 1, 0), "invalid code range"},
		{"overflow", testPDBProcedureRecord(0x1110, "name", 0xffffffff, 1), "invalid code range"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			if _, err := inspectPDBProcedureRecords(testCase.data); err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("record error = %v, want %q", err, testCase.want)
			}
		})
	}
}

func testPDBProcedureRecord(kind uint16, name string, size, section uint32) []byte {
	record := make([]byte, pdbProcedureFixedBytes)
	binary.LittleEndian.PutUint16(record[2:], kind)
	binary.LittleEndian.PutUint32(record[16:], size)
	binary.LittleEndian.PutUint32(record[32:], 2)
	binary.LittleEndian.PutUint16(record[36:], uint16(section))
	record = append(record, name...)
	record = append(record, 0)
	binary.LittleEndian.PutUint16(record, uint16(len(record)-2))
	return record
}

func testPDBProcedureDatabase(symbols []byte) []byte {
	const blockSize = 512
	pdb := make([]byte, 7*blockSize)
	copy(pdb, testPDBSectionContribution())
	binary.LittleEndian.PutUint32(pdb[40:], 7)
	binary.LittleEndian.PutUint32(pdb[44:], 36)
	directory := 3 * blockSize
	binary.LittleEndian.PutUint32(pdb[directory:], 5)
	binary.LittleEndian.PutUint32(pdb[directory+20:], uint32(4+len(symbols)))
	binary.LittleEndian.PutUint32(pdb[directory+24:], 4)
	binary.LittleEndian.PutUint32(pdb[directory+28:], 5)
	binary.LittleEndian.PutUint32(pdb[directory+32:], 6)
	binary.LittleEndian.PutUint16(pdb[5*blockSize+64+34:], 4)
	binary.LittleEndian.PutUint32(pdb[5*blockSize+64+36:], uint32(4+len(symbols)))
	binary.LittleEndian.PutUint32(pdb[6*blockSize:], 4)
	copy(pdb[6*blockSize+4:], symbols)
	return pdb
}
