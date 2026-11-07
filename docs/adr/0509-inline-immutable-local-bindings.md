# ADR 0509: Inline immutable local i32 bindings

- Status: Accepted for bounded prototype
- Date: 2026-11-07
- Extends: [ADR 0493](0493-cross-package-inline-bodies-and-cache-v5.md) and
  [candidate inline v1](../architecture/candidate-inline-v1.md)
- Evidence: [Windows local-binding receipt](../development/candidate-inline-locals-20261107.md)
- Call-family extension: [ADR 0548](0548-inline-same-module-call-graphs.md);
  the no-call v1/v2 identity and local sequencing below remain unchanged.

## Source contract

An existing eligible `@inline` declaration may have zero through sixteen immutable
`let` bindings, in its outer body, followed by the existing arithmetic return
expression. Each binding has a checked i32 type, inferred or explicitly written,
and an initializer from the same expression vocabulary as the return: i32 integer
literals, immutable parameters, preceding local bindings, grouping, unary +/- and
binary +,-,*,/,%. The sixty-four-node budget is shared by all initializer trees and
the returned tree; groups count against that existing budget. Local references
count as references, without substitution or recursively expanding initializers.
Normal name/type/ownership checks remain mandatory before Complete publication.

The declaration remains public, safe, nongeneric, without owner or effects,
returning i32 and taking at most four immutable i32 parameters. Mutable bindings,
assignments, calls in initializers or returns, captures, explicit discards,
intermediate expression statements, control flow, nested body expressions,
aggregate/resource/borrow values and unsupported effects remain closed.
There is no new grammar production or function-tail syntax.

## Ordered execution and identity

Every initializer executes once, in source declaration order, after ordinary
once-only ordered argument evaluation and before the return expression. Unused
bindings still execute and trap. Initializers must not be erased, substituted,
reordered or folded into a different evaluation tree by inline admission.
Existing lowering, checked arithmetic, private C linkage and source spans apply.

A body without locals retains the exact `fairyc.inline-function/v1` tuple.
A body with locals uses `fairyc.inline-function/v2`, with the same JSON fields
`Version`, `Arity`, `Root` and existing node fields `Op`, `Value`, `Children`.
Its root is `bindings`, with empty value and ordered children containing every
initializer followed by the return. Local references use `Op=local` and the
zero-based declaration position as decimal `Value`. Parameter paths retain
`Op=path` and their original zero-based parameter position. Leaf children remain
JSON null. Names, spans, HIR IDs, grouping and equivalent literal spelling do not
enter the canonical tuple. Unused initializers, their ordering, all operators,
reference positions, arity and literal values do enter it.

The existing `fyinlinefn1-sha256-` digest carrier remains a transport identifier;
it carries the digest of either versioned canonical tuple. No-inline and existing
expression-only bytes remain unchanged. Package inline-class derivation,
public-interface v5, cache key v5 and the full digest trailer are unchanged.
A local initializer edit therefore changes the existing package inline class and
invalidates the owner and direct dependents, including edits to unused exports.
No hidden helper closure is introduced because calls remain closed.

## Verification and cache admission

Go source admission is owned by `internal/hir/inline.go`, called from the existing
semantic function checker. Existing MIR lowering emits real binding assignments;
its bounded operation/CFG verifier and fresh-HIR marker admission remain active.
There is no C backend change or relaxed marker verifier.

Fairy source admission remains in `module_inline_declarations`. Its detached body
verifier, `module_inline_ir`, requires an immutable scalar parameter prefix and
an immutable local suffix, an exact binding/return statement sequence, matching
binding-entry and initializer identities, and references only to parameters or
already initialized locals. It reconstructs the ordered canonical tuple and
compares its full digest. Calls, regions, placements and resets remain rejected.
Body, program and project artifact restoration still verify markers and recheck
current sources before cache exposure. Missing, altered and extra markers do not
become authority through the new source family.

## Delivery boundary

Windows checks are recorded separately. Native Linux, full-suite/hosted CI and
publication are deferred. No public API/DB, runtime ABI, dependency, runner,
repository hygiene, default cache-policy or release change is included. Parent
owns staging, commits, shared queue/index updates and the lockfile refresh.
