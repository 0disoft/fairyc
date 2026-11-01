package ast

import "github.com/0disoft/fairyc/internal/syntax"

// Closure is retained syntax, not a first-class runtime callable or ABI type.
type Closure struct {
	Captures []Parameter
	Function Function
}

func lowerClosure(tree *syntax.Tree, node syntax.Node) *Expression {
	v := &Closure{Function: Function{Span: node.Span}}
	for _, id := range node.Children {
		n, ok := tree.Node(id)
		if !ok {
			continue
		}
		switch n.Kind {
		case syntax.KindCapture:
			v.Captures = append(v.Captures, lowerParameter(tree, n))
		case syntax.KindParameter:
			v.Function.Parameters = append(v.Function.Parameters, lowerParameter(tree, n))
		case syntax.KindType:
			v.Function.ResultType = lowerType(tree, n)
		case syntax.KindBlock:
			v.Function.Body = lowerBlock(tree, n)
		}
	}
	return &Expression{Kind: syntax.KindClosure, Span: node.Span, Closure: v}
}
