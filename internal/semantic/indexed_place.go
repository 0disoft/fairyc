package semantic

import (
	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/syntax"
)

// Indexed places are rooted in one named view descriptor. Calls, temporary
// descriptors and raw pointers do not acquire place identity here.
func indexedPlace(expression *ast.Expression) *ast.Expression {
	root := borrowPlaceRoot(expression)
	if root != nil && root.Kind == syntax.KindIndex && len(root.Children) == 2 && assignmentRootPath(root.Children[0]) != nil {
		return root
	}
	return nil
}

func (c *typeChecker) indexedStructPlace(source *ast.Expression) bool {
	index := indexedPlace(source)
	if index == nil {
		return false
	}
	element, nominal := hir.StructViewElement(c.analyzer.expressionTypes[index.Children[0]])
	return nominal && c.analyzer.plainViewStruct(element)
}

func (c *typeChecker) indexedIntegerPlace(source *ast.Expression) bool {
	index := indexedPlace(source)
	if index == nil {
		return false
	}
	element, indexed := hir.IndexedViewElement(c.analyzer.expressionTypes[index.Children[0]])
	return indexed && hir.FixedArrayScalarElement(element)
}

// Only a named integer or plain nominal descriptor yields a stored element address.
// Grouping preserves place identity; calls, temporary views and arrays do not.
func (c *typeChecker) indexedViewBorrow(source *ast.Expression, child *hir.Expression, mutable bool) (hir.Expression, bool) {
	for source != nil && source.Kind == syntax.KindGroup && len(source.Children) == 1 {
		source = source.Children[0]
	}
	if source == nil || source.Kind != syntax.KindIndex || child == nil || !c.indexedStructPlace(source) && !c.indexedIntegerPlace(source) {
		return hir.Expression{}, false
	}
	index := *child
	for index.Kind == "group" && len(index.Children) == 1 {
		index = index.Children[0]
	}
	return indexedElementBorrow(index, mutable)
}

func indexedElementBorrow(index hir.Expression, mutable bool) (hir.Expression, bool) {
	if index.Kind != "view-index" || len(index.Children) != 2 {
		return hir.Expression{}, false
	}
	element, indexed := hir.IndexedViewElement(index.Children[0].Type)
	if !indexed || element != index.Type || mutable && !hir.MutableIndexedView(index.Children[0].Type) {
		return hir.Expression{}, false
	}
	typ := hir.SharedBorrow(element)
	if mutable {
		typ = hir.MutableBorrow(element)
	}
	return hir.Expression{Kind: "view-element-borrow", Type: typ, Span: index.Span, Symbol: hir.NoSymbol, Operator: "[]", Children: index.Children}, true
}

func (f *borrowFlowAnalyzer) evaluateIndexedPlace(index *ast.Expression, mutable, retain bool, environment borrowEnvironment) (borrowValueState, bool) {
	parent := index.Children[0]
	value, reachable := f.evaluateExpression(parent, environment)
	if !reachable {
		return value, false
	}
	f.checkReferenceAccess(parent, value, mutable, environment, span(f.module.file.Path, index.Span))
	base, accessBase := len(f.callBorrows), len(f.callAccesses)
	f.callBorrows = append(f.callBorrows, value)
	f.callAccesses = append(f.callAccesses, borrowCallAccess{value: value, mutable: mutable, span: span(f.module.file.Path, index.Span)})
	defer func() { f.callBorrows = f.callBorrows[:base]; f.callAccesses = f.callAccesses[:accessBase] }()
	if _, reachable = f.evaluateExpression(index.Children[1], environment); !reachable {
		return borrowValueState{}, false
	}
	if !retain {
		return cloneBorrowValue(value), true
	}
	if !mutable {
		return f.retainSharedParent(parent, span(f.module.file.Path, index.Span), value), true
	}
	value = cloneBorrowValue(value)
	for _, origin := range value.origins {
		value.pending = append(value.pending, pendingBorrowLoan{subview: true, owner: f.analyzer.resolved[assignmentRootPath(parent)], borrow: span(f.module.file.Path, index.Span), kind: "mutable", origin: origin})
	}
	return value, true
}
