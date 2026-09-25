package pecoff

import (
	"encoding/binary"
	"strings"
	"testing"
)

var testPDBGUID = [16]byte{0x10, 0x21, 0x32, 0x43, 0x54, 0x65, 0x76, 0x87, 0x98, 0xa9, 0xba, 0xcb, 0xdc, 0xed, 0xfe, 0x0f}

func TestInspectProgramDatabaseAssociationBindsRSDSIdentity(t *testing.T) {
	report, err := InspectProgramDatabaseAssociation(testPDBImage(), testProgramDatabase())
	if err != nil {
		t.Fatal(err)
	}
	if report.GUID != testPDBGUID || report.Age != 7 || report.Path != "program.pdb" ||
		report.PDBVersion != 20000404 || report.PDBSignature != 0x12345678 ||
		report.PDBBlockSize != 512 || report.PDBStreamSize != 28 {
		t.Fatalf("PDB association report = %+v", report)
	}
}

func TestInspectProgramDatabaseAssociationRejectsIdentityAndContainerMutations(t *testing.T) {
	tests := []struct {
		name string
		edit func(*[]byte, *[]byte)
		want string
	}{
		{name: "RSDS guid drifts", edit: func(image, _ *[]byte) {
			(*image)[testREL32ReceiptRaw+0x84]++
		}, want: "do not match"},
		{name: "PDB age drifts", edit: func(_, pdb *[]byte) {
			binary.LittleEndian.PutUint32((*pdb)[4*512+8:], 8)
		}, want: "do not match"},
		{name: "RSDS path is not contained", edit: func(image, _ *[]byte) {
			copy((*image)[testREL32ReceiptRaw+0x98:], "otherxx.pdb")
		}, want: "not the contained"},
		{name: "RSDS identity is empty", edit: func(image, _ *[]byte) {
			for index := 0; index < 16; index++ {
				(*image)[testREL32ReceiptRaw+0x84+index] = 0
			}
		}, want: "empty GUID or age"},
		{name: "MSF magic drifts", edit: func(_, pdb *[]byte) {
			(*pdb)[0] = 'X'
		}, want: "not a bounded MSF 7"},
		{name: "directory block escapes", edit: func(_, pdb *[]byte) {
			binary.LittleEndian.PutUint32((*pdb)[2*512:], 9)
		}, want: "outside the file"},
		{name: "stream block escapes", edit: func(_, pdb *[]byte) {
			binary.LittleEndian.PutUint32((*pdb)[3*512+12:], 9)
		}, want: "outside the file"},
		{name: "information stream amplifies", edit: func(_, pdb *[]byte) {
			binary.LittleEndian.PutUint32((*pdb)[3*512+8:], maxPDBInformationBytes+1)
		}, want: "information stream exceeds"},
		{name: "information stream reuses directory block", edit: func(_, pdb *[]byte) {
			binary.LittleEndian.PutUint32((*pdb)[3*512+12:], 3)
		}, want: "reuses stream directory"},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			image := testPDBImage()
			pdb := testProgramDatabase()
			testCase.edit(&image, &pdb)
			if _, err := InspectProgramDatabaseAssociation(image, pdb); err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("InspectProgramDatabaseAssociation() error = %v, want %q", err, testCase.want)
			}
		})
	}
}

