package semantic

// Forks share immutable evidence. If every branch retains the same arrays, a
// canonical value can be copied once instead of concatenating and sorting it.
// Pending loans and ambiguous declaration ties retain the general merge path.
func cloneUnchangedBorrowJoin(values []borrowValueState) (borrowValueState, bool) {
	if len(values) < 2 {
		return borrowValueState{}, false
	}
	first := values[0]
	if len(first.pending) != 0 {
		return borrowValueState{}, false
	}
	for _, value := range values[1:] {
		if value.known != first.known || len(value.pending) != 0 ||
			len(value.origins) != len(first.origins) || len(value.loans) != len(first.loans) {
			return borrowValueState{}, false
		}
		if len(first.origins) != 0 && &value.origins[0] != &first.origins[0] {
			return borrowValueState{}, false
		}
		if len(first.loans) != 0 && &value.loans[0] != &first.loans[0] {
			return borrowValueState{}, false
		}
	}
	for index := 1; index < len(first.origins); index++ {
		if compareSpan(first.origins[index-1].declaration, first.origins[index].declaration) >= 0 {
			return borrowValueState{}, false
		}
	}
	for index := 1; index < len(first.loans); index++ {
		if first.loans[index-1] >= first.loans[index] {
			return borrowValueState{}, false
		}
	}
	// Keep the join independently writable and normalize empty evidence to nil,
	// exactly as the general concatenation path does.
	return borrowValueState{
		known:   first.known,
		origins: append([]flowOrigin(nil), first.origins...),
		loans:   append([]int(nil), first.loans...),
	}, true
}
