package languageserver

import (
	"container/heap"
	"sort"
)

const maxCompletions = 256

// Retain only the lexically first labels. The cutoff can only decrease, so an
// evicted label cannot re-enter, even when encountered again with new metadata.
// Selected duplicates still use the last declaration, as the original map did.
type completionCandidates struct {
	items  map[string]Completion
	labels completionLabels
}

func (c *completionCandidates) add(item Completion) {
	// Preserve direct map insertion until enough distinct labels need a cap.
	if len(c.labels) == 0 {
		if c.items == nil {
			c.items = make(map[string]Completion)
		}
		c.items[item.Label] = item
		if len(c.items) == maxCompletions {
			c.labels = make(completionLabels, 0, maxCompletions)
			for label := range c.items {
				c.labels = append(c.labels, label)
			}
			heap.Init(&c.labels)
		}
		return
	}
	if _, selected := c.items[item.Label]; selected {
		c.items[item.Label] = item
		return
	}
	if item.Label >= c.labels[0] {
		return
	}
	delete(c.items, c.labels[0])
	c.labels[0] = item.Label
	heap.Fix(&c.labels, 0)
	c.items[item.Label] = item
}

func (c *completionCandidates) sorted() []Completion {
	out := make([]Completion, 0, len(c.items))
	for _, item := range c.items {
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Label < out[j].Label })
	return out
}

// A max-heap exposes the next label to evict at index zero.
type completionLabels []string

func (h completionLabels) Len() int           { return len(h) }
func (h completionLabels) Less(i, j int) bool { return h[i] > h[j] }
func (h completionLabels) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *completionLabels) Push(value any)    { *h = append(*h, value.(string)) }
func (h *completionLabels) Pop() any {
	last := len(*h) - 1
	value := (*h)[last]
	(*h)[last] = ""
	*h = (*h)[:last]
	return value
}
