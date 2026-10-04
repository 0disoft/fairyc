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
	statement.Expression = nil
}

// Resolution already selected the written array and loop-local names. Add only
// a private descriptor and redirect the prepared header paths to that symbol;
// never resolve the written body again or copy the array into a temporary.
func (c *typeChecker) prepareArrayFor(loop *ast.ByteForLoop, prepared *ast.Expression, owner *symbol, want hir.Type, mutable bool) string {
	const shape = "array iteration requires an exact bool or integer item and a named fixed scalar or plain struct array; mutable items require a mutable array"
	element, length, array := hir.FixedArrayParts(owner.typ)
	if !array || element != want || !hir.ExecutableFixedArrayShape(element, length) || mutable && !owner.mutable {
		return shape
	}
	operator := "&"
	if mutable {
		operator = "&mut"
	}
	ownerPath := &ast.Expression{Kind: syntax.KindPath, Span: loop.View.Span, Text: loop.View.Text}
	c.analyzer.resolved[ownerPath] = owner
	source := &ast.Expression{Kind: syntax.KindPrefix, Span: loop.View.Span, Text: operator, Children: []*ast.Expression{ownerPath}}
	return c.prepareForDescriptor(loop, prepared, owner, source, mutable)
}

// Array and direct-subview loops share one descriptor and one element path.
func (c *typeChecker) prepareForDescriptor(loop *ast.ByteForLoop, prepared *ast.Expression, owner *symbol, source *ast.Expression, mutable bool) string {
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
		if loop.Type.Mutable {
			operator = "&mut"
		}
		if indexed.Kind != syntax.KindPrefix || indexed.Text != operator || len(indexed.Children) != 1 {
			return "malformed mutable fixed-array iteration"
		}
		indexed = indexed.Children[0]
	}
	if indexed.Kind != syntax.KindIndex || len(indexed.Children) != 2 || !sameVectorForView(loop.View, indexed.Children[0]) {
		return "malformed fixed-array iteration"
	}
	elementType := loop.Type
	if loop.Type.Kind == ast.TypeKindReference {
		elementType = *loop.Type.Element
	}
	sliceType := ast.TypeReference{Kind: ast.TypeKindSlice, Element: &elementType, Span: loop.View.Span}
	viewType := ast.TypeReference{Kind: ast.TypeKindReference, Mutable: mutable, Element: &sliceType, Span: loop.View.Span}
	view := &ast.Binding{Name: arrayForViewName, Span: loop.View.Span, NameSpan: loop.View.Span, Type: &viewType, Expression: source}
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
