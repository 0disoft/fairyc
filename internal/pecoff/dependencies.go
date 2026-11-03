package pecoff

import (
	"bytes"
	"debug/pe"
	"encoding/binary"
	"fmt"
	"math"
	"strings"
)

type ImportedFunction struct {
	Name    string `json:"name,omitempty"`
	Ordinal uint16 `json:"ordinal,omitempty"`
	IATRVA  uint32 `json:"iatRVA"`
}

type ImportedLibrary struct {
	Name      string             `json:"name"`
	Functions []ImportedFunction `json:"functions"`
}

type DependencyReport struct {
	Libraries []ImportedLibrary `json:"libraries"`
}

// InspectAMD64Dependencies inventories bounded ordinary PE imports. Delay and
// bound import tables are rejected explicitly until a real corpus needs them.
func InspectAMD64Dependencies(image []byte) (DependencyReport, error) {
	var result DependencyReport
	report, err := InspectExecutable(image)
	if err != nil {
		return result, err
	}
	file, err := pe.NewFile(bytes.NewReader(image))
	if err != nil {
		return result, err
	}
	defer file.Close()
	optional := file.OptionalHeader.(*pe.OptionalHeader64)
	for _, index := range []int{11, 13} {
		if optional.DataDirectory[index].VirtualAddress != 0 || optional.DataDirectory[index].Size != 0 {
			return result, fmt.Errorf("dependency inspection does not admit bound/delay import directory %d", index)
		}
	}
	directory := optional.DataDirectory[importDirectoryIndex]
	if directory.VirtualAddress == 0 && directory.Size == 0 {
		return result, nil
	}
	if directory.VirtualAddress == 0 || directory.Size < 20 || directory.Size > 128*20 || uint64(directory.VirtualAddress)+uint64(directory.Size) > math.MaxUint32 {
		return result, fmt.Errorf("invalid import descriptor directory")
	}
	descriptors, err := generatedImageBytes(image, report, directory.VirtualAddress, directory.Size)
	if err != nil {
		return result, err
	}
	seen := make(map[string]bool)
	total := 0
	for offset := 0; offset+20 <= len(descriptors); offset += 20 {
		entry := descriptors[offset : offset+20]
		if bytes.Equal(entry, make([]byte, 20)) {
			return result, nil
		}
		original, stamp := binary.LittleEndian.Uint32(entry), binary.LittleEndian.Uint32(entry[4:])
		nameRVA, iat := binary.LittleEndian.Uint32(entry[12:]), binary.LittleEndian.Uint32(entry[16:])
		if stamp != 0 || iat == 0 {
			return result, fmt.Errorf("bound import or absent import address table")
		}
		name, err := dependencyASCII(image, report, nameRVA, 256)
		if err != nil {
			return result, fmt.Errorf("import DLL: %w", err)
		}
		name = strings.ToLower(name)
		if !strings.HasSuffix(name, ".dll") || name == ".dll" || strings.Contains(name, "..") {
			return result, fmt.Errorf("import DLL is not a contained basename")
		}
		for _, c := range name {
			if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '_' || c == '-' || c == '.') {
				return result, fmt.Errorf("invalid import DLL basename")
			}
		}
		if seen[name] {
			return result, fmt.Errorf("duplicate import DLL %s", name)
		}
		seen[name] = true
		if original == 0 {
			original = iat
		}
		library := ImportedLibrary{Name: name}
		terminated := false
		for index := uint64(0); index < 8192 && total < 8192; index++ {
			lookupRVA, iatRVA := uint64(original)+index*8, uint64(iat)+index*8
			if lookupRVA+8 > math.MaxUint32 || iatRVA+8 > math.MaxUint32 {
				return result, fmt.Errorf("import thunk RVA overflow")
			}
			data, err := generatedImageBytes(image, report, uint32(lookupRVA), 8)
			if err != nil {
				return result, err
			}
			address, err := generatedImageBytes(image, report, uint32(iatRVA), 8)
			if err != nil {
				return result, err
			}
			value := binary.LittleEndian.Uint64(data)
			if value == 0 {
				if binary.LittleEndian.Uint64(address) != 0 {
					return result, fmt.Errorf("unterminated import address table")
				}
				terminated = true
				break
			}
			function := ImportedFunction{IATRVA: uint32(iatRVA)}
			if value&(uint64(1)<<63) != 0 {
				if value & ^(uint64(1)<<63|0xffff) != 0 || value&0xffff == 0 {
					return result, fmt.Errorf("invalid ordinal import")
				}
				function.Ordinal = uint16(value)
			} else {
				if value > math.MaxUint32-2 {
					return result, fmt.Errorf("import name RVA overflow")
				}
				if _, err := generatedImageBytes(image, report, uint32(value), 2); err != nil {
					return result, err
				}
				function.Name, err = dependencyASCII(image, report, uint32(value)+2, 1024)
				if err != nil {
					return result, err
				}
			}
			library.Functions = append(library.Functions, function)
			total++
		}
		if !terminated || len(library.Functions) == 0 {
			return result, fmt.Errorf("empty or unbounded import thunk list")
		}
		result.Libraries = append(result.Libraries, library)
	}
	return result, fmt.Errorf("import descriptor terminator missing")
}

func dependencyASCII(image []byte, report Report, rva uint32, limit uint32) (string, error) {
	if rva == 0 {
		return "", fmt.Errorf("zero import string RVA")
	}
	if name, found, err := dependencyASCIIInSection(image, report, rva, limit); found {
		return name, err
	}
	var output []byte
	for index := uint32(0); index < limit; index++ {
		if uint64(rva)+uint64(index) > math.MaxUint32 {
			return "", fmt.Errorf("import string RVA overflow")
		}
		data, err := generatedImageBytes(image, report, rva+index, 1)
		if err != nil {
			return "", err
		}
		if data[0] == 0 {
			if len(output) == 0 {
				return "", fmt.Errorf("empty import string")
			}
			return string(output), nil
		}
		if data[0] < 33 || data[0] > 126 {
			return "", fmt.Errorf("non-ASCII import string")
		}
		output = append(output, data[0])
	}
	return "", fmt.Errorf("unterminated import string")
}

// Import strings normally fit in one file-backed section. Borrow its validated
// byte range for this synchronous read, then detach only the returned string.
// Cross-section and malformed ranges retain the byte-wise reader's error order.
// Other generatedImageBytes consumers still receive owned copies for masking.
func dependencyASCIIInSection(image []byte, report Report, rva, limit uint32) (string, bool, error) {
	for _, section := range report.Sections {
		if rva < section.VirtualAddress {
			continue
		}
		offset := uint64(rva - section.VirtualAddress)
		span := uint64(min(section.RawSize, section.VirtualSize))
		if offset >= span {
			continue
		}
		start := uint64(section.RawOffset) + offset
		count := min(span-offset, uint64(limit), uint64(math.MaxUint32)-uint64(rva)+1)
		if start > uint64(len(image)) || count > uint64(len(image))-start {
			return "", false, nil
		}
		data := image[start : start+count]
		end := bytes.IndexByte(data, 0)
		if end < 0 {
			return "", false, nil
		}
		if end == 0 {
			return "", true, fmt.Errorf("empty import string")
		}
		for _, value := range data[:end] {
			if value < 33 || value > 126 {
				return "", true, fmt.Errorf("non-ASCII import string")
			}
		}
		return string(data[:end]), true, nil
	}
	return "", false, nil
}
