package pecoff

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"strconv"
)

// ErrNameBudgetExceeded identifies decoded names exceeding the parser budget.
var ErrNameBudgetExceeded = errors.New("PE/COFF name byte budget exceeded")

// VerifyObjectParserBudget bounds allocations in debug/pe for ordinary COFF,
// including data-only fallback callers. It does not replace structural validation.
func VerifyObjectParserBudget(object []byte) error {
	if len(object) < coffFileHeaderBytes || len(object) > maxExecutableBytes {
		return fmt.Errorf("COFF object has invalid size %d", len(object))
	}
	if object[0] == 'M' && object[1] == 'Z' {
		return fmt.Errorf("COFF object must not be a linked PE image")
	}
	symbolCount := binary.LittleEndian.Uint32(object[12:16])
	if symbolCount > maxObjectSymbols {
		return fmt.Errorf("COFF object has invalid symbol count %d", symbolCount)
	}
	if err := verifyCOFFRelocationBudget(object); err != nil {
		return err
	}
	return verifyCOFFNameBudget(object, 0)
}

// debug/pe copies names for each symbol and section reference, even when
// thousands of references share one on-disk name. Bound that expansion before
// calling its parser. Structural validation and malformed-name errors stay there.
func verifyCOFFNameBudget(data []byte, headerOffset int) error {
	header := data[headerOffset : headerOffset+coffFileHeaderBytes]
	symbolStart := uint64(binary.LittleEndian.Uint32(header[8:]))
	symbolCount := binary.LittleEndian.Uint32(header[12:])
	if symbolStart == 0 {
		return nil // debug/pe does not load a string table without this pointer.
	}
	stringStart := symbolStart + uint64(symbolCount)*18
	if stringStart+4 > uint64(len(data)) {
		return nil // The string-table read fails before names are materialized.
	}
	stringSize := uint64(binary.LittleEndian.Uint32(data[stringStart:]))
	var table []byte
	if stringSize > 4 {
		if stringStart+stringSize > uint64(len(data)) {
			return nil // debug/pe rejects the truncated string table first.
		}
		table = data[stringStart+4 : stringStart+stringSize]
	}
	remaining := maxExecutableBytes
	charge := func(raw []byte, section bool) error {
		size := coffExpandedNameBytes(raw, table, section)
		if size > remaining {
			return fmt.Errorf("%w: limit %d bytes", ErrNameBudgetExceeded, maxExecutableBytes)
		}
		remaining -= size
		return nil
	}
	for index := uint64(0); index < uint64(symbolCount); {
		symbol := data[symbolStart+index*18:]
		if err := charge(symbol[:8], false); err != nil {
			return err
		}
		index += 1 + uint64(symbol[17]) // Auxiliary records have no primary name.
	}
	sections := int(binary.LittleEndian.Uint16(header[2:]))
	sectionStart := headerOffset + coffFileHeaderBytes + int(binary.LittleEndian.Uint16(header[16:]))
	for index := 0; index < sections; index++ {
		offset := sectionStart + index*coffSectionHeaderBytes
		if offset+coffSectionHeaderBytes > len(data) {
			break // Still budget complete headers before a later truncated one.
		}
		if err := charge(data[offset:offset+8], true); err != nil {
			return err
		}
	}
	return nil
}

func coffExpandedNameBytes(raw, table []byte, section bool) int {
	var offset uint32
	long := false
	if section && raw[0] == '/' {
		// Match debug/pe's Atoi, including an optional leading plus sign.
		value, err := strconv.Atoi(string(raw[1:coffBudgetCStringBytes(raw)]))
		if err != nil {
			return 0
		}
		offset, long = uint32(value), true
	} else if !section && binary.LittleEndian.Uint32(raw) == 0 {
		offset = binary.LittleEndian.Uint32(raw[4:])
		long = offset != 0
	}
	if long {
		if offset < 4 || uint64(offset-4) > uint64(len(table)) {
			return 0 // The parser rejects this reference without copying its name.
		}
		raw = table[offset-4:]
	}
	return coffBudgetCStringBytes(raw)
}

func coffBudgetCStringBytes(raw []byte) int {
	if end := bytes.IndexByte(raw, 0); end >= 0 {
		return end
	}
	return len(raw)
}
