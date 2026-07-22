# Contained `.fymap` store and current-source validation

- Status: Accepted
- Date: 2026-07-23
- Product source: `docs/product/02-spec.md`
- Store source: `docs/atlas/fymap-store-v1.md`
- Transport prerequisite: `docs/adr/0186-bounded-canonical-fymap-transport-v1.md`

## Context

Canonical map bytes are not enough to make a usable reader. Filesystem publication must not escape
the caller's selected root, interruption must never expose partial semantic truth, retained bytes
need explicit bounds, and a valid old map must not return spans for changed source. A mutable
`current.fymap` pathname would also make Windows replacement and concurrent writer behavior harder
than the content-addressed transport already permits.

The persistent package cache already proved an opened-root, immutable-generation publication
boundary. ADR 0187 reuses its extracted `internal/durablefile` primitive while keeping Atlas schema,
retention, compatibility, and source freshness in `internal/atlasmap`.

## Decision

1. Add an internal opt-in `MapStore` with mandatory caller limits for generation count, total bytes,
   individual map bytes, and temporary-file recovery age. Define no default path or quota.
2. Bind every operation to `os.Root`. The versioned relative layout is
   `fymap-v1/<PackageInstanceId>/<64-lowercase-checksum>.fymap`.
3. Publish only canonical bytes from ADR 0186. Write a permission-restricted exclusive temporary in
   the final directory, sync and close it, then use shared same-directory immutable publication.
   Identical writers converge; different bytes under one checksum-derived name are corruption and
   are repaired.
4. Retain multiple independently checksummed generations when compiler or fact output differs. A
   reader returns exactly one current compatible generation. Multiple valid compatible generations
   are a hard conflict, never an arbitrary last-writer selection.
5. Decode and revalidate every candidate before use, verify that the logical checksum matches its
   filename and its package instance matches its directory, quarantine corrupt generations
   best-effort, and ignore unrelated invalid names.
6. Derive `CurrentPackage` records only from validated workspace metadata and recomputed retained
   source bytes. Require exact package UID, language, instance, coordinate, aggregate source digest,
   source kind, runtime ABI, features, dependencies, package namespace references, and every file
   path/content/line-map/kind identity before trusting spans.
7. Require the caller's exact compiler executable identity and current public-interface hash in
   addition to current source. A mismatch is explicit incompatibility, not a guessed join.
8. Sweep expired unpublished temporaries and evict oldest immutable generations until both caller
   bounds hold. Modification time is only a retention hint and never semantic authority.
9. Keep publication and load failure downstream from compilation. Callers may regenerate maps; they
   cannot turn a map failure into a source diagnostic or build failure.

## Consequences

- Source edits normally change `PackageInstanceId` and therefore select another directory. The
  reader still compares the aggregate and every per-file identity so post-validation mutation or a
  manually contradictory expectation fails closed.
- A checksum is not authentication. The reader additionally validates internal semantics and exact
  current compiler/interface/source compatibility, but this decision defines no signer or remote
  trust policy.
- The opened-root and sync guarantee is the same verified-hit-or-safe-miss boundary as ADR 0169.
  Windows flushes the published file because a portable directory-handle flush contract is absent.
- No ordinary command, default output directory, background writer, watcher, dependency-map
  discovery, query API, or automatic retention policy is enabled.

## Evidence

Tests cover two-package publication, identical convergence, close/reopen loading, exact current
workspace acceptance, aggregate/file/line-map/kind/path/feature staleness, compiler incompatibility,
checksummed conflicting generations, corrupt-file quarantine, generation bounds, stale temporary
recovery, traversal and external-symlink containment, and post-validation source mutation. Separate
process tests kill a real publisher after its temporary closes and after publication; a new process
observes respectively a miss or one fully decoded, current, verified hit.
