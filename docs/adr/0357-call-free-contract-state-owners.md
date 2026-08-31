# ADR 0357: Require call-free contract-state transaction owners

- Status: Accepted experimental compiler foundation
- Date: 2026-08-31
- Implementation: `internal/contractstatemir/lower.go`, `internal/contractstatemir/stable_rebind.go`
- Depends on: ADR 0347, ADR 0355, ADR 0356

## Context

Contract-state transaction MIR is a straight-line side model containing only explicit cell and map
operations plus one static exit. Ordinary Fairy MIR can represent internal and foreign calls, but
the transaction program neither executes a callee nor carries its operations, effects, state-borrow
lifetime, or reentrancy behavior. Accepting an owner whose current HIR body calls another function
would therefore publish two verified representations with different executable meaning.

ADR 0356 closes unrepresented declared effects. It does not exclude an ordinary internal call whose
propagated effects still fit the represented storage-only profile. Exact body-shape admission is a
separate boundary.

## Decision

1. Before lowering a source-independent HIR state transaction, recursively inspect the complete
   current owner body and reject every call expression. Inspection covers statement targets and
   expressions, `for` initializer/condition/post/body components, expression children, nested
   blocks, and match-arm expressions.
2. Apply the same current-body check while rebinding a stable state body to fresh HIR owners. Cached
   state operations cannot bypass the current owner's call-free requirement.
3. Reject both internal and foreign calls. Admitting a proven subset later requires an independent
   decision with transitive operation/effect, state-borrow, capability, and reentrancy evidence.
4. Keep compiler publication atomic. A valid ordinary MIR call in a transaction owner fails at
   `contract-state-mir-lower`, and `compiler.Result` publishes neither ordinary MIR nor state MIR.
5. Keep state-body v1, package-body hash v1, cache key v4, stable compilation v1, persistent compiler
   identity `/2`, language 12, and SDK `0.0.0-stage0.54` unchanged. Current source identity selects
   the build, and fresh-owner rebind rechecks the current HIR body.

## Validation

Direct HIR tests reject a top-level call and a call nested beneath a loop/group expression. Stable
body rebind rejects a call introduced into the fresh current owner. A compiler integration test uses
an otherwise valid internal `helper()` call, reaches `contract-state-mir-lower`, and proves that no
partial executable IR is published. Existing call-free storage-only programs continue to commit and
read `42` through the oracle.

## Compatibility

This narrows a compiler-private experimental lowering and cache-rebind acceptance set. It changes no
Fairy source grammar, public CLI/schema, SDK symbol, runtime ABI, serialized artifact shape, target,
lockfile, or Atlas contract. No version bump is required.

## Consequences

Every accepted current transaction owner is now exactly call-free, including nested expression and
loop positions. This is not a transitive call-graph or reentrancy analysis. Source state syntax,
data-dependent values and control flow, owner-bound borrows, native lowering, runtime persistence,
migration, gas, Merkle proofs, event/block-context operations, chain integration, and deployable
targets remain closed.
