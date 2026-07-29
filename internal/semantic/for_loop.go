package semantic

import (
	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/syntax"
)

// forAsWhile exposes the condition/body fixed point shared by the existing
// Flow analyses. Header clauses are separately analyzed and type checking keeps
// the v1 initializer and post forms scalar-only, so they cannot alter resource,
// Result, or borrow state on a back edge.
func forAsWhile(loop *ast.ForLoop) *ast.Expression {
	if loop == nil || loop.Body == nil {
		return nil
	}
	condition := loop.Condition
	if condition == nil {
		condition = &ast.Expression{Kind: syntax.KindLiteral, Span: loop.Span, Text: "true", Cooked: "true"}
	}
	body := &ast.Expression{Kind: syntax.KindBlock, Span: loop.Body.Span, Block: loop.Body}
	return &ast.Expression{
		Kind:     syntax.KindWhile,
		Span:     loop.Span,
		Children: []*ast.Expression{condition, body},
	}
}
