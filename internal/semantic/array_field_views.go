package semantic

import (
	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/syntax"
)

// A field path is a place only when rooted in a named local or parameter.
// Calls, temporaries, indexing and pointer dereferences do not gain identity.
func arrayFieldRoot(source *ast.Expression) *ast.Expression {
	for source != nil && source.Kind == syntax.KindGroup && len(source.Children) == 1 {
		source = source.Children[0]
	}
	if source == nil || source.Kind != syntax.KindFieldExpr {
		return nil
	}
	return fieldAssignmentRoot(source)
}

func (c *typeChecker) arrayFieldView(source *ast.Expression, field hir.Expression, mutable bool) (hir.Expression, bool) {
	// Repr(C) arrays use raw C members, not the private wrapper-reference
	// representation. Implicit read-index optimization must fall back to a copy.
	if c.reprCArrayFieldExpression(field) {
		return hir.Expression{}, false
	}
	root := arrayFieldRoot(source)
	owner := c.analyzer.resolved[root]
	if root == nil || owner == nil || (owner.kind != "local" && owner.kind != "parameter") || !c.analyzer.inlineArrayField(field.Type) {
		return hir.Expression{}, false
	}
	typ := owner.typ
	if element, _, borrowed := hir.BorrowElement(typ); borrowed {
		typ = element
	}
	if !c.analyzer.plainViewStruct(typ) {
		return hir.Expression{}, false
	}
	var address hir.Expression
	var ok bool
	operator := "&"
	if mutable {
		operator = "&mut"
		address, ok = c.mutableFieldBorrow(source, &field)
	} else {
		address, ok = c.sharedFieldBorrow(source, field)
	}
	if !ok {
		return hir.Expression{}, false
	}
	element, _, _ := hir.FixedArrayParts(field.Type)
	view := hir.SharedBorrow(hir.Slice(element))
	if mutable {
		view = hir.MutableBorrow(hir.Slice(element))
	}
	if !hir.ArrayViewBorrow(address.Type, view, operator) {
		return hir.Expression{}, false
	}
	return hir.Expression{Kind: hir.ExpressionArrayViewBorrow, Type: view, Operator: operator, Span: field.Span, Symbol: hir.NoSymbol, Children: []hir.Expression{address}}, true
}

// Reserve the complete struct owner before bounds and retain it through use.
func (f *borrowFlowAnalyzer) evaluateArrayFieldPlace(expression, selected *ast.Expression, mutable bool, environment borrowEnvironment) (borrowValueState, bool) {
	root := arrayFieldRoot(selected.Children[0])
	owner := f.analyzer.resolved[root]
	if owner == nil {
		return borrowValueState{known: false}, true
	}
	var value borrowValueState
	var reachable bool
	if _, _, borrowed := hir.BorrowElement(owner.typ); borrowed {
		borrow := *expression
		borrow.Text = "&"
		if mutable {
			borrow.Text = "&mut"
		}
		value, reachable = f.evaluateFieldBorrowRoot(&borrow, root, environment)
	} else {
		_, reachable = f.evaluateOwnerPlace(root, environment)
		kind := "shared"
		if mutable {
			kind = "mutable"
		}
		f.checkNamedBorrowCreation(expression, owner, kind, environment)
		origin := f.analyzer.originForSymbol(owner)
		value = borrowValueState{known: true, origins: []flowOrigin{origin}, pending: []pendingBorrowLoan{{owner: owner, borrow: span(f.module.file.Path, expression.Span), kind: kind, origin: origin, subview: mutable}}}
	}
	if !reachable {
		return borrowValueState{}, false
	}
	base, accessBase := len(f.callBorrows), len(f.callAccesses)
	f.callBorrows = append(f.callBorrows, value)
	f.callAccesses = append(f.callAccesses, borrowCallAccess{value: value, mutable: mutable, span: span(f.module.file.Path, expression.Span)})
	defer func() { f.callBorrows = f.callBorrows[:base]; f.callAccesses = f.callAccesses[:accessBase] }()
	for _, bound := range selected.Children[1:] {
		if _, reachable = f.evaluateExpression(bound, environment); !reachable {
			return borrowValueState{}, false
		}
	}
	return value, true
}

func (f *borrowFlowAnalyzer) evaluateArrayFieldStore(statement *ast.Statement, environment borrowEnvironment) bool {
	root := arrayFieldRoot(statement.Target.Children[0])
	held, reachable := f.evaluateArrayFieldPlace(root, statement.Target, true, environment)
	if !reachable {
		return false
	}
	base, accessBase := len(f.callBorrows), len(f.callAccesses)
	priorBorrow, priorLastUse := f.assignmentBorrow, f.assignmentLastUse
	f.callBorrows = append(f.callBorrows, held)
	f.callAccesses = append(f.callAccesses, borrowCallAccess{value: held, mutable: true, store: true, span: span(f.module.file.Path, statement.Target.Span)})
	f.assignmentBorrow = &held
	f.assignmentLastUse = span(f.module.file.Path, statement.Target.Span)
	defer func() {
		f.callBorrows = f.callBorrows[:base]
		f.callAccesses = f.callAccesses[:accessBase]
		f.assignmentBorrow, f.assignmentLastUse = priorBorrow, priorLastUse
	}()
	_, reachable = f.evaluateExpression(statement.Expression, environment)
	return reachable
}
