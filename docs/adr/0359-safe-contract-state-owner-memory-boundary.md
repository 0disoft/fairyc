# ADR 0359: Exclude unsafe and raw-pointer contract-state owners

- Status: Accepted experimental compiler foundation
- Date: 2026-08-31
- Implementation: `internal/contractstatemir/owner_boundary.go`
- Depends on: ADR 0347, ADR 0357, ADR 0358

## Context

ADR 0357 and ADR 0358 make every current contract-state transaction owner call-free and
straight-line. Ordinary HIR can still declare an `unsafe fn`, enter an `unsafe` expression, retain an
operation-level `UnsafeScope`, or transport raw pointers through parameters, results, locals,
expressions, fixed arrays, `Option`, `Result`, safe-borrow elements, or nominal aggregate fields.

Contract-state transaction MIR has no address, allocation identity, lifetime epoch, pointer
provenance, dereference, host-memory, or unsafe-scope operation. Accepting those facts on a current
owner would overstate what the state side model verifies.

## Decision

1. Reject a state transaction owner declared `unsafe`.
2. Recursively reject every current owner-body expression whose kind is `unsafe` or whose
   `UnsafeScope` is present. Inspection covers statement targets and expressions, loop components,
   expression children, nested blocks, and match arms even though calls and dynamic control flow are
   independently closed.
3. Reject raw-pointer types in the owner type, result, drop-glue type, parameters, locals, expression
   types, and cast target types. Recursively inspect safe-borrow elements, fixed arrays, `Option`,
   `Result`, and every referenced struct, enum, or resource field with cycle protection.
4. Continue admitting ordinary safe-borrow and resource types when their reachable type graph has
   no raw pointer. This decision does not create owner-bound state borrows or give resource values a
   transaction-MIR representation.
5. Apply the same boundary during direct HIR transaction lowering and stable state-body rebind to
   fresh current owners. Cached operations cannot bypass the current memory-safety boundary.
6. Keep compiler publication atomic. Otherwise valid ordinary unsafe/raw-pointer MIR fails at
   `contract-state-mir-lower`, and `compiler.Result` publishes neither ordinary nor state MIR.
7. Keep state-body v1, package-body hash v1, cache key v4, stable compilation v1, persistent compiler
   identity `/2`, language 12, and SDK `0.0.0-stage0.54` unchanged. Current source identity selects
   the build, and fresh-owner rebind rechecks the complete current HIR owner.

## Validation

Direct HIR mutation tests reject an unsafe declaration, unsafe body, raw-pointer local, and a raw
pointer nested inside a struct parameter. A positive regression preserves safe-borrow and
pointer-free resource owner types. Stable rebind rejects unsafe declaration/body drift and a fresh
raw-pointer local. Compiler integration tests ordinarily analyze a valid unsafe C-string cast and a
valid raw-pointer parameter, then prove state lowering classifies each at
`contract-state-mir-lower` without publishing partial executable IR.

## Compatibility

This narrows a compiler-private experimental lowering and cache-rebind acceptance set. It changes no
Fairy source grammar, public CLI/schema, SDK symbol, runtime ABI, serialized artifact shape, target,
lockfile, or Atlas contract. No version bump is required.

## Consequences

Every accepted current transaction owner is call-free, straight-line, safe, and free of reachable
raw-pointer types. Safe borrows and resources remain ordinary compiler facts rather than state-MIR
permissions. ADR 0360 later requires an explicit free or static-associated entry and rejects
receiver-bound methods and automatic drop glue. Source state syntax, data-dependent state values,
owner-bound state borrows, potentially trapping ordinary operations, conditional
transaction MIR, native lowering, runtime persistence, migration, gas, Merkle proofs, events, block
context, reentrancy, chain integration, and deployable targets remain closed.
