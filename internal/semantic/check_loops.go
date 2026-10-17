package semantic

import (
	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/syntax"
)

func (c *typeChecker) checkFor(loop *ast.ForLoop) *hir.ForLoop {
	if loop == nil || loop.Body == nil {
		c.analyzer.addUnsupported(c.module.file.Path, c.function.Span, "malformed for statement")
		return nil
	}
	result := &hir.ForLoop{Span: span(c.module.file.Path, loop.Span)}
	if loop.Initializer != nil {
		result.Initializer = c.checkForClause(loop.Initializer)
		if !validForInitializer(loop.Initializer, result.Initializer) {
			c.analyzer.addUnsupported(c.module.file.Path, loop.Initializer.Span, "for initializer outside the scalar declaration or direct-assignment subset")
		}
	}
	if loop.Condition != nil {
		condition, actual := c.checkExpression(loop.Condition, hir.TypeBool)
		result.Condition = condition
		if !plainForHeaderExpression(loop.Condition) {
			c.analyzer.addUnsupported(c.module.file.Path, loop.Condition.Span, "for condition with calls, borrows, blocks, indexing, or other non-scalar header behavior")
		}
		if actual != hir.TypeInvalid && actual != hir.TypeBool {
			c.analyzer.addDiagnostic("E1007", span(c.module.file.Path, loop.Condition.Span))
		}
	}
	c.loopDepth++
	body, _, _ := c.checkBlock(loop.Body, hir.TypeVoid)
	c.loopDepth--
	result.Body = body
	if loop.Post != nil {
		result.Post = c.checkForClause(loop.Post)
		if !validForPost(loop.Post, result.Post) {
			c.analyzer.addUnsupported(c.module.file.Path, loop.Post.Span, "for post clause outside direct scalar assignment")
		}
	}
	return result
}

func (c *typeChecker) checkForClause(statement *ast.Statement) *hir.Statement {
	if statement == nil {
		return nil
	}
	block := &ast.Block{Span: statement.Span, Statements: []ast.Statement{*statement}}
	checked, _, _ := c.checkBlock(block, hir.TypeVoid)
	if len(checked.Statements) != 1 {
		return nil
	}
	result := checked.Statements[0]
	return &result
}

func validForInitializer(source *ast.Statement, checked *hir.Statement) bool {
	if source == nil || checked == nil {
		return false
	}
	if source.Binding != nil {
		return checked.Kind == hir.StatementBinding && checked.Expression != nil &&
			forHeaderScalar(checked.Expression.Type) && plainForHeaderExpression(source.Binding.Expression)
	}
	return validForScalarAssignment(source, checked)
}

func validForPost(source *ast.Statement, checked *hir.Statement) bool {
	return validForScalarAssignment(source, checked)
}

func validForScalarAssignment(source *ast.Statement, checked *hir.Statement) bool {
	return source != nil && checked != nil && source.Kind == syntax.KindAssignment &&
		source.Target != nil && source.Target.Kind == syntax.KindPath &&
		checked.Target != nil && checked.Target.Kind == hir.ExpressionPath &&
		checked.Expression != nil && forHeaderScalar(checked.Target.Type) &&
		checked.Expression.Type == checked.Target.Type && plainForHeaderExpression(source.Expression)
}

func forHeaderScalar(value hir.Type) bool {
	return value == hir.TypeBool || hir.ScalarOrderedType(value)
}

func plainForHeaderExpression(expression *ast.Expression) bool {
	if expression == nil {
		return false
	}
	switch expression.Kind {
	case syntax.KindLiteral, syntax.KindPath:
		return true
	case syntax.KindGroup, syntax.KindBinary, syntax.KindCast:
		for _, child := range expression.Children {
			if !plainForHeaderExpression(child) {
				return false
			}
		}
		return len(expression.Children) > 0
	case syntax.KindPrefix:
		if expression.Text != "+" && expression.Text != "-" && expression.Text != "!" && expression.Text != "~" {
			return false
		}
		return len(expression.Children) == 1 && plainForHeaderExpression(expression.Children[0])
	default:
		return false
	}
}

func (c *typeChecker) checkWhile(expression *ast.Expression, result *hir.Expression) hir.Type {
	if len(expression.Children) != 2 {
		return hir.TypeInvalid
	}
	condition, conditionType := c.checkExpression(expression.Children[0], hir.TypeBool)
	result.Children = appendExpression(result.Children, condition)
	if conditionType != hir.TypeInvalid && conditionType != hir.TypeBool {
		c.analyzer.addDiagnostic("E1007", span(c.module.file.Path, expression.Children[0].Span))
	}
	c.loopDepth++
	body, _ := c.checkExpression(expression.Children[1], hir.TypeVoid)
	c.loopDepth--
	result.Children = appendExpression(result.Children, body)
	return hir.TypeVoid
}
