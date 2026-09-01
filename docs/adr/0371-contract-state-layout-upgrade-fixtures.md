# ADR 0371: Version contract-state layout publication schedules

- Status: Accepted
- Date: 2026-09-01
- Product source: `docs/product/04-contract-profile-candidates.md`
- Kernel source: `docs/contract-state/transaction-kernel-v0.md`
- Fixture source: `spec/fixtures/contract-state-layout/manifest-v0.json`
- Implementation: `internal/contractstatelayoutfixture/`

## Context

`MemoryStore.Upgrade` already admits an identical layout or an append-only extension and rejects a
breaking layout. Static layout comparison alone does not prove the publication boundary. An old
transaction can remain open while the store admits a new layout, so the executable contract must
show that the old overlay cannot cross that change and that a rejected candidate cannot disturb the
current store.

The transaction and conflict manifests have their own historical identities. Rewriting either corpus
to add layout publication would weaken their role as preserved evidence.

## Decision

1. Add `fairyc.contract-state-layout-fixtures/v0` as a separate versioned fixture lane. Historical
   `contract-state` and `contract-state-conflict` manifests remain byte-for-byte unchanged.
2. A changed layout is compatible only when its contract UID is identical, its field count does not
   shrink, and every previous field remains byte-for-byte equal at the same index. Field equality
   includes index, name, kind, key codec identity, value codec identity, and derived namespace
   prefix. New fields may appear only after the complete previous prefix.
3. Every fixture case commits seed bytes, opens a transaction on the base layout, prepares a
   multi-key overlay, and then attempts to publish one candidate layout. This preserves the exact
   time-of-check/time-of-commit boundary.
4. The accepted append-only case must change the layout hash, preserve every old field identity,
   leave committed bytes unchanged, increment the store version once, and make the old transaction
   fail with `ErrLayoutMismatch`. The failed commit publishes no prefix of its overlay and does not
   increment the version.
5. Removal, reorder, rename, field-kind change, map-key codec change, value codec change, and
   contract-namespace change are valid layouts but incompatible extensions. `Upgrade` must return
   `ErrIncompatibleLayout` without changing layout, version, or bytes. Because the rejected proposal
   has no effect, the already-open base-layout transaction remains able to commit atomically.
6. Identical-layout publication remains the kernel's existing no-op. Data migration, codec
   conversion, field reuse, and in-place rewriting remain closed.

## Evidence

`manifest-v0.json` contains one accepted append-only schedule and seven rejected breaking schedules.
The runner compares complete field identities and deep-copied store dumps before and after
publication, checks exact error classes, and reads the final state through a fresh transaction.

The rejected cases also commit the transaction that was open before the proposal. This proves that a
failed upgrade did not silently advance the store version or invalidate live work.

## Consequences

Append-only publication now has executable evidence across the actual store boundary rather than
only a pure layout comparison. A future persistent backend must preserve the same observable rules.
Migration tooling, authorization, crash recovery, Merkle state, native lowering, and deployable
contract targets still require separate decisions and fixtures.
