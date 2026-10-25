package pecoff

import (
	"encoding/binary"
	"fmt"
	"strings"
	"testing"
)

func TestPDBStreamStillChecksUnselectedBlockOwners(t *testing.T) {
	const blocks = 16
	for _, test := range []struct {
		name  string
		block uint32
		want  string
	}{
		{"superblock", 0, "reuses superblock"},
		{"block map", 2, "reuses directory block map"},
		{"directory", 3, "reuses stream directory"},
		{"selected stream", 4, "reuses stream 1"},
		{"same stream", 5, "reuses stream 2"},
	} {
		t.Run(test.name, func(t *testing.T) {
			pdb := manyBlockProgramDatabase(blocks)
			// The second block of unselected stream 2 must not alias any owner.
			binary.LittleEndian.PutUint32(pdb[3*1024+24:], test.block)
			if _, _, err := readPDBStream(pdb, 1); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("unselected stream alias: %v, want %q", err, test.want)
			}
		})
	}
	t.Run("stream zero", func(t *testing.T) {
		pdb := manyBlockProgramDatabase(blocks)
		directory := pdb[3*1024:]
		// Transfer one block from stream 2 to stream 0, retaining the list length.
		binary.LittleEndian.PutUint32(directory[4:], 1024)
		binary.LittleEndian.PutUint32(directory[12:], (blocks-1)*1024)
		binary.LittleEndian.PutUint32(directory[24:], 4)
		if _, _, err := readPDBStream(pdb, 1); err == nil || !strings.Contains(err.Error(), "reuses stream 0") {
			t.Fatalf("stream zero alias: %v", err)
		}
	})
}

func BenchmarkReadPDBStreamBlockOwnership(b *testing.B) {
	for _, blocks := range []int{8192, 32768} {
		b.Run(fmt.Sprint(blocks), func(b *testing.B) {
			pdb := manyBlockProgramDatabase(blocks)
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				selected, blockSize, err := readPDBStream(pdb, 1)
				if err != nil || len(selected) != 28 || blockSize != 1024 {
					b.Fatalf("PDB stream inspection: bytes=%d blockSize=%d err=%v", len(selected), blockSize, err)
				}
			}
		})
	}
}

func manyBlockProgramDatabase(blocks int) []byte {
	const blockSize = 1024
	directoryBytes := 20 + blocks*4
	directoryBlocks := (directoryBytes + blockSize - 1) / blockSize
	infoBlock := 3 + directoryBlocks
	dataBlock := infoBlock + 1
	pdb := make([]byte, (dataBlock+blocks)*blockSize)
	copy(pdb, msf7Magic)
	binary.LittleEndian.PutUint32(pdb[32:], blockSize)
	binary.LittleEndian.PutUint32(pdb[36:], 1)
	binary.LittleEndian.PutUint32(pdb[40:], uint32(dataBlock+blocks))
	binary.LittleEndian.PutUint32(pdb[44:], uint32(directoryBytes))
	binary.LittleEndian.PutUint32(pdb[52:], 2)
	for index := range directoryBlocks {
		binary.LittleEndian.PutUint32(pdb[2*blockSize+index*4:], uint32(3+index))
	}
	directory := pdb[3*blockSize:]
	binary.LittleEndian.PutUint32(directory, 3)
	binary.LittleEndian.PutUint32(directory[4:], pdbNilStreamSize)
	binary.LittleEndian.PutUint32(directory[8:], 28)
	binary.LittleEndian.PutUint32(directory[12:], uint32(blocks*blockSize))
	binary.LittleEndian.PutUint32(directory[16:], uint32(infoBlock))
	for index := range blocks {
		binary.LittleEndian.PutUint32(directory[20+index*4:], uint32(dataBlock+index))
	}
	return pdb
}
