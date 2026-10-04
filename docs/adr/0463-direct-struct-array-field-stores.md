# ADR 0463: Direct struct array field stores

- Status: implemented; Windows native verified, Linux execution deferred
- Date: 2026-10-05
- Related: ADRs 0415, 0449, 0451, 0455, 0462

## Decision

A named mutable fixed array of an admitted plain scalar-field struct (ADR 0462)
supports direct stored-field assignment `array[index].field = value`, nested
scalar and nested plain-struct field paths, checked scalar compound assignment,
and whole plain-struct field replacement. The place root is the array itself;
`array[index]` is not materialized as a copied element. Immutable and temporary
array roots are rejected.

```fairy
var Point[4] polygon = [origin; 4];
polygon[0].y = 9;
polygon[0].x += 10;
polygon[3].x -= 10;
polygon[1].position = Position { .x = -5000000000, .y = 8 };
```

The element type must be an actual nonempty acyclic plain struct with
bool/integer leaves, including nested plain structs, and every written field must
belong to the exact nominal path. Whole-element replacement and element
references stay with ADR 0462; resources, raw pointers, borrowed values, tagged
fields and array fields remain outside this admission.

## Lifetime and evaluation

The whole original array is the loan owner and grouped roots preserve that
identity. Select the storage first, evaluate the index exactly once and check its
bounds before the address exists. During index evaluation hold the selected owner
against replacement, element writes and competing borrows; ordinary reads such as
the fixed length remain allowed, matching the explicit element borrow. Compound
stores read the prior scalar before the right-hand side without reevaluating the
index. The right-hand side then evaluates exactly once; checked arithmetic runs
before the single field store.

While the index evaluates, only competing writes and borrows of the selected
array are rejected; ordinary reads such as its fixed length remain allowed, as
with the explicit element borrow. While the right-hand side evaluates, direct
owner reads, writes and reborrows through the same array are all rejected. Live
shared or exclusive references that reach the array are rejected in either
phase, including loans whose last use falls in the index or the right-hand side.
Loans that ended before the statement are allowed. A bounds trap or
right-hand-side trap performs no store.

## Implementation

`internal/semantic/indexed_place.go` and `internal/semantic/field_store_flow.go`
route an `array-index` assignment root through the same whole-owner place
analysis used by explicit element borrows. `internal/semantic/borrow_creation.go`
shares `checkNamedBorrowCreation` between explicit borrows and these implicit
indexed stores so the source node's liveness covers index and right-hand-side
uses. `internal/semantic/typecheck.go` admits the `array-index` root in
`mutableAssignmentTarget`. `internal/mir/field_store.go` rewrites the stored
`array-index` owner into an internal `array-view-borrow` (`&mut`) followed by the
checked `view-element-borrow`, so existing field-store lowering consumes the
selected element reference.

No SDK, runtime, MIR operation, wire field, grammar production or host flag is
added. The array view, checked index and field store reuse their existing
operations; nominal array operands are still checked against actual local
definitions and checked bounds continue to dominate the address.

## Verification and limits

`internal/compiler/testdata/struct_array_fields.fy` covers nested i64/i32/bool/
u8/u32/usize leaves, scalar and compound stores, whole plain-struct field
replacement, independent element copies, single index evaluation with
right-hand order effects, `.len`-derived indices, in-loop stores and shared-view
observation. The compiler fixture check resolves to verified MIR, round-trips the
private codec, executes in the interpreter and emits through both the Windows
MSVC and Linux C profiles; verifier and codec mutations reject a missing bounds
check, a forged array owner and a shared-receiver store.

Semantic negatives cover immutable, temporary and shared-view roots (`E1008`), a
nominal field mismatch (`E1006`), and live shared/exclusive loans plus owner
reads, writes or borrows reached from the index or the right-hand side (`E2101`).
The Go CLI harness runs the fixture plus end bounds,
bounds-before-right-hand-side, compound bounds, checked i64 overflow
(`integer-overflow`) and a right-hand-side return on Windows through the
`Cdriver.ExecuteWindowsMSVCTarget` path that builds and executes the generated
Fairy EXE: 8 tests pass, accounting for 12 real outcomes across the dev and
release profiles (cache reuse included). The separate compiler's public CLI
`check`/`build`/`run` covers the matrix, the example and one
bounds-before-right-hand-side in dev/release with four diagnostics rejected: 6
public CLI native outcomes pass. The related suite reports 698 PASS, 0 FAIL and 0
SKIP. Evidence: `docs/ops/evidence/struct-array-field-20261005.json` with its
`.cli.json` and `.native.json` companions. Linux actual execution remains
deferred to the owner's batch.

Language /20, SDK stage0.86, runtime /0, MIR/wire schemas, public ABI and target
maturity are unchanged. No CI, installation, benchmark, push or release is
included.
