package semantic

import (
	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/hir"
)

func (c *typeChecker) checkStorageLayout(expression *ast.Expression, result *hir.Expression) hir.Type {
	if expression.Type == nil || len(expression.Children) != 0 || expression.Block != nil ||
		len(expression.TypeArguments) != 0 || (expression.Text != "size" && expression.Text != "alignment") {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "malformed private storage query")
		return hir.TypeInvalid
	}
	typ := c.analyzer.canonicalType(c.module, *expression.Type)
	admitted := typ == hir.TypeBool || typ == hir.TypeU8 || typ == hir.TypeI32 || typ == hir.TypeU32 || typ == hir.TypeUSize ||
		c.analyzer.structTypes[typ] != nil || c.analyzer.resourceTypes[typ] != nil || c.analyzer.enumTypes[typ] != nil
	if !admitted {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "unsupported private storage query type")
		return hir.TypeInvalid
	}
	result.Kind = "storage-layout"
	result.Value = ""
	result.TargetType = typ
	result.Operator = expression.Text
	return hir.TypeUSize
}
