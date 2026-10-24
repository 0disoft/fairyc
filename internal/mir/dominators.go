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
	// Preserve the original fixed point, including unreachable components.
	// Words are independent, so updates can reuse each row without a scratch set.
	for changed := true; changed; {
		changed = false
		for _, block := range function.Blocks {
			if block.ID == function.Entry {
				continue
			}
			row := indices[block.ID]
			incoming := predecessors[block.ID]
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
