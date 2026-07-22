# ADR 0163: Use bounded immutable generations for disposable disk cache records

- Status: Accepted storage primitive
- Date: 2026-07-22
- Implementation: `internal/packagecache/disk.go`
- Stable payload prerequisite: `docs/adr/0162-stable-package-mir-artifact.md`

## Context

Stable package MIR bytes make persistence technically possible, but a mutable `key -> file` layout
would immediately require reliable cross-platform replacement and lock recovery. On Windows, an
open destination can block replacement; a crashed lock owner can leave stale authority; and an
unbounded temp or generation directory would recreate the disk-growth failure Fairy C is intended
to avoid.

## Decision

Add a caller-configured `DiskStore` for generic package-cache records. All four limits are required:
maximum immutable generation count, retained encoded bytes, encoded bytes per record, and abandoned
temporary-file age. There are no defaults.

The v1 layout is `<root>/v1/<cache-key-digest>/<record-sha256>.json`. Record files are canonical,
strict, exact-version JSON and immutable after publication. Writers create, restrict, write, sync,
and close a unique temporary file before renaming it to the content digest. Concurrent writers of
the same bytes converge on the same name. Different valid generations for one cache key are a
producer conflict and therefore a miss; neither is selected by timestamp or filesystem order.

Reads bound allocation before decoding, verify the filename digest, canonical bytes, embedded key,
public-interface metadata, artifact kinds, and record shape. Corrupt generations are best-effort
deleted and counted as rejected. Collection removes caller-expired temporary files and then the
oldest immutable generations until entry and byte bounds both hold. Eviction time is a disposable
hint, never correctness or source truth.

## Evidence

Tests use independent store instances and concurrent writers over one directory, then require a
deep-copy hit. Separate OS processes additionally publish identical payloads concurrently and must
converge on one generation; conflicting process writers must produce a rejected safe miss. A helper
process exits after leaving an unpublished temporary, which caller-aged collection removes. Tests
also inject corruption, enforce entry bounds, and prove oversized rejection preserves an existing
hit.

## Consequences and limits

- No lock file, mutable index, daemon, watcher, background cleanup, or source-of-truth state exists.
- Publication syncs file contents but does not yet claim directory-entry durability across sudden
  power loss on every filesystem. Losing a cache entry remains a safe miss.
- Eviction is creation/modification-time ordered rather than access-time LRU, avoiding a write on
  every hit.
- ADR 0164 connects the store to explicit Sessions and ADR 0166 adds one experimental command. No
  ordinary command, default policy, or cold/warm disk performance claim exists.
- Power-loss directory durability and retained-byte/performance measurements remain required before
  enabling persistence in a compiler command. Process-exit temporary recovery is covered, but it is
  not a filesystem power-cut guarantee.
