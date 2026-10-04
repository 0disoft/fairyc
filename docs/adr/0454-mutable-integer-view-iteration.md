# ADR 0454: Mutable integer view iteration

- Status: implemented; Windows native verified, Linux execution deferred
- Date: 2026-10-04
- Related: ADRs 0415, 0435, 0450, 0453

## Decision

Admit `for (let ref mut T item in values)` for T = u8, i32, i64, u32 or
usize over a named immutable `ref mut T[]` descriptor or parameter. Each item
references the stored element, rather than a copied indexed value. Fixed arrays
and SDK Vec owners first produce a named mutable view. Exact widths and
mutability must match; qualified type resolution follows existing rules.

```fairy
fn void clamp(ref mut i64[] values, i64 lower, i64 upper) {
    for (let ref mut i64 item in values) {
        if (*item < lower) { *item = lower; }
        if (*item > upper) { *item = upper; }
    }
}
```

Shared views cannot yield mutable items. Shared reference item syntax, mutable
descriptor bindings, temporary call/slice iterable roots, direct fixed-array
roots, bool views, raw pointers and owned slices remain outside this admission.
Existing copied integer loops and nominal view iteration retain their behavior.

## Lifetime and lowering

Reuse the existing AST preparation: save length once, iterate an internal usize
index, bounds-check and obtain `ref mut values[index]` before the written body.
Continue runs the hidden increment; break and return do not. Empty views obtain
no element reference. Element stores use checked scalar arithmetic. Explicit
`let T copy = *item` retains an independent value copy.

The current item introduces an exclusive child generation. Parent/alias access
is frozen while the item or a descendant is live; last use releases that child.
The whole owner remains held through the written loop's exit, including break,
continue, return and Result propagation. Structural Vec mutation stays rejected
even after the item's last use. An item returned from a parameter retains its
origin and generation; local-owner escape and competing references retained
across iterations remain rejected. Indices do not establish disjointness.
Independent owners can be iterated in nested loops.

Semantic admission in `internal/semantic/byte_for.go` extends ADR 0450's item
gate and explicitly checks exclusive integer receivers. Existing AST, HIR/MIR,
codec, interpreter and typed C emission paths already implement the stored-cell
borrow added by ADR 0453. No new syntax node, MIR operation, wire field, SDK API,
runtime helper, alias promise or host compiler flag is needed.

## Verification and limits

Source tests cover all five types, last-use parent access, live parent/alias
conflicts, break/continue/return, nested competing loops, item scope, shared
sources/items, descriptor mutability, temporary/direct-array roots, mismatched
integer widths and local-owner escape. Public check tests reject Vec clear/push,
retained returned references and previous-iteration item retention.

The array-backed fixture in `internal/compiler/testdata/mutable_integer_for.fy`
executes after MIR codec round-trip in the interpreter, and emits through both
Windows MSVC and Linux C profiles. It checks copies, ordered stored mutations,
offset/empty subviews, values above i32, continue/break, early Option return,
Result propagation and independent nested owners. Verifier/codec/backend
negative controls reject absent bounds, shared receivers and wrong item types.

Windows dev/release execution checks that matrix, SDK Vec edits and subsequent
owner release, checked overflow for all five types, and a trapped RHS. Public
CLI check/build/run and a clipping example provide standalone use evidence.
Related regressions and Go vet pass; raw results and artifact hashes are in
`docs/ops/evidence/mutable-integer-for-20261004.json` and its companion receipts.

SDK stage0.85, language /20, runtime /0 and target maturity remain unchanged.
Linux actual execution is deferred to the owner's batch. No benchmark, CI,
installation, push or release is included.
