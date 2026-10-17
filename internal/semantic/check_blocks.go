package semantic

import (
	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/syntax"
)

func blockAlwaysTerminates(block *ast.Block) bool {
	if block == nil {
		return false
	}
	for index := range block.Statements {
		statement := &block.Statements[index]
		if statement.Kind == syntax.KindReturn ||
			statement.Kind == syntax.KindAssert && statement.Expression != nil && statement.Expression.Kind == syntax.KindLiteral && statement.Expression.Text == "false" ||
			expressionAlwaysTerminates(statement.Expression) {
			return true
		}
	}
	return block.Tail != nil
}

func expressionAlwaysTerminates(expression *ast.Expression) bool {
	if expression == nil {
		return false
	}
	switch expression.Kind {
	case syntax.KindIf:
		if len(expression.Children) < 3 {
			return false
		}
		thenBranch := expression.Children[1]
		elseBranch := expression.Children[2]
		return thenBranch != nil && elseBranch != nil && blockAlwaysTerminates(thenBranch.Block) &&
			(elseBranch.Kind == syntax.KindIf && expressionAlwaysTerminates(elseBranch) || blockAlwaysTerminates(elseBranch.Block))
	case syntax.KindUnsafe, syntax.KindRegion:
		for _, child := range expression.Children {
			if child != nil && blockAlwaysTerminates(child.Block) {
				return true
			}
		}
	}
	return false
}

