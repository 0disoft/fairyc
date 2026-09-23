package semantic

import (
	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/syntax"
)

// byteForSourceIssue checks the written form before the hidden scalar loop is
// admitted. In particular, a mutable view or a call cannot be made safe by
// the ordinary index/length operations in the prepared loop.
func (c *typeChecker) byteForSourceIssue(loop *ast.ByteForLoop, prepared *ast.Expression) string {
	if loop == nil || loop.Body == nil || loop.View == nil {
		return "malformed byte-view iteration"
	}
	if loop.Type.Kind != ast.TypeKindNamed || loop.Type.Path != "u8" {
		return "byte-view iteration requires an exact u8 item binding"
	}
	if prepared == nil || loop.View.Kind != syntax.KindPath || len(loop.View.Children) != 0 {
		return "byte-view iteration requires a named immutable ref u8[] view"
	}
	value := c.analyzer.resolved[loop.View]
	if value == nil || value.kind != "local" && value.kind != "parameter" || value.mutable || value.typ != hir.SharedBorrow(hir.TypeByteSlice) {
		return "byte-view iteration requires a named immutable ref u8[] view"
	}
	return ""
}
