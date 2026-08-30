# ADR 0353: Publish stable compilation bundles through the persistent cache

- Status: Accepted experimental compiler foundation
- Date: 2026-08-31
- Implementation: `internal/packageartifact/stable_cache.go`,
  `internal/workspacecompiler/session.go`
- Artifact contract: `docs/package/stable-package-compilation-v1.md`
- Depends on: ADR 0164, ADR 0349, ADR 0352

## Context

ADR 0352 created one coordinate-bound package compilation artifact, but the persistent adapter still
published MIR v3 alone. Cache key v4 selected an exact contract-state body hash without requiring
the retrieved record to carry that body. A later combined reassembly path therefore could not
distinguish a complete state-aware record from an older MIR-only generation at the adapter boundary.

## Decision

1. Add the compiler-private cache artifact kind `compilation`. A persistent record contains exactly
   one canonical `fairyc-stable-package-compilation/1` payload rather than one MIR-only payload.
   The generic disk-record v1 envelope and durable publication algorithm remain unchanged.
2. Require every lookup to supply the nonzero ADR 0348 body hash already used to construct package-
   cache key v4. A hit requires exact key, public-interface hash, package instance, nested artifact
   validation, shared coordinate, and state-body hash equality.
3. Treat wrong artifact cardinality or kind, legacy MIR-only records, malformed bytes, package
   mismatch, interface mismatch, and body mismatch as rejected generations. Quarantine uses
   compare-and-delete, so a concurrent valid replacement is preserved. A missing caller expected
   hash is rejected before reading or deleting a valid generation.
4. Advance the workspace persistent compiler identity to
   `fairyc.compiler-stable-package-cache/2` and bind it to the outer compilation, nested MIR, nested
   state-body, and disk-record versions. Existing `/1` cache keys become ordinary misses; no
   migration or compatibility fallback is attempted.
5. On a miss or rejection, lower and verify ordinary MIR first. Only then describe stable state
   bodies, require their hashes to equal the identities used in cache-key construction, and publish
   the complete bundles. Oversized artifacts remain skipped optimizations and store failures remain
   non-authoritative cache failures.
6. On an all-hit lookup, pass only the verified cached MIR members into the existing stable MIR
   reassembly. Do not yet rebind or execute the cached state bodies. Combined MIR/state fresh-symbol
   alignment remains a separate gate.

## Validation

Adapter tests use independent disk-store handles to publish and retrieve a complete bundle carrying
one real cell-write transaction, verify both nested artifacts and operation retention, and prove
caller mutation cannot alter retained bytes. They require a missing expected hash to leave a valid
generation intact, and quarantine interface, package, body, legacy MIR-only, and malformed-payload
contradictions. Existing workspace Session and separate-process tests now exercise compilation
records while still requiring complete hits, zero second lowering, fresh verified MIR reassembly,
and interpreter result `42`. A dedicated identity ratchet requires persistent cache `/2` plus the
outer compilation, nested MIR, nested state-body, and disk-record versions in exact order.

## Compatibility

This changes only disposable compiler-private persistent-cache contents and compiler key identity.
Stable MIR remains `fairyc-stable-package-mir/3`; the outer artifact remains
`fairyc-stable-package-compilation/1`; package-cache key v4, disk-record v1, language
`fairyc-language/12`, public-interface v5, runtime ABI, SDK, target, lockfile, Atlas, and CLI schemas
remain unchanged.

## Consequences

Every new persistent hit now proves that the exact cache-selected state body travels with the MIR
for the same package coordinate. Cache deletion still only causes ordinary compilation. ADR 0354
later adds fresh-owner cached state-body rebind, cross-package symbol alignment, combined transaction
reassembly, and explicit oracle execution. Source syntax, backend lowering, runtime persistence, and
targets remain closed.
