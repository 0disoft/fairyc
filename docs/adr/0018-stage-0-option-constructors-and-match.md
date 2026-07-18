# ADR 0018: Stage 0 Option constructors and shallow match

- Status: Accepted for the Stage 0 prototype
- Date: 2026-07-18
- Product source: `docs/product/02-spec.md`
- Grammar source: `spec/grammar/stage0.ebnf`
- Semantic source: `docs/language/0004-results-traps-and-control-flow.md`

## Context

Stage 0 already names `Option<T>` as the nullable-value abstraction, but it did not fix how source
constructs absence or how shallow matching represents an arm without a payload. Reusing ordinary
functions would collide with imports and would let a user declaration redefine compiler-owned tag
semantics. Giving `None` empty call parentheses would also pretend that absence is a function call.

## Decision

Stage 0 uses `Option::Some(expression)` for presence and the bare path `Option::None` for absence.
Both are compiler-known variant construction and require one contextual `Option<T>` type. They do
not perform ordinary name or function lookup, implicit conversion, or unconstrained inference.

Shallow inspection uses exactly one payload arm and one absence arm:

```fy
match option {
    Option::Some(value) => value,
    Option::None => fallback,
}
```

`Option::Some(binding)` owns one arm-local binding. `Option::None` has no parentheses and creates no
binding. Arm order is source-defined, the scrutinee evaluates once, and both arms produce one
compatible type. `Option::None()`, `Option::None(binding)`, a bare `Option::Some`, wildcard arms,
guards, nested patterns, omitted or duplicate variants, and ref-style payload bindings are outside
this slice.

MIR receives explicit Option construction, discriminant switch, and `Some` payload extraction.
The `None` path never executes a payload extraction. `?` remains Result-only in Stage 0.

## Consequences

- Resolver and type checking distinguish the two compiler-known Option variants from ordinary paths.
- The MIR verifier checks variant-specific operand counts before interpretation or backend handoff.
- Scalar, inert, and resource-bearing Option values may execute in the Stage 0 interpreter.
- Resource-bearing construction, consuming match extraction, `None` cleanup, and active-payload drop
  traces are fixture-backed; user-defined tagged layouts and drop-body effects remain deferred.

## Rejected alternatives

- Unqualified `Some` and `None`: collide with ordinary values and imports.
- `Option::None()`: represents a nullary tag as a call and permits misleading arity variants.
- Implicit conversion from `T`: changes call behavior based on expected type.
- Defining `?` for Option now: conflates absence with typed failure and expands early-return rules.
