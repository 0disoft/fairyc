# ADR 0441: Nested scalar-leaf struct views

- Status: implemented; Windows native, Linux execution deferred
- Date: 2026-10-04
- Related: ADRs 0415, 0436, 0439, 0440

## Decision

Extend ADR 0439 shared `std::vec::view<T>(ref Vec<T>) -> ref T[]` and
ADR 0440 copied-value for-in to acyclic nested structs. For example:

```fairy
struct Span { usize start; usize end; }
struct Token { Span position; i64 value; }
```

Every reachable struct must be an actual nonempty definition. Leaves must be
`bool`, `u8`, `i32`, `i64`, `u32`, `usize`. Repeated sibling types are valid.
Self/indirect cycles, missing definitions, empty nested structs, references,
raw pointers, resources, enums, fixed arrays, Option and Result fields remain
rejected at every depth. Mutable struct slices and fixed-array conversion remain
outside the shared vector-view contract.

## Implementation and semantics

Semantic analysis and the private MIR verifier use `hir.PlainViewStruct` with
stage-owned definition lookup. An explicit DFS stack detects visiting-node cycles
and memoizes completed nodes, doing work proportional to reachable definitions
and fields per invocation without recursive host-stack growth. No guessed source
depth limit is added; other frontend and backend aggregate limits remain owned
by those stages.

An indexed read or for-in item copies the entire nested element value. Mutating
a local copy does not mutate the owner, and a copy survives owner clear. The
view itself retains its whole-owner loan and exact nominal type. Loop-held
loans, returned/tagged origins, bounds, subviews and last-use rules remain as
specified in ADRs 0415 and 0440.

The existing C backend orders aggregate dependencies and copies concrete nested
records through the nominal checked load. The descriptor stays one
`const void *`/`uintptr_t` record. The existing MIR codec retains struct field
references and definitions; no wire schema, runtime symbol or SDK export changes.
SDK stage0.84, language /20, runtime /0 and grammar are unchanged. Compiler
executable identity separates cached generations.

## Verification and limits

See `docs/ops/evidence/struct-nested-20261004.json` for recorded checks and hashes.
`examples/struct_vec_view` now stores a nested `Span` in each token and checks
its range during for-in analysis. Source and MIR rejection tests exercise unsafe
leaves at depth, cycles and missing definitions. Both scalar and nested loop MIR
roundtrip and warm all-hit cache are covered; the graph unit test also exercises
a 10,000-edge chain and a shared-child diamond.

Windows development/release each pass five nested-struct cases, seven existing
struct-iteration cases, eight scalar struct-view cases and twelve token-analyzer
cases (64 native outcomes total). Runtime checks cover nested copy independence,
copy after clear, deep indexed/for-in reads, returned subviews, repeated child
types, large signed values, empty views and checked bounds traps. The existing
token analyzer and scalar struct suites remain regression coverage. Public CLI
check/locked build/run and Go vet are recorded alongside the native receipts.

Linux execution is deferred to the owner's batch. No performance or target
maturity claim, broad suite, benchmark, new runner, installation, CI workflow or
push is included.
