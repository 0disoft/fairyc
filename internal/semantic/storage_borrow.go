package semantic

import (
	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/hir"
)

func (c *typeChecker) checkStorageBorrow(expression *ast.Expression, result *hir.Expression) hir.Type {
	if expression.Type == nil || len(expression.Children) < 2 || expression.Children[0] == nil || expression.Children[1] == nil ||
		expression.Block != nil || len(expression.TypeArguments) != 0 ||
		(expression.Text != "borrow" && expression.Text != "borrow-mut") {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "malformed private storage borrow")
		return hir.TypeInvalid
	}
	typ := hir.TypeInvalid
	if expression.Type.Kind == ast.TypeKindSlice {
		// Unsized slices are descriptors here, never by-value stored elements.
		typ, _ = c.analyzer.canonicalTypeReference(c.module, *expression.Type, false)
	} else {
		typ = c.analyzer.canonicalType(c.module, *expression.Type)
	}
	_, view := hir.IndexedViewElement(hir.SharedBorrow(typ))
	if !view && len(expression.Children) != 2 || view && len(expression.Children) != 3 {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "private storage borrow argument count")
		return hir.TypeInvalid
	}
	admitted := typ == hir.TypeBool || typ == hir.TypeU8 || typ == hir.TypeI32 || typ == hir.TypeI64 || typ == hir.TypeU32 || typ == hir.TypeUSize ||
		c.analyzer.structTypes[typ] != nil || c.analyzer.resourceTypes[typ] != nil || c.analyzer.enumTypes[typ] != nil || view
	if !admitted {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "unsupported private storage borrow type")
		return hir.TypeInvalid
	}
	owner, ownerType := c.checkExpression(expression.Children[0], hir.TypeInvalid)
	pointer, pointerType := c.checkExpression(expression.Children[1], hir.TypeInvalid)
	_, ownerMutable, borrowed := hir.BorrowElement(ownerType)
	element, pointerMutable, raw := hir.RawPointerElement(pointerType)
	mutable := expression.Text == "borrow-mut"
	if !borrowed || ownerMutable != mutable || !raw || element != hir.TypeVoid || (mutable && !pointerMutable) {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "private storage borrow requires matching owner and void pointer mutability")
		return hir.TypeInvalid
	}
	result.Children = appendExpression(result.Children, owner, pointer)
	if view {
		length, lengthType := c.checkExpression(expression.Children[2], hir.TypeUSize)
		if lengthType != hir.TypeUSize {
			c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "private storage view requires a usize element count")
			return hir.TypeInvalid
		}
		result.Children = appendExpression(result.Children, length)
	}
	if len(c.unsafeScopes) == 0 {
		c.analyzer.addDiagnostic("E5101", span(c.module.file.Path, expression.Span))
	} else {
		scope := c.unsafeScopes[len(c.unsafeScopes)-1]
		result.UnsafeScope = &scope
	}
	result.Kind, result.Value, result.TargetType = "storage-borrow", "", typ
	if mutable {
		result.Kind = "storage-borrow-mut"
		return hir.MutableBorrow(typ)
	}
	return hir.SharedBorrow(typ)
}
