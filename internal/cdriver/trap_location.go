package cdriver

import (
	"encoding/binary"
	"fmt"
	"strings"
	"unicode/utf8"
)

const maxTrapControlBytes = 276

// TrapLocation is compiler-private source metadata, using an end-exclusive byte range.
type TrapLocation struct {
	Source             string
	StartByte, EndByte uint64
}

// The legacy category byte remains valid. Extended receipts have a v1, 20-byte
// little-endian header followed by at most 256 UTF-8 source-name bytes.
func decodeTrapControl(exit int32, receipt string) (uint8, *TrapLocation, error) {
	if receipt == "" {
		return 0, nil, nil
	}
	data := []byte(receipt)
	category := data[0]
	if category < 1 || category > 7 || exit != NativeTrapExitBase+int32(category) {
		return 0, nil, fmt.Errorf("invalid trap category or process status")
	}
	if len(data) == 1 {
		return category, nil, nil
	}
	if len(data) < 20 || len(data) > maxTrapControlBytes || data[1] != 1 {
		return 0, nil, fmt.Errorf("invalid trap location header")
	}
	length := int(binary.LittleEndian.Uint16(data[2:4]))
	start, end := binary.LittleEndian.Uint64(data[4:12]), binary.LittleEndian.Uint64(data[12:20])
	source := string(data[20:])
	if length == 0 || length > 256 || len(source) != length || start > end ||
		!utf8.ValidString(source) || strings.ContainsRune(source, 0) {
		return 0, nil, fmt.Errorf("invalid trap source location")
	}
	return category, &TrapLocation{Source: source, StartByte: start, EndByte: end}, nil
}
