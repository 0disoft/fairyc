package semantic

import (
	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/hir"
)

// Explicit borrows used only as call arguments still require compatible access.
// They stay temporary: do not materialize them as function-long local loans.
func (f *borrowFlowAnalyzer) checkBorrowCreation(expression *ast.Expression, environment borrowEnvironment) {
	if f.summarizing || f.probing() || !isBorrowExpression(expression) {
		return
	}
	owner := f.analyzer.resolved[borrowPlaceRoot(expression.Children[0])]
	if owner == nil {
		return
	}
	kind := borrowKind(expression.Text)
	seen := map[hir.Span]bool{}
	report := func(priorOwner *symbol, priorKind string, borrow, lastUse hir.Span) {
		if owner != priorOwner || kind == "shared" && priorKind == "shared" || seen[borrow] {
			return
		}
		seen[borrow] = true
		f.analyzer.addDiagnostic("E2101", span(f.module.file.Path, expression.Span),
			RelatedSpan{Role: "loan", Span: borrow}, RelatedSpan{Role: "last-use", Span: lastUse})
	}
	check := func(value borrowValueState, lastUse hir.Span) {
		for _, id := range value.loans {
			if id >= 0 && id < len(f.loans) {
				loan := f.loans[id]
				report(loan.owner, loan.fact.Kind, loan.fact.Borrow, lastUse)
			}
		}
		for _, pending := range value.pending {
			report(pending.owner, pending.kind, pending.borrow, lastUse)
		}
	}
	for binding, lastUse := range f.liveAfterExpression[expression] {
		if value, exists := environment[binding]; exists {
			check(value, lastUse)
		}
	}
	// Earlier arguments remain borrowed until the enclosing call completes even
	// if their source-variable last use appears before this later argument.
	for _, value := range f.callBorrows {
		check(value, span(f.module.file.Path, expression.Span))
	}
}

// A scalar store retains its target borrow through RHS evaluation. Reading the
// owner directly during that interval would bypass the exclusive reference.
func (f *borrowFlowAnalyzer) checkAssignmentOwnerRead(expression *ast.Expression, owner *symbol) {
	if f.summarizing || f.probing() || f.assignmentBorrow == nil || owner == nil {
		return
	}
	seen := map[hir.Span]bool{}
	report := func(priorOwner *symbol, kind string, borrow hir.Span) {
		if priorOwner != owner || kind != "mutable" || seen[borrow] {
			return
		}
		seen[borrow] = true
		f.analyzer.addDiagnostic("E2101", span(f.module.file.Path, expression.Span),
			RelatedSpan{Role: "loan", Span: borrow},
			RelatedSpan{Role: "last-use", Span: f.assignmentLastUse})
	}
	for _, id := range f.assignmentBorrow.loans {
		if id >= 0 && id < len(f.loans) {
			loan := f.loans[id]
			report(loan.owner, loan.fact.Kind, loan.fact.Borrow)
		}
	}
	for _, pending := range f.assignmentBorrow.pending {
		report(pending.owner, pending.kind, pending.borrow)
	}
}
