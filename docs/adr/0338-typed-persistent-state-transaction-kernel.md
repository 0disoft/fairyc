# ADR 0338: Add a typed persistent-state transaction kernel

- Status: Accepted experimental foundation
- Date: 2026-08-18
- Product source: `docs/product/04-contract-profile-candidates.md`
- Contract source: `docs/contract-state/transaction-kernel-v0.md`
- Depends on: ADR 0001, ADR 0008, ADR 0218, ADR 0328

## Context

A future Fairy C contract profile needs persistent state before chain-specific accounts, events,
ABI formats, or virtual machines matter. Treating state as ordinary mutable globals would erase the
layout, authority, atomicity, and upgrade boundaries that a deterministic contract runtime must
prove.

Adding `state` syntax directly to the Stage 0 parser would be premature. The repository currently has
no state-layout identity, transaction object, write overlay, rollback oracle, conflict rule, or
upgrade compatibility model for such syntax to lower into. Grammar-first work would create a public
shape around semantics that still move underneath it.

The compiler already uses implementation-private executable packages and fixtures to establish a
bounded semantic contract before opening a source or backend surface. Persistent state should follow
the same order.

## Decision

1. Add `internal/contractstate` as the host-independent executable kernel for future persistent
   contract state.
2. Layouts have an ASCII contract UID and ordered `cell` or `map` fields. Every field owns a stable
   index, name, codec identities, and SHA-256 namespace prefix. The complete ordered layout owns a
   separate domain-separated hash.
3. Add generic implementation-side `Cell[T]` and `Map[K, V]` handles. They encode and decode through
   explicit canonical codecs and validate that their complete field descriptor exists in the active
   transaction layout.
4. Supply canonical bool, big-endian `u32`, big-endian `u64`, raw bytes, UTF-8 text, 20-byte, and
   32-byte codecs. `FuncCodec[T]` remains implementation-only and makes its stable ID and deterministic
   byte behavior the caller's responsibility.
5. Add `MemoryStore` as a process-local semantic oracle. Beginning a transaction copies one committed
   snapshot and version. Reads observe the private overlay before that snapshot.
6. Writes and deletes enter a bounded overlay. Commit checks the exact layout hash and snapshot
   version, then applies the complete overlay under one lock in sorted-key order. Layout drift or an
   intervening commit applies nothing.
7. `Execute` commits only after a nil body result. Callback code cannot close a managed transaction
   early through public `Commit` or `Rollback`. A returned error discards the overlay. A panic also
   discards the overlay and is re-raised so trap behavior cannot become success accidentally.
8. Every execution receives explicit limits for distinct writes, aggregate write bytes, value bytes,
   and encoded map-key bytes. Rejected mutations leave prior pending mutations unchanged.
9. Layout upgrade is identical or append-only. Contract UID, existing field count prefix, indexes,
   names, kinds, codec identities, and field prefixes cannot change. Upgrade increments the store
   version and invalidates open transactions.
10. This decision adds no Fairy C keyword, grammar, diagnostic, HIR node, MIR operation, standard
    library API, target, runtime ABI, chain adapter, persistent format, event system, gas schedule, or
    public compatibility promise.

## Validation

Package tests prove:

- deterministic equal-layout hashes and distinct field prefixes;
- append-only compatibility plus rejection of reorder and codec changes;
- typed cell and map commit plus read-only snapshot observation;
- read-your-writes and delete visibility;
- complete rollback on returned error and panic;
- stale-writer rejection without partial publication;
- failure-atomic write-count and encoded-key limits;
- state preservation across append-only upgrade;
- invalidation of transactions opened before an upgrade;
- deep-copy boundaries and race-safe package execution through `go test -race`.

The focused validation commands are:

```text
go test ./internal/contractstate
go vet ./internal/contractstate
go test -race ./internal/contractstate
```

## Compatibility

The language remains `fairyc-language/10`; MIR, runtime ABI, SDK identity, target identities, C ABI,
package schemas, lockfile, Atlas formats, and stable CLI bytes are unchanged. The new hash domains are
experimental implementation identities and do not yet appear in public artifacts.

A later source integration must cut the appropriate language, interface, cache, MIR, diagnostic, and
target compatibility dimensions together. It cannot treat this internal Go API as the public Fairy
surface.

## Consequences

Persistent-state work now has executable behavior to lower into instead of only attractive syntax.
The coarse full-snapshot and store-wide conflict model is intentionally simple and expensive; it is
an oracle, not the intended production backend. A later database or Merkle implementation can be
optimized aggressively while differential tests hold it to the same commit and rollback results.

State declarations, effects, owner-bound state borrows, external-call reentrancy rules, MIR
verification, deterministic target execution, layout migration, Merkle proofs, and gas accounting
remain separate gates. None is implied by this foundation.
