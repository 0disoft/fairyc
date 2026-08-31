# ADR 0363: Freeze a closed-form contract-state owner HIR subset

- Status: Accepted experimental compiler foundation
- Date: 2026-08-31
- Implementation: `internal/contractstatemir/owner_boundary.go`
- Depends on: ADR 0357 through ADR 0362

## Context

ADRs 0357 through 0362 exclude calls, dynamic control flow, unsafe and raw-pointer facts, implicit
entries, ordinary trap edges, owned resource values, and lexical region cleanup from current
contract-state transaction owners. Those checks are intentionally specific, but their combination
is still a blacklist: a newly introduced HIR statement, expression, operator, or expression-local
effect could be accepted by default until another exclusion is added.

That default is unsafe for stable fresh-owner rebind. Cached ordinary MIR may be selected before the
current HIR owner is rechecked, so an unclassified current form must not inherit state-owner
admission merely because older boundary visitors do not recognize it. The current proven owner
subset is small enough to enumerate positively.

## Decision

1. Admit only `binding`, `return`, `discard`, `expression-statement`, and `assignment` statements.
   Assignment admits plain `=` plus bitwise `&=`, `|=`, and `^=`. An allowed statement carrying a
   hidden classic-loop payload is rejected.
2. Admit only literals, paths, grouping, copy-only records and arrays, array-view borrows and length,
   field reads, view length/empty queries, scalar casts, `Result`/`Option`/enum construction, lexical
   blocks, the nontrapping prefix operators `+`, `!`, `~`, `&`, `&mut`, and `*`, and comparison or
   bitwise binary operators.
3. Reject every other statement kind, expression kind, prefix operator, binary operator, assignment
   operator, and every otherwise admitted expression carrying a nonempty expression-local effect
   set. Earlier specific boundaries keep diagnostic priority for calls, control flow, memory,
   entries, traps, resources, and regions.
4. Recursively inspect statement targets and values, expression children, nested blocks, and match
   arms even when an earlier boundary already excludes current match control.
5. Apply the same closed-form check during direct HIR lowering and stable fresh-owner rebind. A new
   HIR form requires an explicit owner-admission decision and regression before cached state
   operations can accompany it.
6. Keep compiler publication atomic. A stage mutation that adds expression-local effects to an
   otherwise valid literal fails at `contract-state-mir-lower`, and `compiler.Result` publishes
   neither ordinary MIR nor state MIR.
7. Keep state-body v1, package-body hash v1, cache key v4, stable compilation v1, persistent compiler
   identity `/2`, language 12, and SDK `0.0.0-stage0.54` unchanged. The allowlist narrows a private
   acceptance boundary without changing serialized or public bytes.

## Validation

One direct classification table enumerates every admitted statement, expression, prefix operator,
binary operator, and assignment operator. Orthogonal negative cases reject future statement and
expression kinds, future operators, hidden loop payloads, expression-local effects, and unknown
forms nested in children, blocks, and arms. Direct transaction lowering and stable fresh-owner
rebind reject unlisted current forms. Compiler integration injects expression-local effects after
valid source analysis, proves ordinary MIR lowering alone would not authorize them for state use,
and requires atomic failure at `contract-state-mir-lower`.

## Compatibility

This changes only a compiler-private experimental lowering and cache-rebind acceptance set. It adds
no Fairy grammar, diagnostic code, public CLI/schema, SDK symbol, runtime ABI, artifact version,
target, lockfile, or Atlas contract. No version bump is required.

## Consequences

Current state owners now fail closed when HIR evolves. Admitting a new form requires an explicit
semantic review rather than relying on absence from historical blacklists. Source state syntax,
data-dependent state values, new owner HIR forms, owner-bound state borrows, conditional transaction
MIR, native lowering, runtime persistence, migration, gas, Merkle proofs, events, block context,
reentrancy, chain integration, and deployable targets remain closed.
