package semantic

import (
	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/syntax"
)

// byteForSourceIssue checks a named integer or shared struct view before the hidden scalar loop is
// admitted. The descriptor binding must be immutable so the length snapshot
// and later indexed reads always use the same view, even for exclusive views.
func (c *typeChecker) byteForSourceIssue(loop *ast.ByteForLoop, prepared *ast.Expression) string {
	if loop == nil || loop.Body == nil || loop.View == nil {
		return "malformed byte-view iteration"
	}
	if (loop.Type.Kind != ast.TypeKindNamed && loop.Type.Kind != ast.TypeKindBoundNominal) || len(loop.Type.Arguments) != 0 {
		return "view iteration requires an exact integer or scalar-field struct item binding"
	}
	wantElement, resolved := c.analyzer.canonicalTypeReference(c.module, loop.Type, false)
	_, integer := hir.IntegerViewElement(hir.Type("[" + string(wantElement) + "]"))
	nominal := c.analyzer.plainViewStruct(wantElement)
	if !resolved || !integer && !nominal {
		return "view iteration requires an exact u8, i32, i64, u32, usize or scalar-field struct item binding"
	}
	shape := loop.Type.Path + "-view iteration requires a named immutable ref " + loop.Type.Path + "[] or ref mut " + loop.Type.Path + "[] view"
	if nominal {
		shape = "struct-view iteration requires an exact named immutable shared struct view"
	}
	if wantElement == hir.TypeU8 {
		shape = "byte-view iteration requires a named immutable ref u8[] or ref mut u8[] view"
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
	if value == nil || value.kind != "local" && value.kind != "parameter" || value.mutable {
		return shape
	}
	if element, indexed := hir.IndexedViewElement(value.typ); !indexed || element != wantElement {
		return shape
	}
	return ""
}

// Hold the view through every body exit, even when break/return means the
// hidden index will never read it again. Release it at the written loop exit.
// This retains the owner loan without creating a shared reborrow: exclusive
// views may still perform element stores and short reborrows inside the body.
func (f *borrowFlowAnalyzer) evaluateViewFor(statement *ast.Statement, environment borrowEnvironment) bool {
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
