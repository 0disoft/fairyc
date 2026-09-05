# ADR 0384: Retain explicit generic function source shapes

- Status: Accepted frontend preparation; semantic specialization remains gated
- Date: 2026-09-05

## Decision

Use C-shaped function declarations with explicit ordered type parameters:

```text
fn T identity<T>(T value) { return value; }
fn i32 main() { return identity<i32>(42); }
```

The parser retains parameter names in a distinct type-parameter node and recognizes a type-argument
list only after a path when its closing angle is followed by a call. Bounded lookahead of at most
256 significant tokens separates this shape from ordinary comparisons. The AST owns ordered
`Function.TypeParameters` and `Expression.TypeArguments`; type arguments are never ordinary value
arguments and must never be erased before specialization.

## Admission sequence

This frontend commit deliberately rejects generic functions in semantic analysis and retains
unsupported specialization expressions. Even an unused type parameter on an otherwise concrete
function cannot bypass the gate. It does not release a new language/SDK version or claim runnable
generics.

The next implementation must connect these AST facts to ADR 0339's bounded planner, clone and
substitute concrete types before ordinary HIR publication, check all admitted instantiations, and
reject implicit inference and unsupported ability or recursive expansion. Package-owned identity,
public-interface/cache transport, and native evidence are required before closing issues 26 and 27.

## Evidence

- `internal/syntax/generic.go`
- `internal/ast/generic_test.go`
- `internal/compiler/generic_syntax_test.go`
- `spec/grammar/stage0.ebnf`
