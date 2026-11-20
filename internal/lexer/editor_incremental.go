package lexer

import (
	"bytes"
	"slices"
	"sort"

	"github.com/0disoft/fairyc/internal/source"
)

const editorCheckpointTokens = 256

// A completed token can have inspected two bytes beyond its end. The
// three-byte operator dispatch and the decimal point followed by a digit
// both need that much. At a clean scanner boundary there is no pending mode.
const editorBoundaryLookahead = 2

type editorLexCheckpoint struct {
	position, tokens, work int
}

// EditorLexSnapshot retains immutable tokens and sparse scanner boundaries.
// Source and tokens share the backing storage owned by the current syntax tree.
type EditorLexSnapshot struct {
	source             []byte
	tokens             []Token
	checkpoints        []editorLexCheckpoint
	work, reusedPrefix int
	reusedSuffix       int
}

func (s *EditorLexSnapshot) CheckpointWeight() int64 {
	if s == nil {
		return 0
	}
	return 128 + int64(cap(s.checkpoints))*24
}

// ReusedPrefix gives the token count already verified against current bytes.
func (s *EditorLexSnapshot) ReusedPrefix() int {
	if s == nil {
		return 0
	}
	return s.reusedPrefix
}

type editorLexState struct {
	previous, next *EditorLexSnapshot
	suffix         []editorLexCheckpoint
	byteShift      int
}

// LexEditor scans from a verified unchanged prefix and can splice in a
// byte-identical suffix after reaching a scanner boundary in both versions.
func LexEditor(file source.File, budget *source.Budget, previous *EditorLexSnapshot) (Result, *EditorLexSnapshot) {
	next := &EditorLexSnapshot{}
	editor := editorLexState{previous: previous, next: next}
	s := scanner{source: append([]byte(nil), file.Bytes...), budget: budget, editor: &editor}
	if previous != nil && !budget.Exhausted() {
		editor.prepare(&s)
	}
	s.run()
	result := Result{Source: s.source, Tokens: s.tokens, Diagnostics: s.diagnostics}
	if budget.Exhausted() || len(s.diagnostics) != 0 {
		return result, nil
	}
	next.source, next.tokens, next.work = s.source, s.tokens, budget.WorkUnits()
	return result, next
}

func (e *editorLexState) prepare(s *scanner) {
	old := e.previous
	compared, prefix := 0, 0
	for _, checkpoint := range old.checkpoints {
		end := checkpoint.position + editorBoundaryLookahead
		if end > min(len(old.source), len(s.source)) || !bytes.Equal(old.source[compared:end], s.source[compared:end]) {
			break
		}
		compared = end
		prefix++
	}
	if prefix != 0 {
		checkpoint := old.checkpoints[prefix-1]
		if s.budget.ReuseSyntax(checkpoint.tokens, 0, checkpoint.work) {
			s.position = checkpoint.position
			e.next.reusedPrefix = checkpoint.tokens
			s.tokens = slices.Clone(old.tokens[:checkpoint.tokens])
			e.next.checkpoints = slices.Clone(old.checkpoints[:prefix])
		}
	}
	if prefix == len(old.checkpoints) {
		return
	}
	if len(e.next.checkpoints) == 0 {
		prefix = 0
	}
	tailBytes := commonEditorSuffix(old.source[s.position:], s.source[s.position:])
	oldStart := len(old.source) - tailBytes
	later := old.checkpoints[prefix:]
	first := sort.Search(len(later), func(i int) bool { return later[i].position >= oldStart })
	e.suffix = later[first:]
	e.byteShift = len(s.source) - len(old.source)
}

func commonEditorSuffix(old, current []byte) int {
	const block = 1024
	shared := min(len(old), len(current))
	n := 0
	for n+block <= shared && bytes.Equal(old[len(old)-n-block:len(old)-n], current[len(current)-n-block:len(current)-n]) {
		n += block
	}
	for n < shared && old[len(old)-n-1] == current[len(current)-n-1] {
		n++
	}
	return n
}

func (e *editorLexState) atBoundary(s *scanner) bool {
	if len(s.diagnostics) != 0 {
		return false
	}
	for len(e.suffix) != 0 && e.suffix[0].position+e.byteShift < s.position {
		e.suffix = e.suffix[1:]
	}
	if len(e.suffix) != 0 && e.suffix[0].position+e.byteShift == s.position && s.position != 0 && e.spliceSuffix(s) {
		return true
	}
	next := e.next
	priorTokens := 0
	if n := len(next.checkpoints); n != 0 {
		priorTokens = next.checkpoints[n-1].tokens
	}
	if len(s.tokens)-priorTokens >= editorCheckpointTokens {
		next.checkpoints = append(next.checkpoints, editorLexCheckpoint{s.position, len(s.tokens), s.budget.WorkUnits()})
	}
	return false
}

func (e *editorLexState) spliceSuffix(s *scanner) bool {
	old, tail := e.previous, e.suffix[0]
	tokens := old.tokens[tail.tokens : len(old.tokens)-1] // EOF is emitted by run.
	workBefore := s.budget.WorkUnits()
	if !s.budget.ReuseSyntax(len(tokens), 0, old.work-tail.work) {
		e.suffix = nil
		return false
	}
	tokenShift, workShift := len(s.tokens)-tail.tokens, workBefore-tail.work
	s.tokens = slices.Grow(s.tokens, len(tokens)+1)
	for _, token := range tokens {
		token.Span.StartByte += e.byteShift
		token.Span.EndByte += e.byteShift
		s.tokens = append(s.tokens, token)
	}
	for _, checkpoint := range e.suffix {
		e.next.checkpoints = append(e.next.checkpoints, editorLexCheckpoint{
			position: checkpoint.position + e.byteShift,
			tokens:   checkpoint.tokens + tokenShift,
			work:     checkpoint.work + workShift,
		})
	}
	e.next.reusedSuffix = len(tokens)
	s.position = len(s.source)
	return true
}
