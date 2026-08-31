# ADR 0362: Require resource-lifetime-free contract-state owners

- Status: Accepted experimental compiler foundation
- Date: 2026-08-31
- Implementation: `internal/contractstatemir/owner_boundary.go`
- Depends on: ADR 0359, ADR 0360, ADR 0361

## Context

Contract-state transaction MIR carries canonical storage operations and one static exit. It has no
operation for an ordinary Fairy resource move, initialization, reassignment drop, explicit burn,
automatic scope-exit drop, lexical region entry, placement cleanup, or reset.

ADR 0359 temporarily preserved pointer-free resource values because they do not violate its
raw-pointer boundary. Ordinary MIR nevertheless gives every owned resource parameter, result, local,
and resource-bearing aggregate a lifetime state. Even an unused owned parameter can emit automatic
drop on function exit. A lexical `region` similarly emits enter and reset operations outside the
state side model. Atomic publication of ordinary and state MIR does not prove that these two
independent lifetime models execute as one failure-atomic transaction.

A safe borrow does not carry release authority, and a resource type used only as the namespace of a
receiver-free static associated function is not an owned runtime value. Those forms do not require
the rejected lifetime transitions.

## Decision

1. Reject owned resource types in current transaction-owner parameters, results, locals, expression
   types, cast target types, and statement targets or values. Recursively inspect fixed arrays,
   `Option`, `Result`, structs, and enums with cycle protection.
2. Reject lexical `region` expressions in statement values, expression children, nested blocks, and
   match arms. `Region.place` remains independently closed by the existing call boundary.
3. Continue admitting safe borrows of pointer-free resource values and resource types used only as
   the `OwnerType` namespace of a receiver-free static associated function. Neither form carries
   ordinary drop authority in the owner.
4. Apply the same current-owner boundary during direct HIR lowering and stable fresh-owner rebind.
   Cached state operations cannot bypass current resource-lifetime checks.
5. Keep compiler publication atomic. An otherwise valid owned-resource function fails at
   `contract-state-mir-lower`, and `compiler.Result` publishes neither ordinary MIR nor state MIR.
6. Keep state-body v1, package-body hash v1, cache key v4, stable compilation v1, persistent compiler
   identity `/2`, language 12, and SDK `0.0.0-stage0.54` unchanged. This narrows compiler-private
   acceptance without changing serialized or public bytes.

## Validation

Type-graph regressions freeze direct and aggregate-nested owned-resource rejection while preserving
shared and mutable resource borrows. Direct boundary tests reject owned parameters, results, locals,
nested resource expressions, and nested lexical regions while preserving a static resource
namespace. Direct HIR lowering rejects an owned parameter and lexical region. Stable rebind rejects
a fresh current owner whose local graph gains a resource type. Compiler integration ordinarily
lowers a valid function with an owned resource parameter, then requires state lowering to fail at
`contract-state-mir-lower` without partial executable IR.

## Compatibility

This changes only a compiler-private experimental lowering and cache-rebind acceptance set. It adds
no Fairy grammar, diagnostic code, public CLI/schema, SDK symbol, runtime ABI, artifact version,
target, lockfile, or Atlas contract. No version bump is required.

## Consequences

Accepted current owners no longer carry ordinary resource release authority or lexical region
cleanup. Safe borrows and receiver-free static resource namespaces remain ordinary HIR facts, not
state-MIR permissions. Source state syntax, data-dependent state values, owner-bound state borrows,
proof-backed resource-lifetime admission, conditional transaction MIR, native lowering, runtime
persistence, migration, gas, Merkle proofs, events, block context, reentrancy, chain integration,
and deployable targets remain closed.
