// Package pecoff validates bounded Windows x64 PE executable structure without
// treating a local artifact as target conformance evidence.
package pecoff

import (
	"bytes"
	"debug/pe"
	"fmt"
)

const (
	maxExecutableBytes = 64 * 1024 * 1024

	machineAMD64               = 0x8664
	executableImage            = 0x0002
	largeAddressAware          = 0x0020
	dynamicLibrary             = 0x2000
	optionalHeader64Magic      = 0x020b
	windowsConsoleSubsystem    = 3
	sectionContainsCode        = 0x00000020
	sectionUninitializedData   = 0x00000080
	sectionExecutable          = 0x20000000
	sectionReadable            = 0x40000000
	sectionWritable            = 0x80000000
	importDirectoryIndex       = 1
	exceptionDirectoryIndex    = 3
	minimumPEFileAlignment     = 512
	maximumPEFileAlignment     = 64 * 1024
	minimumWindowsImageBaseGap = 64 * 1024
	maxSections                = 96
)

// Section is one bounded executable-image section receipt.
type Section struct {
	Name            string
	VirtualAddress  uint32
	VirtualSize     uint32
	RawOffset       uint32
	RawSize         uint32
	Characteristics uint32
}

// Report records the closed PE facts consumed by the Windows host smoke.
// It is internal evidence, not a stable serialized artifact format.
type Report struct {
	Machine                   uint16
	Characteristics           uint16
	ImageBase                 uint64
	EntryPointRVA             uint32
	Subsystem                 uint16
	SectionAlignment          uint32
	FileAlignment             uint32
	SizeOfImage               uint32
	SizeOfHeaders             uint32
	Sections                  []Section
	ImportDirectoryPresent    bool
	ExceptionDirectoryPresent bool
}

