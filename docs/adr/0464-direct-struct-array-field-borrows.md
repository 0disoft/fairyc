# ADR 0464: Direct struct array field borrows

- Status: implemented; Windows native verified, Linux execution deferred
- Date: 2026-10-05
- Related: ADRs 0415, 0449, 0451, 0455, 0462, 0463

## Decision

A named fixed array of an admitted plain scalar-field struct (ADR 0462) admits
direct field references `ref array[index].field` and nested paths such as
`ref mut array[index].inner.field`. The referent is the stored field address in
the original array storage, never a copied element. Scalar fields and whole
nested plain-struct fields are admitted. Shared references also work from
immutable local or by-value parameter array bindings; exclusive references
require a mutable array root. Grouping preserves place identity.

```fairy
var Point[4] polygon = [origin; 4];
points::add_x(ref mut polygon[0].x, 10);
points::add_x(ref mut polygon[3].x, -10);
let ref i64 first_x = ref polygon[0].x;
assert(*first_x == 4000000010);
```

Shared and exclusive reference arguments and returned identity-tagged wrappers
keep their existing origin and generation flow. Resources, raw pointers,
reference-typed struct fields, tagged fields and array fields remain excluded;
exact nominal identity and whole-owner borrow rules still apply.

## Lifetime and evaluation

The whole original array is the loan owner and grouped roots preserve that
identity. A checked element borrow selects the element in the original storage
and the existing field-borrow projection then addresses the field, so a
reference never points at a copied element value. The index evaluates exactly
once and its bounds check dominates the field address.

Sibling fields or elements never prove disjointness. A mutable field child
freezes the root and every alias until its last use; simultaneous mutable
siblings remain rejected. Shared children coexist and read, and an owner write
while a shared child is live is rejected. An exclusive child requires no live
shared or exclusive sibling. Direct call arguments stay held until the enclosing
call completes. References into local or by-value arrays cannot escape, and
temporary or call roots are rejected.

## Implementation

`internal/semantic/field_borrow.go` adds an `arrayFieldBorrow` helper that
rebases each field projection on a checked element borrow of the original array.
The array-field prefix is classified by `indexedPlace` and flows through
`evaluateArrayElementBorrow`/`evaluateArrayPlace` in
`internal/semantic/indexed_place.go` and `internal/semantic/borrow_flow.go`, which
retain the whole-array origin and a pending mutable subview loan instead of
borrowing a copied element. The projection then reuses the existing
`array-view-borrow`/`view-element-borrow` operations and `projectFieldBorrow`.
Shared and exclusive reference calls and returned identity-tagged wrappers reuse
the existing origin checks.
`evaluateFieldBorrowRoot` remains the pre-existing named reference-field path and
is not the new array route. No MIR operation, SDK API, runtime entry or host flag
is added, and grammar productions are unchanged. Language /20, SDK stage0.86 and
runtime /0 stay unchanged.

A direct field-reference argument can terminate its enclosing block. An
`internal/mir/lower.go` `lowerCall` guard now returns as soon as an argument's
block has terminated (`blockTerminated` after `lowerExpression`), instead of
lowering the remaining arguments and the call.
`TestStructArrayFieldBorrowIndexReturn` pins it through interpreter execution and
C emission: when an argument returns early, a later argument's `assert(false)`
and all trailing statements do not run.

## Verification and limits

The related suite reports 805 PASS, 0 FAIL and 0 SKIP. The Windows native harness
runs the matrix, shared-end, exclusive-end, whole-field-end and index-return cases:
7 tests pass, accounting for 10 real outcomes across the dev and release profiles
plus a separate cache reuse test. The separate fixed compiler's public CLI
`check`/`build`/`run` covers the matrix, the example, shared-end, exclusive-end
and index-return in both profiles (10 outcomes), rejects the immutable, loan,
escape and temporary diagnostics, and records raw UTF-8 argv, cwd, stdout, stderr
and exit hashes. Go vet, formatting and the diff check pass.
`TestStructArrayFieldBorrowIndexReturn` also covers interpreter execution and C
emission. Linux actual execution is deferred to the owner's batch.

Evidence: `docs/ops/evidence/struct-array-field-borrow-20261005.json` with its
`.cli.json` and `.native.json` companions. Language /20, SDK stage0.86, runtime /0,
MIR/wire schemas, public ABI and target maturity are unchanged. No CI,
installation, benchmark, push or release is included.
