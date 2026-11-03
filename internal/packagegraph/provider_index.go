package packagegraph

import "math/bits"

// Build only this class's requested paths. In a dense import set, scanning each
// visible module once is cheaper than repeating binary search in every visible
// class for every distinct import. A sparse set retains the lazy search cache.
func prepareVisibleProviderLookup(graph Graph, classes []resolvedClass, visible []int, modules []Module) map[string][]int {
	providers := make(map[string][]int)
	for _, module := range modules {
		for _, path := range module.Imports {
			providers[path] = nil
		}
	}
	if len(providers) == 0 {
		return providers
	}
	moduleCount, searchWork := 0, 0
	for _, index := range visible {
		catalog := graph.Nodes[classes[index].nodes[0]].Modules
		moduleCount += len(catalog)
		searchWork += bits.Len(uint(len(catalog)))
	}
	// Compare work estimates without multiplying counts (and risking overflow).
	if searchWork == 0 || len(providers) <= moduleCount/searchWork {
		clear(providers)
		return providers
	}
	// Visit the same self-first, then canonical dependency order as providerClasses.
	// Never iterate the map to emit providers or source import receipts.
	for _, index := range visible {
		for _, module := range graph.Nodes[classes[index].nodes[0]].Modules {
			if values, requested := providers[module.Path]; requested {
				providers[module.Path] = append(values, index)
			}
		}
	}
	return providers
}
