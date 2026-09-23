package cbackend

import "github.com/0disoft/fairyc/internal/mir"

// reachableBlocksForC omits MIR blocks that have no executable path from the
// entry. MIR verification and canonical block-ID checks run before emission.
func reachableBlocksForC(function mir.Function) []bool {
	reachable := make([]bool, len(function.Blocks))
	queue := []mir.BlockID{function.Entry}
	for len(queue) != 0 {
		id := queue[0]
		queue = queue[1:]
		index := int(id)
		if index < 0 || index >= len(function.Blocks) || reachable[index] {
			continue
		}
		block := &function.Blocks[index]
		reachable[index] = true
		queue = append(queue, block.Terminator.Successors...)
		if block.Terminator.UnwindSuccessor != mir.NoBlock {
			queue = append(queue, block.Terminator.UnwindSuccessor)
		}
	}
	return reachable
}
