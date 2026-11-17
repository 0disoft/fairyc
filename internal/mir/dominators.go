package mir

// This verifier uses a dense fixed point, including unreachable components.
// Bound its quadratic table before allocation; never accept unchecked MIR when
// the analysis exceeds its budget.
const maxDominatorBytes = 64 << 20

func dominatorTableSize(count int) (words, cells int, ok bool) {
	if count < 0 {
		return 0, 0, false
	}
	words = count / 64
	if count%64 != 0 {
		words++
	}
	if words != 0 && count > (maxDominatorBytes/8)/words {
		return 0, 0, false
	}
	return words, count * words, true
}

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

// blockDominators returns nil if the dense table exceeds the verifier budget.
func blockDominators(function *Function, predecessors map[BlockID][]BlockID) *blockDominatorSets {
	indices := make(map[BlockID]int, len(function.Blocks))
	for _, block := range function.Blocks {
		if _, found := indices[block.ID]; !found {
			indices[block.ID] = len(indices)
		}
	}
	count := len(indices)
	words, cells, ok := dominatorTableSize(count)
	if !ok {
		return nil
	}
	sets := &blockDominatorSets{indices: indices, words: words, bits: make([]uint64, cells)}
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
	// Reuse one scratch row so each predecessor ID is resolved once per update,
	// rather than once per word. Never overwrite a row before reading self edges.
	next := make([]uint64, words)
	for changed := true; changed; {
		changed = false
		for _, id := range order {
			if id == function.Entry {
				continue
			}
			row := indices[id]
			sets.intersectPredecessors(next, predecessors[id])
			next[row/64] |= uint64(1) << uint(row%64)
			for word, value := range next {
				position := row*words + word
				if sets.bits[position] != value {
					sets.bits[position] = value
					changed = true
				}
			}
		}
	}
	return sets
}

func (sets *blockDominatorSets) intersectPredecessors(next []uint64, incoming []BlockID) {
	if len(incoming) == 0 {
		clear(next)
		return
	}
	for index, predecessor := range incoming {
		previous, found := sets.indices[predecessor]
		if !found {
			clear(next) // An unknown predecessor contributes the empty set.
			return
		}
		bits := sets.bits[previous*sets.words : (previous+1)*sets.words]
		if index == 0 {
			copy(next, bits)
			continue
		}
		for word, value := range bits {
			next[word] &= value
		}
	}
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
