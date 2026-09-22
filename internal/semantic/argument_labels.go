package semantic

import (
	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/syntax"
)

func (a *analyzer) checkCallArgumentLabels(module *moduleScope, expression *ast.Expression) bool {
	labels := expression.CallLabels
	if labels == nil {
		return true
	}
	reject := func() bool { a.addDiagnostic("E1014", span(module.file.Path, expression.Span)); return false }
	if len(expression.Children) == 0 || expression.Children[0].Kind != syntax.KindPath || labels.ExplicitTypeArguments {
		return reject()
	}
	callee := a.resolved[expression.Children[0]]
	if callee == nil || callee.kind != "function" || callee.function == nil || callee.public || callee.function.Owner != "" || len(callee.function.TypeParameters) != 0 || callee.module != module.file.Module.Path || callee.packageID != module.key.packageID {
		return reject()
	}
	if len(expression.Children)-1 != len(callee.function.Parameters) {
		return true
	} // E1010 owns arity.
	for _, label := range labels.Labels {
		if label.Index < 0 || label.Index >= len(callee.function.Parameters) || label.Name != callee.function.Parameters[label.Index].Name {
			a.addDiagnostic("E1014", span(module.file.Path, label.Span))
			return false
		}
	}
	return true
}
