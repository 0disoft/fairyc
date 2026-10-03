# ADR 0434: Exclusive i64 vector views

- Status: implemented (Windows execution; Linux execution deferred)
- Date: 2026-10-03

## Decision

SDK stage0.81 adds `std::vec::view_mut_i64(ref mut Vec<i64>) -> ref mut i64[]`
over exactly the initialized prefix. The pointer/length descriptor allocates
nothing and copies no elements. Indices and lengths count i64 elements. Empty
vectors, including null-backed storage, are allowed; empty slicing avoids null
pointer arithmetic. Stores and compound assignment preserve checked i64 semantics.

Until the last retained view/alias/subview use, reject owner reads, other owner
borrows, push, reserve, pop, truncate, clear, replacement and move. This includes
structural operations that would not change the allocation or length. Obtain a
fresh view after mutating the vector. Owner-derived and tagged returns retain
the root origin; local escape remains E2102. Shared readers use shortened shared
reborrows. Subviews reuse ADR 0415's whole-parent freeze and generations.
Retained siblings remain rejected even if disjoint or empty. Generic element
slices, mutable for-in and user raw-storage intrinsic admission are not added.

## Implementation and compatibility

SDK-authorized expansion adds the three-argument private form
`storage_borrow_mut<i64[]>(owner, address, element_count)`. The owner must be
exclusive and the opaque address mutable. The wrapper supplies its own Vec
data/length fields; initialized extent, alignment and owner/address association
remain the unsafe SDK producer's obligation. All arguments evaluate once and
are materialized in source order. Flow holds the owner while address/count
expressions run; effectful owner mutation there remains E2101.

The existing `storage-borrow-mut` MIR kind carries the count for this exact
result type. Transfer verification checks operation kind, result and owner
mutability, actual owner/address/count definitions, unsafe provenance and
destination separation. Ordinary/query-only verification still rejects private
transfers. The storage codec round trip reruns verification. C emission constructs
`fy_mut_i64_slice` with `int64_t *` and usize length. Host-storage interpreter
emulation is not added. Fixed-array view operations and the Windows access
workaround from ADR 0433 are reused.

The new SDK export advances stage0.80 to stage0.81 and changes vector source,
public-interface and dependent lock identities. Language `/20`, runtime `/0`,
grammar, MIR/artifact schemas and C runtime symbols remain unchanged. Exact
compiler identity separates disposable cached compiler generations.

## Verification and limits

[The Fairy example](../../examples/vec_i64_mut_view/src/main.fy) checks empty
storage, signed extrema, values above i32, subrange reversal, nested and
returned/tagged mutation, shared-reader reborrowing, owner-visible results,
reserve beyond prior capacity and a fresh view, subsequent push/pop/truncate/
clear, replacement and move after the last retained use.

Public rejection tests cover live-loan owner operations, read/borrow, parent
access, conservative siblings, effectful bounds, derived/tagged origins, local
escape, shared-owner misuse, wrong element type and user intrinsic use. Private
source tests reject wrong owner/address mutability, missing/wrong count, wrong
element type, missing unsafe scope and effectful address/count mutation. Shared
and mutable transfer codec/MIR mutation tests reject kind/result mismatch,
missing counts, actual-local type laundering and bad provenance. C emission is
checked for Windows and Linux profiles.

Eight Windows cases pass in development and release (16 executions): the
example, end/huge/empty stores, reversed/past-end ranges and signed overflow/
underflow. Related generic/SDK/semantic/MIR/backend/codec tests, shared/public
borrow regressions, private compiler transfer tests and Go vet pass. Ordinary
CLI check, both builds and locked runs also pass on a copied example. The
[receipt](../ops/evidence/vec-i64-mut-view-20261003.json) preserves source/SDK/
compiler/image identities and raw outcomes. Images remain in
`.cache/vec-i64-mut-view-native/windows/`.

Linux execution is deferred to the user's batch; the same native test is ready
through the existing harness. C emission is not a Linux execution claim.
No full-repository suite or performance experiment was run. The existing MSVC
volatile-access workaround has unmeasured optimization cost and does not
provide source atomicity or concurrency semantics.
