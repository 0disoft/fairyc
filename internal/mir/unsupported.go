package mir

import "github.com/0disoft/fairyc/internal/hir"

// Unsupported reports a typed source form with no executable lowering yet.
// It is distinct from malformed HIR and internal lowering failures.
type Unsupported struct {
	Feature string
	Span    hir.Span
}

func (e *Unsupported) Error() string { return e.Feature }
