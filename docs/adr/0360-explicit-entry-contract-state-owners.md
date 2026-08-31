# ADR 0360: Require explicit-entry contract-state owners

- Status: Accepted experimental compiler foundation
- Date: 2026-08-31
- Implementation: `internal/contractstatemir/owner_boundary.go`
- Depends on: ADR 0347, ADR 0351, ADR 0359

## Context

The contract-state side table names an existing HIR function and carries only canonical storage
operations plus one static exit. It does not represent an implicit method receiver, receiver mode,
or automatic resource-cleanup invocation.

Stable state function descriptors already retain an owner-qualified logical name, parameter types,
result type, effects, and drop-glue type. They do not retain the separate HIR `Receiver` field.
Consequently a fresh receiver-bound method could match a cached descriptor while changing an
execution fact that the state payload never authenticated. Drop glue is descriptor-bound, but its
automatic scope-exit execution is still outside the explicit transaction-entry model.

Rejecting every type-owned function would be broader than necessary. A static associated function
has no hidden receiver, and its owner-qualified logical name is already part of the stable identity.

## Decision

1. Reject a current state transaction owner whose HIR `Receiver` field is nonempty. Shared,
   exclusive, value, and malformed receiver modes are all outside this closed entry boundary.
2. Reject a current owner whose `DropGlueFor` type is nonempty. Resource cleanup remains ordinary
   compiler behavior and cannot become an implicit state transaction entry.
3. Continue admitting static associated functions whose `OwnerType` is nonempty but `Receiver` and
   `DropGlueFor` are empty. This preserves explicit type-namespaced entry points without claiming
   receiver or cleanup semantics.
4. Apply the boundary after the existing call, dynamic-control-flow, and memory-safety checks in
   direct HIR lowering and stable fresh-owner rebind. Cached state operations cannot bypass it.
5. Keep compiler publication atomic. An otherwise valid receiver-bound method or resource cleanup
   owner fails at `contract-state-mir-lower`, and `compiler.Result` publishes neither ordinary MIR
   nor state MIR.
6. Keep state-body v1, package-body hash v1, cache key v4, stable compilation v1, persistent compiler
   identity `/2`, language 12, and SDK `0.0.0-stage0.54` unchanged. This narrows current acceptance;
   it does not change serialized bytes or public source behavior.

## Validation

Direct HIR tests reject receiver-bound and drop-glue owners while preserving a static associated
function with safe-borrow and pointer-free resource types. Stable rebind rejects fresh receiver
metadata drift that the stable descriptor does not encode and continues rejecting drop-glue drift
through its descriptor. Compiler integration tests analyze a valid receiver-bound method and a valid
resource cleanup function, attach source-independent state operations to each, and require failure at
`contract-state-mir-lower` without partial executable IR.

## Compatibility

This changes only a compiler-private experimental lowering and cache-rebind acceptance set. It adds
no Fairy grammar, diagnostic code, public CLI/schema, SDK symbol, runtime ABI, artifact version,
target, lockfile, or Atlas contract. No version bump is required.

## Consequences

Every accepted current transaction owner now has an explicit entry shape: a free function or static
associated function, never a receiver-bound method or automatic cleanup function. Ordinary methods
and drop glue remain fully available outside the state-owner side table. ADR 0361 later excludes
ordinary checked arithmetic and bounds operations whose conditional trap edges are not represented.
Source state syntax,
data-dependent state values, receiver-bound state borrows, conditional transaction MIR, native
lowering, runtime persistence, migration, gas, Merkle proofs, events, block context, reentrancy,
chain integration, and deployable targets remain closed.
