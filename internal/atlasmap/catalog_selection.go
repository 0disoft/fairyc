package atlasmap

import "slices"

// A nil positions slice represents the admitted node range [0, count).
// Filtered positions are also increasing because filtering preserves node order.
// Both forms are call-local views over the immutable QueryIndex.
type catalogSelection struct {
	positions []int
	count     int
}

func (selection catalogSelection) at(index int) int {
	if selection.positions == nil {
		return index
	}
	return selection.positions[index]
}

func (selection catalogSelection) cursor(position int) (int, bool) {
	if selection.positions == nil {
		return position, position >= 0 && position < selection.count
	}
	return slices.BinarySearch(selection.positions, position)
}
