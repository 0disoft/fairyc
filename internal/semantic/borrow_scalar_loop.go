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
	if e == nil {
		return false
	}
	if result, found := f.neutralScalars[e]; found {
		return result
	}
	if f.neutralScalars == nil {
		f.neutralScalars = make(map[*ast.Expression]bool)
	}
	// Default closed for unknown forms and recursive cycles.
	f.neutralScalars[e] = false
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
				binding := f.analyzer.bindingSymbols[s.Binding]
				if binding == nil || !plain(binding.typ) || !f.borrowNeutralScalar(s.Binding.Expression) {
					return false
				}
			case syntax.KindAssignment:
				if s.Operator != "=" || s.Target == nil || s.Target.Kind != syntax.KindPath || !f.borrowNeutralScalar(s.Target) || !f.borrowNeutralScalar(s.Expression) {
					return false
				}
			case syntax.KindWhile:
				if !f.borrowNeutralScalar(s.Expression) {
					return false
				}
			default:
				return false
			}
		}
		if e.Block.Tail != nil && !f.borrowNeutralScalar(e.Block.Tail) {
			return false
		}
		children = 0
	case syntax.KindLiteral, syntax.KindPath:
		if !plain(f.analyzer.expressionTypes[e]) {
			return false
		}
		children = 0
	case syntax.KindGroup, syntax.KindPrefix, syntax.KindBinary:
		if !plain(f.analyzer.expressionTypes[e]) {
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
		if !f.borrowNeutralScalar(child) {
			return false
		}
	}
	f.neutralScalars[e] = true
	return true
}
