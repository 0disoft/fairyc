package semantic

import (
	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/hir"
)

func (c *typeChecker) checkStorageBorrow(expression *ast.Expression, result *hir.Expression) hir.Type {
	if expression.Type == nil || len(expression.Children) != 2 || expression.Children[0] == nil || expression.Children[1] == nil ||
		expression.Block != nil || len(expression.TypeArguments) != 0 || expression.Text != "borrow" {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "malformed private storage borrow")
		return hir.TypeInvalid
	}
	typ := c.analyzer.canonicalType(c.module, *expression.Type)
	admitted := typ == hir.TypeBool || typ == hir.TypeU8 || typ == hir.TypeI32 || typ == hir.TypeU32 || typ == hir.TypeUSize ||
		c.analyzer.structTypes[typ] != nil || c.analyzer.resourceTypes[typ] != nil || c.analyzer.enumTypes[typ] != nil
	if !admitted {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "unsupported private storage borrow type")
		return hir.TypeInvalid
	}
	owner, ownerType := c.checkExpression(expression.Children[0], hir.TypeInvalid)
	pointer, pointerType := c.checkExpression(expression.Children[1], hir.TypeInvalid)
	_, mutable, borrowed := hir.BorrowElement(ownerType)
	element, _, raw := hir.RawPointerElement(pointerType)
	if !borrowed || mutable || !raw || element != hir.TypeVoid {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "private storage borrow requires a shared owner borrow and a void pointer")
		return hir.TypeInvalid
	}
	result.Children = appendExpression(result.Children, owner, pointer)
	if len(c.unsafeScopes) == 0 {
		c.analyzer.addDiagnostic("E5101", span(c.module.file.Path, expression.Span))
	} else {
		scope := c.unsafeScopes[len(c.unsafeScopes)-1]
		result.UnsafeScope = &scope
	}
	result.Kind, result.Value, result.TargetType = "storage-borrow", "", typ
	return hir.SharedBorrow(typ)
}
