# ADR 0168: Define platform-specific durable publication and crash-safe cache outcomes

- Status: Accepted storage hardening
- Date: 2026-07-22
- Implementation: `internal/packagecache/disk.go`, `internal/packagecache/durability_*.go`
- Storage prerequisite: `docs/adr/0163-bounded-immutable-disk-cache-store.md`

## Context

The disk cache already wrote, synced, and closed a unique temporary file before a same-directory
rename. Readers verified the digest, canonical bytes, embedded key, interface, and artifact shape,
so an incomplete file could not become a hit. The remaining gap was the publication boundary:
ordinary Go rename does not itself state one cross-platform metadata-durability contract, and
process-exit recovery of an abandoned temporary is not evidence about a sudden power interruption.

This cache is disposable derived state. Its required invariant is not that every successful write
survives every storage stack. The invariant is that interruption can leave only an unpublished
temporary, no generation, one complete valid generation, or a conflicting set that becomes a safe
miss. Source compilation must never trust partial bytes or fail because cached state disappeared.

## Decision

Keep the existing content-addressed immutable layout and strengthen its publication primitive by
platform:

- Windows converts both paths to absolute verbatim paths and calls `MoveFileExW` with
  `MOVEFILE_REPLACE_EXISTING | MOVEFILE_WRITE_THROUGH`. This preserves long-path behavior, retains
  same-volume replacement, and requests that the move complete on disk before success.
- Non-Windows builds perform the same-directory `os.Rename` and then open and sync the containing
  directory. Creation of a key directory also syncs the version-root directory before publication.
- The temporary file is still permission-restricted, fully written, file-synced, and closed before
  either publication path runs.
- A failed directory sync or write-through move is a failed cache store. The compiler keeps the
  verified compile result and reports `storeFailed`; it never promotes cache state to authority.
- If publication completed but the durability confirmation failed, the file may remain. Future
  readers independently verify it. Cleanup does not delete an ambiguous generation and risk racing
  a valid concurrent publisher.

The guarantee is crash-safe observation, not universal retention. After process loss or storage
interruption, a reader may get a verified hit or a safe miss. Hardware that lies about flushes,
network filesystems with weaker semantics, controller caches without power protection, and loss of
newly created ancestor directories remain outside a portable language guarantee.

## Evidence

All ordinary disk-store tests now traverse the platform publication primitive. A deterministic
subprocess helper executes the real `DiskStore.Put` path and parks at two checkpoints:

1. after the temporary file has been synced and closed but before publication; the parent kills the
   process and a new store must report a miss with zero immutable generations;
2. after durable publication but before collection; the parent kills the process and a new store
   must return exactly the canonical record with no rejection.

Existing tests still cover same-content process convergence, conflicting-generation safe misses,
corruption rejection, stale temporary collection, bounded eviction, and exact-record quarantine.
These are process-crash and protocol tests. They do not cut power to a physical device or prove a
particular filesystem, controller, hypervisor, or network share honors its flush contract.

## Consequences and limits

- The power-loss boundary is now explicit: retention may degrade to a miss, but partial or ambiguous
  state cannot become executable cache truth.
- Windows publication no longer inherits ordinary `os.Rename`'s lack of a write-through request.
- Unix-like publication now fails closed when directory sync is unavailable instead of silently
  claiming durable metadata.
- Default cache paths and limits still require representative retained-growth and performance
  evidence. This ADR does not enable the cache for ordinary commands.
- Physical power-cut testing may be added for a declared reference storage stack, but it is not a
  portable correctness prerequisite for a disposable verified cache.

## References

- Microsoft `MoveFileExW` and `MOVEFILE_WRITE_THROUGH`:
  <https://learn.microsoft.com/en-us/windows/win32/api/winbase/nf-winbase-movefileexw>
