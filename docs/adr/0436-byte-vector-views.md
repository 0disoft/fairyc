# ADR 0436: Shared and exclusive byte vector views

- Status: implemented (Windows native execution; Linux execution deferred)
- Date: 2026-10-03
- Related: ADRs 0415, 0431, 0434, 0435

## Decision

SDK stage0.82 adds `std::vec::view_u8(ref Vec<u8>) -> ref u8[]` and
`std::vec::view_mut_u8(ref mut Vec<u8>) -> ref mut u8[]`. Both borrow exactly
the initialized prefix without allocating or copying elements. Lengths and
indices count bytes. Capacity beyond length is excluded; empty null-backed
vectors and empty subranges are valid. Bounds and checked byte arithmetic
retain their existing trap behavior.

Shared views and retained aliases/subviews keep the whole vector borrowed,
blocking structural changes, replacement and consuming moves. Exclusive views
also block other owner reads/borrows. Short shared readers and exclusive child
views use the existing reborrow rules. ADR 0415 continues to freeze ancestors
and retained siblings, even when their ranges are disjoint or empty. Derived
and tagged returns preserve the vector origin; local-owner escape remains
E2102. A released view may be followed by growth and a fresh view. No no-op
reserve/truncate exception bypasses the loan.

Both view types support the existing named-view for-in forms. Iteration copies
u8 items, holds the owner through early exits and permits indexed writes only
through exclusive views. Values inserted into later positions are observed by
later loads. There is no implicit Vec-to-slice conversion, generic vector slice
API, byte/i64 reinterpretation, public raw-storage builtin or new foreign ABI.

## Shared implementation and verification boundary

The former i64-specific storage-view path is now shared by the closed u8/i64
set. SDK-authorized expansion accepts the three-argument private shared and
exclusive storage borrows for the two slice types. The unsafe SDK wrapper
provides its Vec owner/data/length fields and proves initialized extent,
alignment and the owner/address association. Argument evaluation is ordered
and materialized before constructing the descriptor. Flow holds the owner
while the address and count expressions run.

MIR transfer verification checks operation/result/owner/address mutability,
actual owner/address/count definitions, usize count, destination separation and
unsafe provenance. Direct private descriptor results must match the callable
return type using actual local definitions, so expanding the closed type set
does not permit relabeling u8 as i64 (or vice versa) by changing a result or
operand annotation. Ordinary and query-only lanes still reject private transfers;
private codec decoding reruns verification.

The C backend selects the existing byte or i64 shared/exclusive descriptor.
Actual native execution exposed that byte-view tagged payloads were outside
its prior capability subset, despite semantic acceptance. The existing tagged
payload admission and layout now recognize u8 views as pointer/length
structures, matching the i64 transport. Shared and exclusive Option payload
length/element checks are included in the native example.
No runtime allocation, new C runtime symbol or interpreter host-storage
emulation is added. Existing i64 emission and its Windows-only workaround
remain intact. The SDK advances from stage0.81 to stage0.82; source/interface
and dependent lock identities refresh. Language `/20`, runtime `/0`, grammar,
MIR and artifact schemas and target maturity remain unchanged.

## Validation and limits

[The Fairy example](../../examples/vec_u8_view/src/main.fy) exercises empty
shared/exclusive views, singleton iteration, 0/127/128/255 bytes, shared
summation, returned/tagged/subrange views, reversal, shared-reader reborrowing,
exclusive for-in/future-position mutation, release followed by growth,
post-growth views, push/pop/truncate/clear, replacement and consuming move.

Source tests reject shared stores, no-op structural mutations under live loans,
replacement/move, early-return consumption, owner reads and competing views,
parent/sibling conflicts, effectful bounds, tagged origins, local escape, wrong
Vec element type and user private builtin use. Private-source tests cover count,
unsafe and effectful argument rejection. Shared/exclusive u8 and i64 MIR codec
mutation tests cover operation/result mismatch, actual-local type laundering,
wrong element result, missing counts and unsafe provenance. Both Linux and
Windows C profiles are checked for exact descriptor selection.

Windows development/release execution covers the example, end/empty/huge
reads, reversed/past-end slices, end/empty stores and u8 overflow/underflow,
plus the existing i64 mutable-vector example as a regression. Owning Go package
tests, public check/build/run and Go vet are recorded in
[the evidence receipt](../ops/evidence/vec-u8-view-20261003.json), together with
source/SDK/compiler identities, image hashes and raw execution outcomes.

Linux native execution remains deferred to the user's batch; C emission is
not a Linux execution result. The native test uses the existing portable Go
harness. No PowerShell runner, dependency install, full-repository suite or
performance experiment is introduced. The i64 MSVC workaround's optimization
cost remains unmeasured; it is not a byte-view workaround or concurrency contract.
