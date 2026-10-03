# ADR 0431: Shared i64 vector views

- Status: Implemented (Windows native evidence; Linux execution deferred)
- Date: 2026-10-03

## Decision

SDK stage0.80 adds `std::vec::view_i64(ref Vec<i64>) -> ref i64[]` over the
initialized prefix. Length, indexing and slicing use i64 element counts.
The descriptor contains the existing `const int64_t *data` and `uintptr_t length`;
construction allocates nothing and copies no elements. Empty storage may have
a null pointer and zero length. Empty slicing does not perform null pointer
arithmetic. Invalid index/range access uses the existing `bounds` trap.

The whole vector remains shared-borrowed until the last retained use, including
aliases, subviews, parameter-derived returns and tagged payloads. Mutation,
replacement and move are forbidden during that interval. `reserve` is blocked
even when the requested capacity would not cause a reallocation. After the
last use, mutation is permitted and callers obtain a new view. Local-owner
escape remains E2102. Mutable i64 views and generic element slices are deferred.

## Implementation and identities

Only SDK-authorized storage expansion admits the three-argument private form
`storage_borrow<i64[]>(owner, data, element_count)`. Ordinary user packages do
not gain this intrinsic. The first argument is a shared safe owner borrow,
the second an opaque raw pointer, and the third a usize. The unsafe caller
proves alignment, initialized extent and address association with the owner;
the public SDK wrapper supplies its own vector fields. All arguments evaluate
once in source order and are materialized before subsequent arguments. Flow
holds the owner while the address and count are evaluated.

The existing private `storage-borrow` MIR operation carries three operands for
this exact result type. Transfer verification checks shape, actual local types
and unsafe provenance. Byte/other-element and mutable descriptors cannot be
substituted. Ordinary and query-only MIR consumers continue rejecting private
storage transfers; the private storage codec round trip reruns verification.
Native C emission creates the existing typed i64 descriptor directly.

The SDK version advances to `0.0.0-stage0.80`. SDK source digests, public
interfaces and dependent locks refresh. Language `/20`, runtime `/0`, grammar,
MIR/artifact schemas and C runtime symbols remain unchanged. This adds an SDK
API and a private transfer form, not a publicly callable raw-parts constructor
or a public C ABI. Executable identity separates compiler generations.

## Evidence and limits

`examples/vec_i64_view` checks empty storage, signed extrema, values above i32,
shared siblings, nested and omitted slices, parameter/tagged returns, subrange
sums, actual growth beyond previous capacity, later push/pop/truncate/clear,
and owner move after the last view use. Windows development and release native
cases also check end-index, empty-index, reversed-range and past-end traps.
Their images and hashes are retained under `.cache/vec-i64-view-native/windows/`.

CLI source rejections cover live-view mutation, exclusive element access,
move/call-move/replacement, view stores, derived/tagged origins, effectful
index/bounds, local escape, wrong element types and user private-intrinsic use.
Private source tests cover malformed/mutable forms, missing unsafe scope and
effectful address/count mutation. Codec/MIR mutations cover missing/wrong count,
local-type laundering, incompatible result/owner/address types and provenance.
Both Windows and Linux C profiles are emitted in tests. Linux native execution
is deferred to the user's batch; no performance measurements or universal
zero-cost claim are made.
