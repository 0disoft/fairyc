# ADR 0460: Copied items from mutable plain struct views

- Status: implemented; Windows native verified, Linux execution deferred
- Date: 2026-10-05
- Related: ADRs 0415, 0440, 0441, 0448, 0457, 0459

## Decision

Admit `for (let Token item in items)` over an immutable named binding or
parameter of exact type `ref mut Token[]`, in addition to existing shared
struct view support. The exact nominal element remains a nonempty acyclic
plain struct graph with scalar leaves. Qualified types resolve normally;
same-layout unrelated nominals are not interchangeable. Mutable descriptor
bindings, computed iterable roots, resource/pointer/borrow/array/tagged field
graphs and fixed struct arrays remain outside this admission.

Each iteration loads an independent whole-element value immediately before
the body runs. The `let` item is immutable; a separate `var Token copy = item`
can be changed. The copy includes nested scalar-field structs. Body writes
through the original mutable descriptor are allowed, including whole-element
replacement and short element-reference borrows under existing lifetime rules.
Writing the original cannot change the current item, and changing a local copy
cannot change the original. Future iterations load current storage and observe
earlier body writes to not-yet-visited elements. Length is captured once; this
is not an initial snapshot of the complete collection.

```fairy
var usize index = 0;
for (let Record before in items) {
    items[index].value = clamp(before.value);
    report(before.value, items[index].value);
    index += 1;
}
```

The whole owner remains held through the written loop, so clear/reserve/push,
replacement and moves remain rejected even at break/continue/return exits.
Existing shared/mutable element-reference loans still restrict competing
access; copied items do not bypass those loans. A returned or retained value
copy has no reference to the original storage and remains usable after owner
cleanup. References to the local copied item cannot escape. Nested copied
loops, empty/named offset subviews, shadowing and Result propagation use the
existing loop and cleanup semantics. Shared and exclusive item behavior is
unchanged.

## Implementation and adjacent defect

`semantic/byte_for.go` removes only the shared-source restriction for copied
plain struct items. The existing `view-index` HIR/MIR load, typed bounds checks
and whole-struct C load are reused; no new runtime helper or representation is
introduced. Mutable reference items still require a mutable source.

The public clamp example combines a copied integer array loop with an SDK Vec.
Generic expansion detached its retained source iterable from the resolved bound
path, leaving the synthetic array borrow attached to an unresolved place.
`semantic/array_for.go` now creates an owner path explicitly bound to the actual
symbol selected by the checked bound path. Written source provenance remains
intact. The minimal generic identity/array regression executes after MIR
round-trip and confirms repeatable source analysis. Existing array lifetime
and retained-source loop checks remain in the related lane.

## Verification and compatibility

Flow and public checks cover independent nested-field copies, parent stores,
short exclusive borrows, retained/returned values, nesting, scope, local-copy
escape, item immutability, existing live-loan conflicts and structural owner
changes. Compiler tests prove a matching mutable-view `view-index` value load,
codec round-trip and repeatable encoding, plus rejection of missing bounds,
wrong nominal results and forged receiver annotations. Both Windows MSVC and
Linux C profiles emit the positive program.

The stored SDK matrix checks original/current-copy independence, visibility of
future-element writes, whole replacement, named subviews, empty/singleton views,
break/continue/return, Result propagation, nesting/shadowing and copies surviving
owner clear. Windows dev/release runs also check arithmetic overflow. A separate
compiler executable runs public checks and locked builds of the matrix and the
two-module clamp example at `examples/copied_mutable_struct_quickstart`, comparing
exact exit/stdout/stderr. Hashes and raw receipts are recorded in
`docs/ops/evidence/copied-mutable-struct-for-20261005.json` and companions.
Related regressions, Go vet and diff hygiene pass.

Language /20, SDK stage0.86, runtime /0, SDK exports, ABI and MIR/wire schemas
are unchanged. Compiler identity separates cached generations. Linux actual
execution remains deferred to the owner's batch. No benchmark, installation,
CI change, push or release is included.
