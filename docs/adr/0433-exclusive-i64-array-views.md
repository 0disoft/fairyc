# ADR 0433: Exclusive i64 fixed-array views

- Status: implemented
- Date: 2026-10-03
- Owner: [Borrow and return origins](../language/0002-borrows-and-return-origins.md)

## Contract

Admit `let ref mut i64[] view = ref mut data` where data is a mutable executable
`i64[N]` fixed array. The existing array-size bound applies. Descriptors borrow
storage; creating/slicing them does not allocate or copy array elements.
Indices and lengths count i64 elements, not bytes. Element reads, writes and
checked compound assignment preserve the full signed range. Length, emptiness
and explicit/omitted half-open slices retain checked element bounds.
Receiver, index/bounds and RHS evaluation use the existing ordered lowering.
Bounds failure traps before an indexed store; overflow uses checked i64 traps.

Reuse ADR 0415's whole-parent freeze and subview generations. Live descendants
block parent, copied parent and original owner access. A retained child alias
keeps its loan; last use releases the restriction. Retained sibling views are
rejected even for disjoint ranges. Nested and parameter-derived returns,
including the existing single safe-borrow leaf in Option/Result, retain origin
and exclusive generations. Shared readers use a shortened shared reborrow.
Local escape and shared-to-exclusive promotion remain rejected.

Mutable Vec views, generic element views, mutable for-in iteration, interval
alias proofs and i64-to-byte reinterpretation are not added.

## Implementation

HIR's indexed-view classification includes exclusive i64 descriptors. Semantic
place admission and subview flow reuse existing exclusive borrowing. MIR
lowering/verifying checks matching element types, actual receiver/index/store
value definitions and dominating bounds checks. The interpreter retains array
backing and offsets across mutable slices and shared reborrows. Codec round
trips use the existing MIR shape. Private C emits `fy_mut_i64_slice` with an
`int64_t *` and element length; shared conversion explicitly constructs the
const descriptor. Generated C record names reserve the new descriptor.

No public SDK export, foreign ABI, runtime symbol or serialized field is added.
Language `/20`, SDK stage0.80, runtime `/0` and artifact versions remain unchanged
under the existing additive admission policy. Compiler executable identity
separates disposable caches.

## Windows optimized execution

MSVC 14.51.36231 `/O2` returned unchanged 4000000000 after a three-element
subview reversal whose expected first value was INT64_MIN. MIR interpretation
and `/Od` were correct. A smaller positive-only reversal passed. Volatile stores
alone did not repair the failing case. Preventing function inlining repaired
the called writer, but the same inline local loop still failed.

Windows transport therefore uses volatile-qualified i64 view loads when the
emitted program contains an exclusive i64 descriptor, and volatile i64 stores.
The full example, called reversal and local-loop reversal now pass both profiles.
Shared-only i64 programs, ordinary array loads and Linux emission retain normal
loads. The earlier Windows byte-store workaround remains in place. No noinline
attribute or general optimization disable is retained. The precise upstream
optimizer mechanism is not established. This is an internal correctness
workaround, not source volatile/atomic or concurrency semantics; optimization
cost has not been measured.

## Evidence and limits

[The Fairy example](../../examples/i64_mut_view/src/main.fy) reverses a subrange,
preserves signed extrema and values beyond i32, uses nested/returned/tagged
views and a shared reader, checks empty end slices and bound evaluation order,
and verifies original-array values through element reads after loan release.
Semantic cases reject parent/owner/alias/sibling conflicts, bound mutation,
overlapping arguments, unsafe promotion and local escape. Compiler tests check
MIR round-trip execution, wrong receiver/value/bound metadata, missing guards,
interpreter bounds/overflow traps and both target C profiles.

Windows native verification passes 11 cases in development and release (22
executions), including the minimized reversal cases, end/huge/empty/reversed/
past-end bounds and signed overflow/underflow. Ordinary CLI check, both builds
and locked runs pass on a copied example. Related HIR/semantic/MIR/interpreter/
backend/codec tests, selected compiler regressions and Go vet pass. The
[evidence receipt](../ops/evidence/i64-mut-view-20261003.json) records source,
compiler, SDK and image identities with raw native outcomes and CLI argv/cwd.
Images remain in `.cache/i64-mut-view-native/windows/` and the test suite can
regenerate them. Linux native execution is deferred to the user's later batch;
Linux C emission is checked. No full-repository suite or benchmark was run.
