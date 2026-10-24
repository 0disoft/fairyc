package languageserver

import (
	"sort"
	"unicode/utf8"
)

// Sparse checkpoints bound rescanning without retaining a position per byte.
const positionCheckpointBytes = 1024

type positionCheckpoint struct {
	offset   int
	position Position
}

// A positionIndex belongs to one immutable source in one analysis snapshot.
// It extends only as far as requested, so a diagnostic at byte zero is cheap.
type positionIndex struct {
	text        []byte
	checkpoints []positionCheckpoint
	scanned     int
	position    Position
}

func newPositionIndex(text []byte) *positionIndex {
	return &positionIndex{text: text, checkpoints: []positionCheckpoint{{}}}
}

func (index *positionIndex) bytePosition(offset int) Position {
	offset = max(0, min(offset, len(index.text)))
	if offset >= index.scanned {
		for index.scanned < offset {
			r, width := utf8.DecodeRune(index.text[index.scanned:])
			if index.scanned+width > offset {
				break
			}
			index.scanned += width
			if r == '\n' {
				index.position.Line++
				index.position.Character = 0
			} else if r != '\r' {
				index.position.Character++
				if r > 0xffff {
					index.position.Character++
				}
			}
			last := index.checkpoints[len(index.checkpoints)-1]
			if index.scanned-last.offset >= positionCheckpointBytes {
				index.checkpoints = append(index.checkpoints, positionCheckpoint{index.scanned, index.position})
			}
		}
		return index.position
	}
	point := sort.Search(len(index.checkpoints), func(i int) bool {
		return index.checkpoints[i].offset > offset
	}) - 1
	checkpoint := index.checkpoints[point]
	position := bytePosition(index.text[checkpoint.offset:], offset-checkpoint.offset)
	if position.Line == 0 {
		position.Character += checkpoint.position.Character
	}
	position.Line += checkpoint.position.Line
	return position
}

func (a *analysis) sourceRange(filename string, text []byte, start, end int) Range {
	if a.positions == nil {
		a.positions = make(map[string]*positionIndex)
	}
	index := a.positions[filename]
	if index == nil {
		index = newPositionIndex(text)
		a.positions[filename] = index
	}
	return Range{index.bytePosition(start), index.bytePosition(end)}
}
