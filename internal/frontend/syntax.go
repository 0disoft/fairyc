// Package frontend composes the Stage 0 syntax pipeline behind one compiler-facing boundary.
package frontend

import (
	"errors"
	"slices"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/source"
	"github.com/0disoft/fairyc/internal/syntax"
)

// SyntaxResult contains the lossless syntax tree and, for diagnostic-free input, its lowered AST.
type SyntaxResult struct {
	Tree *syntax.Tree
	AST  *ast.Module
}

// AnalyzeSyntax parses one immutable source, verifies lossless reconstruction, and lowers clean syntax.
func AnalyzeSyntax(file source.File, limits source.Limits) (SyntaxResult, error) {
	tree := syntax.Parse(file, limits)
	reconstructed, ok := tree.Reconstruct()
	if !ok || !slices.Equal(reconstructed, file.Bytes) {
		return SyntaxResult{}, errors.New("lossless syntax reconstruction failed")
	}
	result := SyntaxResult{Tree: tree}
	if !tree.ErrorFree() {
		return result, nil
	}
	module, err := ast.Lower(tree)
	if err != nil {
		return SyntaxResult{}, errors.New("lower diagnostic-free syntax tree")
	}
	result.AST = module
	return result, nil
}
