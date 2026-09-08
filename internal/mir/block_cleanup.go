package mir

import "github.com/0disoft/fairyc/internal/hir"

// Block results must be captured before cleanup can invalidate their backing
// resource. Region and loop callers retain their own cleanup/reset accounting.
func (l *functionLowerer) hasValueBlockCleanup(scope hir.Span) bool {
	for _, resource := range l.result.Resources {
		if sameSpanSource(resource.Binding, scope) && resource.Binding.StartByte >= scope.StartByte &&
			resource.Binding.EndByte <= scope.EndByte && l.live[resource.Symbol] {
			return true
		}
	}
	return false
}

func (l *functionLowerer) finishValueBlockCleanup(scope hir.Span) {
	if cleanupContainsDrop(l.emitCleanupDeclaredWithin(l.live, scope)) {
		l.result.EliminatedForms = append(l.result.EliminatedForms, EliminatedFormFact{Form: "implicit-drop", RemainingCount: 0})
	}
}
