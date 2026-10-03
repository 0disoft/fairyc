package semantic

import "github.com/0disoft/fairyc/internal/ast"

func (f *borrowFlowAnalyzer) evaluateFieldAssignment(statement *ast.Statement, environment borrowEnvironment) bool {
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
