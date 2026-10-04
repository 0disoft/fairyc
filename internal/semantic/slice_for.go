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
		// Use the same explicit subview borrow as ordinary bindings and calls.
		borrow := &ast.Expression{Kind: syntax.KindPrefix, Span: checked.Span, Text: operator, Children: []*ast.Expression{checked}}
		return c.prepareForDescriptor(loop, prepared, owner, borrow, mutableItem)
	} else {
		element, indexed := hir.IndexedViewElement(owner.typ)
		if !indexed || element != want || owner.mutable || mutableItem && !hir.MutableIndexedView(owner.typ) {
			return shape
		}
		mutable = hir.MutableIndexedView(owner.typ)
	}
	return c.prepareForDescriptor(loop, prepared, owner, &source, mutable)
}
