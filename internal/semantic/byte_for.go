package semantic

import (
	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/syntax"
)

// byteForSourceIssue checks the written form before the hidden scalar loop is
// admitted. In particular, a mutable view or a call cannot be made safe by
// the ordinary index/length operations in the prepared loop.
func (c *typeChecker) byteForSourceIssue(loop *ast.ByteForLoop, prepared *ast.Expression) string {
	if loop == nil || loop.Body == nil || loop.View == nil {
		return "malformed byte-view iteration"
	}
	if loop.Type.Kind != ast.TypeKindNamed || loop.Type.Path != "u8" {
		return "byte-view iteration requires an exact u8 item binding"
	}
	if prepared == nil || loop.View.Kind != syntax.KindPath || len(loop.View.Children) != 0 {
		return "byte-view iteration requires a named immutable ref u8[] view"
	}
	if prepared.Kind != syntax.KindBlock || prepared.Block == nil || len(prepared.Block.Statements) != 2 {
		return "malformed byte-view iteration"
	}
	bound := prepared.Block.Statements[0].Binding
	if bound == nil || bound.Expression == nil || bound.Expression.Kind != syntax.KindFieldExpr ||
		bound.Expression.Text != "len" || len(bound.Expression.Children) != 1 {
		return "malformed byte-view iteration"
	}
	view := bound.Expression.Children[0]
	if view == nil || view.Kind != syntax.KindPath || view.Text != loop.View.Text || view.Span != loop.View.Span {
		return "malformed byte-view iteration"
	}
	// AST cloning may detach the source node from the prepared bound expression.
	// Resolve the actual checked path instead of relying on shared pointers.
	value := c.analyzer.resolved[view]
	if value == nil || value.kind != "local" && value.kind != "parameter" || value.mutable || value.typ != hir.SharedBorrow(hir.TypeByteSlice) {
		return "byte-view iteration requires a named immutable ref u8[] view"
	}
	return ""
}
