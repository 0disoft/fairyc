package syntax

import (
	"bytes"
	"slices"

	"github.com/0disoft/fairyc/internal/lexer"
	"github.com/0disoft/fairyc/internal/source"
)

// EditorParseSnapshot owns the checkpoints for one current-grammar parse.
// Its tree and child lists are immutable for as long as the snapshot is retained.
type EditorParseSnapshot struct {
	path              string
	limits            source.Limits
	tree              *Tree
	lexer             *lexer.EditorLexSnapshot
	prefixes          []parsePrefix
	reusedItems       int
	reusedSuffixItems int
}

type parsePrefix struct {
	items, nodes                 int
	position, lastIndex, lastEnd int
	observed, syntheticTokens    int
}

type editorParseState struct {
	previous, next            *EditorParseSnapshot
	observed, syntheticTokens int
	suffix                    editorParseSuffix
}

func (state *editorParseState) lex(file source.File, budget *source.Budget) lexer.Result {
	var previous *lexer.EditorLexSnapshot
	if state.previous != nil {
		previous = state.previous.lexer
	}
	result, next := lexer.LexEditor(file, budget, previous)
	state.next.lexer = next
	return result
}

// ParseEditor reuses verified tokens and top-level subtrees when the current
// parser reaches the matching declaration boundaries.
func ParseEditor(file source.File, limits source.Limits, previous *EditorParseSnapshot) (*Tree, *EditorParseSnapshot) {
	if previous != nil && (previous.path != file.RelativePath || previous.limits != limits) {
		previous = nil
	}
	next := &EditorParseSnapshot{path: file.RelativePath, limits: limits}
	tree := parse(file, limits, parseCurrent, &editorParseState{previous: previous, next: next})
	if !tree.ErrorFree() {
		return tree, nil
	}
	next.tree = tree
	return tree, next
}

// CheckpointWeight bounds the retained prefix metadata, including unused capacity.
func (s *EditorParseSnapshot) CheckpointWeight() int64 {
	return int64(cap(s.prefixes))*64 + int64(len(s.path)) + 128 + s.lexer.CheckpointWeight()
}

// ReusedItems reports the count of top-level items restored in the prefix.
func (s *EditorParseSnapshot) ReusedItems() int { return s.reusedItems }

func (p *parser) recordEditorPrefix(items int) {
	if p.splitShift || p.budget.Exhausted() || len(p.tree.Diagnostics) != 0 {
		return
	}
	p.editor.next.prefixes = append(p.editor.next.prefixes, parsePrefix{
		items: items, nodes: len(p.tree.Nodes),
		position: p.position, lastIndex: p.lastFullIndex, lastEnd: p.lastEndByte,
		observed: p.editor.observed, syntheticTokens: p.editor.syntheticTokens,
	})
}

func (p *parser) restoreEditorPrefix() []NodeID {
	if p.editor == nil || p.editor.previous == nil || len(p.tree.Diagnostics) != 0 {
		return nil
	}
	previous := p.editor.previous
	if len(previous.prefixes) == 0 {
		return nil
	}
	old := previous.tree
	through := previous.prefixes[len(previous.prefixes)-1].observed
	match := min(p.editor.next.lexer.ReusedPrefix(), through+1)
	for match <= through && match < min(len(p.tree.Tokens), len(old.Tokens)) {
		token := old.Tokens[match]
		if token != p.tree.Tokens[match] ||
			!bytes.Equal(old.Source[token.Span.StartByte:token.Span.EndByte], p.tree.Source[token.Span.StartByte:token.Span.EndByte]) {
			break
		}
		match++
	}
	for i := len(previous.prefixes) - 1; i >= 0; i-- {
		prefix := previous.prefixes[i]
		if prefix.observed >= match {
			continue
		}
		// advance charges one work unit per consumed significant token; each
		// contextual split charges one additional work unit and synthetic token.
		if !p.budget.ReuseSyntax(prefix.syntheticTokens, prefix.nodes, prefix.position+prefix.syntheticTokens) {
			// Reparse to emit the same diagnostic at the first exhausted token.
			return nil
		}
		p.position = prefix.position
		p.lastFullIndex, p.lastEndByte = prefix.lastIndex, prefix.lastEnd
		p.editor.observed, p.editor.syntheticTokens = prefix.observed, prefix.syntheticTokens
		p.tree.Nodes = slices.Clone(old.Nodes[:prefix.nodes])
		p.tree.Module = old.Module
		p.editor.next.prefixes = slices.Clone(previous.prefixes[:i+1])
		p.editor.next.reusedItems = prefix.items - previous.prefixes[0].items
		return slices.Clone(old.Nodes[old.Root].Children[:prefix.items])
	}
	return nil
}
