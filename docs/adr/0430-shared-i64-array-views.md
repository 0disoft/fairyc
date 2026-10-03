# ADR 0430: Shared i64 fixed-array views

- Status: Implemented (Windows native evidence; Linux execution deferred)
- Date: 2026-10-03

## Decision

Admit `ref i64[]` as a read-only, non-owning view over a directly named `i64[N]`
local or by-value parameter. A contextual `ref values` conversion retains its
array's origin and whole-owner shared loan. Arrays retain the existing positive
length and 4,096-element implementation bound. No new allocation, implicit
element conversion, user generic aggregate, Vec view or mutable i64 view is added.
Unsized `i64[]` by value and `ref mut i64[]` remain unsupported.

The view supports `.len`, `.is_empty`, checked `view[index]`, and half-open
`view[start..end]`, including omitted bounds and empty subviews at the end.
Length and bounds count i64 elements, not bytes. The receiver and explicit
indices/bounds evaluate once in source order; index requires `index < len`,
slice requires `start <= end <= len`. Failure is the ordinary `bounds` trap.
The selected receiver's loan remains held while an effectful index or slice
bound runs. This also closes the corresponding shared byte-view Flow gap.

Shared siblings, aliases, nested slices and parameter-derived returned views
retain the original loan. Existing Option/Result origin tracking also applies;
the C tagged-value transport accepts the shared i64 descriptor as a payload.
Mutation/replacement of the owner is rejected until the last retained view use.
Returning a local array view or a view of a by-value parameter is E2102;
ambiguous-origin return rules are unchanged. No view element store is admitted.

## Implementation and compatibility

HIR adds `[i64]` to the closed borrowed-slice vocabulary. Existing MIR array-view
borrowing, view metadata, checked index and checked slice operations carry the
new exact owner/element types. Verifier checks derive actual receiver and bound
types from local definitions, rejecting byte/i64 descriptor laundering and
missing bounds guards. Interpreter views retain the original array backing and
element offset. Artifact round trips preserve the type and signed values.

Private C uses `fy_i64_slice { const int64_t *data; uintptr_t length; }`, with
typed element loads and pointer offsets only after checked edges. Descriptor
creation/slicing does not copy array elements or call an allocator. This is an
internal calling representation, not a public C ABI or a timing guarantee.
Generated-name collision checks reserve `fy_i64_slice`.

Public-interface spellings use `ref i64[]`; function artifact identities and
semantic snapshot builtin identities retain the new type. No grammar spelling,
MIR field, serialized schema, SDK export or runtime symbol is added. Existing
compiler-executable cache identities invalidate prior executable generations.
Language `/20`, SDK stage0.79 and runtime `/0` are unchanged, as with the earlier
additive array/byte-view admissions. No contract-state target is newly admitted.

## Evidence and limits

The example at `examples/i64_view` exercises signed extrema, values beyond i32,
sum over a subrange, omitted/empty/nested slices, simultaneous shared siblings,
parameter-derived and tagged returns, and owner mutation after the last use.
Compiler tests execute a MIR encode/decode round trip, emit both target profiles,
and reject altered owner/result/bound types and missing slice checks. Semantic
tests reject live-loan owner mutation, competing exclusive borrowing, effectful
bound/index mutation, local escape, wrong element types and view stores.

Windows native tests cover the example, a local-only descriptor, ordered bound
side effects and index/slice traps in low/optimized profiles. Raw images and
source/image hashes are retained in `.cache/i64-view-native/receipt.json`.
The ordinary CLI `check`, development/release `build` and locked `run` also
pass on a copied example workspace. Related semantic/compiler regressions,
MIR/interpreter/backend/interface/artifact/snapshot tests and `go vet` pass.
A broader compiler package run exceeded its five-minute limit inside existing
private Vec native experiments; it is not a full-suite pass and was not repeated.
An obsolete storage-query rejection that still treated admitted i64 as invalid
now checks the unadmitted mutable i64 view instead.
Linux C emission and test-binary cross compilation are checked. The
`TestI64ViewLinuxNative` cases are ready, but Linux compiler execution is
deliberately deferred to the user's batch validation. Benchmarks and physical power-loss
tests are outside this feature; no performance improvement is claimed.
