package semantic

import "github.com/0disoft/fairyc/internal/ast"

func (f *borrowFlowAnalyzer) evaluateFieldAssignment(statement *ast.Statement, environment borrowEnvironment) bool {
	if index := indexedPlace(statement.Target); index != nil {
		var held borrowValueState
		var reachable bool
		if indexedArrayOwner(f.analyzer, index) != nil {
			// Select original array storage and hold the whole owner until store.
			// The parent node's liveness includes loans last used in index or RHS.
			held, reachable = f.evaluateArrayPlace(assignmentRootPath(index.Children[0]), index, true, environment)
		} else {
			held, reachable = f.evaluateIndexedPlace(index, true, false, environment)
		}
		if !reachable {
			return false
		}
		base, accessBase := len(f.callBorrows), len(f.callAccesses)
		priorBorrow, priorLastUse := f.assignmentBorrow, f.assignmentLastUse
		f.callBorrows = append(f.callBorrows, held)
		f.callAccesses = append(f.callAccesses, borrowCallAccess{value: held, mutable: true, span: span(f.module.file.Path, statement.Target.Span)})
		f.assignmentBorrow = &held
		f.assignmentLastUse = span(f.module.file.Path, statement.Target.Span)
		defer func() {
			f.callBorrows = f.callBorrows[:base]
			f.callAccesses = f.callAccesses[:accessBase]
			f.assignmentBorrow = priorBorrow
			f.assignmentLastUse = priorLastUse
		}()
		_, reachable = f.evaluateExpression(statement.Expression, environment)
		return reachable
	}
	root := fieldAssignmentRoot(statement.Target)
	binding := f.analyzer.resolved[root]
	if _, reachable := f.evaluateExpression(statement.Target, environment); !reachable {
		return false
	}
	// A selected reference receiver still owns its loan until the final store,
	// even when a var reference is rebound while evaluating the RHS.
	base := len(f.callBorrows)
	priorBorrow, priorLastUse := f.assignmentBorrow, f.assignmentLastUse
	if held, ok := environment[binding]; isSafeBorrowSymbol(binding) && ok {
		held = cloneBorrowValue(held)
		f.callBorrows = append(f.callBorrows, held)
		f.assignmentBorrow = &held
		f.assignmentLastUse = span(f.module.file.Path, statement.Target.Span)
	}
	_, reachable := f.evaluateExpression(statement.Expression, environment)
	f.assignmentBorrow, f.assignmentLastUse = priorBorrow, priorLastUse
	f.callBorrows = f.callBorrows[:base]
	if reachable {
		f.checkOwnerAssignment(statement, binding, environment)
	}
	return reachable
}
