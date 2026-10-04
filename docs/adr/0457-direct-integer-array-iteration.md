# ADR 0457: Direct integer array iteration

- Status: implemented; Windows native verified, Linux execution deferred
- Date: 2026-10-04
- Related: ADRs 0415, 0435, 0453, 0454, 0455

## Decision

Permit direct `for (let T item in data)` and
`for (let ref mut T item in data)` over named local fixed arrays of u8, i32,
i64, u32 and usize. Copied items are independent scalar values; exclusive
items reference the original element. Mutable items require a mutable array
binding. By-value array parameters support copied iteration of their local
storage. Exact element types and the existing `1..=4096` length guard apply.

```fairy
var i64[3] values = [1, 4000000000, 3];
for (let ref mut i64 value in values) { *value += 1; }
for (let i64 value in values) { assert(value > 0); }
```

An internal immutable descriptor borrows original storage once, then uses the
existing checked view loop. No array value copy is introduced by the loop.
Length is captured once; indices advance in ascending order. `continue` runs
the hidden increment; `break`, return and Result propagation use ordinary
loop cleanup. Bounds checks dominate element access, and scalar stores retain
checked arithmetic and receiver-before-RHS evaluation.

Bool/struct arrays, zero-length arrays, array fields, raw pointers, shared
reference items, and literal/call/slice iterable roots remain outside this
admission. Existing named-view and authenticated SDK Vec loops retain their
behavior.

## Borrow and name identity

The whole array remains borrowed through the written loop's exit. Copied loops
allow shared original-array reads but reject writes/replacement and competing
exclusive borrows. Exclusive loops reject separate original-array reads and
writes even after the current item's last use. The hidden exclusive descriptor
is a child generation of the array owner; each mutable item is its descendant.
Different indices do not establish disjointness. After loop exit the original
array can be accessed again. Local/by-value-parameter element references cannot
escape, and retaining competing item references across iterations is rejected.
Independent arrays can be used in nested loops; copied shared loops can nest on
the same array. The item's written name may shadow the iterable's name.

`internal/semantic/array_for.go` validates the actual resolved array type before
introducing a private lexical descriptor. Only prepared header paths are bound
to its fresh symbol. Written body symbols are not resolved again, so outer
bindings and shadowing retain their source identity. Before a fresh analysis,
`resolve.go` resets this type-dependent preparation from the written source.
Repeated checking of the same retained AST produces identical encoded MIR.

The existing array-view borrow, view length/index/bounds, element-address and
scalar-loop MIR operations are reused. No syntax production, MIR opcode,
artifact schema, SDK export, runtime symbol or host flag is added. Language
`/20`, SDK stage0.86, runtime `/0` and target maturity stay unchanged. Exact
compiler revision/executable identity separates older compiler/cache behavior.

## Verification and limits

Source Flow tests exercise all five types, copied/mutable items, owner access,
shadowing, independent nesting, shared nesting, competing aliases, replacement,
break/continue/return, last-item-use retention, item scope and local escape.
Unadmitted types, immutable mutable-loops, widths and temporary roots are rejected.
Public check tests additionally reject retaining an item across iterations.

The `internal/compiler/testdata/array_for.fy` matrix executes after verified MIR
codec round-trip and emits through Windows MSVC and Linux C profiles. It checks
copied values and original stored changes, first/last and single elements,
values above i32, continue/break, early return, Result propagation, independent
nested arrays, item shadowing and by-value array parameters. Negative verifier
mutations reject removed bounds, shared receivers, wrong element widths and
forged array owners before codec or C emission.

Windows dev/release native executions cover the matrix, checked overflow for
each of the five types and a trapped RHS. A separately built compiler runs
public check/locked build/run on the matrix and the clamp example at
`examples/integer_array_for_quickstart`; exact output is checked. Related
stage regressions and Go vet pass. Raw executions and compiler/source/SDK/image
hashes are recorded in `docs/ops/evidence/array-for-20261004.json` and its
companion receipts.

Linux actual execution remains deferred to the owner's batch. No benchmark,
installation, CI change, push or release is included.
