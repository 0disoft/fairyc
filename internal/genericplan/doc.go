// Package genericplan defines the first executable semantic kernel for bounded,
// statically dispatched Fairy C generics.
//
// The package deliberately owns no source syntax, AST, HIR, MIR, runtime table,
// reflection object, or backend lowering. It validates immutable generic
// declarations, exact capability implementations, coherence, specialization
// dependencies, and deterministic monomorphization identities before those
// concepts are admitted into the language frontend.
//
// A Catalog is immutable after construction. Planning uses only explicit type
// arguments and exact implementations, so the same catalog and requests always
// produce the same dependency-ordered specialization plan.
package genericplan
