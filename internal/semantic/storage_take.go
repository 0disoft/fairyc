package semantic

import (
	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/hir"
)

func (c *typeChecker) checkStorageTake(expression *ast.Expression, result *hir.Expression) hir.Type {
	if expression.Type == nil || len(expression.Children) != 1 || expression.Children[0] == nil ||
		expression.Block != nil || len(expression.TypeArguments) != 0 || expression.Text != "take" {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "malformed private storage take")
		return hir.TypeInvalid
	}
	typ := c.analyzer.canonicalType(c.module, *expression.Type)
	admitted := typ == hir.TypeBool || typ == hir.TypeU8 || typ == hir.TypeI32 || typ == hir.TypeU32 || typ == hir.TypeUSize ||
		c.analyzer.structTypes[typ] != nil || c.analyzer.resourceTypes[typ] != nil || c.analyzer.enumTypes[typ] != nil
	if !admitted {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "unsupported private storage take type")
		return hir.TypeInvalid
	}
	pointer, pointerType := c.checkExpression(expression.Children[0], hir.RawPointer(hir.TypeVoid, true))
	if pointerType != hir.RawPointer(hir.TypeVoid, true) {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "private storage take requires a mutable void pointer")
		return hir.TypeInvalid
	}
	result.Children = appendExpression(result.Children, pointer)
	if len(c.unsafeScopes) == 0 {
		c.analyzer.addDiagnostic("E5101", span(c.module.file.Path, expression.Span))
	} else {
		scope := c.unsafeScopes[len(c.unsafeScopes)-1]
		result.UnsafeScope = &scope
	}
	result.Kind, result.Value, result.TargetType = "storage-take", "", typ
	return typ
}
