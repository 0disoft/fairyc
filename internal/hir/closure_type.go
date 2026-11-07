package hir

// TypeClosureI32 is the canonical copied closure value leaf. It is distinct
// from a C callback; this identity alone grants no executable admission.
var TypeClosureI32 Type = builtin("fn i32(i32)")

// IsClosureI32 identifies only the exact canonical leaf, not wrappers or
// arbitrary safe function signatures retained by the source parser.
func IsClosureI32(value Type) bool { return value == TypeClosureI32 }
