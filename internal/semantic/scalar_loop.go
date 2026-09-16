package semantic

import (
	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/syntax"
)

// The closed scalar subset cannot create loans, modify borrow environments or
// capture loop exits. With no incoming borrow state its fixed point is empty.
// Only the probe is skipped: the ordinary observable pass still visits the body.
func (f *borrowFlowAnalyzer) canSkipScalarLoopProbe(e *ast.Expression, environment borrowEnvironment) bool {
	return !f.summarizing && len(environment) == 0 && len(f.loans) == 0 && len(f.callBorrows) == 0 && f.borrowNeutralScalar(e)
}

func (f *borrowFlowAnalyzer) borrowNeutralScalar(e *ast.Expression) bool {
	if f.neutralScalars == nil {
		f.neutralScalars = make(map[*ast.Expression]bool)
	}
	return f.analyzer.flowNeutralScalar(e, f.neutralScalars)
}

// A closed scalar body cannot produce, handle or transfer Result obligations,
// record result-handling facts, or capture loop exits. Keep the first visit and
// omit only its duplicate back-edge visit when the incoming state is empty.
func (a *analyzer) canSkipScalarResultBackEdge(e *ast.Expression, environment resultEnvironment) bool {
	if len(environment) != 0 {
		return false
	}
	if _, typed := a.expressionTypes[e]; !typed {
		return false // Do not extend this optimization to synthesized for loops.
	}
	if a.resultNeutralScalars == nil {
		a.resultNeutralScalars = make(map[*ast.Expression]bool)
	}
	return a.flowNeutralScalar(e, a.resultNeutralScalars)
}

// The typed AST is immutable during flow analysis. Each caller owns its memo;
// entries are bounded by visited expressions and live only for that analysis.
func (a *analyzer) flowNeutralScalar(e *ast.Expression, memo map[*ast.Expression]bool) bool {
	if e == nil {
		return false
	}
	if result, found := memo[e]; found {
		return result
	}
	// Default closed for unknown forms and recursive cycles.
	memo[e] = false
	plain := func(typ hir.Type) bool { return typ == hir.TypeI32 || typ == hir.TypeBool }
	children := -1
	switch e.Kind {
	case syntax.KindWhile:
		children = 2
	case syntax.KindBlock:
		if e.Block == nil || len(e.Children) != 0 {
			return false
		}
		for _, s := range e.Block.Statements {
			if s.For != nil {
				return false
			}
			switch s.Kind {
			case syntax.KindBinding:
				if s.Binding == nil {
					return false
				}
				binding := a.bindingSymbols[s.Binding]
				if binding == nil || !plain(binding.typ) || !a.flowNeutralScalar(s.Binding.Expression, memo) {
					return false
				}
			case syntax.KindAssignment:
				if s.Operator != "=" || s.Target == nil || s.Target.Kind != syntax.KindPath || !a.flowNeutralScalar(s.Target, memo) || !a.flowNeutralScalar(s.Expression, memo) {
					return false
				}
			case syntax.KindWhile:
				if !a.flowNeutralScalar(s.Expression, memo) {
					return false
				}
			default:
				return false
			}
		}
		if e.Block.Tail != nil && !a.flowNeutralScalar(e.Block.Tail, memo) {
			return false
		}
		children = 0
	case syntax.KindLiteral, syntax.KindPath:
		if !plain(a.expressionTypes[e]) {
			return false
		}
		children = 0
	case syntax.KindGroup, syntax.KindPrefix, syntax.KindBinary:
		if !plain(a.expressionTypes[e]) {
			return false
		}
		children = 1
		if e.Kind == syntax.KindPrefix && e.Text != "+" && e.Text != "-" {
			return false
		}
		if e.Kind == syntax.KindBinary {
			children = 2
			switch e.Text {
			case "+", "-", "*", "==", "!=", "<", "<=", ">", ">=":
			default:
				return false
			}
		}
	default:
		return false
	}
	if len(e.Children) != children {
		return false
	}
	for _, child := range e.Children {
		if !a.flowNeutralScalar(child, memo) {
			return false
		}
	}
	memo[e] = true
	return true
}
