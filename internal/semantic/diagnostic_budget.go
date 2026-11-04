package semantic

import "github.com/0disoft/fairyc/internal/hir"

// DefaultSemanticReportLimit is the default combined limit for admitted
// semantic diagnostics and unsupported facts.
const DefaultSemanticReportLimit = 256

// SemanticReportExhaustedCode is the terminal marker emitted when the semantic
// report budget omits later diagnostics or unsupported facts.
const SemanticReportExhaustedCode = "E0011"

type analysisOptions struct {
	reportLimit    int
	resourceStates bool
}

// diagnosticBudget admits diagnostics and unsupported facts as one bounded
// reporting stream. The marker is emitted once and does not consume an admitted
// fact slot, so a complete report contains at most limit admitted facts plus
// the terminal E0011 marker.
type diagnosticBudget struct {
	limit     int
	admitted  int
	exhausted bool
}

func newDiagnosticBudget(limit int) diagnosticBudget {
	if limit <= 0 {
		limit = DefaultSemanticReportLimit
	}
	return diagnosticBudget{limit: limit}
}

func (b *diagnosticBudget) admit() (admitted, marker bool) {
	if b.limit <= 0 {
		b.limit = DefaultSemanticReportLimit
	}
	if b.exhausted {
		return false, false
	}
	if b.admitted < b.limit {
		b.admitted++
		return true, false
	}
	b.exhausted = true
	return false, true
}

func semanticReportExhaustedDiagnostic(primary hir.Span) Diagnostic {
	return Diagnostic{Code: SemanticReportExhaustedCode, Primary: primary}
}
