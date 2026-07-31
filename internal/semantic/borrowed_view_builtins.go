package semantic

import (
	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/syntax"
)

const (
	strFromRawPartsPath              = "str::from_raw_parts"
	cstrFromRawPartsPath             = "CStr::from_raw_parts"
	byteSliceFromRawPartsPath        = "byte_slice::from_raw_parts"
	mutableByteSliceFromRawPartsPath = "byte_slice::from_raw_parts_mut"
)

type borrowedViewConstructor struct {
	viewType       hir.Type
	ownerMustBeMut bool
	dataMustBeMut  bool
}

func borrowedViewConstructorPath(value string) (borrowedViewConstructor, bool) {
	switch value {
	case strFromRawPartsPath:
		return borrowedViewConstructor{viewType: hir.SharedBorrow(hir.TypeStr)}, true
	case cstrFromRawPartsPath:
		return borrowedViewConstructor{viewType: hir.SharedBorrow(hir.TypeCStr)}, true
	case byteSliceFromRawPartsPath:
		return borrowedViewConstructor{viewType: hir.SharedBorrow(hir.TypeByteSlice)}, true
	case mutableByteSliceFromRawPartsPath:
		return borrowedViewConstructor{
			viewType:       hir.MutableBorrow(hir.TypeByteSlice),
			ownerMustBeMut: true,
			dataMustBeMut:  true,
		}, true
	default:
		return borrowedViewConstructor{}, false
	}
}

func isBorrowedViewConstructorCall(expression *ast.Expression) bool {
	if expression == nil || expression.Kind != syntax.KindCall || len(expression.Children) == 0 {
		return false
	}
	callee := expression.Children[0]
	if callee == nil || callee.Kind != syntax.KindPath {
		return false
	}
	_, ok := borrowedViewConstructorPath(callee.Text)
	return ok
}

func isCompilerBuiltinPath(value string) bool {
	if value == sourceLocationPath {
		return true
	}
	if isBuiltinVariantPath(value) {
		return true
	}
	_, ok := borrowedViewConstructorPath(value)
	return ok
}

func (c *typeChecker) checkBorrowedViewConstructor(
	expression *ast.Expression,
	result *hir.Expression,
	path string,
	constructor borrowedViewConstructor,
) hir.Type {
	result.Kind = "borrowed-view-from-raw-parts"
	result.Operator = path

	arguments := expression.Children[1:]
	if len(arguments) != 3 {
		c.analyzer.addDiagnostic("E1010", span(c.module.file.Path, expression.Span))
	}

	argumentTypes := make([]hir.Type, len(arguments))
	for index, argument := range arguments {
		expected := hir.TypeInvalid
		if index == 2 {
			expected = hir.TypeUSize
		}
		child, actual := c.checkExpression(argument, expected)
		result.Children = appendExpression(result.Children, child)
		argumentTypes[index] = actual
	}

	valid := len(argumentTypes) == 3
	if len(argumentTypes) > 0 {
		_, mutable, borrowed := hir.BorrowElement(argumentTypes[0])
		if !borrowed || mutable != constructor.ownerMustBeMut {
			expected := "shared"
			if constructor.ownerMustBeMut {
				expected = "exclusive"
			}
			c.analyzer.addUnsupported(c.module.file.Path, arguments[0].Span, "raw-parts owner that is not an "+expected+" safe borrow")
			valid = false
		}
	}
	if len(argumentTypes) > 1 {
		element, mutable, pointer := hir.RawPointerElement(argumentTypes[1])
		if !pointer || element != hir.TypeU8 && element != hir.TypeVoid || constructor.dataMustBeMut && !mutable {
			c.analyzer.addUnsupported(c.module.file.Path, arguments[1].Span, "raw-parts data that is not a byte or opaque raw pointer")
			valid = false
		}
	}
	if len(argumentTypes) > 2 && argumentTypes[2] != hir.TypeInvalid && argumentTypes[2] != hir.TypeUSize {
		c.requireType(
			"E1006",
			span(c.module.file.Path, arguments[2].Span),
			argumentTypes[2],
			hir.TypeUSize,
			span(c.module.file.Path, expression.Span),
		)
		valid = false
	}

	if len(c.unsafeScopes) == 0 {
		c.analyzer.addDiagnostic("E5101", span(c.module.file.Path, expression.Span))
	} else {
		scope := c.unsafeScopes[len(c.unsafeScopes)-1]
		result.UnsafeScope = &scope
	}
	if !valid {
		return hir.TypeInvalid
	}
	return constructor.viewType
}