func (c *typeChecker) checkBlock(block *ast.Block, expectedTail hir.Type) (hir.Block, hir.Type, bool) {
	result := hir.Block{Span: span(c.module.file.Path, block.Span)}
	hasReturn := false
	for index := range block.Statements {
		statement := &block.Statements[index]
		hirStatement := hir.Statement{Kind: statementKind(statement.Kind), Span: span(c.module.file.Path, statement.Span), Binding: hir.NoSymbol}
		switch {
		case statement.Binding != nil:
			binding := statement.Binding
			symbol := c.analyzer.bindingSymbols[binding]
			if symbol == nil {
				continue
			}
			expected := hir.TypeInvalid
			if binding.Type != nil {
				expected = c.canonicalType(*binding.Type)
			}
			expression, actual := c.checkExpression(binding.Expression, expected)
			if expected != hir.TypeInvalid {
				c.requireType(
					"E1006",
					span(c.module.file.Path, binding.Expression.Span),
					actual,
					expected,
					span(c.module.file.Path, binding.Type.Span),
				)
				symbol.typ = expected
			} else {
				symbol.typ = actual
			}
			hirStatement.Binding = symbol.id
			hirStatement.Expression = expression
			localSpan := span(c.module.file.Path, binding.NameSpan)
			c.hir.Locals = append(c.hir.Locals, hir.Local{
				Symbol:  symbol.id,
				Name:    binding.Name,
				Type:    symbol.typ,
				Mutable: binding.Mutable,
				Span:    localSpan,
			})
			c.addType(localSpan, symbol.typ)
		case statement.Kind == syntax.KindReturn:
			hasReturn = true
			expression, actual := c.checkExpression(statement.Expression, c.resultType)
			hirStatement.Expression = expression
			if statement.Expression == nil {
				actual = hir.TypeVoid
			}
			primary := statement.Span
			if statement.Expression != nil {
				primary = statement.Expression.Span
			}
			c.requireType(
				"E1009",
				span(c.module.file.Path, primary),
				actual,
				c.resultType,
				span(c.module.file.Path, c.function.ResultType.Span),
			)
		case statement.Kind == syntax.KindAssignment:
			target, targetType := c.checkExpression(statement.Target, hir.TypeInvalid)
			value, actual := c.checkExpression(statement.Expression, targetType)
			if derefAssignmentTarget(statement.Target) &&
				((!hir.FixedArrayScalarElement(targetType) && !c.analyzer.plainViewStruct(targetType)) || !directDerefBorrowPath(statement.Target.Children[0])) {
				c.analyzer.addUnsupported(c.module.file.Path, statement.Target.Span, "dereference assignment requires a named mutable reference to an admitted scalar or plain struct")
			}
			hirStatement.Operator = statement.Operator
			if hirStatement.Operator == "" {
				hirStatement.Operator = "="
			}
			hirStatement.Target = target
			hirStatement.Expression = value
			if statement.Target != nil && !c.mutableAssignmentTarget(statement.Target, target) {
				c.analyzer.addDiagnostic("E1008", span(c.module.file.Path, statement.Target.Span))
			}
			if statement.Expression != nil {
				primary := span(c.module.file.Path, statement.Expression.Span)
				expectedSpan := span(c.module.file.Path, statement.Target.Span)
				if target != nil && target.Kind == hir.ExpressionFieldExpression &&
					(c.analyzer.structTypes[targetType] != nil || c.analyzer.structTypes[actual] != nil) {
					c.requireDetailedType(primary, actual, targetType, expectedSpan)
				} else {
					c.requireType("E1006", primary, actual, targetType, expectedSpan)
				}
			}
			if hirStatement.Operator != "=" {
				validPlace := target != nil && (target.Kind == hir.ExpressionPath || target.Kind == hir.ExpressionArrayIndex || target.Kind == hir.ExpressionViewIndex || target.Kind == hir.ExpressionFieldExpression || derefAssignmentTarget(statement.Target))
				if !validPlace || !hir.CompoundAssignmentType(hirStatement.Operator, targetType) {
					c.analyzer.addUnsupported(c.module.file.Path, statement.Span, "compound assignment "+hirStatement.Operator+" on "+targetType.String())
				}
			}
		case statement.Kind == syntax.KindAssert:
			expression, actual := c.checkExpression(statement.Expression, hir.TypeBool)
			hirStatement.Expression = expression
			if statement.Expression != nil && actual != hir.TypeInvalid && actual != hir.TypeBool {
				c.analyzer.addDiagnostic("E1007", span(c.module.file.Path, statement.Expression.Span))
			}
		case statement.Kind == syntax.KindDefer:
			expression, actual := c.checkExpression(statement.Expression, hir.TypeVoid)
			hirStatement.Expression = expression
			if !validDeferredCall(statement.Expression, expression, actual) {
				c.analyzer.addUnsupported(c.module.file.Path, statement.Span, "defer outside an internal void call with copy-scalar arguments")
			}
		case statement.Kind == syntax.KindExprStmt || statement.Kind == syntax.KindDiscard:
			expression, _ := c.checkExpression(statement.Expression, hir.TypeInvalid)
			hirStatement.Expression = expression
		case statement.Kind == syntax.KindBurn:
			expression, _ := c.checkExpression(statement.Expression, hir.TypeInvalid)
			hirStatement.Expression = expression
			c.analyzer.validateResourceBurn(c.module, statement.Expression)
		case statement.Kind == syntax.KindWhile:
			expression, _ := c.checkExpression(statement.Expression, hir.TypeVoid)
			hirStatement.Expression = expression
		case statement.Kind == syntax.KindFor:
			hirStatement.For = c.checkFor(statement.For)
		case statement.Kind == syntax.KindByteFor:
			reason := c.byteForSourceIssue(statement.ByteFor, statement.Expression)
			issueSpan := statement.Span
			if len(c.analyzer.vectorForTypes) != 0 && statement.ByteFor != nil &&
				statement.ByteFor.Type.Kind == ast.TypeKindReference && reason != "" {
				reason, issueSpan = c.vectorForSourceIssue(statement.ByteFor, statement.Expression)
			}
			if reason != "" {
				c.analyzer.addUnsupported(c.module.file.Path, issueSpan, reason)
			} else {
				hirStatement.Expression, _ = c.checkExpression(statement.Expression, hir.TypeVoid)
				hirStatement.Kind = hir.StatementExpressionStatement
			}
		case statement.Kind == syntax.KindBreak || statement.Kind == syntax.KindContinue:
			if c.loopDepth == 0 {
				c.analyzer.addDiagnostic("E1012", span(c.module.file.Path, statement.Span))
			}
		case statement.Kind == syntax.KindRegion:
			expression, nestedReturn := c.checkRegion(statement.Expression, expectedTail)
			hirStatement.Expression = expression
			hasReturn = hasReturn || nestedReturn
		default:
			c.analyzer.addUnsupported(c.module.file.Path, statement.Span, "statement "+string(statement.Kind))
		}
		result.Statements = append(result.Statements, hirStatement)
	}
	resultType := hir.TypeVoid
	if block.Tail != nil {
		result.Tail, resultType = c.checkExpression(block.Tail, expectedTail)
	}
	return result, resultType, hasReturn
}

func validDeferredCall(source *ast.Expression, checked *hir.Expression, actual hir.Type) bool {
	if source == nil || source.Kind != syntax.KindCall || checked == nil || checked.Kind != hir.ExpressionCall ||
		actual != hir.TypeVoid || checked.Symbol == hir.NoSymbol || checked.ForeignABI != "" || len(checked.Children) == 0 {
		return false
	}
	for index := 1; index < len(checked.Children); index++ {
		if !hir.FixedArrayScalarElement(checked.Children[index].Type) {
			return false
		}
	}
	return true
}
