package pecoff

import (
	"bytes"
	"debug/pe"
	"encoding/binary"
	"fmt"
	"math"
)

const (
	debugDirectoryIndex    = 6
	imageDebugDirectoryLen = 28
	imageDebugTypeCodeView = 2
	maxPDBInformationBytes = 4 * 1024 * 1024
	pdbNilStreamSize       = math.MaxUint32
)

var msf7Magic = []byte("Microsoft C/C++ MSF 7.00\r\n\x1aDS\x00\x00\x00")

// ProgramDatabaseReport binds one final PE RSDS record to the identity in PDB stream 1.
type ProgramDatabaseReport struct {
	GUID          [16]byte
	Age           uint32
	Path          string
	PDBVersion    uint32
	PDBSignature  uint32
	PDBBlockSize  uint32
	PDBStreamSize uint32
}

// InspectProgramDatabaseAssociation validates the final image's unique RSDS record and requires
// its GUID and age to match the bounded MSF 7 PDB information stream.
func InspectProgramDatabaseAssociation(image, pdb []byte) (ProgramDatabaseReport, error) {
	imageReport, err := InspectExecutable(image)
	if err != nil {
		return ProgramDatabaseReport{}, fmt.Errorf("inspect PDB-associated image: %w", err)
	}
	file, err := pe.NewFile(bytes.NewReader(image))
	if err != nil {
		return ProgramDatabaseReport{}, fmt.Errorf("reparse PDB-associated image: %w", err)
	}
	defer file.Close()
	optional, ok := file.OptionalHeader.(*pe.OptionalHeader64)
	if !ok || len(optional.DataDirectory) <= debugDirectoryIndex {
		return ProgramDatabaseReport{}, fmt.Errorf("PDB-associated image has no PE32+ debug directory slot")
	}
	directory := optional.DataDirectory[debugDirectoryIndex]
	if directory.VirtualAddress == 0 || directory.Size == 0 || directory.Size%imageDebugDirectoryLen != 0 {
		return ProgramDatabaseReport{}, fmt.Errorf("PDB-associated image has an invalid debug directory")
	}
	directoryOffset, _, err := mapImageRVA(imageReport, directory.VirtualAddress, directory.Size)
	if err != nil {
		return ProgramDatabaseReport{}, fmt.Errorf("map PE debug directory: %w", err)
	}
	var rsds []byte
	for offset := directoryOffset; offset < directoryOffset+int(directory.Size); offset += imageDebugDirectoryLen {
		entry := image[offset : offset+imageDebugDirectoryLen]
		if binary.LittleEndian.Uint32(entry[12:]) != imageDebugTypeCodeView {
			continue
		}
		size := binary.LittleEndian.Uint32(entry[16:])
		address := binary.LittleEndian.Uint32(entry[20:])
		pointer := binary.LittleEndian.Uint32(entry[24:])
		mapped, _, mapErr := mapImageRVA(imageReport, address, size)
		if mapErr != nil || uint64(pointer)+uint64(size) > uint64(len(image)) || mapped != int(pointer) {
			return ProgramDatabaseReport{}, fmt.Errorf("PE CodeView debug data has inconsistent RVA and file offsets")
		}
		if rsds != nil {
			return ProgramDatabaseReport{}, fmt.Errorf("PDB-associated image repeats its CodeView debug entry")
		}
		rsds = image[int(pointer):int(uint64(pointer)+uint64(size))]
	}
	if len(rsds) < 25 || !bytes.Equal(rsds[:4], []byte("RSDS")) {
		return ProgramDatabaseReport{}, fmt.Errorf("PDB-associated image has no bounded RSDS record")
	}
	terminator := bytes.IndexByte(rsds[24:], 0)
	if terminator < 1 || 24+terminator+1 != len(rsds) {
		return ProgramDatabaseReport{}, fmt.Errorf("PE RSDS path is missing, unterminated, or has trailing bytes")
	}
	path := string(rsds[24 : 24+terminator])
	if path != "program.pdb" {
		return ProgramDatabaseReport{}, fmt.Errorf("PE RSDS path %q is not the contained program.pdb identity", path)
	}
	var guid [16]byte
	copy(guid[:], rsds[4:20])
	age := binary.LittleEndian.Uint32(rsds[20:24])
	if guid == ([16]byte{}) || age == 0 {
		return ProgramDatabaseReport{}, fmt.Errorf("PE RSDS identity has an empty GUID or age")
	}
	pdbIdentity, err := inspectPDBInformationStream(pdb)
	if err != nil {
		return ProgramDatabaseReport{}, err
	}
	if guid != pdbIdentity.GUID || age != pdbIdentity.Age {
		return ProgramDatabaseReport{}, fmt.Errorf("PE RSDS and PDB stream identities do not match")
	}
	return ProgramDatabaseReport{
		GUID: guid, Age: age, Path: path, PDBVersion: pdbIdentity.Version,
		PDBSignature: pdbIdentity.Signature, PDBBlockSize: pdbIdentity.BlockSize,
		PDBStreamSize: pdbIdentity.StreamSize,
	}, nil
}

type pdbInformationIdentity struct {
	GUID       [16]byte
	Age        uint32
	Version    uint32
	Signature  uint32
	BlockSize  uint32
	StreamSize uint32
}

