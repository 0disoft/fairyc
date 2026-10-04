# ADR 0452: Plain struct dereference replacement

- Status: implemented; Windows native verified, Linux execution deferred
- Date: 2026-10-04
- Related: ADRs 0401, 0415, 0441, 0445, 0448, 0450, 0451

## Decision

Admit `*target = value` through a named `ref mut T` (grouping the named
reference is allowed), where T is an actual nonempty acyclic plain struct
with scalar leaves, using the existing ADR 0441 graph predicate. The RHS
must have the exact nominal type. Replacement copies all scalar/nested values;
resources, pointers, arrays, borrowed and tagged fields remain outside this
admission. Shared receivers, compound stores, call-result/temporary reference
roots and implicit resource moves stay closed. No new syntax or SDK API is added.

```fairy
fn void replace(ref mut Token target, Token value) {
    *target = value;
}
let ref mut Token token = ref mut items[0];
replace(token, replacement);
```

Named local borrows, projected struct children, indexed element borrows,
mutable for-in items and parameter-origin returned/tagged references share
this rule. This supersedes the whole-dereference exclusion in ADRs 0401,
0450 and 0451; other exclusions remain in place.

## Evaluation and lifetime

Save the reference descriptor before evaluating the RHS once. Rebinding that
variable during the RHS changes later uses, but not the selected destination.
Flow retains the original selected loan through the final store. Live children
or shared aliases block replacement; parent/owner/competing accesses and owner
moves remain subject to existing borrow generations. Reads through the selected
reference (including `*target = *target`) and shared child reads whose last use
precedes the store are allowed. Last use releases the hold as before.

Indexed address selection still requires its checked bounds branch before RHS
evaluation. A return or trap in the RHS has no store or normal continuation.
This feature changes stored values without reallocating the owner or transferring
cleanup responsibility.

## Implementation

Reuse MIR `deref-store` and its existing wire fields. Keep the scalar
`ValidDerefStoreShape` boundary; a sibling `ValidPlainStructDerefStoreShape`
checks the declaration-owned plain graph and closed metadata. Verification
checks the concrete receiver and value local types, so operand annotations
cannot relabel a shared reference or another nominal value. Malformed stores
are rejected before codec and backend consumption. The source checker removes
obsolete rejection cases rather than retaining contradictory feature gates.

The interpreter keeps the destination structState identity and replaces its
fields with deep value copies. Indexed storage, returned references and named
owners observe the same replacement; copies of the source or old value remain
independent. C emission uses the existing typed indirect assignment with no new
runtime symbol, host flag or alias-optimization promise.

## Verification and limits

Source tests cover grouped targets, indexed and for-in references, self-copy,
shared child last use, live child/shared conflicts, RHS parent conflicts and
exact nominal/nonplain/resource/shared/temporary/compound rejection.
Round-trip tests execute nested replacement and copy independence, reject
malformed receiver/value/metadata/declaration mutations, and emit both C profiles.
An indexed MIR mutation removing the bounds check is rejected. Interpreter tests
check array-backed referent identity and source/load copy independence. A
terminating RHS has no store and a returning RHS preserves the old value.

`internal/compiler/testdata/struct_deref_store.fy` exercises values above i32,
i64 max, shared/mutable returns, Option payloads and mutable iteration. Windows
dev/release native checks cover that source, bounds, RHS assertion/overflow and
reference rebinding: ten outcomes. Standalone public CLI check/build/run passes
both profiles and rejects live owner and child conflicts. Related regressions,
cold/warm workspace cache checks and Go vet pass.

Evidence: `docs/ops/evidence/struct-deref-store-20261004.json`. Compiler/source/SDK
digests, source fixtures, native image hashes and raw outcomes are retained.
Linux actual execution remains deferred to the owner's batch; Linux C emission
is verified. SDK stage0.85, language /20, runtime /0 and target maturity remain
unchanged. No DB, dependency, CI/runner, benchmark, install, push or release change.
