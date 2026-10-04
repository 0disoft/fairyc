package semantic

import (
	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/syntax"
)

const arrayForViewName = "\x00fairyc_array_for_view"

// A retained source AST can be checked by a fresh analyzer. Discard our prior
// type-dependent descriptor before name resolution so its symbols and array
// eligibility are established again in the new analysis.
func resetArrayForPreparation(statement *ast.Statement) {
	if statement == nil || statement.ByteFor == nil || statement.Expression == nil || statement.Expression.Block == nil {
		return
	}
	block := statement.Expression.Block
	if len(block.Statements) != 3 || block.Statements[0].Binding == nil || block.Statements[0].Binding.Name != arrayForViewName {
		return
	}
	bound := block.Statements[1].Binding
	classic := block.Statements[2].For
	if bound == nil || bound.Expression == nil || len(bound.Expression.Children) != 1 || classic == nil || classic.Body == nil || len(classic.Body.Statements) == 0 {
		return
	}
	item := classic.Body.Statements[0].Binding
	if item == nil || item.Expression == nil {
		return
	}
	indexed := item.Expression
	if indexed.Kind == syntax.KindPrefix && (indexed.Text == "&mut" || indexed.Text == "&") && len(indexed.Children) == 1 {
		indexed = indexed.Children[0]
	}
	if indexed.Kind != syntax.KindIndex || len(indexed.Children) != 2 {
		return
	}
	view := statement.ByteFor.View
	bound.Expression.Children[0] = view
	indexed.Children[0] = &ast.Expression{Kind: syntax.KindPath, Span: view.Span, Text: view.Text}
	block.Statements = block.Statements[1:]
}

// Resolution already selected the written array and loop-local names. Add only
// a private descriptor and redirect the prepared header paths to that symbol;
// never resolve the written body again or copy the array into a temporary.
func (c *typeChecker) prepareArrayFor(loop *ast.ByteForLoop, prepared *ast.Expression, owner *symbol, want hir.Type, mutable bool) string {
	const shape = "array iteration requires an exact integer item and a named fixed integer array; mutable items require a mutable array"
	element, length, array := hir.FixedArrayParts(owner.typ)
	if !array || element != want || element == hir.TypeBool || !hir.ExecutableFixedArrayShape(element, length) || mutable && !owner.mutable {
		return shape
	}
	classic := prepared.Block.Statements[1].For
	if classic == nil || classic.Body == nil || len(classic.Body.Statements) == 0 {
		return "malformed fixed-array iteration"
	}
	item := classic.Body.Statements[0].Binding
	if item == nil || item.Expression == nil || item.Type == nil || item.Name != loop.Name || c.canonicalType(*item.Type) != c.canonicalType(loop.Type) {
		return "malformed fixed-array iteration"
	}
	indexed := item.Expression
	if loop.Type.Kind == ast.TypeKindReference {
		operator := "&"
		if mutable {
			operator = "&mut"
		}
		if indexed.Kind != syntax.KindPrefix || indexed.Text != operator || len(indexed.Children) != 1 {
			return "malformed mutable fixed-array iteration"
		}
		indexed = indexed.Children[0]
	}
	if indexed.Kind != syntax.KindIndex || len(indexed.Children) != 2 || indexed.Children[0].Kind != syntax.KindPath || indexed.Children[0].Text != loop.View.Text {
		return "malformed fixed-array iteration"
	}
	elementType := loop.Type
	if loop.Type.Kind == ast.TypeKindReference {
		elementType = *loop.Type.Element
	}
	sliceType := ast.TypeReference{Kind: ast.TypeKindSlice, Element: &elementType, Span: loop.View.Span}
	viewType := ast.TypeReference{Kind: ast.TypeKindReference, Mutable: mutable, Element: &sliceType, Span: loop.View.Span}
	operator := "&"
	if mutable {
		operator = "&mut"
	}
	// Generic expansion may detach the retained source view from the checked
	// bound path. Bind our new owner path to the symbol selected by that path;
	// the written loop view is source provenance, not a resolved place token.
	ownerPath := &ast.Expression{Kind: syntax.KindPath, Span: loop.View.Span, Text: loop.View.Text}
	c.analyzer.resolved[ownerPath] = owner
	view := &ast.Binding{Name: arrayForViewName, Span: loop.View.Span, NameSpan: loop.View.Span, Type: &viewType,
		Expression: &ast.Expression{Kind: syntax.KindPrefix, Span: loop.View.Span, Text: operator, Children: []*ast.Expression{ownerPath}}}
	value := &symbol{id: c.analyzer.allocateSymbol(), packageID: owner.packageID, kind: "local", name: arrayForViewName,
		module: owner.module, source: owner.source, declaration: loop.View.Span, typeSpan: loop.View.Span}
	c.analyzer.bindingSymbols[view] = value
	path := func() *ast.Expression {
		p := &ast.Expression{Kind: syntax.KindPath, Span: loop.View.Span, Text: arrayForViewName}
		c.analyzer.resolved[p] = value
		return p
	}
	bound := prepared.Block.Statements[0].Binding
	bound.Expression.Children[0] = path()
	indexed.Children[0] = path()
	prepared.Block.Statements = append([]ast.Statement{{Kind: syntax.KindBinding, Span: loop.View.Span, Binding: view}}, prepared.Block.Statements...)
	return ""
}

// Materialize the implicit array view once, then retain it through all loop
// exits. This is the same whole-owner loan as an explicit named-view loop.
func (f *borrowFlowAnalyzer) evaluateArrayFor(prepared *ast.Expression, environment borrowEnvironment) bool {
	view := &prepared.Block.Statements[0]
	if view.Binding == nil || view.Binding.Name != arrayForViewName {
		return true // Type checking rejects malformed preparation.
	}
	binding := f.analyzer.bindingSymbols[view.Binding]
	defer delete(environment, binding)
	value, reachable := f.evaluateExpression(view.Binding.Expression, environment)
	if !reachable {
		return false
	}
	// The implicit exclusive descriptor is a child generation of the whole
	// array. Keep direct owner reads frozen even after this iteration's item
	// dies; the next iteration still accesses the original storage.
	if view.Binding.Type.Mutable {
		for i := range value.pending {
			value.pending[i].subview = true
		}
	}
	environment[binding] = f.materialize(value, binding)
	base := len(f.callBorrows)
	f.callBorrows = append(f.callBorrows, environment[binding])
	defer func() { f.callBorrows = f.callBorrows[:base] }()
	rest := &ast.Expression{Kind: syntax.KindBlock, Span: prepared.Span,
		Block: &ast.Block{Span: prepared.Block.Span, Statements: prepared.Block.Statements[1:]}}
	_, reachable = f.evaluateExpression(rest, environment)
	return reachable
}
