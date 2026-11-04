package semantic

import (
	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/diagnostic"
)

// Summary inference and observable analysis share one allowance per function.
// Expressions cost one unit; block visits also charge incoming map entries.
const defaultBorrowWorkLimit = 1 << 20
const borrowWorkLimitFeature = "safe-borrow analysis work budget exceeded"

type borrowWorkBudget struct {
	remaining int
	exhausted bool
}

func (a *analyzer) borrowBudget(function *ast.Function) *borrowWorkBudget {
	if a.borrowWork == nil {
		a.borrowWork = make(map[*ast.Function]*borrowWorkBudget)
	}
	if budget := a.borrowWork[function]; budget != nil {
		return budget
	}
	limit := a.borrowWorkLimit
	if limit <= 0 {
		limit = defaultBorrowWorkLimit
	}
	budget := &borrowWorkBudget{remaining: limit}
	a.borrowWork[function] = budget
	return budget
}

func (f *borrowFlowAnalyzer) consumeBorrowWork(at diagnostic.Span, units int) bool {
	if f.analyzer.canceled() {
		return false
	}
	budget := f.analyzer.borrowBudget(f.function)
	if budget.exhausted {
		return false
	}
	if units < 0 || units > budget.remaining {
		budget.exhausted = true
		f.analyzer.addUnsupported(f.module.file.Path, at, borrowWorkLimitFeature)
		return false
	}
	budget.remaining -= units
	return true
}

func (a *analyzer) borrowWorkExhausted(function *ast.Function) bool {
	budget := a.borrowWork[function]
	return budget != nil && budget.exhausted
}
