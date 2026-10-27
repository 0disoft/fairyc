package atlasmap

import (
	"container/heap"
	"iter"
)

// Each adjacency list is already in canonical relation order. Merge its unread
// suffix lazily so Around can stop at its budget without collecting every edge.
func (index *QueryIndex) frontierRelations(
	frontier []int,
	direction QueryDirection,
	filter map[RelationKind]struct{},
	seen map[int]struct{},
) iter.Seq[int] {
	return func(yield func(int) bool) {
		pending := make(relationFrontier, 0, len(frontier))
		for _, position := range frontier {
			symbol := index.nodes[position].symbol.Visible
			if values := index.outgoing[symbol]; len(values) != 0 && (direction == DirectionOutgoing || direction == DirectionBoth) {
				pending = append(pending, values)
			}
			if values := index.incoming[symbol]; len(values) != 0 && (direction == DirectionIncoming || direction == DirectionBoth) {
				pending = append(pending, values)
			}
		}
		heap.Init(&pending)
		previous := -1
		for len(pending) != 0 {
			relation := pending[0][0]
			if len(pending[0]) == 1 {
				// Remove the cursor in place; heap.Pop would box its slice
				// into an interface and allocate for every exhausted list.
				last := len(pending) - 1
				pending[0] = pending[last]
				pending[last] = nil
				pending = pending[:last]
			} else {
				pending[0] = pending[0][1:]
			}
			if len(pending) != 0 {
				heap.Fix(&pending, 0)
			}
			if relation == previous {
				continue // Self-calls and edges between frontier nodes occur twice.
			}
			previous = relation
			if _, selected := seen[relation]; selected {
				continue
			}
			if _, included := filter[index.relations[relation].Kind]; included && !yield(relation) {
				return
			}
		}
	}
}

// Heap entries borrow adjacency slices; only the local slice headers change.
type relationFrontier [][]int

func (values relationFrontier) Len() int           { return len(values) }
func (values relationFrontier) Less(i, j int) bool { return values[i][0] < values[j][0] }
func (values relationFrontier) Swap(i, j int)      { values[i], values[j] = values[j], values[i] }
func (values *relationFrontier) Push(value any)    { *values = append(*values, value.([]int)) }
func (values *relationFrontier) Pop() any {
	last := len(*values) - 1
	value := (*values)[last]
	(*values)[last] = nil
	*values = (*values)[:last]
	return value
}
