// Package prevrevision migrates the immediately preceding Fairy C source revision without
// exposing that grammar through the product compiler entry point.
package prevrevision

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/0disoft/fairyc/internal/lexer"
	"github.com/0disoft/fairyc/internal/source"
	"github.com/0disoft/fairyc/internal/syntax"
)

// Result is one in-memory migration. Callers own persistence and atomic replacement.
type Result struct {
	Source  []byte
	Changed bool
}

type edit struct {
	start       int
	end         int
	replacement string
}

// MigrateDeclarations converts declaration order, structural type spelling, resource cleanup,
// and function tail returns. It refuses syntax-invalid input and overlapping edits.
func MigrateDeclarations(path string, input []byte) (Result, error) {
	tree := syntax.Parse(source.NewFile(path, input), source.DefaultLimits())
	if len(tree.Diagnostics) != 0 {
		return Result{}, fmt.Errorf("previous revision parse produced %d diagnostics", len(tree.Diagnostics))
	}
	edits, err := declarationEdits(tree)
	if err != nil {
		return Result{}, err
	}
	if len(edits) == 0 {
		return Result{Source: slices.Clone(input)}, nil
	}
	output, err := applyEdits(input, edits)
	if err != nil {
		return Result{}, err
	}
	verified := syntax.Parse(source.NewFile(path, output), source.DefaultLimits())
	if len(verified.Diagnostics) != 0 {
		return Result{}, fmt.Errorf("migrated declaration parse produced %d diagnostics", len(verified.Diagnostics))
	}
	return Result{Source: output, Changed: true}, nil
}

func declarationEdits(tree *syntax.Tree) ([]edit, error) {
	var edits []edit
	for index := range tree.Nodes {
		node := tree.Nodes[index]
		switch node.Kind {
		case syntax.KindFunction:
			if node.Text == "drop" {
				block, found := childOfKind(tree, node, syntax.KindBlock)
				if !found {
					return nil, errors.New("drop declaration has no body")
				}
				edits = append(edits, edit{start: node.Span.StartByte, end: block.Span.StartByte, replacement: "cleanup "})
				continue
			}
			legacy, fnToken := legacyFunction(tree, node)
			if !legacy {
				continue
			}
			block, found := childOfKind(tree, node, syntax.KindBlock)
			if !found {
				return nil, fmt.Errorf("function %q has no body", node.Text)
			}
			if commentsBetween(tree, fnToken+1, block.TokenStart) {
				return nil, fmt.Errorf("function %q header contains comments and requires manual migration", node.Text)
			}
			header, err := renderFunctionHeader(tree, node)
			if err != nil {
				return nil, err
			}
			edits = append(edits, edit{start: tree.Tokens[fnToken].Span.EndByte, end: block.Span.StartByte, replacement: header})
			if tail, found := childOfKind(tree, block, syntax.KindTail); found && len(tail.Children) == 1 {
				expression, ok := tree.Node(tail.Children[0])
				if ok {
					edits = append(edits,
						edit{start: expression.Span.StartByte, end: expression.Span.StartByte, replacement: "return "},
						edit{start: expression.Span.EndByte, end: expression.Span.EndByte, replacement: ";"},
					)
				}
			}
		case syntax.KindForeign:
			legacy, fnToken := legacyFunction(tree, node)
			if !legacy {
				continue
			}
			if commentsBetween(tree, fnToken+1, node.TokenEnd) {
				return nil, fmt.Errorf("foreign function %q header contains comments and requires manual migration", node.Text)
			}
			header, err := renderFunctionHeader(tree, node)
			if err != nil {
				return nil, err
			}
			semicolon := tokenIndex(tree, node, ";")
			if semicolon < 0 {
				return nil, fmt.Errorf("foreign function %q has no terminator", node.Text)
			}
			edits = append(edits, edit{start: tree.Tokens[fnToken].Span.EndByte, end: tree.Tokens[semicolon].Span.StartByte, replacement: header})
		case syntax.KindField, syntax.KindConst, syntax.KindBinding:
			typeNode, found := childOfKind(tree, node, syntax.KindType)
			if !found || node.NameSpan.StartByte >= typeNode.Span.StartByte {
				continue
			}
			start := node.NameSpan.StartByte
			if node.Kind == syntax.KindConst || node.Kind == syntax.KindBinding {
				keyword := tokenIndex(tree, node, "const", "let", "var")
				if keyword < 0 {
					continue
				}
				start = tree.Tokens[keyword].Span.EndByte
			}
			edits = append(edits, edit{
				start: start, end: typeNode.Span.EndByte,
				replacement: " " + renderType(tree, typeNode) + " " + node.Text,
			})
		}
	}
	return edits, nil
}

func legacyFunction(tree *syntax.Tree, node syntax.Node) (bool, int) {
	fn := tokenIndex(tree, node, "fn")
	if fn < 0 {
		return false, -1
	}
	next := nextSignificant(tree, fn+1, node.TokenEnd)
	after := nextSignificant(tree, next+1, node.TokenEnd)
	return next >= 0 && after >= 0 && tree.Tokens[next].Kind == lexer.Identifier &&
		tree.Tokens[next].Text(tree.Source) == node.Text && tree.Tokens[after].Text(tree.Source) == "(", fn
}

