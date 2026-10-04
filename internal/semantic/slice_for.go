package semantic

import (
	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/syntax"
)

// Direct slices are admitted only over a named array or immutable descriptor.
// Reuse the resolved header tree: effectful bounds must keep their symbols and
// execute once, before the length snapshot and every element access.
func (c *typeChecker) prepareSliceFor(loop *ast.ByteForLoop, prepared *ast.Expression, checked *ast.Expression, want hir.Type, mutableItem bool) string {
	const shape = "subview iteration requires an exact item type and a named fixed array or immutable view; mutable items require mutable storage"
	if !sameVectorForView(loop.View, checked) || checked.Kind != syntax.KindSlice || len(checked.Children) != 3 {
		return shape
	}
	root := checked.Children[0]
	if root == nil || root.Kind != syntax.KindPath || len(root.Children) != 0 {
		return shape
	}
	owner := c.analyzer.resolved[root]
	if owner == nil || owner.kind != "local" && owner.kind != "parameter" {
		return shape
	}
	source := *checked
	source.Children = append([]*ast.Expression(nil), checked.Children...)
	mutable := mutableItem
	if element, length, array := hir.FixedArrayParts(owner.typ); array {
		if element != want || !hir.ExecutableFixedArrayShape(element, length) || mutableItem && !owner.mutable {
			return shape
		}
		operator := "&"
		if mutableItem {
			operator = "&mut"
		}
		// Whole-array conversion needs a typed binding. Keep it in a private
		// expression block so ordinary slice checking receives a view, without
		// extending array slicing elsewhere or copying array storage.
		elementType := loop.Type
		if elementType.Kind == ast.TypeKindReference {
			elementType = *elementType.Element
		}
		sliceType := ast.TypeReference{Kind: ast.TypeKindSlice, Element: &elementType, Span: root.Span}
		viewType := ast.TypeReference{Kind: ast.TypeKindReference, Mutable: mutableItem, Element: &sliceType, Span: root.Span}
		const name = "\x00fairyc_slice_for_array"
		binding := &ast.Binding{Name: name, Span: root.Span, NameSpan: root.Span, Type: &viewType,
			Expression: &ast.Expression{Kind: syntax.KindPrefix, Span: root.Span, Text: operator, Children: []*ast.Expression{root}}}
		value := &symbol{id: c.analyzer.allocateSymbol(), packageID: owner.packageID, kind: "local", name: name,
			module: owner.module, source: owner.source, declaration: root.Span, typeSpan: root.Span}
		c.analyzer.bindingSymbols[binding] = value
		path := &ast.Expression{Kind: syntax.KindPath, Span: root.Span, Text: name}
		c.analyzer.resolved[path] = value
		source.Children[0] = path
		block := &ast.Expression{Kind: syntax.KindBlock, Span: checked.Span, Block: &ast.Block{Span: checked.Span,
			Statements: []ast.Statement{{Kind: syntax.KindBinding, Span: root.Span, Binding: binding}}, Tail: &source}}
		return c.prepareForDescriptor(loop, prepared, owner, block, mutableItem)
	} else {
		element, indexed := hir.IndexedViewElement(owner.typ)
		if !indexed || element != want || owner.mutable || mutableItem && !hir.MutableIndexedView(owner.typ) {
			return shape
		}
		mutable = hir.MutableIndexedView(owner.typ)
	}
	return c.prepareForDescriptor(loop, prepared, owner, &source, mutable)
}
