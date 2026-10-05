package semantic

import (
	"slices"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/hir"
)

func (f *borrowFlowAnalyzer) evaluateBorrowedIndex(expression *ast.Expression, environment borrowEnvironment) (borrowValueState, bool) {
	receiver, reachable := f.evaluateExpression(expression.Children[0], environment)
	if !reachable {
		return receiver, false
	}
	base, accessBase := len(f.callBorrows), len(f.callAccesses)
	f.callBorrows = append(f.callBorrows, receiver)
	f.callAccesses = append(f.callAccesses, borrowCallAccess{value: receiver, mutable: false, span: span(f.module.file.Path, expression.Span)})
	defer func() { f.callBorrows = f.callBorrows[:base]; f.callAccesses = f.callAccesses[:accessBase] }()
	return f.evaluateExpression(expression.Children[1], environment)
}

func (f *borrowFlowAnalyzer) evaluateSubview(expression *ast.Expression, environment borrowEnvironment) (borrowValueState, bool) {
	if len(expression.Children) != 3 {
		return borrowValueState{known: false}, true
	}
	receiverExpression := expression.Children[0]
	receiver, reachable := f.evaluateExpression(receiverExpression, environment)
	if !reachable {
		return receiver, false
	}
	mutable := hir.MutableIndexedView(f.analyzer.expressionTypes[expression])
	base, accessBase := len(f.callBorrows), len(f.callAccesses)
	if mutable {
		f.checkReferenceAccess(receiverExpression, receiver, true, environment, span(f.module.file.Path, expression.Span))
	}
	// Every selected receiver remains borrowed while effectful bounds run,
	// including a shared view whose previous alias has reached its last use.
	// Pure length reads are allowed; competing mutation is not.
	f.callBorrows = append(f.callBorrows, receiver)
	f.callAccesses = append(f.callAccesses, borrowCallAccess{value: receiver, mutable: mutable, span: span(f.module.file.Path, expression.Span)})
	defer func() { f.callBorrows = f.callBorrows[:base]; f.callAccesses = f.callAccesses[:accessBase] }()
	for _, bound := range expression.Children[1:] {
		if _, reachable = f.evaluateExpression(bound, environment); !reachable {
			return borrowValueState{}, false
		}
	}
	if mutable {
		receiver = cloneBorrowValue(receiver)
		for _, origin := range receiver.origins {
			receiver.pending = append(receiver.pending, pendingBorrowLoan{
				subview: true, owner: f.analyzer.resolved[assignmentRootPath(receiverExpression)],
				borrow: span(f.module.file.Path, expression.Span), kind: "mutable", origin: origin,
			})
		}
	}
	return receiver, true
}

// Subview generations distinguish descendants from ancestors and copied
// parent descriptors. Origins alone cannot distinguish those references.
func (f *borrowFlowAnalyzer) checkSubviewPathAccess(expression *ast.Expression, owner *symbol, environment borrowEnvironment) {
	if f.summarizing || f.probing() || owner == nil {
		return
	}
	access, found := environment[owner]
	if !found {
		access = borrowValueState{known: true, origins: []flowOrigin{f.analyzer.originForSymbol(owner)}}
	}
	primary := span(f.module.file.Path, expression.Span)
	report := func(loan, lastUse hir.Span) {
		for _, diagnostic := range f.analyzer.diagnostics {
			if diagnostic.Code == "E2101" && diagnostic.Primary == primary {
				return
			}
		}
		f.analyzer.addDiagnostic("E2101", primary, RelatedSpan{Role: "loan", Span: loan}, RelatedSpan{Role: "last-use", Span: lastUse})
	}
	check := func(retained borrowValueState, lastUse hir.Span) {
		if !borrowValuesOverlap(access, retained) {
			return
		}
		for _, id := range retained.loans {
			if id >= 0 && id < len(f.loans) && f.loans[id].subview && !slices.Contains(access.loans, id) {
				report(f.loans[id].fact.Borrow, lastUse)
			}
		}
		for _, pending := range retained.pending {
			if pending.subview && !slices.ContainsFunc(access.pending, func(candidate pendingBorrowLoan) bool {
				return candidate.subview && candidate.borrow == pending.borrow && sameFlowOrigin(candidate.origin, pending.origin)
			}) {
				report(pending.borrow, lastUse)
			}
		}
	}
	for binding, lastUse := range f.liveAfterExpression[expression] {
		if retained, exists := environment[binding]; exists {
			check(retained, lastUse)
		}
	}
	for _, held := range f.callBorrows {
		check(held, primary)
	}
}

func (f *borrowFlowAnalyzer) evaluateSubviewIndexStore(statement *ast.Statement, environment borrowEnvironment) bool {
	target := statement.Target
	if len(target.Children) != 2 {
		return true
	}
	if arrayFieldRoot(target.Children[0]) != nil && hir.ScalarArrayField(f.analyzer.expressionTypes[target.Children[0]]) {
		return f.evaluateArrayFieldStore(statement, environment)
	}
	receiver, reachable := f.evaluateExpression(target.Children[0], environment)
	if !reachable {
		return false
	}
	base := len(f.callBorrows)
	f.callBorrows = append(f.callBorrows, receiver)
	defer func() { f.callBorrows = f.callBorrows[:base] }()
	if _, reachable = f.evaluateExpression(target.Children[1], environment); !reachable {
		return false
	}
	if _, reachable = f.evaluateExpression(statement.Expression, environment); !reachable {
		return false
	}
	if owner := indexedArrayOwner(f.analyzer, target); owner != nil {
		f.checkOwnerAssignment(statement, owner, environment)
	}
	return true
}
