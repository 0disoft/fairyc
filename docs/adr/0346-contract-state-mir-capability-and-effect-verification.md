# ADR 0346: Verify contract-state transaction capability and effects

- Status: Accepted experimental compiler foundation
- Date: 2026-08-31
- MIR source: `docs/mir/contract-state-transaction-mir-v0.md`
- Effect source: `docs/language/0008-effects-and-capabilities-v0.md`
- Depends on: ADR 0338, ADR 0342, ADR 0344, ADR 0345

## Context

The compiler-private contract-state MIR already verified layout, operation, and exit shape before
opening an oracle transaction. Its operations did not identify the transaction authority they used,
and the program did not carry the function-boundary effects required by those operations. A future
HIR lowerer could therefore construct executable state access without proving either invariant.

## Decision

1. Give every transaction-MIR program one compiler-created opaque `TransactionCapability` identity.
   Its token has nonzero storage identity and cannot be fabricated outside `contractstatemir`.
   Copying the handle preserves one identity rather than creating independent authority.
2. Require every state operation to carry exactly the program capability. Missing or different
   identities fail with `CSM1004` before the oracle captures a snapshot.
3. Carry a canonical sorted, unique set of released on-chain HIR effects on each program. State
   reads require `storage_read`; writes and deletes require `storage_write`. Missing, noncanonical,
   unknown, or host-only effects fail with `CSM1005`.
4. Permit additional released on-chain effects because a complete source function may conservatively
   declare authority used by operations outside this straight-line state submodel.
5. Keep `contractstate.Execute` as the sole managed lifecycle owner. The capability is compiler proof
   metadata and does not expose public commit, rollback, or mutable transaction handles.
6. Keep source grammar, ordinary typed function HIR, stable MIR, package artifacts and caches,
   interpreter, backend, runtime, SDK, and target identities unchanged.

## Validation

Unit tests prove that distinct fresh capability identities do not compare equal, operations using a
different identity fail, missing storage effects fail, noncanonical effect sets fail, host effects
fail, and a valid mixed read/write transaction still commits and observes read-your-writes through
the failure-atomic oracle.

## Compatibility

This is a compiler-private in-memory model cut. The language remains `fairyc-language/12`;
public-interface v5, package-cache key v3, stable MIR, runtime ABI, SDK, target, lockfile, Atlas, and
CLI bytes remain unchanged.

## Consequences

The next slice can lower source-independent HIR state accesses only by allocating one capability per
entry transaction, attaching it to every lowered operation, and preserving declared effects. Source
syntax, data-dependent values, calls, branches, state-borrow lifetimes, external-call exclusion,
cached body transport, and deterministic target execution remain closed gates.
