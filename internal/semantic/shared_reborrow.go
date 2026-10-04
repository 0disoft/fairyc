package semantic

import (
	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/syntax"
)

// Calls may shorten an exclusive reference to the same shared referent.
// Explicit plain-field projection has its separate ADR 0446 admission.
// Bindings, returns, casts and tagged construction retain exact-type rules.
func (c *typeChecker) sharedCallArgument(source **ast.Expression, child *hir.Expression, actual, expected hir.Type) (*hir.Expression, hir.Type) {
	element, mutable, borrow := hir.BorrowElement(actual)
	if child == nil || !borrow || !mutable || expected != hir.SharedBorrow(element) {
		return child, actual
	}
	original := *source
	*source = &ast.Expression{Kind: ast.PrivateSharedReborrow, Span: original.Span, Children: []*ast.Expression{original}}
	c.analyzer.expressionTypes[*source] = expected
	return &hir.Expression{Kind: string(ast.PrivateSharedReborrow), Type: expected, Span: child.Span, Symbol: hir.NoSymbol, Children: []hir.Expression{*child}}, expected
}

type borrowCallAccess struct {
	value   borrowValueState
	mutable bool
	span    hir.Span
}

func borrowValuesOverlap(left, right borrowValueState) bool {
	for _, a := range left.origins {
		for _, b := range right.origins {
			if sameFlowOrigin(a, b) {
				return true
			}
		}
	}
	return false
}

// Shared children preserve root loans and also freeze their named reference
// parent. This parent loan handles references received as parameters, which
// have an origin but no local root loan.
func (f *borrowFlowAnalyzer) sharedReborrowValue(expression *ast.Expression, value borrowValueState) borrowValueState {
	if len(expression.Children) != 1 {
		return borrowValueState{known: false}
	}
	return f.retainSharedParent(expression.Children[0], span(f.module.file.Path, expression.Span), value)
}

// Retain named-parent loans for shared children. Reference descriptor rebinding
// keeps the child's selected origin; referent writes are checked by origin.
func (f *borrowFlowAnalyzer) retainSharedParent(parent *ast.Expression, at hir.Span, value borrowValueState) borrowValueState {
	value = cloneBorrowValue(value)
	parent = assignmentRootPath(parent)
	if parent != nil && parent.Kind == syntax.KindPath {
		owner := f.analyzer.resolved[parent]
		for _, origin := range value.origins {
			value.pending = append(value.pending, pendingBorrowLoan{owner: owner, borrow: at, kind: "shared", origin: origin})
		}
	}
	return value
}

func (f *borrowFlowAnalyzer) checkReferenceAccess(expression *ast.Expression, value borrowValueState, mutable bool, environment borrowEnvironment, primary hir.Span) {
	if f.summarizing || f.probing() || !value.known {
		return
	}
	seen := make(map[hir.Span]bool)
	report := func(loan, lastUse hir.Span) {
		if seen[loan] {
			return
		}
		seen[loan] = true
		for _, diagnostic := range f.analyzer.diagnostics {
			// A rejected earlier argument already explains this call conflict.
			if diagnostic.Code == "E2101" && (diagnostic.Primary == primary || diagnostic.Primary == loan) {
				return
			}
		}
		f.analyzer.addDiagnostic("E2101", primary, RelatedSpan{Role: "loan", Span: loan}, RelatedSpan{Role: "last-use", Span: lastUse})
	}
	for _, prior := range f.callAccesses {
		if (mutable || prior.mutable) && borrowValuesOverlap(value, prior.value) {
			report(prior.span, span(f.module.file.Path, expression.Span))
		}
	}
	if !mutable {
		return
	}
	for binding, lastUse := range f.liveAfterExpression[expression] {
		_, exclusive, leaves := safeBorrowLeaf(binding.typ)
		if leaves == 0 || exclusive {
			continue
		}
		retained, found := environment[binding]
		if found && borrowValuesOverlap(value, retained) {
			report(span(binding.source, binding.declaration), lastUse)
		}
	}
}

func (f *borrowFlowAnalyzer) checkCallReferenceArgument(call *ast.Expression, index int, argument *ast.Expression, value borrowValueState, environment borrowEnvironment) {
	callee := f.analyzer.callableForCall(call)
	if callee == nil || callee.function == nil || index >= len(callee.function.Parameters) {
		return
	}
	parameter := f.analyzer.parameter[&callee.function.Parameters[index]]
	if parameter == nil {
		return
	}
	_, mutable, borrowed := hir.BorrowElement(parameter.typ)
	if !borrowed {
		return
	}
	f.checkReferenceAccess(argument, value, mutable, environment, span(f.module.file.Path, argument.Span))
	f.callAccesses = append(f.callAccesses, borrowCallAccess{value: value, mutable: mutable, span: span(f.module.file.Path, argument.Span)})
}

func (f *borrowFlowAnalyzer) checkReferenceStore(target *ast.Expression, environment borrowEnvironment) {
	if target == nil || target.Kind == syntax.KindPath {
		return
	}
	primary := span(f.module.file.Path, target.Span)
	if index := indexedPlace(target); index != nil {
		target = index.Children[0]
	}
	for len(target.Children) > 0 && (target.Kind == syntax.KindPrefix || target.Kind == syntax.KindFieldExpr || target.Kind == syntax.KindIndex || target.Kind == syntax.KindGroup) {
		target = target.Children[0]
	}
	root := assignmentRootPath(target)
	if binding := f.analyzer.resolved[root]; isSafeBorrowSymbol(binding) {
		if value, found := environment[binding]; found {
			f.checkReferenceAccess(root, value, true, environment, primary)
		}
	}
}

// An earlier argument remains held while later argument blocks execute.
func (f *borrowFlowAnalyzer) checkHeldOwnerWrite(statement *ast.Statement, owner *symbol) {
	for _, held := range f.callBorrows {
		for _, id := range held.loans {
			if id >= 0 && id < len(f.loans) && f.loans[id].owner == owner {
				f.reportOwnerWrite(statement, f.loans[id].fact.Borrow, span(f.module.file.Path, statement.Target.Span))
			}
		}
		for _, pending := range held.pending {
			if pending.owner == owner {
				f.reportOwnerWrite(statement, pending.borrow, span(f.module.file.Path, statement.Target.Span))
			}
		}
	}
}

func (f *borrowFlowAnalyzer) reportOwnerWrite(statement *ast.Statement, loan, lastUse hir.Span) {
	primary := span(f.module.file.Path, statement.Target.Span)
	for _, diagnostic := range f.analyzer.diagnostics {
		if diagnostic.Code == "E2101" && diagnostic.Primary == primary {
			return
		}
	}
	f.analyzer.addDiagnostic("E2101", primary, RelatedSpan{Role: "loan", Span: loan}, RelatedSpan{Role: "last-use", Span: lastUse})
}
