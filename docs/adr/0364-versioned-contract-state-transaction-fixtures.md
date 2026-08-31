# ADR 0364: Version contract-state transaction execution fixtures

- Status: Accepted
- Date: 2026-08-31

## Context

The experimental contract-state kernel and compiler-private transaction MIR already prove
failure-atomic commit, reject, trap, delete, and resource-limit behavior in package-local Go tests.
Those tests are valuable implementation checks, but they are not a durable cross-package corpus:
the cases, encoded values, expected reads, receipts, and final store observations are embedded in
test code rather than owned by one versioned contract.

This is an admission gap. A later source, interpreter, backend, or persistence implementation could
claim compatibility while selecting only the convenient parts of the existing unit tests.

## Decision

1. `spec/fixtures/contract-state/manifest-v0.json` is the versioned source of truth for the first
   transaction execution corpus. Its schema fixes the layout, explicit limits, canonical encoded
   operation values, exit, expected reads, receipt facts, and committed store entries.
2. `internal/contractstatefixture` strictly decodes the manifest, rejects unknown JSON fields and
   duplicate identities, constructs source-independent HIR, lowers through
   `contractstatemir.LowerHIRTransactions`, re-enters MIR verification, and executes the resulting
   program against a fresh semantic-oracle store.
3. V0 contains exactly five cases:
   - commit with read-your-write visibility;
   - explicit reject rollback;
   - explicit trap rollback with exact category preservation;
   - pending delete visibility and committed absence;
   - write-count-limit failure with rollback.
4. The runner checks exact read order and bytes, store version, receipt write count, and physical
   field-prefix/value observations. Every case starts from a new empty store so state cannot leak
   between fixtures.
5. This corpus does not admit source syntax, conditional or data-dependent operations, native
   lowering, production persistence, chain accounting, or concurrency scheduling. Those require
   separately versioned cases and implementation evidence.

## Verification

The fixture package ratchets the manifest to exactly five cases. A separate schema test keeps the
JSON operation, exit, and result vocabularies byte-for-byte aligned with the runner. The full Go
suite executes the corpus through HIR lowering, verified transaction MIR, and the semantic oracle.

## Consequences

Contract-state transaction semantics now have a reusable admission artifact rather than only
implementation-local tests. A future execution path can be required to consume the same manifest
and produce the same observations without treating the experimental in-memory store as a
production persistence design.
