# ADR 0361: Exclude potentially trapping contract-state owner operations

- Status: Accepted experimental compiler foundation
- Date: 2026-08-31
- Implementation: `internal/contractstatemir/owner_boundary.go`
- Depends on: ADR 0347, ADR 0358, ADR 0360

## Context

Contract-state transaction MIR carries one canonical linear storage-operation sequence and one
static commit, reject, or trap exit. ADR 0358 excludes explicit dynamic control flow, but it
temporarily admits ordinary arithmetic, assignments, and indexing. Fairy ordinary MIR lowers
checked arithmetic, division, remainder, shifts, array or view indexing, and view slicing into a
conditional success edge plus a non-returning trap edge.

The state side table does not identify one ordinary trap check, its operands, its success edge, or
its relation to the static transaction exit. Accepting such an owner could let ordinary MIR trap
while the state program commits the same fixed operation sequence. A successful constant example
does not prove the operation safe for every runtime input, and the current boundary owns no range
analysis or trap-equivalence proof.

This is distinct from the state program's explicit static `trap(category)` exit. That exit remains a
fully represented transaction outcome and is not removed by this decision.

## Decision

1. Reject retained HIR unary `-`; binary `+`, `-`, `*`, `/`, `%`, `<<`, and `>>`; and the matching
   compound-assignment operators. These operations can trap on overflow, zero division or
   remainder, signed division overflow, invalid shift, or overflowing left shift.
2. Reject array indexing, byte-view indexing, and byte-view slicing in expression and assignment
   targets because their bounds checks can trap.
3. Recursively inspect statement targets and expressions, classic-loop components and bodies,
   expression children, nested blocks, and match arms. Earlier call and dynamic-control checks keep
   diagnostic priority for forms they already own.
4. Continue admitting nontrapping bitwise `&`, `|`, `^`, unary `~`, comparisons, matching bitwise
   compound assignments, value-preserving scalar casts, bindings, plain assignments, groups,
   blocks, and receiver-free static associated functions.
5. Apply the same current-owner check during direct HIR lowering and stable fresh-owner rebind.
   Cached operation payloads cannot bypass the trap boundary.
6. Keep compiler publication atomic. An otherwise valid ordinary arithmetic owner fails at
   `contract-state-mir-lower`, and `compiler.Result` publishes neither ordinary MIR nor state MIR.
7. Keep state-body v1, package-body hash v1, cache key v4, stable compilation v1, persistent compiler
   identity `/2`, language 12, and SDK `0.0.0-stage0.54` unchanged. This narrows current acceptance
   without changing serialized or public bytes.

## Validation

One direct classification table freezes every rejected arithmetic and compound-assignment operator,
all three bounds-checked HIR forms, and the admitted bitwise, comparison, plain-assignment, and
scalar-cast forms. Direct lowering also rejects nested bounds access and checked compound assignment
while preserving a static associated owner with safe bitwise work. Stable rebind rejects a fresh
owner whose ordinary body gains division. Compiler integration analyzes and ordinarily lowers a
valid parameter-dependent `i32` addition, then requires state lowering to fail at
`contract-state-mir-lower` without partial executable IR.

## Compatibility

This changes only a compiler-private experimental lowering and cache-rebind acceptance set. It adds
no Fairy grammar, diagnostic code, public CLI/schema, SDK symbol, runtime ABI, artifact version,
target, lockfile, or Atlas contract. No version bump is required.

## Consequences

Accepted current owners no longer retain ordinary HIR operations that can branch to a runtime trap.
The state program may still select its own explicit static trap exit. ADR 0362 later excludes owned
resource values and lexical region cleanup. Source state syntax,
data-dependent state values, proof-backed ordinary trap admission, receiver-bound state borrows,
proof-backed resource lifetime admission, conditional transaction MIR, native lowering, runtime
persistence, migration, gas, Merkle proofs, events, block context, reentrancy, chain integration,
and deployable targets remain closed.
