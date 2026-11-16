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
	root := borrowPlaceRoot(expression.Children[0])
	if index := indexedPlace(root); index != nil {
		if _, _, array := hir.FixedArrayParts(f.analyzer.expressionTypes[index.Children[0]]); array {
			root = assignmentRootPath(index.Children[0])
		}
	}
	owner := f.analyzer.resolved[root]
	f.checkNamedBorrowCreation(expression, owner, borrowKind(expression.Text), environment)
}

// Implicit indexed stores use the same live-loan checks as explicit borrows.
// The source node supplies expression liveness, including index and RHS uses.
func (f *borrowFlowAnalyzer) checkNamedBorrowCreation(expression *ast.Expression, owner *symbol, kind string, environment borrowEnvironment) {
	if f.summarizing || f.probing() || owner == nil {
		return
	}
	seen := map[hir.Span]bool{}
	check := func(value borrowValueState, lastUse hir.Span) {
		for _, borrow := range borrowCreationConflicts(value, f.loans, owner, kind) {
			if seen[borrow] {
				continue
			}
			seen[borrow] = true
			f.analyzer.addDiagnostic("E2101", span(f.module.file.Path, expression.Span),
				RelatedSpan{Role: "loan", Span: borrow}, RelatedSpan{Role: "last-use", Span: lastUse})
		}
	}
	f.visitLiveOwnerBindings(owner, f.liveAfterExpression[expression], environment, check)
	// Earlier arguments remain borrowed until the enclosing call completes even
	// if their source-variable last use appears before this later argument.
	for _, value := range f.callBorrows {
		check(value, span(f.module.file.Path, expression.Span))
	}
}

// Conflict selection only reads its inputs. The returned slice belongs to the
// caller; valid shared accesses and unrelated owners require no allocation.
func borrowCreationConflicts(value borrowValueState, loans []borrowLoanRecord, owner *symbol, kind string) []hir.Span {
	var conflicts []hir.Span
	for _, id := range value.loans {
		if id < 0 || id >= len(loans) {
			continue
		}
		loan := loans[id]
		if loan.owner != owner || kind == "shared" && loan.fact.Kind == "shared" {
			continue
		}
		conflicts = append(conflicts, loan.fact.Borrow)
	}
	for _, pending := range value.pending {
		if pending.owner != owner || kind == "shared" && pending.kind == "shared" {
			continue
		}
		conflicts = append(conflicts, pending.borrow)
	}
	return conflicts
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
