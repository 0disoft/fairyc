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
	afterCR  bool
}

// A positionIndex belongs to one immutable source in an analysis or document snapshot.
// It extends only as far as requested, so a diagnostic at byte zero is cheap.
type positionIndex struct {
	text        []byte
	checkpoints []positionCheckpoint
	scanned     int
	position    Position
	afterCR     bool
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
				index.afterCR = false
			} else if r == '\r' {
				// Keep this set until LF, even through text following a lone CR.
				index.afterCR = true
			} else {
				index.position.Character++
				if r > 0xffff {
					index.position.Character++
				}
			}
			last := index.checkpoints[len(index.checkpoints)-1]
			if index.scanned-last.offset >= positionCheckpointBytes {
				index.checkpoints = append(index.checkpoints, positionCheckpoint{index.scanned, index.position, index.afterCR})
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

// positionByte reuses sparse checkpoints for incoming UTF-16 request positions.
// Checkpoints within a long line avoid rescanning its prefix on every request.
// A checkpoint after CR cannot admit columns on that line: the linear oracle
// stops at the first CR even if no LF follows it.
func (index *positionIndex) positionByte(p Position) (int, bool) {
	if p.Line < 0 || p.Character < 0 {
		return 0, false
	}
	for index.scanned < len(index.text) && (index.position.Line < p.Line ||
		index.position.Line == p.Line && !index.afterCR && index.position.Character < p.Character) {
		index.bytePosition(min(len(index.text), index.scanned+positionCheckpointBytes))
	}
	point := max(0, sort.Search(len(index.checkpoints), func(i int) bool {
		position := index.checkpoints[i].position
		return position.Line > p.Line || position.Line == p.Line && position.Character > p.Character
	})-1)
	if index.checkpoints[point].afterCR {
		point = max(0, sort.Search(len(index.checkpoints), func(i int) bool {
			return index.checkpoints[i].position.Line >= p.Line
		})-1)
	}
	checkpoint := index.checkpoints[point]
	column := p.Character
	if checkpoint.position.Line == p.Line {
		column -= checkpoint.position.Character
	}
	offset, ok := positionByte(index.text[checkpoint.offset:], Position{Line: p.Line - checkpoint.position.Line, Character: column})
	if !ok {
		return 0, false
	}
	return checkpoint.offset + offset, true
}

func (d *document) positionByte(p Position) (int, bool) {
	if d.positionIndex == nil {
		d.positionIndex = newPositionIndex(d.Text)
	}
	return d.positionIndex.positionByte(p)
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
