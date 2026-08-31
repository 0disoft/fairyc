# ADR 0355: Publish ordinary and contract-state MIR atomically in one compiler result

- Status: Accepted experimental compiler foundation
- Date: 2026-08-31
- Implementation: `internal/compiler/check.go`, `internal/workspacecompiler/session.go`
- Depends on: ADR 0347, ADR 0354

## Context

The ordinary compiler result published verified executable MIR, while the workspace-cache
coordinator exposed verified contract-state transactions through a second sibling field. That split
made equivalent cold and cached compilations have different result ownership and allowed a caller to
observe ordinary MIR even if later contract-state lowering failed.

Source admission is still closed, but source-independent HIR state transactions already pass exact
layout, effect, capability, body-identity, cache, rebind, and oracle checks. The shared compiler
boundary should therefore own their publication before any source or native/runtime surface depends
on them.

## Decision

1. Extend `compiler.Result` with ordered verified compiler-private state transactions. `Verified`
   now means semantic analysis completed and both ordinary MIR and every present contract-state MIR
   program passed their owning lowering and verifier boundaries.
2. Lower and verify contract-state transactions only after ordinary MIR verifies, but publish
   neither executable representation until both succeed. Classify a contract-state lowering or
   verifier failure as `contract-state-mir-lower`, distinct from ordinary MIR lowering and
   verification.
3. Remove the duplicate transaction field from `workspacecompiler.Result`. Cold/fallback,
   process-local all-hit, and persistent all-hit paths all publish transactions through the nested
   `compiler.Result`. Process-local all-hit lowers current source-independent HIR transactions;
   persistent all-hit retains the ADR 0354 cached fresh-symbol reassembly.
4. Keep compilation side-effect free. A successful result contains executable compiler-private
   transaction programs, but neither `compiler.Check`, `compiler.LowerAnalyzed`, nor
   `workspacecompiler.Session.Check` begins or commits state.
5. Keep stable MIR v3, stable state-body v1, stable compilation v1, cache key v4, persistent compiler
   identity `/2`, language 12, and SDK `0.0.0-stage0.54` unchanged. This is a process-local compiler
   API boundary and changes no serialized or public language artifact.

## Validation

An ordinary compiler regression injects one source-independent cell layout and write-then-read
transaction into a completed package analysis. The unified result must bind the transaction to the
same function symbol as ordinary MIR and explicit oracle execution must commit and read `42`.

A negative regression removes the required storage effects. Contract-state verification must fail
at `contract-state-mir-lower`, and the compiler must publish no ordinary MIR, no state transaction,
and no verified result. The existing independent-session persistent cache regression must continue
to return the cached transaction under the same nested compiler-result field, create fresh authority,
skip ordinary MIR lowering, and commit/read `42`.

## Compatibility

`compiler.Result` and `workspacecompiler.Result` are internal Go APIs. No Fairy source grammar,
public CLI/schema, package/cache bytes, SDK symbol, runtime ABI, target, lockfile, or Atlas contract
changes. No version bump is required.

## Consequences

All current compilation paths now share one atomic verified-result boundary for ordinary and
contract-state MIR. Source declarations and access expressions, data-dependent control flow,
owner-bound borrows, external-call exclusion, native lowering, runtime persistence, migration, gas,
Merkle proofs, reentrancy, chain integration, and deployable targets remain closed.

ADR 0356 later narrows accepted transaction authority to the storage effects represented by this
compiler-private MIR.
