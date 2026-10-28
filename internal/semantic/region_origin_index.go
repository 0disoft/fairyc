package semantic

import "github.com/0disoft/fairyc/internal/hir"

// Match sameFlowOrigin: allocation sites do not distinguish lexical origins.
type flowOriginKey struct {
	name        string
	declaration hir.Span
}

type regionOriginIndex struct {
	lastUses map[*symbol]hir.Span
	escaping map[flowOriginKey]struct{}
}

// Build only for functions that emit Region facts, after bindings and origins
// are resolved. Uses and returns are scanned once, rather than per allocation.
func (a *analyzer) indexRegionOrigins(module *moduleScope, inventory *flowInventory, origins map[*symbol]flowOrigin) regionOriginIndex {
	index := regionOriginIndex{lastUses: make(map[*symbol]hir.Span)}
	after := make(map[*symbol]int)
	for _, expression := range inventory.expressions {
		if !a.isRegionAllocationCall(expression) {
			continue
		}
		if binding := a.bindingForExpression(expression); binding != nil {
			after[binding] = expression.Span.EndByte
		}
	}
	for _, expression := range inventory.expressions {
		binding := a.resolved[expression]
		end, found := after[binding]
		if !found || expression.Span.StartByte < end {
			continue
		}
		use := span(binding.source, expression.Span)
		if previous, found := index.lastUses[binding]; !found || use.EndByte > previous.EndByte {
			index.lastUses[binding] = use
		}
	}
	for _, returned := range inventory.returns {
		for _, origin := range a.expressionOrigins(module, returned, origins) {
			if index.escaping == nil {
				index.escaping = make(map[flowOriginKey]struct{})
			}
			index.escaping[flowOriginKey{name: origin.name, declaration: origin.declaration}] = struct{}{}
		}
	}
	return index
}
