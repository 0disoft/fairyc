package semantic

import "github.com/0disoft/fairyc/internal/hir"

// borrowOwnerBindingIndex reverses the environment relation for owner
// conflict checks. A binding can acquire candidates in sibling forks, so the
// ordered set is deliberately cumulative; each check filters it against the
// environment and live snapshot for that exact program point.
type borrowOwnerBindingIndex map[*symbol]*borrowOwnerBindingSet

type borrowOwnerBindingSet struct {
	ordered []*symbol
	seen    map[*symbol]struct{}
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
		set = &borrowOwnerBindingSet{seen: make(map[*symbol]struct{})}
		(*index)[owner] = set
	}
	if _, found := set.seen[binding]; found {
		return
	}
	set.seen[binding] = struct{}{}
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
	candidates := f.ownerBindings.candidates(owner)
	for _, binding := range candidates {
		lastUse, retained := live[binding]
		if !retained {
			continue
		}
		if value, found := environment[binding]; found {
			visit(value, lastUse)
		}
	}
	return len(candidates)
}
