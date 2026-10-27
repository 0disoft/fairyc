package semantic

import "github.com/0disoft/fairyc/internal/hir"

// safeBorrowLeaf reports the one safe-borrow leaf admitted by the Stage 0
// Option/Result return shapes. The count remains explicit so callers never
// silently choose one origin if a malformed or future type contains several.
// Counts beyond int's range saturate rather than wrapping to zero or negative.
func safeBorrowLeaf(value hir.Type) (element hir.Type, mutable bool, count int) {
	search := safeBorrowLeafSearch{}
	result := search.visit(value)
	return result.element, result.mutable, result.count
}

type safeBorrowLeafResult struct {
	element hir.Type
	mutable bool
	count   int
}

// Type identities are immutable and can share compound children. Keep completed
// summaries local to one search, including types that contain no borrow at all.
type safeBorrowLeafSearch struct {
	results map[hir.Type]safeBorrowLeafResult
}

func (search *safeBorrowLeafSearch) visit(value hir.Type) safeBorrowLeafResult {
	if element, mutable, ok := hir.BorrowElement(value); ok {
		return safeBorrowLeafResult{element: element, mutable: mutable, count: 1}
	}
	if result, found := search.results[value]; found {
		return result
	}
	var result safeBorrowLeafResult
	if payload, ok := hir.OptionElement(value); ok {
		result = search.visit(payload)
	} else if success, failure, ok := hir.ResultElements(value); ok {
		left, right := search.visit(success), search.visit(failure)
		result = left
		if left.count == 0 {
			result = right
		}
		const maxCount = int(^uint(0) >> 1)
		if left.count > maxCount-right.count {
			result.count = maxCount
		} else {
			result.count = left.count + right.count
		}
	} else {
		return safeBorrowLeafResult{element: hir.TypeInvalid}
	}
	if search.results == nil {
		search.results = make(map[hir.Type]safeBorrowLeafResult)
	}
	search.results[value] = result
	return result
}

func containsSafeBorrow(value hir.Type) bool {
	_, _, count := safeBorrowLeaf(value)
	return count > 0
}
