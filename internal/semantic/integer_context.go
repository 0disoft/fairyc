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
func contextualIntegerTree(expression *ast.Expression) bool {
	if expression == nil {
		return false
	}
	switch expression.Kind {
	case syntax.KindLiteral:
		return isIntegerLiteral(expression.Text)
	case syntax.KindGroup:
		return len(expression.Children) == 1 && contextualIntegerTree(expression.Children[0])
	case syntax.KindPrefix:
		return (expression.Text == "+" || expression.Text == "-" || expression.Text == "~") &&
			len(expression.Children) == 1 && contextualIntegerTree(expression.Children[0])
	case syntax.KindBinary:
		return integerValueOperator(expression.Text) && len(expression.Children) == 2 &&
			contextualIntegerTree(expression.Children[0]) && contextualIntegerTree(expression.Children[1])
	}
	return false
}
