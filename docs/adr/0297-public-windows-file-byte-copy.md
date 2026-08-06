# ADR 0297: Copy one Windows file into a new byte-for-byte destination

- Status: Accepted
- Date: 2026-08-06
- Product source: `docs/product/02-spec.md`
- Runtime source: `docs/runtime/runtime-stdlib-v0.md`
- Depends on: ADR 0264, ADR 0275, ADR 0291

## Context

Public `std::fs` can open, read, write, create, remove, and rename files, but a safe program still
has to implement its own copy loop. A copy helper is useful only if its overwrite, sharing,
partial-failure, metadata, and recovery behavior are explicit. Forwarding directly to `CopyFileW`
would imply target-owned metadata and link behavior without exposing a completed-byte count.

The existing Windows File component already owns bounded ASCII path conversion, exact
`FileError`, partial counts, and synchronous handle I/O. It can provide a byte-copy operation
without allocation, a temporary-name policy, or a second public error type.

## Decision

1. Move the compiler-bundled SDK from `0.0.0-stage0.33` to `0.0.0-stage0.34`. Keep language
   `fairyc-language/10` and runtime ABI `fairyc-runtime/0` unchanged.
2. Add public `std::fs::copy_file_bytes(ref CStr source, ref CStr destination) ->
   Result<usize, FileError>` with the existing `filesystem` effect. The operation copies file
   contents only and returns the exact number of destination bytes completed on success.
3. Apply the existing nonempty, NUL-terminated, bounded ASCII conversion to both paths before any
   open. Invalid source or destination input therefore has no filesystem effect.
4. Open the source once with `GENERIC_READ`, `OPEN_EXISTING`, and no sharing. Open the destination
   once with `GENERIC_WRITE`, `CREATE_NEW`, and no sharing. A live source handle can make the copy
   fail with the exact sharing error; an existing destination is never opened, truncated, or
   replaced.
5. Copy synchronously through one fixed 65,536-byte stack buffer. Read until EOF and complete every
   read chunk with one or more writes. A zero-progress successful write fails with stable status
   `4` rather than looping.
6. On a later read, write, or close failure, close both owned handles and return the first observed
   failure with the exact number of bytes already written in `FileError.completed`. The visible
   destination prefix remains caller-owned recovery state; the operation does not delete or roll
   it back.
7. Copy no timestamps, attributes, ACLs, alternate data streams, sparse ranges, encryption policy,
   or other metadata. Do not flush storage, publish through a temporary name, provide cancellation
   or progress callbacks, retry external changes, or claim atomicity or directory-entry durability.
8. Use ordinary `CreateFileW` path resolution. Source reparse points therefore retain target
   behavior and no containment, no-follow, portable link, or identity-pinning promise is added.
9. Prove raw invalid-path preflight, a 65,537-byte multi-window copy, exact destination collision,
   missing-source no-create behavior, live-source sharing failure, exact bytes, and cleanup under
   low and optimized MSVC. Prove public success, collision preservation, missing-source no-create,
   exact bytes, and cleanup through generated Fairy at both profiles.

## Consequences

- Safe programs can copy ordinary file bytes without implementing a source-level read/write loop.
- The name `copy_file_bytes` avoids implying a metadata clone or directory operation.
- No-replace creation prevents accidental destination data loss, while explicit completed-prefix
  preservation makes recovery honest rather than pretending the operation is transactional.
- Portable Unicode paths, Linux public linkage, metadata copy, atomic publication, recursive
  directory copy, cancellation, progress, durability barriers, and no-follow policies remain
  separate contracts.
- The additive SDK and private target-symbol change updates the source digest, public-interface
  hash, package instance, generated locks, dependent identities, and whole-package cache keys.

## Evidence

- `internal/cstdlib/stage0_windows_file.c` owns exclusive handles and the bounded copy loop.
- `internal/cstdlib/stage0_windows_file_probe.c` executes raw success and failure paths.
- `internal/cstdlibcontract/` owns target admission and component closure.
- `internal/sdkstdlib/sources/std/fs.fy` owns the public wrapper.
- `examples/std_fs_quickstart/src/path_ops.fy` executes the generated-Fairy lifecycle.
- `cmd/fairyc-windows-host-smoke/` compiles, inspects, and executes both evidence paths.