// InspectExecutable parses and validates one bounded AMD64 PE32+ console image.
func InspectExecutable(image []byte) (Report, error) {
	if len(image) < 64 || len(image) > maxExecutableBytes {
		return Report{}, fmt.Errorf("PE executable has invalid size %d", len(image))
	}
	if image[0] != 'M' || image[1] != 'Z' {
		return Report{}, fmt.Errorf("PE executable is missing the DOS MZ signature")
	}
	file, err := pe.NewFile(bytes.NewReader(image))
	if err != nil {
		return Report{}, fmt.Errorf("parse PE executable: %w", err)
	}
	defer file.Close()
	if file.FileHeader.Machine != machineAMD64 {
		return Report{}, fmt.Errorf("PE executable machine is %#x, want AMD64", file.FileHeader.Machine)
	}
	if file.FileHeader.Characteristics&executableImage == 0 || file.FileHeader.Characteristics&largeAddressAware == 0 || file.FileHeader.Characteristics&dynamicLibrary != 0 {
		return Report{}, fmt.Errorf("PE executable has invalid image characteristics %#x", file.FileHeader.Characteristics)
	}
	optional, ok := file.OptionalHeader.(*pe.OptionalHeader64)
	if !ok || optional == nil || optional.Magic != optionalHeader64Magic {
		return Report{}, fmt.Errorf("PE executable is not PE32+")
	}
	if optional.Subsystem != windowsConsoleSubsystem {
		return Report{}, fmt.Errorf("PE executable subsystem is %d, want Windows console", optional.Subsystem)
	}
	if optional.AddressOfEntryPoint == 0 {
		return Report{}, fmt.Errorf("PE executable has no entry point")
	}
	if optional.ImageBase == 0 || optional.ImageBase%minimumWindowsImageBaseGap != 0 {
		return Report{}, fmt.Errorf("PE executable image base %#x is not 64 KiB aligned", optional.ImageBase)
	}
	if !validFileAlignment(optional.FileAlignment) || !powerOfTwo(optional.SectionAlignment) || optional.SectionAlignment < optional.FileAlignment {
		return Report{}, fmt.Errorf("PE executable has invalid section/file alignment %d/%d", optional.SectionAlignment, optional.FileAlignment)
	}
	if optional.SizeOfHeaders == 0 || optional.SizeOfHeaders%optional.FileAlignment != 0 || uint64(optional.SizeOfHeaders) > uint64(len(image)) {
		return Report{}, fmt.Errorf("PE executable has invalid header size %d", optional.SizeOfHeaders)
	}
	if optional.SizeOfImage == 0 || optional.SizeOfImage%optional.SectionAlignment != 0 {
		return Report{}, fmt.Errorf("PE executable has invalid image size %d", optional.SizeOfImage)
	}
	if len(file.Sections) == 0 || len(file.Sections) > maxSections {
		return Report{}, fmt.Errorf("PE executable has invalid section count %d", len(file.Sections))
	}

	report := Report{
		Machine:                   file.FileHeader.Machine,
		Characteristics:           file.FileHeader.Characteristics,
		ImageBase:                 optional.ImageBase,
		EntryPointRVA:             optional.AddressOfEntryPoint,
		Subsystem:                 optional.Subsystem,
		SectionAlignment:          optional.SectionAlignment,
		FileAlignment:             optional.FileAlignment,
		SizeOfImage:               optional.SizeOfImage,
		SizeOfHeaders:             optional.SizeOfHeaders,
		Sections:                  make([]Section, 0, len(file.Sections)),
		ImportDirectoryPresent:    directoryPresent(optional.DataDirectory[importDirectoryIndex]),
		ExceptionDirectoryPresent: directoryPresent(optional.DataDirectory[exceptionDirectoryIndex]),
	}
	names := make(map[string]struct{}, len(file.Sections))
	entrySectionFound := false
	textSectionFound := false
	previousVirtualAddress := uint32(0)
	previousVirtualEnd := uint64(0)
	for index, section := range file.Sections {
		if section.Name == "" {
			return Report{}, fmt.Errorf("PE executable section %d has no name", index)
		}
		if _, duplicate := names[section.Name]; duplicate {
			return Report{}, fmt.Errorf("PE executable repeats section name %q", section.Name)
		}
		names[section.Name] = struct{}{}
		if section.VirtualAddress == 0 || section.VirtualAddress%optional.SectionAlignment != 0 || index > 0 && section.VirtualAddress <= previousVirtualAddress {
			return Report{}, fmt.Errorf("PE executable section %q has invalid virtual address %#x", section.Name, section.VirtualAddress)
		}
		previousVirtualAddress = section.VirtualAddress
		if section.Size > 0 {
			if section.Offset < optional.SizeOfHeaders || section.Offset%optional.FileAlignment != 0 || section.Size%optional.FileAlignment != 0 || uint64(section.Offset)+uint64(section.Size) > uint64(len(image)) {
				return Report{}, fmt.Errorf("PE executable section %q has an invalid raw range", section.Name)
			}
		}
		virtualSpan := max(section.VirtualSize, section.Size)
		virtualEnd := uint64(section.VirtualAddress) + uint64(virtualSpan)
		if virtualSpan == 0 || virtualEnd > uint64(optional.SizeOfImage) {
			return Report{}, fmt.Errorf("PE executable section %q has an invalid virtual range", section.Name)
		}
		if index > 0 && uint64(section.VirtualAddress) < previousVirtualEnd {
			return Report{}, fmt.Errorf("PE executable section %q overlaps the previous virtual range", section.Name)
		}
		previousVirtualEnd = virtualEnd
		containsCode := section.Characteristics&sectionContainsCode != 0
		executable := section.Characteristics&sectionExecutable != 0
		readable := section.Characteristics&sectionReadable != 0
		if section.Name == ".text" {
			if !containsCode || !executable || !readable {
				return Report{}, fmt.Errorf("PE executable .text section is not readable executable code")
			}
			textSectionFound = true
		}
		entry := optional.AddressOfEntryPoint
		if uint64(entry) >= uint64(section.VirtualAddress) && uint64(entry) < virtualEnd {
			if !containsCode || !executable {
				return Report{}, fmt.Errorf("PE executable entry point is not in executable code")
			}
			entrySectionFound = true
		}
		report.Sections = append(report.Sections, Section{
			Name:            section.Name,
			VirtualAddress:  section.VirtualAddress,
			VirtualSize:     section.VirtualSize,
			RawOffset:       section.Offset,
			RawSize:         section.Size,
			Characteristics: section.Characteristics,
		})
	}
	if !textSectionFound {
		return Report{}, fmt.Errorf("PE executable has no .text section")
	}
	if !entrySectionFound {
		return Report{}, fmt.Errorf("PE executable entry point is outside executable sections")
	}
	return report, nil
}

func validFileAlignment(value uint32) bool {
	return powerOfTwo(value) && value >= minimumPEFileAlignment && value <= maximumPEFileAlignment
}

func powerOfTwo(value uint32) bool {
	return value != 0 && value&(value-1) == 0
}

func directoryPresent(value pe.DataDirectory) bool {
	return value.VirtualAddress != 0 && value.Size != 0
}