func inspectPDBInformationStream(pdb []byte) (pdbInformationIdentity, error) {
	if len(pdb) < 56 || len(pdb) > maxExecutableBytes || !bytes.Equal(pdb[:len(msf7Magic)], msf7Magic) {
		return pdbInformationIdentity{}, fmt.Errorf("program database is not a bounded MSF 7 file")
	}
	blockSize := binary.LittleEndian.Uint32(pdb[32:])
	numBlocks := binary.LittleEndian.Uint32(pdb[40:])
	directoryBytes := binary.LittleEndian.Uint32(pdb[44:])
	blockMap := binary.LittleEndian.Uint32(pdb[52:])
	if blockSize < 512 || blockSize > 64*1024 || blockSize&(blockSize-1) != 0 || numBlocks == 0 ||
		uint64(numBlocks)*uint64(blockSize) != uint64(len(pdb)) || directoryBytes < 12 || directoryBytes > uint32(len(pdb)) || blockMap >= numBlocks {
		return pdbInformationIdentity{}, fmt.Errorf("program database has an invalid MSF 7 superblock")
	}
	directoryBlockCount64 := (uint64(directoryBytes) + uint64(blockSize) - 1) / uint64(blockSize)
	if directoryBlockCount64 > math.MaxUint32 || directoryBlockCount64*4 > uint64(blockSize) {
		return pdbInformationIdentity{}, fmt.Errorf("program database uses an unsupported multi-block directory map")
	}
	directoryBlockCount := uint32(directoryBlockCount64)
	blockMapOffset := uint64(blockMap) * uint64(blockSize)
	directory := make([]byte, 0, uint64(directoryBlockCount)*uint64(blockSize))
	usedBlocks := map[uint32]string{0: "superblock", blockMap: "directory block map"}
	for index := uint32(0); index < directoryBlockCount; index++ {
		entry := blockMapOffset + uint64(index)*4
		block := binary.LittleEndian.Uint32(pdb[entry:])
		if block >= numBlocks {
			return pdbInformationIdentity{}, fmt.Errorf("program database directory references block %d outside the file", block)
		}
		if owner, duplicate := usedBlocks[block]; duplicate {
			return pdbInformationIdentity{}, fmt.Errorf("program database directory reuses %s block %d", owner, block)
		}
		usedBlocks[block] = "stream directory"
		start := uint64(block) * uint64(blockSize)
		directory = append(directory, pdb[start:start+uint64(blockSize)]...)
	}
	directory = directory[:directoryBytes]
	if len(directory) < 4 {
		return pdbInformationIdentity{}, fmt.Errorf("program database stream directory is truncated")
	}
	streamCount := binary.LittleEndian.Uint32(directory)
	if streamCount < 2 || streamCount > 1<<16 || uint64(4+streamCount*4) > uint64(len(directory)) {
		return pdbInformationIdentity{}, fmt.Errorf("program database has an invalid stream count %d", streamCount)
	}
	sizes := make([]uint32, streamCount)
	position := uint64(4)
	for index := range sizes {
		sizes[index] = binary.LittleEndian.Uint32(directory[position:])
		position += 4
	}
	var information []byte
	for streamIndex, size := range sizes {
		if size == pdbNilStreamSize || size == 0 {
			continue
		}
		blocks64 := (uint64(size) + uint64(blockSize) - 1) / uint64(blockSize)
		if streamIndex == 1 && size > maxPDBInformationBytes {
			return pdbInformationIdentity{}, fmt.Errorf("program database information stream exceeds %d bytes", maxPDBInformationBytes)
		}
		if blocks64 > math.MaxUint32 || position+blocks64*4 > uint64(len(directory)) {
			return pdbInformationIdentity{}, fmt.Errorf("program database stream %d block list is truncated", streamIndex)
		}
		blocks := uint32(blocks64)
		var stream []byte
		if streamIndex == 1 {
			stream = make([]byte, 0, uint64(blocks)*uint64(blockSize))
		}
		for blockIndex := uint32(0); blockIndex < blocks; blockIndex++ {
			block := binary.LittleEndian.Uint32(directory[position:])
			position += 4
			if block >= numBlocks {
				return pdbInformationIdentity{}, fmt.Errorf("program database stream %d references block %d outside the file", streamIndex, block)
			}
			if owner, duplicate := usedBlocks[block]; duplicate {
				return pdbInformationIdentity{}, fmt.Errorf("program database stream %d reuses %s block %d", streamIndex, owner, block)
			}
			usedBlocks[block] = fmt.Sprintf("stream %d", streamIndex)
			if streamIndex == 1 {
				start := uint64(block) * uint64(blockSize)
				stream = append(stream, pdb[start:start+uint64(blockSize)]...)
			}
		}
		if streamIndex == 1 {
			information = stream[:size]
		}
	}
	if position != uint64(len(directory)) || len(information) < 28 {
		return pdbInformationIdentity{}, fmt.Errorf("program database has an invalid information stream")
	}
	var guid [16]byte
	copy(guid[:], information[12:28])
	return pdbInformationIdentity{
		Version: binary.LittleEndian.Uint32(information), Signature: binary.LittleEndian.Uint32(information[4:]),
		Age: binary.LittleEndian.Uint32(information[8:]), GUID: guid,
		BlockSize: blockSize, StreamSize: sizes[1],
	}, nil
}
