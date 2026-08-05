# ADR 0270: Add shared Windows file-size observation

- Status: Accepted
- Date: 2026-08-05
- Product source: `docs/product/02-spec.md`
- Runtime source: `docs/runtime/runtime-stdlib-v0.md`
- Depends on: ADR 0123, ADR 0147, ADR 0264, ADR 0269

## Context

The public Windows File surface can read bytes and control the current position, but it cannot
observe the file's total byte length. A caller therefore cannot plan a bounded buffer, validate a
header's declared extent, or distinguish an empty file without reading to EOF. The existing live
`HANDLE` already owns this information, and Windows exposes it without changing the file pointer.

A general metadata record would also need timestamps, file kinds, attributes, links, and portable
normalization rules. Those policies are not required to expose one exact byte-size observation and
would make this vertical slice substantially less predictable.

## Decision

1. The compiler-bundled SDK version moves from `0.0.0-stage0.15` to `0.0.0-stage0.16`. Language
   remains `fairyc-language/8` and runtime ABI remains `fairyc-runtime/0`.
2. `std::fs` adds `size(ref File) -> Result<usize, FileError>` with the `filesystem` effect. The
   shared borrow makes the absence of ownership or cursor mutation explicit.
3. The target component calls `GetFileSizeEx` and adds
   `fy_std_windows_file_size_v0(const void *)`. Its fixed result is
   `{ status: u32, os_code: u32, size: uint64 }` and crosses the current Windows x86-64 MSVC
   boundary as Fairy `usize` without adding unsupported Stage 0 `u64` source types.
4. Success returns the exact byte length and does not change the current position. A closed File
   returns status 3 and OS code 0. A kernel failure returns status 1, the exact Win32 error, and size
   0. Every outcome preserves ownership and position and performs no allocation or output.
5. The API observes the size associated with the already-open handle. It does not add path-based
   metadata, timestamps, file kinds, permissions, links, directory information, change monitoring,
   truncation, or a concurrency snapshot guarantee against external writers.

## Compatibility

The Fairy surface and closed target-support symbol set are additive, but exact SDK selection is not
lock-compatible. The SDK source digest, package instance, public-interface hash, dependent
identities, generated locks, and whole-package cache keys change. The new C symbol is a pre-1
target-support contract rather than a stable public native ABI.

## Evidence

- Go tests require the new symbol to remain Windows-target-bound, selected only with the Windows
  File component, and present in entry-reachable public quickstart C emission.
- The hand-authored separate-translation-unit C probe observes size 9 before and after seeking,
  proves the cursor stays at 9 or 5, rejects a closed File with status 3, and reports exact
  `ERROR_INVALID_HANDLE` after the underlying handle is externally closed.
- The private wrapper and compiler-bundled SDK quickstart observe size 9 through a shared File
  borrow before subsequent position/read checks and observe size 0 for a newly truncated empty
  file. Low and optimized generated-Fairy MSVC images return 42.

## Consequences

Windows Stage 0 programs can inspect an open file's byte length without consuming the owner,
moving its cursor, allocating storage, or exposing raw handles. General metadata, portable file
kinds and timestamps, path-based queries, external-write consistency, truncation, and Linux public
linkage remain separate contracts.
