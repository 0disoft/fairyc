# ADR 0547: Flat Option transport of extended scalars

- Status: Accepted for bounded Go and Fairy source/artifact transport
- Date: 2026-11-07
- Extends ADR 0494's wrapper boundary; preserves ADR 0387 aggregate admissions
- Go evidence: [execution record](../development/go-flat-option-transport-20261107.md)
- Fairy evidence: [execution record](../development/candidate-flat-option-transport-20261107.md)
- Go cache evidence: [execution record](../development/go-flat-option-cache-20261107.md)
- Fairy CLI/cache evidence: [execution record](../development/candidate-flat-option-cli-cache-20261107.md)
- Preparation: [bounded implementation plan](../development/composite-generic-transport-plan-20261107.md)

## Decision

Admit `Option<S>` when S is exactly one of `i8`, `i16`, `isize`, `u16`, `u64`,
`f32` or `f64`. The payload is one direct named scalar, with no inner wrapper.
This applies both to ordinary source spelling and to generic source: a written
`Option<u16>` and `Option<T>` specialized with `T=u16` denote the same value
type. Generic functions may also spell that scalar directly in their signatures.

Admitted behavior is contextual `.some(value)` / `.none`, local value copies,
reassignment, ordinary call arguments, explicit generic forwarding, returns and
exhaustive `switch` extraction. A flat Option of these scalars is inert and
copyable. Copying and passing it preserves the original value without resource
ownership or nonempty cleanup. None never authorizes inactive payload access.

The generic argument domain stays scalar: `relay<u16>(value)` may transport an
Option value; `id<Option<u16>>(value)` remains unsupported. Ordinary semantic,
Flow and verified MIR checks still apply after structural specialization.
Original source ASTs and source locations remain unchanged.

## Closed boundaries

This decision does not admit extended scalars inside nested Option, Result,
references, views, fixed arrays or pointers. Examples in the generic signature
path that remain unsupported are `Option<Option<T>>`, `Result<T,i32>`,
`Result<Option<T>,i32>`, `Option<Result<T,i32>>`, `ref T`, `ref Option<T>`,
`Option<ref T>`, `ref T[]`, `T[2]` and `ptr T` with T from the seven new types.
Previously admitted original-scalar wrappers and aggregate policies retain their
existing behavior. This is no new generic struct/resource field or layout
contract; in particular it does not make extended scalar arrays or arbitrary
aggregate nesting executable.

Extended-scalar literals, arithmetic, comparisons and implicit conversions are
unchanged. Direct extern-C scalar types and Option-by-value FFI are unchanged.
Deferred Option capture is outside this slice. No backend, target, trust,
registry, chain, provider, public default or release version is selected here.

## Implementation boundary

Go's generic signature restriction exempts only the complete flat Option shape.
It retains authenticated expansion origins, resolve-error precedence, written
composite spans, duplicate compaction and the existing report budget. Written
ordinary Option signatures already resolve structurally; their executable C
transport is the corresponding new admission.

The Go C preparation path has a separate flat-Option candidate and payload
layout branch. The shared Result/aggregate scalar-payload predicates remain
closed. Recursive candidate discovery cannot use a flat Option candidate to
admit a surrounding nested Option or Result. The existing target-owned scalar
layout and ordinary Option tag/payload representation supply C types and layout
assertions; no boxing, allocation or new wire kind is introduced.

Fairy source admission must likewise allow only a root Option's direct scalar
payload, whether written or substituted. An ordinary depth-one exception is
insufficient because it would admit other wrappers. Descriptor handles remain
source-only; failed resolution must roll back partial graph materialization.
Existing scalar tags and Option kind are sufficient. The coordinator owns the
Fairy implementation and its independent execution receipt.

## Evidence and remaining work

The Go execution record owns the exact commands and results. It covers all seven
types, ordinary and generic spellings, private/explicit/package analysis,
immutable source ASTs, two emitted C profiles, actual Windows C11/C17 low and
optimized payload execution, and independent negative wrapper/FFI boundaries.
The C observer supplies signed minima, unsigned maxima, finite floats and both
zero signs; source code does not synthesize new scalar literals or compare those
scalars numerically.

Fairy's separate execution record establishes graph canonicalization/rollback,
the current-source restored-artifact C pipeline in four Windows producer/consumer
combinations, 24-case Go diagnostic comparison and complete source/locked checks.
These are independently observed integration results, not consequences inferred
from the Go result. The Go cache follow-up establishes memory/new-session and
actual new-process disk restoration without lowering, source/helper/payload
invalidation, inner-artifact rejection/repair and four restored C executions.
The Fairy full CLI/cache follow-up passes both Windows compiler profiles, with
real check/build/run/native-test dispatch, current packet scalar observations,
dependency body invalidation, source refusal and selected-generation repair.
Native Linux replay,
full-main Atlas, consecutive self-build, distribution refresh and hosted CI are
not claimed by this ADR. Existing cache/interface encodings remain unchanged;
their existing source/compiler identity checks remain required.
