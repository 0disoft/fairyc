package semantic

import (
	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/hir"
)

// checkStorageElementAddress keeps typed element layout and unsafe provenance
// attached to the SDK-only address calculation. The caller proves the index
// is within its initialized allocation before reconstructing a safe borrow.
func (c *typeChecker) checkStorageElementAddress(expression *ast.Expression, result *hir.Expression) hir.Type {
	if expression.Type == nil || len(expression.Children) != 2 || expression.Children[0] == nil || expression.Children[1] == nil ||
		expression.Block != nil || len(expression.TypeArguments) != 0 || expression.Text != "element-address" {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "malformed private storage element address")
		return hir.TypeInvalid
	}
	typ := c.analyzer.canonicalType(c.module, *expression.Type)
	admitted := typ == hir.TypeBool || typ == hir.TypeU8 || typ == hir.TypeI32 || typ == hir.TypeI64 || typ == hir.TypeU32 || typ == hir.TypeUSize ||
		c.analyzer.structTypes[typ] != nil || c.analyzer.resourceTypes[typ] != nil || c.analyzer.enumTypes[typ] != nil
	if !admitted {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "unsupported private storage element address type")
		return hir.TypeInvalid
	}
	base, baseType := c.checkExpression(expression.Children[0], hir.RawPointer(hir.TypeVoid, true))
	index, indexType := c.checkExpression(expression.Children[1], hir.TypeUSize)
	if baseType != hir.RawPointer(hir.TypeVoid, true) || indexType != hir.TypeUSize {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "private storage element address requires an exact mutable void pointer and usize index")
		return hir.TypeInvalid
	}
	result.Children = appendExpression(result.Children, base, index)
	if len(c.unsafeScopes) == 0 {
		c.analyzer.addDiagnostic("E5101", span(c.module.file.Path, expression.Span))
	} else {
		scope := c.unsafeScopes[len(c.unsafeScopes)-1]
		result.UnsafeScope = &scope
	}
	result.Kind, result.Value, result.TargetType = "storage-element-address", "", typ
	return hir.RawPointer(hir.TypeVoid, true)
}
