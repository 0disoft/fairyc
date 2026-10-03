package semantic

import (
	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/syntax"
)

// byteForSourceIssue checks a shared u8/i64 view before the hidden scalar loop is
// admitted. In particular, a mutable view or a call cannot be made safe by
// the ordinary index/length operations in the prepared loop.
func (c *typeChecker) byteForSourceIssue(loop *ast.ByteForLoop, prepared *ast.Expression) string {
	if loop == nil || loop.Body == nil || loop.View == nil {
		return "malformed byte-view iteration"
	}
	if loop.Type.Kind != ast.TypeKindNamed || loop.Type.Path != "u8" && loop.Type.Path != "i64" || len(loop.Type.Arguments) != 0 {
		return "view iteration requires an exact u8 or i64 item binding"
	}
	wantType := hir.SharedBorrow(hir.TypeByteSlice)
	shape := "byte-view iteration requires a named immutable ref u8[] view"
	if loop.Type.Path == "i64" {
		wantType = hir.SharedBorrow(hir.TypeI64Slice)
		shape = "i64-view iteration requires a named immutable ref i64[] view"
	}
	if prepared == nil || loop.View.Kind != syntax.KindPath || len(loop.View.Children) != 0 {
		return shape
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
	if value == nil || value.kind != "local" && value.kind != "parameter" || value.mutable || value.typ != wantType {
		return shape
	}
	return ""
}

// Hold the i64 view through every body exit, even when break/return means the
// hidden index will never read it again. Release it at the written loop exit.
func (f *borrowFlowAnalyzer) evaluateI64ViewFor(statement *ast.Statement, environment borrowEnvironment) bool {
	prepared := statement.Expression
	if prepared == nil || prepared.Block == nil || len(prepared.Block.Statements) != 2 {
		return true // Malformed source is rejected by the type-checking gate.
	}
	bound := prepared.Block.Statements[0].Binding
	if bound == nil || bound.Expression == nil || len(bound.Expression.Children) != 1 {
		return true
	}
	view, reachable := f.evaluateExpression(bound.Expression.Children[0], environment)
	if !reachable {
		return false
	}
	base := len(f.callBorrows)
	f.callBorrows = append(f.callBorrows, view)
	defer func() { f.callBorrows = f.callBorrows[:base] }()
	_, reachable = f.evaluateExpression(prepared, environment)
	return reachable
}
