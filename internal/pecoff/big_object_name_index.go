package pecoff

import (
	"encoding/binary"
	"slices"
	"strings"
)

type bigCOFFNameSpan struct {
	start, end uint32
}

// Index referenced C-string spans in offset order. Repeated references and
// suffixes reuse a terminator, so no string-table byte is scanned twice.
// Invalid offsets and unterminated names are still diagnosed in symbol order.
func indexBigCOFFNameSpans(symbols []byte, table string) []bigCOFFNameSpan {
	offsets := make([]uint32, 0, len(symbols)/coffBigObjectSymbolBytes)
	for position := 0; position < len(symbols); position += coffBigObjectSymbolBytes {
		field := symbols[position : position+8]
		if binary.LittleEndian.Uint32(field) == 0 {
			offset := binary.LittleEndian.Uint32(field[4:])
			if offset >= 4 && uint64(offset) < uint64(len(table)) {
				offsets = append(offsets, offset)
			}
		}
	}
	slices.Sort(offsets)
	var spans []bigCOFFNameSpan
	for _, offset := range offsets {
		if len(spans) > 0 && offset <= spans[len(spans)-1].end {
			continue
		}
		end := strings.IndexByte(table[offset:], 0)
		if end < 0 {
			spans = append(spans, bigCOFFNameSpan{start: offset, end: uint32(len(table))})
			break
		}
		spans = append(spans, bigCOFFNameSpan{start: offset, end: offset + uint32(end)})
	}
	return spans
}
