package mir

// blockDominatorSets stores one bit per candidate instead of a hash entry.
// Dense indices keep storage independent of the numeric block IDs.
type blockDominatorSets struct {
	indices map[BlockID]int
	words   int
	bits    []uint64
}

func (sets *blockDominatorSets) contains(target, candidate BlockID) bool {
	row, targetFound := sets.indices[target]
	column, candidateFound := sets.indices[candidate]
	return targetFound && candidateFound && sets.bits[row*sets.words+column/64]&(uint64(1)<<uint(column%64)) != 0
}

func blockDominators(function *Function, predecessors map[BlockID][]BlockID) *blockDominatorSets {
	indices := make(map[BlockID]int, len(function.Blocks))
	for _, block := range function.Blocks {
		if _, found := indices[block.ID]; !found {
			indices[block.ID] = len(indices)
		}
	}
	count := len(indices)
	words := (count + 63) / 64
	sets := &blockDominatorSets{indices: indices, words: words, bits: make([]uint64, count*words)}
	for row := range count {
		for word := range words {
			sets.bits[row*words+word] = ^uint64(0)
		}
		if remainder := count % 64; remainder != 0 {
			sets.bits[(row+1)*words-1] = (uint64(1) << uint(remainder)) - 1
		}
	}
	if entry, found := indices[function.Entry]; found {
		clear(sets.bits[entry*words : (entry+1)*words])
		sets.bits[entry*words+entry/64] = uint64(1) << uint(entry%64)
	}
	order := dominatorBlockOrder(function, indices, predecessors)
	// Preserve the original fixed point, including unreachable components.
	// Words are independent, so updates can reuse each row without a scratch set.
	for changed := true; changed; {
		changed = false
		for _, id := range order {
			if id == function.Entry {
				continue
			}
			row := indices[id]
			incoming := predecessors[id]
			for word := range words {
				next := uint64(0)
				if len(incoming) > 0 {
					next = ^uint64(0)
					for _, predecessor := range incoming {
						previous, found := indices[predecessor]
						if !found {
							next = 0
							break
						}
						next &= sets.bits[previous*words+word]
					}
				}
				if word == row/64 {
					next |= uint64(1) << uint(row%64)
				}
				position := row*words + word
				if sets.bits[position] != next {
					sets.bits[position] = next
					changed = true
				}
			}
		}
	}
	return sets
}

// Visit predecessors before their users instead of depending on block storage
// order. DAGs then settle in one update pass plus confirmation. Cycles still use
// the same fixed point, and disconnected components retain their old semantics.
func dominatorBlockOrder(function *Function, indices map[BlockID]int, predecessors map[BlockID][]BlockID) []BlockID {
	seen := make([]bool, len(indices))
	order := make([]BlockID, 0, len(indices))
	if entry, found := indices[function.Entry]; found {
		seen[entry] = true // The entry row is constant, even with incoming backedges.
		order = append(order, function.Entry)
	}
	type frame struct {
		id   BlockID
		next int
	}
	var stack []frame
	for _, block := range function.Blocks {
		if seen[indices[block.ID]] {
			continue
		}
		seen[indices[block.ID]] = true
		stack = append(stack, frame{id: block.ID})
		for len(stack) > 0 {
			current := &stack[len(stack)-1]
			incoming := predecessors[current.id]
			if current.next < len(incoming) {
				predecessor := incoming[current.next]
				current.next++
				if row, found := indices[predecessor]; found && !seen[row] {
					seen[row] = true
					stack = append(stack, frame{id: predecessor})
				}
				continue
			}
			order = append(order, current.id)
			stack = stack[:len(stack)-1]
		}
	}
	return order
}
