# ADR 0358: Require straight-line contract-state transaction owners

- Status: Accepted experimental compiler foundation
- Date: 2026-08-31
- Implementation: `internal/contractstatemir/lower.go`, `internal/contractstatemir/stable_rebind.go`
- Depends on: ADR 0347, ADR 0355, ADR 0357

## Context

Contract-state transaction MIR contains one canonical linear operation sequence and one static
commit, reject, or trap exit. Ordinary Fairy MIR additionally supports conditional branches, loops,
short-circuit evaluation, Result propagation, and assertions. The state side model does not carry a
condition, branch target, loop bound, conditional trap, or alternative operation sequence.

ADR 0357 prevents unrepresented callee execution but leaves an owner body able to select ordinary
behavior dynamically while its state transaction always executes the same side-table plan. That is
too broad a current-owner boundary for later source integration or cache consumption.

## Decision

1. Reject dynamic control flow anywhere in the complete current owner HIR body. The closed set is
   `if`, `match`, expression or statement `while`, classic `for`, `break`, `continue`, `assert`,
   Result propagation, and short-circuit `&&` or `||`.
2. Recursively inspect statement targets and expressions, loop components and bodies, expression
   children, nested blocks, and match-arm expressions. Ordinary groups, arithmetic, casts, bindings,
   assignments, blocks, regions, and unconditional returns remain structurally admitted when they
   contain neither calls nor dynamic control flow.
3. Apply the same check during direct HIR transaction lowering and stable state-body rebind to fresh
   current owners. Cached operation payloads cannot bypass the current straight-line requirement.
4. Keep compiler publication atomic. An otherwise valid ordinary conditional owner fails at
   `contract-state-mir-lower`, and `compiler.Result` publishes neither ordinary nor state MIR.
5. Keep state-body v1, package-body hash v1, cache key v4, stable compilation v1, persistent compiler
   identity `/2`, language 12, and SDK `0.0.0-stage0.54` unchanged. Current source identity selects
   the build, and fresh-owner rebind rechecks the current HIR body.

## Validation

Direct HIR mutation tests reject an owner `if`, nested short-circuit expression, and assertion while
existing straight-line storage-only execution remains valid. Stable rebind rejects a fresh owner
containing a `match`. A compiler integration test first analyzes and ordinarily lowers a valid Fairy
`if` expression, then proves state lowering classifies it at `contract-state-mir-lower` without
publishing partial executable IR.

## Compatibility

This narrows a compiler-private experimental lowering and cache-rebind acceptance set. It changes no
Fairy source grammar, public CLI/schema, SDK symbol, runtime ABI, serialized artifact shape, target,
lockfile, or Atlas contract. No version bump is required.

## Consequences

Every accepted current transaction owner is now call-free and straight-line under the exact closed
HIR set above. ADR 0359 later excludes unsafe code and reachable raw-pointer types while preserving
ordinary safe-borrow and pointer-free resource types. This does not derive state operations from
ordinary expressions or admit data-dependent state values. ADR 0360 later excludes receiver-bound
methods and automatic drop glue without rejecting static associated functions. Source state syntax, owner-bound state
borrows, conditional transaction MIR,
native lowering, runtime persistence, migration, gas, Merkle proofs, events, block context,
reentrancy, chain integration, and deployable targets remain closed.
