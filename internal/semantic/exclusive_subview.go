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
	f.checkHeldReferentRead(expression, expression.Children[0], receiver)
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
	ordinaryRead := expression != f.ownerReadPlace && !isSafeBorrowSymbol(owner)
	report := func(loan, lastUse hir.Span) {
		for _, diagnostic := range f.analyzer.diagnostics {
			if diagnostic.Code == "E2101" && diagnostic.Primary == primary {
				return
			}
			// Consuming resource expressions already have the more specific move
			// diagnostic. Do not also diagnose the root read inside that move.
			if diagnostic.Code == "E2105" && diagnostic.Primary.Source == primary.Source &&
				diagnostic.Primary.StartByte <= primary.StartByte && diagnostic.Primary.EndByte >= primary.EndByte {
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
			if id >= 0 && id < len(f.loans) && !slices.Contains(access.loans, id) {
				loan := f.loans[id]
				if loan.subview || ordinaryRead && loan.owner == owner && loan.fact.Kind == "mutable" {
					report(loan.fact.Borrow, lastUse)
				}
			}
		}
		for _, pending := range retained.pending {
			if (pending.subview || ordinaryRead && pending.owner == owner && pending.kind == "mutable") && !slices.ContainsFunc(access.pending, func(candidate pendingBorrowLoan) bool {
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

// Only the selected root node is exempt. An index callback that reads the
// same variable has a different source node and still checks its live loans.
func (f *borrowFlowAnalyzer) evaluateOwnerPlace(expression *ast.Expression, environment borrowEnvironment) (borrowValueState, bool) {
	prior := f.ownerReadPlace
	f.ownerReadPlace = fieldAssignmentRoot(expression)
	defer func() { f.ownerReadPlace = prior }()
	return f.evaluateExpression(expression, environment)
}

// Closed storage constructors obtain data/length metadata from the selected
// owner during the same acquisition. Exempt only a direct field projection
// of that owner; effectful metadata expressions and other owners remain reads.
func (f *borrowFlowAnalyzer) evaluateConstructorMetadata(expression, ownerExpression *ast.Expression, environment borrowEnvironment) (borrowValueState, bool) {
	if isBorrowExpression(ownerExpression) {
		ownerExpression = ownerExpression.Children[0]
	}
	owner := fieldAssignmentRoot(ownerExpression)
	root := fieldAssignmentRoot(expression)
	if owner != nil && root != nil && root != assignmentRootPath(expression) && f.analyzer.resolved[owner] == f.analyzer.resolved[root] {
		return f.evaluateOwnerPlace(expression, environment)
	}
	return f.evaluateExpression(expression, environment)
}

func (f *borrowFlowAnalyzer) evaluateSubviewIndexStore(statement *ast.Statement, environment borrowEnvironment) bool {
	target := statement.Target
	if len(target.Children) != 2 {
		return true
	}
	if arrayFieldRoot(target.Children[0]) != nil && f.analyzer.inlineArrayField(f.analyzer.expressionTypes[target.Children[0]]) {
		return f.evaluateArrayFieldStore(statement, environment)
	}
	receiver, reachable := f.evaluateOwnerPlace(target.Children[0], environment)
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
