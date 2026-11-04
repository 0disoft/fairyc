package packagesource

import (
	"archive/zip"
	"encoding/binary"
	"errors"
)

const (
	zipDirectorySignature   = 0x02014b50
	zipEndSignature         = 0x06054b50
	zip64EndSignature       = 0x06064b50
	zip64LocatorSignature   = 0x07064b50
	zipDirectoryHeaderBytes = 46
	zipEndBytes             = 22
	zip64EndBytes           = 56
	zip64LocatorBytes       = 20
)

var errArchiveEntryLimit = errors.New("ZIP entry limit exceeded")

// admitArchiveDirectory walks framing without allocating per-entry metadata.
// archive/zip still owns all ZIP decoding and integrity checks. Its reader scans
// beyond the advertised directory size/count, so admission must do the same.
func admitArchiveDirectory(data []byte) error {
	end := -1
	for index := len(data) - zipEndBytes; index >= max(0, len(data)-65*1024); index-- {
		if binary.LittleEndian.Uint32(data[index:]) != zipEndSignature {
			continue
		}
		if int(binary.LittleEndian.Uint16(data[index+20:])) > len(data)-index-zipEndBytes {
			return zip.ErrFormat
		}
		end = index
		break
	}
	if end < 0 {
		return zip.ErrFormat
	}
	records := uint64(binary.LittleEndian.Uint16(data[end+10:]))
	size := uint64(binary.LittleEndian.Uint32(data[end+12:]))
	offset := uint64(binary.LittleEndian.Uint32(data[end+16:]))
	if records == 0xffff || size == 0xffffffff || offset == 0xffffffff {
		locator := end - zip64LocatorBytes
		if locator >= 0 && binary.LittleEndian.Uint32(data[locator:]) == zip64LocatorSignature &&
			binary.LittleEndian.Uint32(data[locator+4:]) == 0 && binary.LittleEndian.Uint32(data[locator+16:]) == 1 {
			position := binary.LittleEndian.Uint64(data[locator+8:])
			if len(data) < zip64EndBytes || position > uint64(len(data)-zip64EndBytes) {
				return zip.ErrFormat
			}
			end = int(position)
			if binary.LittleEndian.Uint32(data[end:]) != zip64EndSignature {
				return zip.ErrFormat
			}
			records = binary.LittleEndian.Uint64(data[end+32:])
			size = binary.LittleEndian.Uint64(data[end+40:])
			offset = binary.LittleEndian.Uint64(data[end+48:])
		}
	}
	if records == 0 || records > maxArchivePathEntries {
		return errArchiveEntryLimit
	}
	if size > uint64(end) || offset > 1<<63-1 {
		return zip.ErrFormat
	}
	start := end - int(size)
	// Match archive/zip's prefix adjustment, including its preference for a
	// valid unadjusted header when a positive inferred prefix is misleading.
	if offset < uint64(start) && validArchiveDirectoryHeader(data, int(offset)) {
		start = int(offset)
	}
	for count := 0; ; count++ {
		next, ok := archiveDirectoryRecordEnd(data, start)
		if !ok {
			return nil
		}
		if count == maxArchivePathEntries {
			return errArchiveEntryLimit
		}
		start = next
	}
}

func archiveDirectoryRecordEnd(data []byte, start int) (int, bool) {
	if start < 0 || start > len(data)-zipDirectoryHeaderBytes || binary.LittleEndian.Uint32(data[start:]) != zipDirectorySignature {
		return 0, false
	}
	header := data[start:]
	size := zipDirectoryHeaderBytes + int(binary.LittleEndian.Uint16(header[28:])) +
		int(binary.LittleEndian.Uint16(header[30:])) + int(binary.LittleEndian.Uint16(header[32:]))
	if size > len(header) {
		return 0, false
	}
	return start + size, true
}

// ZIP64 extra fields are the only variable-header fields whose contents can
// make archive/zip reject its prefix probe. Other extra metadata is best effort.
func validArchiveDirectoryHeader(data []byte, start int) bool {
	if _, ok := archiveDirectoryRecordEnd(data, start); !ok {
		return false
	}
	header := data[start:]
	nameBytes := int(binary.LittleEndian.Uint16(header[28:]))
	extraBytes := int(binary.LittleEndian.Uint16(header[30:]))
	extra := header[zipDirectoryHeaderBytes+nameBytes : zipDirectoryHeaderBytes+nameBytes+extraBytes]
	headerOffset := uint64(binary.LittleEndian.Uint32(header[42:]))
	for len(extra) >= 4 {
		tag, size := binary.LittleEndian.Uint16(extra), int(binary.LittleEndian.Uint16(extra[2:]))
		extra = extra[4:]
		if size > len(extra) {
			break
		}
		field := extra[:size]
		extra = extra[size:]
		if tag != 1 {
			continue
		}
		for index, required := range []bool{binary.LittleEndian.Uint32(header[24:]) == 0xffffffff, binary.LittleEndian.Uint32(header[20:]) == 0xffffffff, headerOffset == 0xffffffff} {
			if !required {
				continue
			}
			if len(field) < 8 {
				return false
			}
			// Only the header offset stops being a sentinel after a ZIP64
			// field is read; 32-bit size sentinels remain unchanged.
			if index == 2 {
				headerOffset = binary.LittleEndian.Uint64(field)
			}
			field = field[8:]
		}
	}
	return true
}
