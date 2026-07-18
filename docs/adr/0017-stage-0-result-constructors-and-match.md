# ADR 0017: Stage 0 Result constructors and shallow match

- Status: Accepted for the Stage 0 prototype
- Date: 2026-07-18
- Product source: `docs/product/02-spec.md`
- Grammar source: `spec/grammar/stage0.ebnf`
- Semantic source: `docs/language/0004-results-traps-and-control-flow.md`

## Context

The initial Result fixtures deliberately accepted typed parameters so they could prove must-use,
return, discard, and `?` behavior without choosing constructor syntax. That kept the first Flow
contract honest, but it now blocks source programs from creating or inspecting a `Result` value.
Unqualified `Ok(value)` and `Err(error)` would also occupy ordinary value names and collide with
user functions or imports.

## Decision

Stage 0 uses the compiler-known qualified forms `Result::Ok(expression)` and
`Result::Err(expression)`. They are variant construction, not function lookup or implicit
conversion. The enclosing expected `Result<T, E>` type supplies the other payload type, so a
constructor is accepted only in a context with one known canonical Result type. Stage 0 does not
add expression-level generic arguments or unconstrained constructor inference.

Shallow inspection uses the existing pattern grammar with the same qualified identities:

```fy
match result {
    Result::Ok(value) => value,
    Result::Err(error) => fallback(error),
}
```

A Result match contains exactly one `Result::Ok(binding)` arm and one
`Result::Err(binding)` arm. Arm order is source-defined, each binding is scoped to its arm, and
both arm expressions must have one compatible type. Guards, nested patterns, omitted variants,
duplicate variants, wildcard Result arms, and ref-style payload bindings are not in this slice.

Constructor payloads and the match scrutinee evaluate exactly once. Binding a payload by value
moves it when its type is non-copyable. MIR receives explicit variant construction, discriminant
switch, and payload extraction; neither an interpreter nor a backend reconstructs Result behavior
from source syntax.

`Result` is compiler-owned in type, constructor, and pattern-head position. This decision does not
reserve unrelated qualified paths or choose `Option` constructor spelling; Option adopts a
separate executable contract when its payload and absence behavior are implemented.

## Consequences

- Resolver and type checking must distinguish compiler-known variants from ordinary functions.
- Result constructors without a contextual `Result<T, E>` are rejected instead of guessed.
- Flow, MIR, verifier, interpreter, fixtures, and CLI examples must agree on the qualified variant
  identities and exactly-once behavior before construction is reported as implemented.
- Resource-bearing Result payload construction and active-payload drop traces remain deferred even
  after the scalar and inert tagged slice becomes executable.

## Rejected alternatives

- Unqualified `Ok` and `Err`: collide with ordinary value names and imports.
- `Result<T, E>::Ok(value)`: adds expression-level generic path syntax solely for constructors.
- Implicit conversion from `T` or `E`: makes call behavior and control flow depend on expected type.
- Backend-only tagged lowering: bypasses the MIR semantic oracle and duplicates language behavior.
