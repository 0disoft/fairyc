package syntax

import (
	"bytes"
	"slices"

	"github.com/0disoft/fairyc/internal/diagnostic"
	"github.com/0disoft/fairyc/internal/lexer"
)

type editorParseSuffix struct {
	firstToken            int
	tokenShift, byteShift int
	positionShift         int
	boundaries            []parsePrefix
}

func (p *parser) prepareEditorSuffix() {
	if p.editor == nil || p.editor.previous == nil || len(p.tree.Diagnostics) != 0 {
		return
	}
	previous := p.editor.previous
	if len(previous.prefixes) < 2 {
		return
	}
	old, current := previous.tree, p.tree
	tokenShift := len(current.Tokens) - len(old.Tokens)
	byteShift := len(current.Source) - len(old.Source)
	first := len(current.Tokens)
	for first > p.lastFullIndex+1 && first-1-tokenShift >= 0 {
		before := old.Tokens[first-1-tokenShift]
		after := current.Tokens[first-1]
		beforeBytes := old.Source[before.Span.StartByte:before.Span.EndByte]
		before.Span = shiftEditorSpan(before.Span, byteShift)
		if before != after || !bytes.Equal(beforeBytes, current.Source[after.Span.StartByte:after.Span.EndByte]) {
			break
		}
		first--
	}
	last := previous.prefixes[len(previous.prefixes)-1]
	p.editor.suffix = editorParseSuffix{
		firstToken: first, byteShift: byteShift, tokenShift: tokenShift,
		positionShift: len(p.significant) - 1 - last.position,
		boundaries:    previous.prefixes,
	}
}

// Run only at a top-level item boundary. The exact token suffix includes all
// possible forward lookahead, and the current parser has finished the header.
func (p *parser) restoreEditorSuffix(children []NodeID) []NodeID {
	if p.editor == nil || len(p.editor.suffix.boundaries) < 2 || len(p.tree.Diagnostics) != 0 {
		return nil
	}
	suffix := &p.editor.suffix
	if p.currentFullIndex() < suffix.firstToken {
		return nil
	}
	position := p.position - suffix.positionShift
	for len(suffix.boundaries) > 1 && suffix.boundaries[0].position < position {
		suffix.boundaries = suffix.boundaries[1:]
	}
	if len(suffix.boundaries) < 2 || suffix.boundaries[0].position != position {
		return nil
	}
	prefixes := suffix.boundaries
	suffix.boundaries = nil
	base, last := prefixes[0], prefixes[len(prefixes)-1]
	synthetic := last.syntheticTokens - base.syntheticTokens
	if !p.budget.ReuseSyntax(synthetic, last.nodes-base.nodes, last.position-base.position+synthetic) {
		// Ordinary parsing locates the exact point of budget exhaustion.
		return nil
	}
	old := p.editor.previous.tree
	nodeShift := len(p.tree.Nodes) - base.nodes
	roots := old.Nodes[old.Root].Children[base.items:last.items]
	p.appendEditorSuffixNodes(old.Nodes[base.nodes:last.nodes], int(roots[0])-base.nodes, nodeShift)
	p.rebaseEditorSuffix(prefixes, len(children), nodeShift)
	for _, child := range roots {
		children = append(children, shiftEditorNode(child, nodeShift))
	}
	return children
}

func (p *parser) appendEditorSuffixNodes(nodes []Node, firstItem, nodeShift int) {
	suffix := &p.editor.suffix
	p.tree.Nodes = slices.Grow(p.tree.Nodes, len(nodes)+1)
	for index, node := range nodes {
		node.Span = shiftEditorSpan(node.Span, suffix.byteShift)
		if node.NameSpan != (diagnostic.Span{}) {
			node.NameSpan = shiftEditorSpan(node.NameSpan, suffix.byteShift)
		}
		if node.DocumentationSpan != (diagnostic.Span{}) {
			node.DocumentationSpan = shiftEditorSpan(node.DocumentationSpan, suffix.byteShift)
		}
		node.TokenStart += suffix.tokenShift
		node.TokenEnd += suffix.tokenShift
		if nodeShift != 0 && len(node.Children) != 0 {
			node.Children = slices.Clone(node.Children)
			for i, child := range node.Children {
				node.Children[i] = shiftEditorNode(child, nodeShift)
			}
		}
		if index == firstItem && supportsDeclarationDocumentation(node.Kind) {
			// Leading documentation may still touch the changed region.
			node.DocumentationSpan, _ = lexer.AttachedDocumentationSpan(p.tree.Tokens, p.tree.Source, node.TokenStart)
		}
		p.tree.Nodes = append(p.tree.Nodes, node)
	}
}

func (p *parser) rebaseEditorSuffix(prefixes []parsePrefix, items, nodeShift int) {
	editor := p.editor
	base := prefixes[0]
	itemShift := items - base.items
	syntheticShift := editor.syntheticTokens - base.syntheticTokens
	for _, boundary := range prefixes[1:] {
		boundary.items += itemShift
		boundary.nodes += nodeShift
		boundary.position += editor.suffix.positionShift
		boundary.lastIndex += editor.suffix.tokenShift
		boundary.lastEnd += editor.suffix.byteShift
		boundary.observed = max(editor.observed, boundary.observed+editor.suffix.tokenShift)
		boundary.syntheticTokens += syntheticShift
		editor.next.prefixes = append(editor.next.prefixes, boundary)
	}
	last := editor.next.prefixes[len(editor.next.prefixes)-1]
	p.position, p.lastFullIndex, p.lastEndByte = last.position, last.lastIndex, last.lastEnd
	editor.observed, editor.syntheticTokens = last.observed, last.syntheticTokens
	editor.next.reusedSuffixItems = last.items - items
}

func shiftEditorSpan(span diagnostic.Span, shift int) diagnostic.Span {
	return diagnostic.Span{StartByte: span.StartByte + shift, EndByte: span.EndByte + shift}
}

func shiftEditorNode(node NodeID, shift int) NodeID {
	if node == NoNode {
		return node
	}
	return node + NodeID(shift)
}
