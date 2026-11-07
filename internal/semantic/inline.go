package semantic

import (
	"strings"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/syntax"
)

func (c *typeChecker) checkInlineCallPath(callee *ast.Expression) bool {
	if c.function == nil {
		return true
	}
	for _, attribute := range c.function.Attributes {
		if attribute.Name == "inline" && (callee == nil || callee.Kind != syntax.KindPath || strings.Contains(callee.Text, "::")) {
			c.analyzer.addUnsupported(c.module.file.Path, c.function.Span, "@inline calls require a bare same-module declaration name")
			return false
		}
	}
	return true
}

func (a *analyzer) finalizeInlineFunctions(typed *hir.Module) {
	// Membership comes from checked declarations, not spellings in a call path.
	declarations := make(map[hir.SymbolID]*hir.Function)
	for i := range typed.Functions {
		fn := &typed.Functions[i]
		if fn.InlineBody != "" {
			declarations[fn.Symbol] = fn
		}
	}
	resolve := func(symbol hir.SymbolID) *hir.Function { return declarations[symbol] }
	identities := make(map[hir.SymbolID]string)
	for i := range typed.Functions {
		fn := &typed.Functions[i]
		if _, requested := declarations[fn.Symbol]; !requested {
			continue
		}
		id, err := hir.InlineFunctionID(*fn, resolve)
		if err != nil {
			a.addUnsupportedRecord(Unsupported{Feature: err.Error(), Span: fn.Span})
			continue
		}
		identities[fn.Symbol] = id
	}
	for i := range typed.Functions {
		fn := &typed.Functions[i]
		if _, requested := declarations[fn.Symbol]; requested {
			fn.InlineBody = identities[fn.Symbol]
		}
	}
}
