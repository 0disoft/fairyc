package semantic

import "github.com/0disoft/fairyc/internal/hir"

func (c *typeChecker) requireDetailedType(primary hir.Span, actual, expected hir.Type, expectedSpan hir.Span) {
	if actual == hir.TypeInvalid || expected == hir.TypeInvalid || actual == expected {
		return
	}
	related := []RelatedSpan{{Role: "expected-type", Span: expectedSpan}}
	for _, entry := range []struct {
		typ  hir.Type
		role string
	}{
		{expected, "expected-type-declaration"},
		{actual, "actual-type-declaration"},
	} {
		if info := c.analyzer.structTypes[entry.typ]; info != nil {
			related = append(related, RelatedSpan{
				Role: entry.role,
				Span: span(info.symbol.source, info.symbol.declaration),
			})
		}
	}
	c.analyzer.diagnostics = append(c.analyzer.diagnostics, Diagnostic{
		Code: "E1006", Primary: primary,
		Related:      related,
		ExpectedType: expected, ActualType: actual,
	})
}
