# ADR 0368: Version contract-state map conflict schedules

- Status: Accepted
- Date: 2026-08-31

## Context

ADR 0367 versions overlapping cell transactions, but it does not demonstrate whether optimistic
serialization is scoped to a cell, field, or complete store version. `MemoryStore` compares the
transaction's base version with the store version before inspecting pending write keys. Therefore
two overlapping writers conflict even when they update distinct map keys.

Without a durable map schedule, a future backend could incorrectly implement per-key last-writer
success while still passing the cell-only historical corpus.

## Decision

1. Preserve `contract-state-conflict/manifest-v0.json` and `schema-v0.json` byte-for-byte as the
   three-schedule cell baseline.
2. Add v1 with explicit cell/map field kinds. It retains all three cell schedules and appends
   exactly three map schedules:
   - a winner commits key A while a stale transaction has key B in its overlay; key B still
     conflicts and publishes nothing;
   - an already-open reader does not observe key A committed after its snapshot;
   - a fresh transaction observes key A and then commits key B at version two.
3. The v1 manifest owns exact physical storage-key vectors for both canonical `bytes20/v1` keys.
   Final-state comparison permits multiple entries in one map field but rejects duplicate physical
   entry identities.
4. The runner selects action vocabulary by schema version. V0 remains cell-only; v1 adds
   `map-read` and `map-write` with mandatory canonical key identity and bytes.
5. This decision fixes store-wide optimistic serialization for the experimental oracle. It does not
   choose sharding, key-level locking, retry policy, production database isolation, or consensus.

## Verification

Tests ratchet historical v0 to three schedules and current v1 to six. Both schemas must match their
version-specific runner vocabulary. V1 checks exact map snapshot reads, conflict error and zero
receipt, winner/version preservation, fresh observation, physical keys, and canonical values.

## Consequences

Backend candidates cannot infer that disjoint map keys permit overlapping commits under the current
contract. Any later key-granular concurrency model is a semantic revision requiring a new fixture
version rather than an optimization hidden beneath v1.
