package semantic

import "github.com/0disoft/fairyc/internal/hir"

// Keep first-occurrence order for diagnostic evidence. Small obligations need
// no index; larger unions share one index across every branch of the merge.
type resultProducerSet struct {
	values  []hir.Span
	seen    map[hir.Span]struct{}
	indexed bool
}

func (set *resultProducerSet) add(values ...hir.Span) {
	for _, value := range values {
		if !set.indexed {
			duplicate := false
			for _, existing := range set.values {
				if existing == value {
					duplicate = true
					break
				}
			}
			if duplicate {
				continue
			}
			if len(set.values) == 8 {
				if set.seen == nil {
					set.seen = make(map[hir.Span]struct{})
				} else {
					clear(set.seen)
				}
				set.indexed = true
				for _, existing := range set.values {
					set.seen[existing] = struct{}{}
				}
			}
		} else if _, found := set.seen[value]; found {
			continue
		}
		if set.indexed {
			set.seen[value] = struct{}{}
		}
		set.values = append(set.values, value)
	}
}
