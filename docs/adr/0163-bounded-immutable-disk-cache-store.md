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
and close a unique temporary file before durably publishing it to the content digest. ADR 0168
defines Windows write-through rename and non-Windows directory sync. Concurrent writers of
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
process exits after leaving an unpublished temporary, which caller-aged collection removes.
Deterministic real-`Put` crash checkpoints additionally kill a subprocess after its synced temporary
closes and after publication: a fresh reader must observe respectively a miss with no immutable
generation or one canonical hit. Tests also inject corruption, enforce entry bounds, and prove
oversized rejection preserves an existing hit.

## Consequences and limits

- No lock file, mutable index, daemon, watcher, background cleanup, or source-of-truth state exists.
- ADR 0168 requests host-specific publication durability but deliberately guarantees only verified
  hit or safe miss after interruption. No portable contract can force a filesystem, controller,
  hypervisor, or network share to honor flushes.
- Eviction is creation/modification-time ordered rather than access-time LRU, avoiding a write on
  every hit.
- ADR 0164 connects the store to explicit Sessions and ADR 0166 adds one experimental command. No
  ordinary command, default policy, or cold/warm disk performance claim exists.
- Retained-growth, reference-environment, and accepted performance evidence remain required before
  ordinary-command defaults. Process-crash checkpoints do not simulate physical device power cuts.
