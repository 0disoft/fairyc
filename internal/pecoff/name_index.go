package pecoff

import (
	"slices"
	"strings"
)

type coffNameSpan struct {
	start, end uint32
}

// Callers collect in-range offsets and retain format-specific diagnostics.
// Sorting those offsets lets references to the same C-string (including
// suffixes) share a single terminator scan. len(table) marks an unclosed span.
func indexCOFFNameSpans(offsets []uint32, table string) []coffNameSpan {
	slices.Sort(offsets)
	var spans []coffNameSpan
	for _, offset := range offsets {
		if len(spans) > 0 && offset <= spans[len(spans)-1].end {
			continue
		}
		end := strings.IndexByte(table[offset:], 0)
		if end < 0 {
			spans = append(spans, coffNameSpan{start: offset, end: uint32(len(table))})
			break
		}
		spans = append(spans, coffNameSpan{start: offset, end: offset + uint32(end)})
	}
	return spans
}
