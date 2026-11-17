package semantic

import (
	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/diagnostic"
)

// Summary inference and observable analysis share one allowance per function.
// Expressions cost one unit; block visits also charge incoming map entries.
// Liveness copies and joins charge every input entry before allocating a map.
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

func (f *borrowFlowAnalyzer) cloneBorrowLive(at diagnostic.Span, live borrowLiveMap) (borrowLiveMap, bool) {
	if !f.consumeBorrowWork(at, len(live)) {
		return nil, false
	}
	return cloneBorrowLive(live), true
}

func (f *borrowFlowAnalyzer) withoutBorrowLive(at diagnostic.Span, live borrowLiveMap, binding *symbol) borrowLiveMap {
	if _, found := live[binding]; !found {
		return live
	}
	result, ok := f.cloneBorrowLive(at, live)
	if !ok {
		return live
	}
	delete(result, binding)
	return result
}

func (f *borrowFlowAnalyzer) mergeBorrowLive(at diagnostic.Span, values ...borrowLiveMap) borrowLiveMap {
	// Charge separately to avoid overflowing a sum of input sizes. Reject the
	// whole join before allocation; a partial map cannot certify liveness.
	for _, value := range values {
		if !f.consumeBorrowWork(at, len(value)) {
			return nil
		}
	}
	return mergeBorrowLive(values...)
}
