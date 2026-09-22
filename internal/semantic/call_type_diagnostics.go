package semantic

import "github.com/0disoft/fairyc/internal/hir"

func (c *typeChecker) requireCallArgumentType(primary hir.Span, actual, expected hir.Type, expectedSpan hir.Span) {
	if actual == hir.TypeInvalid || expected == hir.TypeInvalid || actual == expected {
		return
	}
	c.analyzer.diagnostics = append(c.analyzer.diagnostics, Diagnostic{
		Code: "E1006", Primary: primary,
		Related:      []RelatedSpan{{Role: "expected-type", Span: expectedSpan}},
		ExpectedType: expected, ActualType: actual,
	})
}
