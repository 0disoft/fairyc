package ast

import (
	"github.com/0disoft/fairyc/internal/lexer"
	"github.com/0disoft/fairyc/internal/syntax"
)

// functionModifiers reads only the declaration prefix. A result type may itself
// contain unsafe extern C fn, which does not grant an unsafe function body.
func functionModifiers(tree *syntax.Tree, node syntax.Node) (public, unsafe bool) {
	start := max(node.TokenStart, 0)
	for _, id := range node.Children {
		if child, ok := tree.Node(id); ok && child.Kind == syntax.KindAttribute {
			start = max(start, child.TokenEnd)
		}
	}
	end := min(node.TokenEnd, len(tree.Tokens))
	for i := start; i < end; i++ {
		token := tree.Tokens[i]
		if token.Kind != lexer.Keyword {
			continue
		}
		switch token.Text(tree.Source) {
		case "fn":
			return public, unsafe
		case "pub":
			public = true
		case "unsafe":
			unsafe = true
		}
	}
	return public, unsafe
}
