# ADR 0344: Add a compiler-private contract-state transaction MIR

- Status: Accepted experimental foundation
- Date: 2026-08-31
- Product source: `docs/product/04-contract-profile-candidates.md`
- MIR source: `docs/mir/contract-state-transaction-mir-v0.md`
- State source: `docs/contract-state/transaction-kernel-v0.md`
- Depends on: ADR 0218, ADR 0328, ADR 0338, ADR 0342

## Context

ADR 0338 established canonical typed layouts and a failure-atomic process-local transaction oracle,
but no compiler representation could yet carry that layout or choose commit, reject, and trap edges.
Opening Fairy syntax at this point would still force grammar and HIR to target an unspecified MIR
boundary.

Putting experimental operations directly into the stable whole-program MIR would also be premature.
That artifact feeds the interpreter, C backend, package caches, and codecs, so an optional field would
quietly widen compatibility surfaces before source and target admission gates are ready.

## Decision

1. Add `internal/contractstatemir` as a separate compiler-private transaction submodel.
2. Carry an exact contract UID, layout hash, ordered field identities, canonical key/value types,
   and namespace prefixes. Verification reconstructs the oracle layout and rejects any drift.
3. Admit straight-line cell and map read, write, and delete operations using canonical encoded bytes.
4. Admit exactly three explicit exits: commit, reject, and categorized trap.
5. Verify the complete layout, every operation, every encoded operand, and the exit before beginning
   an oracle transaction.
6. Execute through managed `contractstate.Execute`, preserving atomic commit, returned-error
   rollback, trap rollback and re-panic, layout checks, conflicts, and limits.
7. Add compiler-private encoded cell and map handles inside `internal/contractstate`. They admit only
   the seven built-in canonical codecs and do not change the typed `Cell[T]` and `Map[K,V]` API.
8. Keep stable Fairy source grammar, HIR, whole-program MIR, MIR codecs, package/interface/cache
   identities, interpreter, C backend, target runtime, SDK, and runtime ABI unchanged.

## Validation

Focused Go tests prove:

- compiler-private and typed layouts produce identical hashes, field prefixes, and ordering;
- changed prefixes, noncanonical values, and invalid trap categories fail verification;
- reads observe prior writes in one transaction;
- committed deletes and map writes publish as one overlay;
- reject, trap, and write-limit failure publish no state;
- trap rollback preserves the exact category before the trap is re-panicked.

This is a process-local compiler/oracle boundary. Native generated-Fairy execution is not applicable
until a deterministic target backend consumes the model.

## Compatibility

The language remains `fairyc-language/12`. The stable whole-program MIR, runtime ABI, SDK identity,
target identities, package schemas, lockfile, Atlas formats, and CLI bytes are unchanged. The new Go
package and `CSM1xxx` verifier codes are experimental implementation identities only.

## Consequences

The next source-independent compiler slice has an executable layout and transaction target instead
of prose alone. It can add HIR state declarations, explicit storage effects, and interface/cache
identity without committing to source spelling first.

The submodel cannot yet express data-dependent writes, control flow, calls, ownership borrows, or
external interaction. It must either be integrated into stable MIR with a coordinated compatibility
cut or be replaced explicitly once those semantics are proven; it must not become a second permanent
general-purpose IR.
