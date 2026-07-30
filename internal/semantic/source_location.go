package semantic

import (
	"encoding/hex"
	"fmt"
	"slices"
	"strconv"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/diagnostic"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/source"
	"github.com/0disoft/fairyc/internal/syntax"
)

const sourceLocationPath = "source_location"

func isSourceLocationCall(expression *ast.Expression) bool {
	return expression != nil && expression.Kind == syntax.KindCall && len(expression.Children) > 0 &&
		expression.Children[0] != nil && expression.Children[0].Kind == syntax.KindPath &&
		expression.Children[0].Text == sourceLocationPath
}

func (a *analyzer) rejectSourceLocationName(path, name string, nameSpan diagnostic.Span) bool {
	if name != sourceLocationPath {
		return false
	}
	a.addDiagnostic("E4401", span(path, nameSpan))
	return true
}

func (c *typeChecker) checkSourceLocation(expression *ast.Expression, result *hir.Expression) hir.Type {
	if len(expression.Children) != 1 {
		c.analyzer.addDiagnostic("E1010", span(c.module.file.Path, expression.Span))
		for _, argument := range expression.Children[1:] {
			child, _ := c.checkExpression(argument, hir.TypeInvalid)
			result.Children = appendExpression(result.Children, child)
		}
		return hir.TypeInvalid
	}

	location, ok := sourceLocationValue(c.module, expression.Children[0].Span)
	if !ok {
		c.analyzer.addDiagnostic("E4402", span(c.module.file.Path, expression.Span))
		return hir.TypeInvalid
	}
	result.Kind = string(syntax.KindLiteral)
	result.Value = strconv.Quote(location)
	result.Cooked = hex.EncodeToString([]byte(location))
	result.Symbol = hir.NoSymbol
	result.Children = nil
	return hir.SharedBorrow(hir.TypeStr)
}

func sourceLocationValue(module *moduleScope, calleeSpan diagnostic.Span) (string, bool) {
	if module == nil || module.file.Module == nil {
		return "", false
	}
	packageID, relativePath := hir.SourcePathParts(module.file.Path)
	if packageID != module.key.packageID {
		return "", false
	}
	file := source.NewFile(relativePath, nil)
	expectedModule, err := file.ExpectedModule()
	if err != nil || expectedModule != module.file.Module.Path {
		return "", false
	}
	starts := module.file.Module.SourceLineStarts
	size := module.file.Module.SourceSize
	if size <= 0 || len(starts) == 0 || starts[0] != 0 || !slices.IsSorted(starts) || calleeSpan.StartByte < 0 ||
		calleeSpan.EndByte <= calleeSpan.StartByte || calleeSpan.EndByte > size {
		return "", false
	}
	for index, start := range starts {
		if start < 0 || start > size || index > 0 && start <= starts[index-1] {
			return "", false
		}
	}
	lineIndex, _ := slices.BinarySearch(starts, calleeSpan.StartByte)
	if lineIndex >= len(starts) || starts[lineIndex] > calleeSpan.StartByte {
		lineIndex--
	}
	if lineIndex < 0 {
		return "", false
	}
	line := lineIndex + 1
	column := calleeSpan.StartByte - starts[lineIndex] + 1
	return fmt.Sprintf("%s:%d:%d@%d", relativePath, line, column, calleeSpan.StartByte), true
}
