package semantic

import (
	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/diagnostic"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/syntax"
)

// vectorForSourceIssue validates the written loop before the authenticated
// SDK candidate's prepared calls enter ordinary type checking and Flow.
func (c *typeChecker) vectorForSourceIssue(loop *ast.ByteForLoop, prepared *ast.Expression) (string, diagnostic.Span) {
	const shape = "read-only SDK Vec iteration requires a shared ref item and a named immutable vector view or explicit place borrow"
	const malformed = "malformed read-only SDK Vec iteration"
	if loop == nil {
		return shape, diagnostic.Span{}
	}
	if loop.Type.Kind != ast.TypeKindReference || loop.Type.Mutable || loop.Type.Element == nil {
		return shape, loop.Type.Span
	}
	if loop.Body == nil || loop.View == nil {
		return shape, loop.Span
	}
	if !ast.VectorForEligibleView(loop.View) {
		return shape, loop.View.Span
	}
	if loop.PreparedVector == nil {
		return shape, loop.Span
	}
	if prepared == nil || prepared.Kind != syntax.KindBlock || prepared.Block == nil ||
		len(prepared.Block.Statements) != 3 || prepared.Block.Tail != nil {
		return malformed, loop.Span
	}
	owner := prepared.Block.Statements[0].Binding
	bound := prepared.Block.Statements[1].Binding
	classic := prepared.Block.Statements[2].For
	if owner == nil || owner.Type == nil || owner.Type.Kind != ast.TypeKindReference || owner.Type.Mutable ||
		!sameVectorForView(loop.View, owner.Expression) || bound == nil || bound.Expression == nil ||
		classic == nil || classic.Body == nil || len(classic.Body.Statements) == 0 {
		return malformed, loop.Span
	}
	item := classic.Body.Statements[0].Binding
	if item == nil || item.Name != loop.Name || item.Type == nil || item.Expression == nil ||
		item.Expression.Kind != syntax.KindCall || bound.Expression.Kind != syntax.KindCall {
		return malformed, loop.Span
	}
	ownerType := c.canonicalType(*owner.Type)
	vectorType, mutable, borrowed := hir.BorrowElement(ownerType)
	if !borrowed || mutable || vectorType == hir.TypeInvalid {
		return shape, loop.View.Span
	}
	info := c.analyzer.resourceTypes[vectorType]
	if info == nil || info.symbol == nil || !c.analyzer.vectorForTypes[vectorTypeKey{
		info.symbol.packageID, info.symbol.module, info.symbol.name,
	}] {
		return "read-only SDK Vec iteration requires an authenticated Vec owner", loop.View.Span
	}
	if c.canonicalType(loop.Type) != c.canonicalType(*item.Type) {
		return malformed, loop.Span
	}
	if owner.Expression.Kind == syntax.KindPath {
		value := c.analyzer.resolved[owner.Expression]
		if value == nil || value.kind != "local" && value.kind != "parameter" ||
			value.mutable || value.typ != ownerType {
			return shape, loop.View.Span
		}
	}
	return "", loop.Span
}

func sameVectorForView(source, checked *ast.Expression) bool {
	if source == nil || checked == nil || source.Kind != checked.Kind ||
		source.Text != checked.Text || source.Span != checked.Span ||
		len(source.Children) != len(checked.Children) {
		return false
	}
	for index := range source.Children {
		if !sameVectorForView(source.Children[index], checked.Children[index]) {
			return false
		}
	}
	return true
}
