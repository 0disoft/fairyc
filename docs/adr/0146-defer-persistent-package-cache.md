# ADR 0146: Defer persistent package-cache storage until artifacts are real

- Status: Accepted defer decision
- Date: 2026-07-21
- Product source: `docs/product/02-spec.md`
- Cache source: `docs/package/whole-package-cache-v1.md`
- Workspace command source: `docs/cli/command-contract.md`

## Context

The repository has a complete whole-package cache key and a bounded process-local LRU store. The
new workspace command supplies a real package-aware semantic and verified-MIR path, but every
invocation is still a cold process. More importantly, `typed-hir`, `mir`, and `object` cache
artifacts are only closed kind names: none has a compiler-owned byte encoding, decoder, verifier,
or package-aware object producer.

Persisting the existing opaque test bytes would create a directory format before there is anything
real to reuse. Choosing default disk limits from fixture payloads would be measurement theater and
would directly conflict with Fairy C's small-build-artifact goal.

## Decision

1. Do not add persistent cache files, directories, indexes, locks, cleanup commands, or default disk
   budgets in the current slice.
2. Keep `MemoryStore` as the only implementation. It remains caller-bounded and process-local.
3. Reopen persistence only after one compiler-owned artifact has canonical encode/decode plus
   verification, and a package-aware build path can demonstrate a correct cache hit.
4. Required evidence is comparable cold and warm command measurements, bytes written and retained,
   hit/miss/eviction counts, invalidation mutations, corrupt-entry fallback, concurrent-process
   behavior, and cleanup recovery. A configured benchmark or measurement intent must own the run.
5. A persistent store must remain disposable: corrupt, incompatible, absent, locked, or full cache
   state becomes a miss or bounded warning, never source truth or a failed correct build.
6. Disk limits, entry limits, sharding, index format, compression, and collection policy stay
   `UNDECIDED` until that evidence exists.

## Consequences

- Stage 0 avoids inventing the exact disk-growth problem Fairy C is supposed to distinguish itself
  from.
- The cache key and in-memory API remain useful prerequisites rather than fake persistence proof.
- Package-aware SDK/build work comes first because it creates the artifact and repeat-build path
  that future cache measurements need.

## Reopen gate

- canonical compiler-private artifact bytes exist and round-trip through a verifier;
- workspace build/run produces that artifact for at least one supported Windows path;
- a configured measurement intent captures comparable cold/warm time, allocations, reads/writes,
  retained bytes, and correctness outcomes;
- the proposed persistent policy shows bounded positive ROI instead of merely producing a hit.

## Current evidence update (2026-07-22)

ADR 0149 now supplies a bounded verified whole-program MIR codec, ADRs 0157-0158 preserve
process-local cold, warm, and edit-class observations, and ADR 0159 exposes that reuse through a
bounded foreground workspace-check session. None of this writes or reads cache state across a
process boundary. Process-memory measurement, representative workloads, corrupt persistent-entry
recovery, concurrent-process behavior, cleanup recovery, and a justified retained-disk policy
therefore keep the persistence gate closed.
