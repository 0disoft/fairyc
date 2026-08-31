# ADR 0367: Version overlapping contract-state transaction conflicts

- Status: Accepted
- Date: 2026-08-31

## Context

The contract-state oracle has package-local tests for optimistic serialization, but the versioned
transaction corpora execute each program against a fresh store with no overlapping transaction
lifetime. They therefore cannot act as durable evidence for snapshot selection, stale commit
conflicts, or the absence of partial state publication.

A store conflict is not a compiler publication failure. Verified ordinary and contract-state IR
already exist before execution begins. Erasing executable IR after an optimistic commit conflict
would incorrectly couple compiler authority to runtime storage scheduling.

## Decision

1. Add `spec/fixtures/contract-state-conflict/manifest-v0.json` as a separate deterministic schedule
   corpus owned by `internal/contractstateconflictfixture`.
2. V0 contains exactly three cell schedules:
   - two transactions start at version zero; the winner commits one field, the stale writer's
     disjoint overlay conflicts, and only the winner remains published at version one;
   - an already-open reader retains its version-zero snapshot after an overlapping writer commits;
   - a transaction begun after a commit observes that value and can publish a second disjoint field
     at version two.
3. Every step names one transaction lifetime and one action. The strict runner rejects unknown JSON,
   duplicate case/step/field identities, transaction-name reuse, invalid action-specific fields,
   and schedules that leave a transaction open.
4. Conflict requires `contractstate.ErrConflict`, a zero receipt, no stale-overlay entries, and no
   store-version increment. It does not alter or make claims about compiler IR publication.
5. Keep the compiler-published transaction fixture v0/v1 bytes unchanged. Parallel goroutine
   scheduling, map conflicts, layout upgrades, retry policy, production databases, and distributed
   consensus remain separate evidence.

## Verification

The runner constructs one canonical layout and fresh `MemoryStore` per case, begins explicitly
overlapping transactions, checks every read and commit receipt, requires all lifetimes to close, and
compares the final store by exact physical cell key and canonical value bytes. The test ratchets v0
to exactly three schedules and keeps schema action/status vocabularies aligned with the runner.

## Consequences

The semantic oracle now has durable snapshot and conflict evidence without pretending that a
process-local schedule format is a production isolation-level or retry contract. Backend candidates
must map these observable state outcomes separately from compiler executable-IR publication.
