# ADR 0356: Admit only represented storage effects in contract-state MIR

- Status: Accepted experimental compiler foundation
- Date: 2026-08-31
- Implementation: `internal/contractstatemir/verify.go`
- Depends on: ADR 0346, ADR 0355

## Context

ADR 0346 allowed every released on-chain effect on a contract-state transaction program, including
`block_context`, `emit`, and `external_call`, because the state side model did not own the complete
function result. ADR 0355 now publishes ordinary and contract-state MIR together, but the
contract-state verifier still cannot prove an effect for which it has no operation.

Carrying unrepresented authority is unsafe preparation for source admission: a later consumer could
mistake an accepted effect declaration for verified block, event, or external-call semantics even
though the current transaction program contains only cell/map reads, writes, and deletes.

## Decision

1. Keep `Program.Effects` canonical, sorted, and unique, but admit only `storage_read` and
   `storage_write`. Those are the only effects derived from current transaction operations.
2. Continue rejecting unknown and host-only effects with `CSM1005`. Also reject released but
   unrepresented `block_context`, `emit`, and `external_call` with `CSM1005` and the explicit reason
   that no contract-state transaction-MIR operation represents the effect.
3. Apply the same verifier on direct programs, source-independent HIR lowering, stable body
   encoding/decoding, fresh-owner rebind, persistent cache lookup, and unified compiler-result
   publication. No transport path may bypass the closed profile.
4. Do not infer complete call-graph exclusion from this effect check. ADR 0357 separately requires
   the current transaction owner body to be call-free; admitting any internal or foreign call still
   requires transitive operation/effect, state-borrow, capability, and reentrancy evidence.
5. Keep state-body v1, package-body hash v1, cache key v4, stable compilation v1, persistent compiler
   identity `/2`, language 12, and SDK `0.0.0-stage0.54` unchanged. Existing bytes already carry the
   exact effect set and re-enter the current verifier under an exact compiler-build identity.

## Validation

Direct verifier tests reject all three unrepresented on-chain effects while preserving valid
storage-only read/write execution. HIR lowering rejects an owner declaring `external_call` plus the
required storage effect. Stable body encoding rejects the same otherwise-canonical payload.
`compiler.LowerAnalyzed` classifies the failure as `contract-state-mir-lower` and publishes no
ordinary MIR, state transaction, or verified result.

## Compatibility

This narrows a compiler-private experimental verifier acceptance set. It changes no Fairy source
grammar, public CLI/schema, SDK symbol, runtime ABI, serialized artifact shape, target, lockfile, or
Atlas contract. No version bump is required.

## Consequences

An accepted transaction program now carries only authority justified by its represented storage
operations. ADR 0357 later closes every current owner-body call position. Source state syntax,
data-dependent values and control flow, call admission, owner-bound borrows, event/block-context operations, native lowering, runtime persistence, migration,
gas, Merkle proofs, reentrancy, chain integration, and deployable targets remain closed.
