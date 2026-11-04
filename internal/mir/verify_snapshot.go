package mir

// Bound cumulative map entries copied for joins/forks. Straight-line blocks
// transfer ownership and do not consume this budget.
const maxVerificationSnapshotEntries = 1 << 20

func cloneVerificationSnapshot(state verificationState, remaining *int) (verificationState, bool) {
	needed := len(state.live) + len(state.flags) + len(state.regions)
	if needed < 0 || needed > *remaining {
		return verificationState{}, false
	}
	*remaining -= needed
	return cloneVerificationState(state), true
}

func verificationJoinPoints(function *Function) []bool {
	incoming := make([]uint8, len(function.Blocks))
	if function.Entry >= 0 && int(function.Entry) < len(incoming) {
		incoming[function.Entry] = 1
	}
	for _, block := range function.Blocks {
		for _, successor := range block.Terminator.Successors {
			if successor >= 0 && int(successor) < len(incoming) && incoming[successor] < 2 {
				incoming[successor]++
			}
		}
	}
	joins := make([]bool, len(incoming))
	for index, count := range incoming {
		joins[index] = count > 1
	}
	return joins
}