func renderFunctionHeader(tree *syntax.Tree, node syntax.Node) (string, error) {
	resultType, found := childOfKind(tree, node, syntax.KindType)
	if !found {
		return "", fmt.Errorf("function %q has no result type", node.Text)
	}
	parameters := make([]string, 0, 4)
	for _, childID := range node.Children {
		child, ok := tree.Node(childID)
		if !ok || child.Kind != syntax.KindParameter {
			continue
		}
		if child.Cooked == "mut" {
			return "", fmt.Errorf("function %q has mutable parameter %q and requires manual migration", node.Text, child.Text)
		}
		parameterType, ok := childOfKind(tree, child, syntax.KindType)
		if !ok {
			return "", fmt.Errorf("parameter %q has no type", child.Text)
		}
		parameters = append(parameters, renderType(tree, parameterType)+" "+child.Text)
	}
	return " " + renderType(tree, resultType) + " " + node.Text + "(" + strings.Join(parameters, ", ") + ") ", nil
}

func renderType(tree *syntax.Tree, node syntax.Node) string {
	children := make([]syntax.Node, 0, len(node.Children))
	for _, id := range node.Children {
		child, ok := tree.Node(id)
		if ok {
			children = append(children, child)
		}
	}
	childType := func(index int) string {
		if index < 0 || index >= len(children) || children[index].Kind != syntax.KindType {
			return ""
		}
		return renderType(tree, children[index])
	}
	switch syntax.TypeForm(node.Cooked) {
	case syntax.TypeFormReference:
		return "ref " + childType(0)
	case syntax.TypeFormReferenceMut:
		return "ref mut " + childType(0)
	case syntax.TypeFormPointer:
		return "ptr " + childType(0)
	case syntax.TypeFormPointerMut:
		return "ptr mut " + childType(0)
	case syntax.TypeFormSlice:
		return childType(0) + "[]"
	case syntax.TypeFormArray:
		length := ""
		if len(children) > 1 {
			length = strings.TrimSpace(children[1].Text)
		}
		return childType(0) + "[" + length + "]"
	case syntax.TypeFormGrouped:
		return "(" + childType(0) + ")"
	case syntax.TypeFormForeignFunction:
		if len(children) == 0 {
			return strings.TrimSpace(node.Text)
		}
		parameters := make([]string, 0, len(children)-1)
		for index := 0; index < len(children)-1; index++ {
			parameters = append(parameters, renderType(tree, children[index]))
		}
		return "unsafe extern C fn " + renderType(tree, children[len(children)-1]) + "(" + strings.Join(parameters, ", ") + ")"
	case syntax.TypeFormNamed:
		if len(children) == 0 {
			return strings.TrimSpace(node.Text)
		}
		name := children[0].Text
		if len(children) == 1 {
			return name
		}
		arguments := make([]string, 0, len(children)-1)
		for index := 1; index < len(children); index++ {
			arguments = append(arguments, renderType(tree, children[index]))
		}
		return name + "<" + strings.Join(arguments, ", ") + ">"
	default:
		return strings.TrimSpace(node.Text)
	}
}

func childOfKind(tree *syntax.Tree, node syntax.Node, kind syntax.Kind) (syntax.Node, bool) {
	for _, id := range node.Children {
		child, ok := tree.Node(id)
		if ok && child.Kind == kind {
			return child, true
		}
	}
	return syntax.Node{}, false
}

func tokenIndex(tree *syntax.Tree, node syntax.Node, values ...string) int {
	wanted := make(map[string]struct{}, len(values))
	for _, value := range values {
		wanted[value] = struct{}{}
	}
	for index := max(node.TokenStart, 0); index < min(node.TokenEnd, len(tree.Tokens)); index++ {
		if _, ok := wanted[tree.Tokens[index].Text(tree.Source)]; ok {
			return index
		}
	}
	return -1
}

func nextSignificant(tree *syntax.Tree, start, end int) int {
	for index := max(start, 0); index < min(end, len(tree.Tokens)); index++ {
		if tree.Tokens[index].Kind != lexer.Trivia {
			return index
		}
	}
	return -1
}

func commentsBetween(tree *syntax.Tree, start, end int) bool {
	for index := max(start, 0); index < min(end, len(tree.Tokens)); index++ {
		if tree.Tokens[index].Kind != lexer.Trivia {
			continue
		}
		text := tree.Tokens[index].Text(tree.Source)
		if strings.Contains(text, "//") || strings.Contains(text, "/*") {
			return true
		}
	}
	return false
}

func applyEdits(input []byte, edits []edit) ([]byte, error) {
	slices.SortFunc(edits, func(left, right edit) int {
		if left.start != right.start {
			return right.start - left.start
		}
		return right.end - left.end
	})
	output := slices.Clone(input)
	previousStart := len(input) + 1
	for _, current := range edits {
		if current.start < 0 || current.end < current.start || current.end > len(input) {
			return nil, fmt.Errorf("invalid migration edit %d..%d", current.start, current.end)
		}
		if current.end > previousStart {
			return nil, fmt.Errorf("overlapping migration edit %d..%d", current.start, current.end)
		}
		output = append(output[:current.start], append([]byte(current.replacement), output[current.end:]...)...)
		previousStart = current.start
	}
	return output, nil
}
