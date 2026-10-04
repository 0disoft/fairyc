# ADR 0458: Shared integer reference iteration

- Status: implemented; Windows native verified, Linux execution deferred
- Date: 2026-10-05
- Related: ADRs 0415, 0435, 0453, 0454, 0455, 0457

## Decision

Admit `for (let ref T item in source)` for u8, i32, i64, u32 and usize.
The source is a named fixed integer array or an immutable named binding of
exact type `ref T[]` or `ref mut T[]`. Items point at original stored elements.
The array binding may be immutable or mutable; a shared item cannot store.
Array parameters refer to their local by-value storage. Existing fixed-array
length limits remain `1..=4096`; named views may be empty or sliced before use.
Bool/struct shared items, computed iterable roots and width conversions remain
outside this admission. Existing copied and exclusive item loops keep their
contracts. No new syntax production, SDK export or runtime helper is introduced.

```fairy
fn Option<ref i64> find(ref i64[] values, i64 needle) {
    for (let ref i64 value in values) {
        if (*value == needle) { return .some(value); }
    }
    return .none;
}
```

The prepared loop captures length once and walks checked ascending indices.
Each item uses the existing shared view-element borrow. Direct arrays use the
existing private array descriptor of ADR 0457, without copying array storage.
`continue` advances the hidden index; break, return and Result propagation
retain ordinary loop cleanup. Returning an item from a borrowed parameter or
retaining it in `Option<ref T>` uses existing owner provenance and liveness.
Local and by-value array references cannot escape. Owner mutation and exclusive
reborrows remain blocked throughout the written loop, including after an item's
last use; after exit, any still-live retained reference continues to constrain
the owner. Shared reads and nested shared array loops are allowed.

## Preparation and Flow boundaries

Syntax lowering keeps shared-reference source loops intact. After authenticated
SDK Vec preparation, name resolution prepares remaining named integer loops.
SDK preparation tracks lexically declared array/view parameters and bindings,
including local shadowing, so it does not register Vec calls for those sources.
Semantic analysis still validates exact resolved types and source shapes.
Retained array AST preparation is reset before a fresh analysis, preserving
written body symbols and deterministic MIR.

Vec and array descriptor blocks both have three statements. Flow must distinguish
them by the first private binding identity rather than statement count. The
previous array dispatcher could skip Vec Flow entirely; this change restores
ordinary Vec owner-binding evaluation. Existing Vec tests now reject owner
mutation/exclusive aliases/moves and direct/aliased local item escape again.
Unsupported Vec forms retain exact source diagnostic spans. Mixed integer
array/view/Vec loops are checked through the authenticated candidate lane.

Named mutable integer views acquire a shared loop access in addition to the
held descriptor, preventing writes through the view or its alias even after
the current item's last read. Array descriptors retain the same whole-owner
loan. Multiple disjoint mutable siblings are not admitted by this change.

## Verification and compatibility

Source Flow tests cover all five integer types, shared/mutable views, fixed
arrays, returned references, nesting, owner writes/exclusive reborrows, aliases,
shared-item stores, local/by-value escape, descriptor mutability and item scope.
The stored matrix executes after MIR codec round-trip, checks repeated retained
AST analysis, and emits through Windows MSVC and Linux C profiles. It includes
values above i32, first/last elements, empty subviews, no-match/early-return
searches, break/continue, Result propagation, nesting and shadowing. Verifier
mutations reject removed bounds, mutable results, wrong widths and forged array
owners before codec and C emission.

Windows dev/release native runs check the matrix and a trapped shared-item body.
A separately built compiler checks source errors and runs locked builds of the
matrix and `examples/shared_integer_ref_quickstart`; exact exit/stdout/stderr
and compiler/source/SDK/image hashes are recorded in
`docs/ops/evidence/shared-integer-for-20261005.json` and companion receipts.
Related regressions and Go vet pass. Language /20, SDK stage0.86, runtime /0,
MIR/wire schemas and ABI are unchanged; compiler identity separates caches.

Linux actual execution is deferred to the owner's batch. No performance claim,
benchmark, installation, CI change, push or release is included.
