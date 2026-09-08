# ADR 0390: Public Windows SDK vector

- Status: Accepted
- Date: 2026-09-09
- Depends on: ADR 0388 and ADR 0389

## Decision

SDK `0.0.0-stage0.58` admits the move-only `std::vec::Vec<T>` through ordinary
manifest loading and stable `check`, `build`, `run`, and native `test` on Windows
AMD64 MSVC. Language revision stays `fairyc-language/14`; runtime ABI stays
`fairyc-runtime/0`. This is SDK-owned template admission, not general user-defined
generic aggregate or nominal-argument function admission.

The public operations are explicit generic functions: `empty<T>`, `len<T>`,
`is_empty<T>`, `capacity<T>`, `reserve<T>`, `push<T>`, `pop<T>`, `clear<T>`, and
`at<T>`. Empty construction does not allocate. Reserve takes a minimum element
capacity and returns `Result<usize, u32>` using the allocation-status vocabulary.
Push consumes one element and returns `Result<usize, T>`; `.err` returns the
original element. Allocation failure preserves the vector. Pop returns
`Option<T>` and transfers ownership. Clear drops the initialized elements in
reverse order, returns their count, and retains capacity. Cleanup drops the
remaining initialized prefix and releases its allocation. Spare capacity is not
initialized storage and is never exposed to safe code.

`at<T>(ref Vec<T>, usize)` returns an owner-bound immutable `ref T`. An invalid
index traps with `assertion-failed`. While the reference is live, mutation or
movement of the vector is rejected. Type arguments are supported builtin scalars
or directly named non-generic structs/resources whose current native storage
layout is admitted. Zero-sized, over-aligned, borrowed and composite type
arguments, nested generic containers, mutable access and iteration remain outside
this release. Generic element layout is checked by the native backend; semantic
check alone is not a promise of target representability.

## Loading, authority and compatibility

The SDK keeps its existing six-module core. An exact resolved `std::vec` import
selects a core-plus-vector bundle under the same standard SDK identity, with its
own complete source digest. Test-only imports select that bundle only in the
ephemeral test workspace; the production lock remains based on production source.
The loader rebuilds the SDK selection after test overlays, including local
dependencies. Programs that do not import Vec keep the prior compiler and Atlas
paths. There is no new CLI flag, manifest field, implicit import or host discovery.

Every storage-aware check/cache/native path authenticates the exact bundled
source bytes, ASTs, ordered paths, module catalog, identity and resolved owner.
Only SDK owners may use private storage operations, generic aggregates or nominal
template arguments. User generic functions keep the builtin-scalar restriction.
Native consumers cannot declare private vector runtime symbols. Cache keys retain
the SDK-owned discriminator and exact current-source analysis; core/vector mode
transitions use one active cache family. The private candidate identity remains a
separate test lane, not a user-facing SDK option.

Existing lockfiles must be refreshed once for the SDK version change. Runtime
support composes one heap implementation with the private vector transport and
the existing product trap channel. No vector C ABI or portable cache format is
published. Interpreter-backed workspace tests report that `--native` is required
instead of treating storage-aware MIR as an internal compiler error. Test listing
and source diagnostics remain available without executing the program. Atlas
inspection of Vec source and non-Windows execution are not promoted here.

## Evidence

`TestWindowsVectorStableCLI` executes ordinary check/build/run/native-test dispatch
under low and optimized MSVC, checks executable creation and exact output/exit
status, and requires a source diagnostic for forbidden consumer templates.
The native test harness also preserves trap-then-pass isolation, captured output
and JSON source locations. The default-loaded two-package workspace obtains
memory and reopened disk-cache hits, preserves owner-bound element borrows and
executes resource cleanup through the independent allocation probe. Invalid source,
forged SDK data and unadmitted consumer templates cannot change cache statistics.
`examples/std_vec_quickstart` demonstrates scalar push, borrow, pop and clear.

The failure-injection and initialized-prefix evidence remains owned by ADR 0388;
borrow/alias restrictions remain owned by ADR 0389. This decision does not claim
sanitizer coverage, non-Windows runtime support or an independent compiler lineage.
