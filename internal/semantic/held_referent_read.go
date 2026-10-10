package semantic

import (
	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/hir"
)

// Reading a reference descriptor is distinct from reading its referent. Calls
// hold exclusive arguments through later argument evaluation, including reads
// through aliases. An ordinary selected store may read its own RHS storage.
func (f *borrowFlowAnalyzer) checkHeldReferentRead(expression, receiver *ast.Expression, value borrowValueState) {
	if f.summarizing || f.probing() || !value.known || receiver == f.ownerReadPlace {
		return
	}
	primary := span(f.module.file.Path, expression.Span)
	for _, prior := range f.callAccesses {
		if prior.store || !prior.mutable || !borrowValuesOverlap(value, prior.value) {
			continue
		}
		for _, diagnostic := range f.analyzer.diagnostics {
			if diagnostic.Code == "E2101" && (diagnostic.Primary == primary || diagnostic.Primary == prior.span) {
				return
			}
		}
		f.analyzer.addDiagnostic("E2101", primary,
			RelatedSpan{Role: "loan", Span: prior.span},
			RelatedSpan{Role: "last-use", Span: span(f.module.file.Path, receiver.Span)})
		return
	}
}

func (f *borrowFlowAnalyzer) evaluateReferentRead(expression *ast.Expression, environment borrowEnvironment) (borrowValueState, bool) {
	value, reachable := f.evaluateChildren(expression.Children, environment)
	if reachable && len(expression.Children) == 1 {
		receiver := expression.Children[0]
		if _, _, borrowed := hir.BorrowElement(f.analyzer.expressionTypes[receiver]); borrowed {
			f.checkHeldReferentRead(expression, receiver, value)
		}
	}
	return value, reachable
}
