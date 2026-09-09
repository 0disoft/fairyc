package source

import "github.com/0disoft/fairyc/internal/diagnostic"

// Limits are implementation guards, not a stable language compatibility surface.
// They remain caller-configurable until measurements justify published defaults.
type Limits struct {
	MaxBytes       int
	MaxTokens      int
	MaxNodes       int
	MaxDepth       int
	MaxRepairs     int
	MaxDiagnostics int
	MaxWorkUnits   int
}

// DefaultLimits bounds one batch parse without claiming a released public limit.
func DefaultLimits() Limits {
	return Limits{
		MaxBytes:       8 << 20,
		MaxTokens:      1 << 20,
		MaxNodes:       1 << 20,
		MaxDepth:       512,
		MaxRepairs:     256,
		MaxDiagnostics: 256,
		MaxWorkUnits:   16 << 20,
	}
}

// Budget follows one immutable source through lexing, parsing, recovery, and lowering.
type Budget struct {
	limits      Limits
	tokens      int
	nodes       int
	depth       int
	repairs     int
	diagnostics int
	workUnits   int
	exhausted   bool
}

// NewBudget charges the source bytes before any token or node allocation.
func NewBudget(limits Limits, sourceBytes int) *Budget {
	budget := &Budget{limits: limits}
	if sourceBytes < 0 || sourceBytes > limits.MaxBytes {
		budget.exhausted = true
	}
	return budget
}

// Exhausted reports whether any shared counter crossed its configured limit.
func (b *Budget) Exhausted() bool {
	return b == nil || b.exhausted
}

// ChargeToken accounts for one real or synthetic token.
func (b *Budget) ChargeToken() bool {
	if b == nil || b.exhausted {
		return false
	}
	b.tokens++
	if b.tokens > b.limits.MaxTokens {
		b.exhausted = true
	}
	return !b.exhausted
}

// ChargeNode accounts for one CST node.
func (b *Budget) ChargeNode() bool {
	if b == nil || b.exhausted {
		return false
	}
	b.nodes++
	if b.nodes > b.limits.MaxNodes {
		b.exhausted = true
	}
	return !b.exhausted
}

// ChargeWork accounts for scanning, materialized decimal bytes and parser decisions.
// Negative/unrepresentable charges exhaust the budget rather than wrapping it.
func (b *Budget) ChargeWork(units int) bool {
	if b == nil || b.exhausted {
		return false
	}
	if units < 0 || b.workUnits > b.limits.MaxWorkUnits || units > b.limits.MaxWorkUnits-b.workUnits {
		b.exhausted = true
		return false
	}
	b.workUnits += units
	return true
}

// Enter charges one recursive grammar level.
func (b *Budget) Enter() bool {
	if b == nil || b.exhausted {
		return false
	}
	b.depth++
	if b.depth > b.limits.MaxDepth {
		b.exhausted = true
	}
	return !b.exhausted
}

// Leave releases one recursive grammar level.
func (b *Budget) Leave() {
	if b != nil && b.depth > 0 {
		b.depth--
	}
}

// ChargeRepair accounts for one parser recovery action.
func (b *Budget) ChargeRepair() bool {
	if b == nil || b.exhausted {
		return false
	}
	b.repairs++
	if b.repairs > b.limits.MaxRepairs {
		b.exhausted = true
	}
	return !b.exhausted
}

// ChargeDiagnostic accounts for one independently reported diagnostic.
func (b *Budget) ChargeDiagnostic() bool {
	if b == nil || b.exhausted {
		return false
	}
	b.diagnostics++
	if b.diagnostics > b.limits.MaxDiagnostics {
		b.exhausted = true
	}
	return !b.exhausted
}

// ExhaustionDiagnostic returns the stable failure reported when the shared budget is spent.
func ExhaustionDiagnostic(position int) diagnostic.Diagnostic {
	return diagnostic.Diagnostic{
		Code:   "E0010",
		Span:   diagnostic.Span{StartByte: position, EndByte: position},
		Repair: diagnostic.RepairNone,
	}
}
