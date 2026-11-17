package semantic

import (
	"cmp"
	"math/bits"
	"slices"

	"github.com/0disoft/fairyc/internal/hir"
)

// borrowOwnerBindingIndex reverses the environment relation for owner
// conflict checks. A binding can acquire candidates in sibling forks, so the
// ordered set is deliberately cumulative; each check filters it against the
// environment and live snapshot for that exact program point.
type borrowOwnerBindingIndex map[*symbol]*borrowOwnerBindingSet

type borrowOwnerBindingSet struct {
	ordered   []*symbol
	positions map[*symbol]int
}

func (index *borrowOwnerBindingIndex) add(owner, binding *symbol) {
	if owner == nil || binding == nil {
		return
	}
	if *index == nil {
		*index = make(borrowOwnerBindingIndex)
	}
	set := (*index)[owner]
	if set == nil {
		set = &borrowOwnerBindingSet{positions: make(map[*symbol]int)}
		(*index)[owner] = set
	}
	if _, found := set.positions[binding]; found {
		return
	}
	set.positions[binding] = len(set.ordered)
	set.ordered = append(set.ordered, binding)
}

func (index borrowOwnerBindingIndex) candidates(owner *symbol) []*symbol {
	if owner == nil || index == nil || index[owner] == nil {
		return nil
	}
	return index[owner].ordered
}

func (index *borrowOwnerBindingIndex) addValue(value borrowValueState, binding *symbol, loans []borrowLoanRecord) {
	if binding == nil {
		return
	}
	for _, loanID := range value.loans {
		if loanID < 0 || loanID >= len(loans) {
			continue
		}
		index.add(loans[loanID].owner, binding)
	}
	for _, pending := range value.pending {
		index.add(pending.owner, binding)
	}
}

func (f *borrowFlowAnalyzer) visitLiveOwnerBindings(owner *symbol, live borrowLiveMap, environment borrowEnvironment,
	visit func(borrowValueState, hir.Span)) int {
	set := f.ownerBindings[owner]
	if owner == nil || set == nil {
		return 0
	}
	candidates := set.ordered
	inspected := len(candidates)
	// Prefer the live side only when scanning and sorting it is cheaper than
	// scanning the owner's cumulative history. Divide to avoid size overflow.
	if len(live) < len(candidates)/(bits.Len(uint(len(live)))+1) {
		var scratch [8]*symbol
		candidates = scratch[:0]
		for binding := range live {
			if _, found := set.positions[binding]; found {
				candidates = append(candidates, binding)
			}
		}
		// Map iteration must not reorder diagnostics or which report the budget
		// admits first. Restore the original owner-index insertion order.
		slices.SortFunc(candidates, func(left, right *symbol) int {
			return cmp.Compare(set.positions[left], set.positions[right])
		})
		inspected = len(live)
	}
	for _, binding := range candidates {
		lastUse, retained := live[binding]
		if !retained {
			continue
		}
		if value, found := environment[binding]; found {
			visit(value, lastUse)
		}
	}
	return inspected
}
