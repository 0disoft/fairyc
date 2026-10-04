package semantic

import (
	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/syntax"
)

// byteForSourceIssue checks a named integer/plain struct view before the hidden
// scalar loop is admitted. The descriptor binding must be immutable so the length snapshot
// and later indexed reads always use the same view, even for exclusive views.
func (c *typeChecker) byteForSourceIssue(loop *ast.ByteForLoop, prepared *ast.Expression) string {
	if loop == nil || loop.Body == nil || loop.View == nil {
		return "malformed byte-view iteration"
	}
	itemType := loop.Type
	mutableItem := itemType.Kind == ast.TypeKindReference && itemType.Mutable && itemType.Element != nil
	referenceItem := itemType.Kind == ast.TypeKindReference && itemType.Element != nil
	if referenceItem {
		itemType = *itemType.Element
	}
	if (itemType.Kind != ast.TypeKindNamed && itemType.Kind != ast.TypeKindBoundNominal) || len(itemType.Arguments) != 0 {
		return "view iteration requires an exact integer or scalar-field struct item binding"
	}
	wantElement, resolved := c.analyzer.canonicalTypeReference(c.module, itemType, false)
	_, integer := hir.IntegerViewElement(hir.Type("[" + string(wantElement) + "]"))
	nominal := c.analyzer.plainViewStruct(wantElement)
	if !resolved || !integer && !nominal {
		return "view iteration requires an exact u8, i32, i64, u32, usize or scalar-field struct item binding"
	}
	shape := loop.Type.Path + "-view iteration requires a named immutable ref " + loop.Type.Path + "[] or ref mut " + loop.Type.Path + "[] view"
	if nominal {
		shape = "struct-view iteration requires an exact named immutable shared or mutable struct view"
	}
	if mutableItem {
		shape = "mutable view iteration requires an exact ref mut integer or struct item and a named immutable mutable view"
	} else if referenceItem {
		shape = "shared reference iteration requires an exact integer or plain struct item and a named immutable view"
	}
	if wantElement == hir.TypeU8 && !mutableItem {
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
	if value == nil || value.kind != "local" && value.kind != "parameter" {
		return shape
	}
	if _, _, array := hir.FixedArrayParts(value.typ); array {
		return c.prepareArrayFor(loop, prepared, value, wantElement, mutableItem)
	}
	if value.mutable {
		return shape
	}
	if element, indexed := hir.IndexedViewElement(value.typ); !indexed || element != wantElement || mutableItem && !hir.MutableIndexedView(value.typ) {
		return shape
	}
	if referenceItem {
		classic := prepared.Block.Statements[1].For
		if classic == nil || classic.Body == nil || len(classic.Body.Statements) == 0 {
			return "malformed mutable view iteration"
		}
		item := classic.Body.Statements[0].Binding
		operator := "&"
		if mutableItem {
			operator = "&mut"
		}
		if item == nil || item.Name != loop.Name || item.Type == nil || c.canonicalType(*item.Type) != c.canonicalType(loop.Type) ||
			item.Expression == nil || item.Expression.Kind != syntax.KindPrefix || item.Expression.Text != operator || len(item.Expression.Children) != 1 {
			return "malformed mutable view iteration"
		}
		index := item.Expression.Children[0]
		if index.Kind != syntax.KindIndex || len(index.Children) != 2 || index.Children[0].Kind != syntax.KindPath || index.Children[0].Text != loop.View.Text {
			return "malformed mutable view iteration"
		}
	}
	return ""
}

// Hold the view through every body exit, even when break/return means the
// hidden index will never read it again. Release it at the written loop exit.
// This retains the owner loan without creating a shared reborrow: exclusive
// views may still perform element stores and short reborrows inside the body.
func (f *borrowFlowAnalyzer) evaluateViewFor(statement *ast.Statement, environment borrowEnvironment) bool {
	prepared := statement.Expression
	if prepared != nil && prepared.Block != nil && len(prepared.Block.Statements) == 3 {
		if first := prepared.Block.Statements[0].Binding; first != nil && first.Name == arrayForViewName {
			return f.evaluateArrayFor(prepared, environment)
		}
		// SDK Vec preparation holds its owner through an ordinary block binding.
		_, reachable := f.evaluateExpression(prepared, environment)
		return reachable
	}
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
	base, accessBase := len(f.callBorrows), len(f.callAccesses)
	f.callBorrows = append(f.callBorrows, view)
	if statement.ByteFor.Type.Kind == ast.TypeKindReference && !statement.ByteFor.Type.Mutable {
		f.callAccesses = append(f.callAccesses, borrowCallAccess{value: view, mutable: false, span: span(f.module.file.Path, statement.ByteFor.View.Span)})
	}
	defer func() { f.callBorrows = f.callBorrows[:base]; f.callAccesses = f.callAccesses[:accessBase] }()
	_, reachable = f.evaluateExpression(prepared, environment)
	return reachable
}
