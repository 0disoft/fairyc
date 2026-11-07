# ADR 0550: Flat Result success-payload transport

- Status: Accepted for bounded prototype
- Date: 2026-11-07
- Extends: [ADR 0547](0547-flat-option-generic-transport.md)
- Go execution: [receipt](../development/go-flat-result-transport-20261107.md)
- Go cache: [receipt](../development/go-flat-result-cache-20261107.md)
- Fairy source/artifact/native: [receipt](../development/candidate-flat-result-transport-20261107.md)

## Scope

Admit a root `Result<S,u32>` with a direct named success payload S from
`i8`, `i16`, `isize`, `u16`, `u64`, `f32`, `f64`. A scalar generic argument may
substitute that direct success leaf in `Result<T,u32>`. Existing Result forms
and ordinary scalar transport retain their behavior. The error payload must
resolve to exactly u32; this does not generalize error transport.

This is a shallow permission. References, pointers, arrays, wrapper nesting,
storage/defer capture and composite generic arguments containing the extension
retain their independent admission limits. In particular `Result<S,i32>`,
`Result<u32,S>`, `Result<Option<S>,u32>` and `id<Result<S,u32>>` do not gain
admission. No source scalar literals, arithmetic or comparison are added.

## Ownership and execution

Existing `.ok`/`.err` construction, inert value copy, reassignment, ordinary and
generic parameters/results, exhaustive extraction and `?` use existing Result
checking and lowering. Must-handle obligations still apply independently to
each copy. The `?` operand executes once, and error-type mismatch diagnostics
retain their current source origin. Failed type-graph admission restores the
whole previous graph, including an interned success leaf.

Go and Fairy retain their existing tagged representation and active-member
semantics. Native observers must inspect the active payload only, including
signed minima, unsigned maxima and floating signed zero. There is no new
Result-by-value foreign ABI, allocation, boxing or exposed layout guarantee.

## Identity and delivery

Use current concrete type vocabulary, public-interface spelling, specialization
identity, verified artifact transport and cache restoration. Existing no-Result
and previously admitted Result bytes retain their identities. Source and
restored-artifact/native tests must be independent of each other's output as an
oracle. Whole CLI/cache acceptance remains a separately recorded gate.

The Go compiler/backend owner, Fairy type/source/artifact owner and cache/CLI
owner use disjoint write scopes. The coordinator owns integration, source lock,
architecture/queue updates and small Windows-verified commits. Linux execution
remains batched. No dependencies, runner changes or remote actions are needed.
