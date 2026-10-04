package semantic

import (
	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/syntax"
)

// Indexed places are rooted in one named view descriptor or fixed array.
// Calls, temporary descriptors and raw pointers do not acquire place identity.
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

// Named integer arrays and integer/plain nominal view descriptors yield stored
// element addresses. Grouping preserves identity; temporary roots do not.
func (c *typeChecker) indexedViewBorrow(source *ast.Expression, child *hir.Expression, mutable bool) (hir.Expression, bool) {
	for source != nil && source.Kind == syntax.KindGroup && len(source.Children) == 1 {
		source = source.Children[0]
	}
	if source == nil || source.Kind != syntax.KindIndex || child == nil {
		return hir.Expression{}, false
	}
	if borrowed, ok := c.indexedArrayBorrow(source, child, mutable); ok {
		return borrowed, true
	}
	if !c.indexedStructPlace(source) && !c.indexedIntegerPlace(source) {
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

// Direct fixed-array places lower through an internal view of the original
// storage. Do not lower the indexed value first: that would reference a copy.
func (c *typeChecker) indexedArrayBorrow(source *ast.Expression, child *hir.Expression, mutable bool) (hir.Expression, bool) {
	index := indexedPlace(source)
	if index == nil {
		return hir.Expression{}, false
	}
	owner := c.analyzer.resolved[assignmentRootPath(index.Children[0])]
	if owner == nil || (owner.kind != "local" && owner.kind != "parameter") || mutable && !owner.mutable {
		return hir.Expression{}, false
	}
	element, length, array := hir.FixedArrayParts(owner.typ)
	if !array || element == hir.TypeBool || !hir.ExecutableFixedArrayShape(element, length) {
		return hir.Expression{}, false
	}
	lowered := *child
	for lowered.Kind == "group" && len(lowered.Children) == 1 {
		lowered = lowered.Children[0]
	}
	if lowered.Kind != "array-index" || len(lowered.Children) != 2 || lowered.Type != element || lowered.Children[0].Type != owner.typ {
		return hir.Expression{}, false
	}
	view := hir.SharedBorrow(hir.Type("[" + string(element) + "]"))
	operator := "&"
	if mutable {
		view = hir.MutableBorrow(hir.Type("[" + string(element) + "]"))
		operator = "&mut"
	}
	if !hir.ArrayViewBorrow(owner.typ, view, operator) {
		return hir.Expression{}, false
	}
	lowered.Kind = "view-index"
	lowered.Children = []hir.Expression{
		{Kind: "array-view-borrow", Type: view, Span: lowered.Children[0].Span, Symbol: hir.NoSymbol, Operator: operator, Children: []hir.Expression{lowered.Children[0]}},
		lowered.Children[1],
	}
	return indexedElementBorrow(lowered, mutable)
}

func (f *borrowFlowAnalyzer) evaluateArrayElementBorrow(expression, index *ast.Expression, environment borrowEnvironment) (borrowValueState, bool) {
	parent := index.Children[0]
	owner := f.analyzer.resolved[assignmentRootPath(parent)]
	if owner == nil {
		return borrowValueState{known: false}, true
	}
	if _, reachable := f.evaluateExpression(parent, environment); !reachable {
		return borrowValueState{}, false
	}
	f.checkBorrowCreation(expression, environment)
	origin := f.analyzer.originForSymbol(owner)
	mutable := expression.Text == "&mut"
	value := borrowValueState{known: true, origins: []flowOrigin{origin}, pending: []pendingBorrowLoan{{owner: owner, borrow: span(f.module.file.Path, expression.Span), kind: borrowKind(expression.Text), origin: origin}}}
	base, accessBase := len(f.callBorrows), len(f.callAccesses)
	f.callBorrows = append(f.callBorrows, value)
	f.callAccesses = append(f.callAccesses, borrowCallAccess{value: value, mutable: mutable, span: span(f.module.file.Path, expression.Span)})
	defer func() { f.callBorrows = f.callBorrows[:base]; f.callAccesses = f.callAccesses[:accessBase] }()
	if _, reachable := f.evaluateExpression(index.Children[1], environment); !reachable {
		return borrowValueState{}, false
	}
	// Exclusive elements freeze the whole owner and existing view aliases until
	// last use. Array indices do not establish disjointness.
	value.pending[0].subview = mutable
	return value, true
}
