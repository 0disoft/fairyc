# ADR 0365: Version contract-state map execution fixtures

- Status: Accepted
- Date: 2026-08-31

## Context

ADR 0364 created a durable five-case transaction corpus, but its layout and operations exercise
only direct cells. The transaction MIR and semantic oracle also expose typed map read, write, and
delete operations with a domain-separated physical entry key and an independent encoded-key byte
limit. Leaving that half of the storage model only in Go unit tests would make the versioned corpus
an incomplete admission gate.

The historical v0 manifest must remain reproducible. Extending it in place would silently change an
accepted fixture identity and make its explicit five-case ratchet misleading.

## Decision

1. Preserve `manifest-v0.json` and `schema-v0.json` unchanged as the five-case cell baseline.
2. Add `manifest-v1.json` and `schema-v1.json`. V1 retains all five v0 cases and appends exactly
   three map cases:
   - canonical `bytes20/v1` key write plus same-transaction read and commit;
   - pending map write, delete, and absent read before commit;
   - a 20-byte canonical key rejected by a 19-byte execution limit with no publication.
3. V1 records the exact 32-byte domain-separated committed map storage key as
   `storageKeyHex`. The runner compares that vector and the stored value independently of a second
   implementation of the key-derivation algorithm.
4. `internal/contractstatefixture` accepts only the explicit v0 and v1 schema identities. Tests
   keep both schema vocabularies aligned with the runner and ratchet the historical corpus to five
   cases and the current corpus to eight.
5. The extension changes no source syntax, HIR or MIR format, public API, target, runtime,
   persistence backend, or compatibility version. Map concurrency, conflict scheduling, and layout
   upgrades remain separate fixture slices.

## Verification

The current runner executes both manifests through source-independent HIR lowering, transaction
MIR verification, and fresh in-memory stores. V1 requires exact read order and bytes, committed
storage key and value, receipt/store versions, write counts, delete visibility, error category, and
rollback after the map-key limit failure.

## Consequences

The versioned corpus now covers both storage shapes represented by contract-state transaction MIR.
Future source, backend, runtime, or persistence candidates can be required to satisfy the eight-case
v1 corpus while historical v0 remains independently executable.

ADR 0366 later routes both corpora through the compiler's atomic ordinary/state MIR publication
boundary before oracle execution.
