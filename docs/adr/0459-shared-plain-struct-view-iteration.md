# ADR 0459: Shared plain struct view iteration

- Status: implemented; Windows native verified, Linux execution deferred
- Date: 2026-10-05
- Related: ADRs 0415, 0435, 0450, 0451, 0452, 0458

## Decision

Admit `for (let ref Token item in items)` over immutable named bindings or
parameters of exact type `ref Token[]` or `ref mut Token[]`. The item references
an original stored element rather than a copied struct. Qualified nominal types
resolve normally; same-layout unrelated nominals remain distinct. The referent
must satisfy the existing nonempty acyclic plain struct graph with scalar leaves,
including nested structs. Resource, pointer, borrowed, array and tagged fields,
mutable descriptor bindings, computed iterable roots and fixed struct arrays
remain outside this admission. Copied struct loops still require shared views;
mutable items still require mutable views.

```fairy
pub fn Option<ref Token> find(ref Token[] tokens, u32 kind) {
    for (let ref Token token in tokens) {
        if (token.kind == kind) { return .some(token); }
    }
    return .none;
}
```

A named subview may be used, including an empty subview. Length is snapshotted
once; checked ascending element addresses feed ordinary scalar-loop control.
Continue advances the hidden index; break, return and Result propagation use
existing cleanup rules. Shared items permit nested field reads and shared field
borrows. Explicit `*item` copies the struct; shared field and whole-item stores
are rejected.

## Lifetime and lowering

The ADR 0458 shared loop access applies to both shared and mutable struct
views. Owner/view writes, structural moves or mutation, and competing exclusive
reborrows remain blocked through the written loop's exit, even after the current
item's last read. Shared reads and nested shared loops may coexist. A returned
item or field reference retains the original owner provenance; references into
local owners cannot escape. Returning/retaining `Option<ref Token>` or
`Option<ref Span>` uses existing tagged-reference lifetime summaries. After loop
exit, retained live references continue to restrict the owner; last use restores
ordinary access. No disjoint-field/index promise or concurrent mutable sibling
admission is added.

Semantic admission reuses the existing prepared reference-view loop and
`view-element-borrow` HIR/MIR operations of ADR 0451. Integer behavior remains
unchanged; AST preparation's helper is named `PrepareReferenceViewLoop` to match
its expanded role. Existing typed bounds, receiver provenance, exact nominal
identity and mutability checks remain the authority before codec or C emission.

Deferred shared-reference loops have no prepared expression during generic
expansion. Their written body must still expand actual user calls and local
types. `genericsource/clone.go` now applies the active transform to that body
when `Statement.Expression` is absent. Already-prepared loops expand their body
through the prepared expression and keep the retained source copy inert.
Synthetic Vec candidate calls remain inert until SDK authentication. This fixes
unexpanded SDK calls/local Vec types inside deferred loops without registering
unauthenticated synthetic vector calls.

## Verification and limits

Flow cases cover shared/mutable parents, nested fields, direct/Option field
returns, retained shared items, nested loops, post-loop access, owner/alias writes,
competing exclusive calls, shared stores, item scope and returned-reference
retention. Nominal mismatch, temporary roots, mutable descriptors, unadmitted
graphs and exclusive items from shared sources are rejected. Public checks
reject owner clear/move, parent writes, nested-field stores, retained-return
invalidation and local-owner escape.

Compiler tests check stable MIR encoding across repeated source analysis,
codec round-trip and shared element addresses without whole-struct index loads.
Removed bounds, forged mutability and same-layout nominal substitution are
rejected by the verifier, codec and C emitter. Windows MSVC and Linux C profiles
emit the positive program. Related struct/integer/Vec regressions pass.

The stored SDK Vec matrix exercises growth, first/last/no-match search, empty
and offset views, mutable-parent/shared-item traversal, nested fields, retained
items, break/continue, Result propagation, nesting/shadowing, and generic calls
with independent local Vec ownership in the written loop body. Windows
ordinary/release native execution also covers a singleton and a body trap.
A separately built compiler checks errors and executes locked builds of the
matrix and `examples/shared_struct_ref_quickstart`. The two-module token search
prints exact output and demonstrates a shared returned reference into a subview.
Compiler/source/SDK/image hashes and raw execution receipts are recorded in
`docs/ops/evidence/shared-struct-for-20261005.json` and companion files.

Language /20, SDK stage0.86, runtime /0, SDK exports, MIR/wire schemas and ABI
are unchanged. Exact compiler identity separates cache generations. Linux
actual execution is deferred to the owner's batch. No benchmark, installation,
CI change, push or release is included.
