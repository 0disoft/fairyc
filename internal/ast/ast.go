// Package ast lowers error-free compact syntax into the first Stage 0 AST boundary.
package ast

import (
	"errors"
	"strings"

	"github.com/0disoft/fairyc/internal/syntax"
)

// Module is the parser fixture-facing portion of the lowered AST.
type Module struct {
	Path     string
	Bindings []Binding
	Literals []Literal
}

// Binding records a source declaration and its expression tree.
type Binding struct {
	Name       string
	Expression *Expression
}

// Literal records raw and cooked values without selecting a target type.
type Literal struct {
	Raw    string
	Kind   string
	Cooked string
}

// Expression is a small structural AST used before name resolution and typing.
type Expression struct {
	Kind     syntax.Kind
	Text     string
	Cooked   string
	Children []*Expression
}

// Lower rejects diagnostic CSTs and converts source concepts without semantic analysis.
func Lower(tree *syntax.Tree) (*Module, error) {
	if tree == nil {
		return nil, errors.New("syntax tree is nil")
	}
	if !tree.ErrorFree() {
		return nil, errors.New("diagnostic syntax tree cannot lower to AST")
	}
	module := &Module{Path: tree.Module}
	visited := make(map[syntax.NodeID]struct{}, len(tree.Nodes))
	var walk func(syntax.NodeID)
	walk = func(id syntax.NodeID) {
		if id == syntax.NoNode {
			return
		}
		if _, found := visited[id]; found {
			return
		}
		visited[id] = struct{}{}
		node, found := tree.Node(id)
		if !found {
			return
		}
		if node.Kind == syntax.KindBinding && len(node.Children) > 0 {
			expression := lowerExpression(tree, node.Children[len(node.Children)-1])
			module.Bindings = append(module.Bindings, Binding{Name: node.Text, Expression: expression})
		}
		if node.Kind == syntax.KindLiteral {
			module.Literals = append(module.Literals, Literal{
				Raw:    node.Text,
				Kind:   classifyLiteral(node.Text),
				Cooked: node.Cooked,
			})
		}
		for _, child := range node.Children {
			walk(child)
		}
	}
	walk(tree.Root)
	return module, nil
}

func lowerExpression(tree *syntax.Tree, id syntax.NodeID) *Expression {
	node, found := tree.Node(id)
	if !found {
		return nil
	}
	expression := &Expression{
		Kind:   node.Kind,
		Text:   node.Text,
		Cooked: node.Cooked,
	}
	for _, child := range node.Children {
		if lowered := lowerExpression(tree, child); lowered != nil {
			expression.Children = append(expression.Children, lowered)
		}
	}
	return expression
}

func classifyLiteral(raw string) string {
	switch {
	case raw == "true" || raw == "false":
		return "boolean"
	case raw == "null":
		return "null"
	case strings.HasPrefix(raw, "b\""):
		return "byte-string"
	case strings.HasPrefix(raw, "c\""):
		return "c-string"
	case strings.HasPrefix(raw, "\""):
		return "string"
	case strings.ContainsAny(raw, ".eE") && !strings.HasPrefix(raw, "0x"):
		return "float"
	default:
		return "integer"
	}
}
