package pecoff

import (
	"bytes"
	"debug/pe"
	"encoding/binary"
	"fmt"
)

const (
	runtimeFunctionBytes = 12
	unwindHeaderBytes    = 4

	unwindFlagExceptionHandler = 1
	unwindFlagUnwindHandler    = 2
	unwindFlagChainInfo        = 4
)

// UnwindReport records bounded Microsoft x64 exception-directory structure.
type UnwindReport struct {
	RuntimeFunctions   int
	UniqueUnwindInfos  int
	HandlerInfos       int
	ChainedInfos       int
	Version2Infos      int
	MaximumUnwindCodes int
}

// InspectAMD64Unwind validates the linked image's Microsoft x64 exception table
// and the bounded UNWIND_INFO records it references.
func InspectAMD64Unwind(image []byte) (UnwindReport, error) {
	executable, err := InspectExecutable(image)
	if err != nil {
		return UnwindReport{}, err
	}
	file, err := pe.NewFile(bytes.NewReader(image))
	if err != nil {
		return UnwindReport{}, fmt.Errorf("parse PE unwind image: %w", err)
	}
	defer file.Close()
	optional, ok := file.OptionalHeader.(*pe.OptionalHeader64)
	if !ok || optional == nil {
		return UnwindReport{}, fmt.Errorf("PE unwind image has no PE32+ header")
	}
	directory := optional.DataDirectory[exceptionDirectoryIndex]
	if directory.VirtualAddress == 0 || directory.Size == 0 || directory.Size%runtimeFunctionBytes != 0 {
		return UnwindReport{}, fmt.Errorf("PE unwind exception directory has invalid RVA/size %#x/%d", directory.VirtualAddress, directory.Size)
	}
	directoryOffset, directorySection, err := mapImageRVA(executable, directory.VirtualAddress, directory.Size)
	if err != nil {
		return UnwindReport{}, fmt.Errorf("map PE unwind exception directory: %w", err)
	}
	if directorySection.Name != ".pdata" || directorySection.Characteristics&sectionReadable == 0 || directorySection.Characteristics&sectionExecutable != 0 {
		return UnwindReport{}, fmt.Errorf("PE unwind exception directory is not readable non-executable .pdata")
	}
	count := int(directory.Size / runtimeFunctionBytes)
	report := UnwindReport{RuntimeFunctions: count}
	seenUnwind := make(map[uint32]struct{}, count)
	previousBegin := uint32(0)
	previousEnd := uint32(0)
	for index := 0; index < count; index++ {
		entryOffset := directoryOffset + index*runtimeFunctionBytes
		begin := binary.LittleEndian.Uint32(image[entryOffset:])
		end := binary.LittleEndian.Uint32(image[entryOffset+4:])
		unwindRVA := binary.LittleEndian.Uint32(image[entryOffset+8:])
		if begin == 0 || begin >= end || unwindRVA == 0 || unwindRVA%4 != 0 {
			return UnwindReport{}, fmt.Errorf("PE unwind runtime function %d has invalid begin/end/info %#x/%#x/%#x", index, begin, end, unwindRVA)
		}
		if index > 0 && (begin < previousBegin || begin < previousEnd) {
			return UnwindReport{}, fmt.Errorf("PE unwind runtime function %d is unsorted or overlapping", index)
		}
		if err := requireExecutableRVARange(executable, begin, end); err != nil {
			return UnwindReport{}, fmt.Errorf("PE unwind runtime function %d: %w", index, err)
		}
		previousBegin = begin
		previousEnd = end
		if _, duplicate := seenUnwind[unwindRVA]; duplicate {
			continue
		}
		seenUnwind[unwindRVA] = struct{}{}
		codes, flags, version, err := inspectUnwindInfo(image, executable, unwindRVA, end-begin)
		if err != nil {
			return UnwindReport{}, fmt.Errorf("PE unwind runtime function %d: %w", index, err)
		}
		report.UniqueUnwindInfos++
		if flags&(unwindFlagExceptionHandler|unwindFlagUnwindHandler) != 0 {
			report.HandlerInfos++
		}
		if flags&unwindFlagChainInfo != 0 {
			report.ChainedInfos++
		}
		if version == 2 {
			report.Version2Infos++
		}
		if codes > report.MaximumUnwindCodes {
			report.MaximumUnwindCodes = codes
		}
	}
	return report, nil
}

