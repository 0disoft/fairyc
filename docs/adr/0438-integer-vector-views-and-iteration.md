# ADR 0438: Integer vector views and iteration

- Status: implemented; Windows native, Linux execution deferred
- Date: 2026-10-04
- Related: ADRs 0415, 0435, 0436, 0437

## Decision

SDK stage0.83 adds shared/exclusive initialized-prefix views for Vec<i32>,
Vec<u32> and Vec<usize> through `view_i32`/`view_mut_i32`, `view_u32`/
`view_mut_u32` and `view_usize`/`view_mut_usize`. Exact element types are
required. Bounds count elements, capacity is excluded, and empty null-backed
vectors/subranges are valid. Descriptor construction allocates/copies no elements.

Named immutable view descriptors and parameters support copied-value for-in
for all five executable integer view elements, including the new fixed-array
views. Length is snapshotted once; checked reads happen at each body entry.
Stores to future positions are visible. The whole owner stays held across all
loop exits, including early return. Reference items, mutable descriptor bindings,
call/slice-expression iterables and implicit Vec conversion stay unsupported.

Live views/returned derivatives/tagged payloads retain owner loans. Shared loans
block structural mutation/replacement/move; exclusive loans also block other
owner reads/borrows. No-op reserve/truncate receives no exception. Short shared
readers and exclusive subviews use existing reborrow rules. ADR 0415 still freezes
ancestors and retained siblings, including disjoint/empty ranges. Local owner
escape is E2102. Released loans permit growth, fresh views and consuming moves.

## Implementation and identities

The SDK-owned private storage path uses the closed HIR integer-view set, exact
three-operand unsafe transfers and initialized/aligned Vec storage. MIR transfer
verification checks actual owner/address/count definitions, result/operation
mutability, usize length, unsafe provenance and destination separation. Return
verification prevents relabeling descriptors. Ordinary/query-only paths still
reject these transfers. The existing mutation oracle covers every new element
and both mutabilities through private codec decoding.

C storage views reuse centralized exact pointer/length descriptors; runtime C
symbols are unchanged. The i64 MSVC workaround remains restricted to i64. No
host-storage interpretation, public raw builtin or foreign ABI is introduced.
AST loop preparation now admits the three scalar item spellings while semantic
analysis continues to enforce exact matching views. Interface serialization,
snapshot generic slices and function/body identities recognize the same set,
so package cache artifacts cannot lose the new element types.

SDK source/interface and dependent lock identities refresh at stage0.83.
Following ADR 0430-0437, language /20, runtime /0, grammar, MIR/artifact and
interface schemas stay unchanged; compiler executable identity separates cached
compiler generations. This is no released compiler or target-maturity claim.

## Verification and limits

The example exercises fixed-array and vector iteration, signed/unsigned extrema,
empty/singleton views, returned/tagged/subrange origins, shared readers, in-loop
stores, growth after release and vector lifecycle operations. 102 source
rejection cases cover owner changes, no-op mutations, move, early-return move,
parent/sibling conflicts, tagged origins, effectful bounds, local escape,
wrong Vec element and user private builtins. Twelve MIR mutations per element
and mutability reject missing/wrong lengths, local-annotation laundering,
wrong element/mutability/kind, typed addresses, missing unsafe and aliasing.

Windows native development/release executes 44 cases: the example, end/huge
reads, empty stores, reversed/past-end slices and checked overflow/underflow in
loops. Updated-compiler u8/i64 native examples run in both profiles. Owning
package tests, fixed-array loop interpreter/codec checks, focused compiler/cache
checks, Go vet, public CLI and identities are in
[the receipt](../ops/evidence/vec-integer-views-20261004.json).

Linux actual execution remains deferred to the user's batch; emitted Linux C
is not runtime proof. Full repository/scanner experiments are not repeated.
No PS1 runner, dependency installation, push or performance claim is added.
