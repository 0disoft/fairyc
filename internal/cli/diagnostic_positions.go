package cli

import "bytes"

// Fixed byte checkpoints keep memory independent of line count, including
// newline-only input. Each lookup rescans at most one 4 KiB chunk.
const diagnosticPositionChunk = 4096

type diagnosticBytePosition struct {
	line, column int
}

func (position diagnosticBytePosition) advance(contents []byte) diagnosticBytePosition {
	lines := bytes.Count(contents, []byte{'\n'})
	position.line += lines
	if lines == 0 {
		position.column += len(contents)
	} else {
		position.column = len(contents) - bytes.LastIndexByte(contents, '\n')
	}
	return position
}

// The source is borrowed for one report only; no index survives source edits.
type diagnosticPositionIndex struct {
	source      []byte
	checkpoints []diagnosticBytePosition
}

func (index *diagnosticPositionIndex) position(offset int) (int, int) {
	offset = max(0, min(offset, len(index.source)))
	if offset == 0 {
		return 1, 1
	}
	if len(index.checkpoints) == 0 {
		index.checkpoints = append(index.checkpoints, diagnosticBytePosition{1, 1})
	}
	chunk := offset / diagnosticPositionChunk
	for len(index.checkpoints) <= chunk {
		last := len(index.checkpoints) - 1
		start := last * diagnosticPositionChunk
		next := index.checkpoints[last].advance(index.source[start : start+diagnosticPositionChunk])
		index.checkpoints = append(index.checkpoints, next)
	}
	position := index.checkpoints[chunk].advance(index.source[chunk*diagnosticPositionChunk : offset])
	return position.line, position.column
}