func inspectUnwindInfo(image []byte, executable Report, rva, functionSize uint32) (int, uint8, uint8, error) {
	headerOffset, section, err := mapImageRVA(executable, rva, unwindHeaderBytes)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("map UNWIND_INFO %#x: %w", rva, err)
	}
	if section.Characteristics&sectionReadable == 0 || section.Characteristics&sectionExecutable != 0 {
		return 0, 0, 0, fmt.Errorf("UNWIND_INFO %#x is not in readable non-executable data", rva)
	}
	versionAndFlags := image[headerOffset]
	version := versionAndFlags & 0x07
	flags := versionAndFlags >> 3
	prologSize := image[headerOffset+1]
	codeCount := int(image[headerOffset+2])
	frame := image[headerOffset+3]
	if version != 1 && version != 2 || flags > 7 || flags&unwindFlagChainInfo != 0 && flags&(unwindFlagExceptionHandler|unwindFlagUnwindHandler) != 0 {
		return 0, 0, 0, fmt.Errorf("UNWIND_INFO %#x has invalid version/flags %d/%#x", rva, version, flags)
	}
	if uint32(prologSize) > functionSize || frame&0x0f == 0 && frame>>4 != 0 {
		return 0, 0, 0, fmt.Errorf("UNWIND_INFO %#x has invalid prolog/frame data", rva)
	}
	alignedSlots := codeCount
	if alignedSlots%2 != 0 {
		alignedSlots++
	}
	minimumBytes := unwindHeaderBytes + alignedSlots*2
	if flags&(unwindFlagExceptionHandler|unwindFlagUnwindHandler) != 0 {
		minimumBytes += 4
	}
	if flags&unwindFlagChainInfo != 0 {
		minimumBytes += runtimeFunctionBytes
	}
	recordOffset, _, err := mapImageRVA(executable, rva, uint32(minimumBytes))
	if err != nil {
		return 0, 0, 0, fmt.Errorf("map complete UNWIND_INFO %#x: %w", rva, err)
	}
	previousCodeOffset := 256
	epilogSeen := false
	for slot := 0; slot < codeCount; {
		codeOffset := int(image[recordOffset+unwindHeaderBytes+slot*2])
		opAndInfo := image[recordOffset+unwindHeaderBytes+slot*2+1]
		op := opAndInfo & 0x0f
		opInfo := opAndInfo >> 4
		if version == 2 && op == 6 {
			consumed := 1
			if !epilogSeen && opInfo&1 == 0 {
				consumed = 2
			}
			if slot+consumed > codeCount {
				return 0, 0, 0, fmt.Errorf("UNWIND_INFO %#x has a truncated version-2 epilog descriptor", rva)
			}
			epilogSeen = true
			slot += consumed
			continue
		}
		if version == 2 && op == 7 {
			slot++
			continue
		}
		if codeOffset > int(prologSize) || codeOffset > previousCodeOffset {
			return 0, 0, 0, fmt.Errorf("UNWIND_INFO %#x has unsorted code offset %d", rva, codeOffset)
		}
		consumed, err := unwindOperationSlots(op, opInfo)
		if err != nil || slot+consumed > codeCount {
			return 0, 0, 0, fmt.Errorf("UNWIND_INFO %#x has invalid unwind operation %#x/%#x", rva, op, opInfo)
		}
		previousCodeOffset = codeOffset
		slot += consumed
	}
	tail := recordOffset + unwindHeaderBytes + alignedSlots*2
	if flags&(unwindFlagExceptionHandler|unwindFlagUnwindHandler) != 0 {
		handlerRVA := binary.LittleEndian.Uint32(image[tail:])
		if handlerRVA == 0 || requireExecutableRVARange(executable, handlerRVA, handlerRVA+1) != nil {
			return 0, 0, 0, fmt.Errorf("UNWIND_INFO %#x has invalid handler RVA %#x", rva, handlerRVA)
		}
	}
	if flags&unwindFlagChainInfo != 0 {
		begin := binary.LittleEndian.Uint32(image[tail:])
		end := binary.LittleEndian.Uint32(image[tail+4:])
		chainedUnwind := binary.LittleEndian.Uint32(image[tail+8:])
		if begin == 0 || begin >= end || chainedUnwind == 0 || chainedUnwind%4 != 0 || requireExecutableRVARange(executable, begin, end) != nil {
			return 0, 0, 0, fmt.Errorf("UNWIND_INFO %#x has invalid chained runtime function", rva)
		}
	}
	return codeCount, flags, version, nil
}

func unwindOperationSlots(operation, operationInfo uint8) (int, error) {
	switch operation {
	case 0, 2, 3:
		return 1, nil
	case 1:
		switch operationInfo {
		case 0:
			return 2, nil
		case 1:
			return 3, nil
		default:
			return 0, fmt.Errorf("UWOP_ALLOC_LARGE has invalid operation info")
		}
	case 4, 8:
		return 2, nil
	case 5, 9:
		return 3, nil
	case 10:
		if operationInfo > 1 {
			return 0, fmt.Errorf("UWOP_PUSH_MACHFRAME has invalid operation info")
		}
		return 1, nil
	default:
		return 0, fmt.Errorf("unsupported unwind operation")
	}
}

func mapImageRVA(executable Report, rva, size uint32) (int, Section, error) {
	if size == 0 {
		return 0, Section{}, fmt.Errorf("RVA %#x has zero size", rva)
	}
	end := uint64(rva) + uint64(size)
	for _, section := range executable.Sections {
		sectionStart := uint64(section.VirtualAddress)
		sectionRawEnd := sectionStart + uint64(section.RawSize)
		if uint64(rva) < sectionStart || end > sectionRawEnd {
			continue
		}
		offset := uint64(section.RawOffset) + uint64(rva-section.VirtualAddress)
		if offset+uint64(size) > uint64(maxExecutableBytes) {
			return 0, Section{}, fmt.Errorf("RVA %#x maps beyond the bounded image", rva)
		}
		return int(offset), section, nil
	}
	return 0, Section{}, fmt.Errorf("RVA range %#x..%#x is not file-backed", rva, end)
}

func requireExecutableRVARange(executable Report, begin, end uint32) error {
	if begin >= end {
		return fmt.Errorf("executable RVA range %#x..%#x is empty", begin, end)
	}
	for _, section := range executable.Sections {
		sectionStart := uint64(section.VirtualAddress)
		sectionEnd := sectionStart + uint64(max(section.VirtualSize, section.RawSize))
		if uint64(begin) >= sectionStart && uint64(end) <= sectionEnd && section.Characteristics&sectionContainsCode != 0 && section.Characteristics&sectionExecutable != 0 {
			return nil
		}
	}
	return fmt.Errorf("RVA range %#x..%#x is not executable code", begin, end)
}
