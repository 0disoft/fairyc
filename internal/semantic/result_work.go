package semantic

import "github.com/0disoft/fairyc/internal/diagnostic"

const defaultResultWorkLimit = 1 << 20
const resultWorkLimitFeature = "Result analysis work budget exceeded"

type resultWorkBudget struct {
	remaining int
	exhausted bool
}

func newResultWorkBudget(limit int) *resultWorkBudget {
	if limit <= 0 {
		limit = defaultResultWorkLimit
	}
	return &resultWorkBudget{remaining: limit}
}

// All branches and back-edge visits share the current function's allowance.
// Blocks charge incoming map entries too, before their environment is traversed.
func (a *analyzer) consumeResultWork(source string, at diagnostic.Span, units int) bool {
	if a.canceled() {
		return false
	}
	if a.resultWork == nil {
		a.resultWork = newResultWorkBudget(a.resultWorkLimit)
	}
	budget := a.resultWork
	if budget.exhausted {
		return false
	}
	if units < 0 || units > budget.remaining {
		budget.exhausted = true
		a.addUnsupported(source, at, resultWorkLimitFeature)
		return false
	}
	budget.remaining -= units
	return true
}
