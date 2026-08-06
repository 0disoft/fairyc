# ADR 0309: Public Windows File allocated size

- Status: Accepted
- Date: 2026-08-06

## Context

Fairy C exposes logical file length, same-handle kind, read-only state, link count, and identity, but
cannot observe the allocation length reported by the filesystem. Storage diagnostics therefore
cannot distinguish logical bytes from the storage reservation visible through the open handle.
Timestamps remain blocked on a portable signed time and epoch contract, while attribute mutation
has broader access and state-preservation requirements.

## Decision

SDK `0.0.0-stage0.43` adds:

```fairyc
std::fs::allocated_size(ref File) -> Result<usize, FileError>
```

The private Windows adapter `fy_std_windows_file_alloc_size_v0` validates the existing File
storage, rejects closed storage, and performs one `GetFileInformationByHandleEx(FileStandardInfo)`
query. It maps nonnegative `FILE_STANDARD_INFO.AllocationSize` to `usize`, preserves exact query
errors, and rejects an impossible negative target value as arithmetic overflow. It does not
allocate, reopen a path, retain a pointer, move the cursor, or change the file.

The value is a momentary filesystem-reported allocation length. It is not a portable promise of
physical disk consumption, block shape, compression accounting, sparse extents, quotas, billing,
durability, or stability under concurrent changes.

## Evidence

- SDK and closed-symbol tests require the public function, exact target symbol, component closure,
  and SDK `0.0.0-stage0.43`.
- The dedicated raw metadata probe observes allocation at least as large as a written payload,
  preserves cursor and logical size, and covers closed and invalid storage under low and optimized
  MSVC.
- The checked-in generated-Fairy quickstart observes allocation at least as large as logical size
  through ordinary, read-only, and both names of a two-link file while preserving cursor and size
  under low and optimized MSVC.

## Consequences

Fairy C gains a small C-friendly storage diagnostic without a compound metadata record or new
resource machinery. SDK identity, locks, and dependent cache keys change; language and runtime ABI
do not. Timestamps, extent enumeration, compression/sparse classification, allocation mutation,
attribute mutation, portable Unicode paths, and Linux public linkage remain separate gates.