func TestReadPDBStreamSelectsBoundedDBI(t *testing.T) {
	const blockSize = 512
	pdb := make([]byte, 6*blockSize)
	copy(pdb, testProgramDatabase())
	binary.LittleEndian.PutUint32(pdb[40:], 6)
	binary.LittleEndian.PutUint32(pdb[44:], 28)
	directory := 3 * blockSize
	binary.LittleEndian.PutUint32(pdb[directory:], 4)
	binary.LittleEndian.PutUint32(pdb[directory+12:], pdbNilStreamSize)
	binary.LittleEndian.PutUint32(pdb[directory+16:], 64)
	binary.LittleEndian.PutUint32(pdb[directory+20:], 4)
	binary.LittleEndian.PutUint32(pdb[directory+24:], 5)
	for index := 0; index < 64; index++ {
		pdb[5*blockSize+index] = byte(index)
	}
	stream, size, err := readPDBStream(pdb, 3)
	if err != nil || size != blockSize || len(stream) != 64 {
		t.Fatalf("DBI stream = %d bytes, block size %d, err %v", len(stream), size, err)
	}
	for index, value := range stream {
		if value != byte(index) {
			t.Fatalf("DBI byte %d = %d", index, value)
		}
	}
	if _, err := InspectProgramDatabaseAssociation(testPDBImage(), pdb); err != nil {
		t.Fatalf("association after DBI extraction: %v", err)
	}
	if _, _, err := readPDBStream(pdb, 4); err == nil || !strings.Contains(err.Error(), "is missing") {
		t.Fatalf("missing stream error = %v", err)
	}
	binary.LittleEndian.PutUint32(pdb[directory+24:], 4)
	if _, _, err := readPDBStream(pdb, 3); err == nil || !strings.Contains(err.Error(), "reuses stream 1") {
		t.Fatalf("duplicate DBI block error = %v", err)
	}
}

func testPDBImage() []byte {
	image := testREL32Image()
	debugSection := testSectionHeaderOffset + coffSectionHeaderBytes
	binary.LittleEndian.PutUint32(image[debugSection+8:], 0x100)
	dataDirectory := testOptionalHeaderOffset + 112 + debugDirectoryIndex*8
	binary.LittleEndian.PutUint32(image[dataDirectory:], 0x2040)
	binary.LittleEndian.PutUint32(image[dataDirectory+4:], imageDebugDirectoryLen)
	directory := testREL32ReceiptRaw + 0x40
	binary.LittleEndian.PutUint32(image[directory+12:], imageDebugTypeCodeView)
	path := []byte("program.pdb\x00")
	size := uint32(24 + len(path))
	binary.LittleEndian.PutUint32(image[directory+16:], size)
	binary.LittleEndian.PutUint32(image[directory+20:], 0x2080)
	binary.LittleEndian.PutUint32(image[directory+24:], uint32(testREL32ReceiptRaw+0x80))
	rsds := testREL32ReceiptRaw + 0x80
	copy(image[rsds:], "RSDS")
	copy(image[rsds+4:], testPDBGUID[:])
	binary.LittleEndian.PutUint32(image[rsds+20:], 7)
	copy(image[rsds+24:], path)
	return image
}

func testProgramDatabase() []byte {
	const blockSize = 512
	pdb := make([]byte, 5*blockSize)
	copy(pdb, msf7Magic)
	binary.LittleEndian.PutUint32(pdb[32:], blockSize)
	binary.LittleEndian.PutUint32(pdb[36:], 1)
	binary.LittleEndian.PutUint32(pdb[40:], 5)
	binary.LittleEndian.PutUint32(pdb[44:], 16)
	binary.LittleEndian.PutUint32(pdb[52:], 2)
	binary.LittleEndian.PutUint32(pdb[2*blockSize:], 3)
	directory := 3 * blockSize
	binary.LittleEndian.PutUint32(pdb[directory:], 2)
	binary.LittleEndian.PutUint32(pdb[directory+4:], pdbNilStreamSize)
	binary.LittleEndian.PutUint32(pdb[directory+8:], 28)
	binary.LittleEndian.PutUint32(pdb[directory+12:], 4)
	information := 4 * blockSize
	binary.LittleEndian.PutUint32(pdb[information:], 20000404)
	binary.LittleEndian.PutUint32(pdb[information+4:], 0x12345678)
	binary.LittleEndian.PutUint32(pdb[information+8:], 7)
	copy(pdb[information+12:], testPDBGUID[:])
	return pdb
}
