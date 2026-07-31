package semantic

import "github.com/0disoft/fairyc/internal/hir"

// safeBorrowLeaf reports the one safe-borrow leaf admitted by the Stage 0
// Option/Result return shapes. The count remains explicit so callers never
// silently choose one origin if a malformed or future type contains several.
func safeBorrowLeaf(value hir.Type) (element hir.Type, mutable bool, count int) {
	if element, mutable, ok := hir.BorrowElement(value); ok {
		return element, mutable, 1
	}
	if payload, ok := hir.OptionElement(value); ok {
		return safeBorrowLeaf(payload)
	}
	if success, failure, ok := hir.ResultElements(value); ok {
		successElement, successMutable, successCount := safeBorrowLeaf(success)
		failureElement, failureMutable, failureCount := safeBorrowLeaf(failure)
		if successCount > 0 {
			element, mutable = successElement, successMutable
		} else if failureCount > 0 {
			element, mutable = failureElement, failureMutable
		}
		return element, mutable, successCount + failureCount
	}
	return hir.TypeInvalid, false, 0
}

func containsSafeBorrow(value hir.Type) bool {
	_, _, count := safeBorrowLeaf(value)
	return count > 0
}
