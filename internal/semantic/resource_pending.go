package semantic

import "github.com/0disoft/fairyc/internal/hir"

// These values have left their source owners but have not yet been installed
// into the enclosing record or passed to its callee. An abrupt exit drops them.
type resourcePendingValue struct {
	typ        hir.Type
	scopeDepth int
}

func (a *analyzer) retainResourcePending(typ hir.Type, mode resourceUse) {
	if mode == resourceMove && a.isResource(typ) {
		a.resourcePending = append(a.resourcePending, resourcePendingValue{typ, len(a.resourceScopes)})
	}
}

func (a *analyzer) checkResourcePendingEffects(at hir.Span, scopeDepth int, seen map[*symbol]struct{}) {
	if scopeDepth < 0 {
		return
	}
	for _, value := range a.resourcePending {
		if value.scopeDepth >= scopeDepth {
			a.checkResourceValueEffectsOnce(&symbol{typ: value.typ}, at, seen)
		}
	}
}
