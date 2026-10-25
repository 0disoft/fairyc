package semantic

import (
	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/syntax"
)

func integerBinaryContext(operator string, expected hir.Type) hir.Type {
	if integerValueOperator(operator) && hir.CheckedArithmeticType(expected) {
		return expected
	}
	return hir.TypeInvalid
}

func integerPeerContext(peer, fallback hir.Type) hir.Type {
	if hir.CheckedArithmeticType(peer) {
		return peer
	}
	return fallback
}

func integerValueOperator(operator string) bool {
	switch operator {
	case "+", "-", "*", "/", "%", "&", "^", "|", "<<", ">>":
		return true
	}
	return false
}

func contextualIntegerOperator(operator string) bool {
	if integerValueOperator(operator) {
		return true
	}
	switch operator {
	case "==", "!=", "<", "<=", ">", ">=":
		return true
	}
	return false
}

// Only source integer literals and their pure operator trees are flexible.
// Typed names (including folded global constants), calls and casts are anchors.
func (a *analyzer) contextualIntegerTree(expression *ast.Expression) (flexible bool) {
	if expression == nil {
		return false
	}
	if expression.Kind != syntax.KindGroup && expression.Kind != syntax.KindPrefix && expression.Kind != syntax.KindBinary {
		return expression.Kind == syntax.KindLiteral && isIntegerLiteral(expression.Text)
	}
	// Classification depends only on immutable syntax, not the expected type.
	// Share it across type checking and compile-time evaluation in this analysis.
	if value, found := a.integerTrees[expression]; found {
		return value
	}
	if a.integerTrees == nil {
		a.integerTrees = make(map[*ast.Expression]bool)
	}
	defer func() { a.integerTrees[expression] = flexible }()
	switch expression.Kind {
	case syntax.KindGroup:
		return len(expression.Children) == 1 && a.contextualIntegerTree(expression.Children[0])
	case syntax.KindPrefix:
		return (expression.Text == "+" || expression.Text == "-" || expression.Text == "~") &&
			len(expression.Children) == 1 && a.contextualIntegerTree(expression.Children[0])
	case syntax.KindBinary:
		return integerValueOperator(expression.Text) && len(expression.Children) == 2 &&
			a.contextualIntegerTree(expression.Children[0]) && a.contextualIntegerTree(expression.Children[1])
	}
	return false
}
