# ADR 0288: Append one Windows file read into an owned byte buffer

- Status: Accepted
- Date: 2026-08-06
- Product source: `docs/product/02-spec.md`
- Runtime source: `docs/runtime/runtime-stdlib-v0.md`
- Depends on: ADR 0246, ADR 0264, ADR 0285

## Context

Public `std::fs::read` can fill a caller-owned mutable byte view, while `std::bytes::Buffer` can
reserve, append, resize, and expose only its initialized prefix. Safe code cannot read directly into
spare Buffer capacity: `view_mut` intentionally excludes uninitialized storage, and growing through
`resize_zeroed` before I/O would publish zero-filled bytes that were never read. Requiring a
temporary fixed array adds a copy and fixed bound to otherwise ordinary buffered file reads.

The File and Buffer implementations already own the necessary invariants, but neither component
may reach into the other's private storage. A bounded adapter can close that gap without exposing
capacity pointers, retaining a borrow, or adding a general uninitialized-memory surface.

## Decision

1. Move the compiler-bundled SDK from `0.0.0-stage0.26` to `0.0.0-stage0.27`. Keep language
   `fairyc-language/10` and runtime ABI `fairyc-runtime/0` unchanged.
2. Add public
   `std::fs::read_append(ref mut File, ref mut bytes::Buffer, usize maximum) ->
   Result<usize, ReadAppendError>` with the existing `filesystem` effect. It performs at most one
   synchronous Windows read and returns the exact appended count, including zero at EOF.
3. Preserve the Buffer's existing initialized prefix. Reserve capacity for `old_len + maximum`,
   read only into the spare suffix, and increase length only by the successful completed count.
   A zero maximum validates both owners, performs no allocation or OS read, and succeeds with zero.
4. Allocation failure occurs before file I/O and leaves the file cursor, Buffer allocation, bytes,
   capacity, and length unchanged. A file failure leaves bytes and length unchanged; capacity may
   have grown during successful reserve and remains reusable. The adapter retains no pointer and
   never exposes spare bytes through the public Buffer view.
5. Add public inert `ReadAppendError` with exact file status, Windows code, and allocation status
   accessors. Exactly one failure domain is nonzero: file failures carry allocation status zero;
   allocation failures carry file status and OS code zero. Success requires both statuses zero.
6. Add private target symbol `fy_std_windows_file_read_append_v0` in a dedicated component ordered
   after and dependency-closed over Windows File, SystemHeap, and Buffer. The raw record and symbol
   remain Windows x86-64 MSVC pre-1 implementation compatibility data.
7. Prove generated-Fairy prefix preservation, a five-byte read followed by a four-byte short read,
   EOF without growth, write-only-handle failure with initialized bytes preserved, explicit close,
   and automatic Buffer cleanup under low and optimized MSVC. Separately force the first growth
   allocation to fail and require statuses `0/1`, unchanged len/capacity, no file I/O, consuming
   close, and zero retained probe allocations at both profiles.

## Consequences

- Safe callers can accumulate bounded chunks without temporary arrays, copying, or public access to
  uninitialized storage.
- `maximum` remains caller policy. This is not `read_to_end`, and it adds no hidden loop, file-size
  trust, implicit seek, buffering policy, cancellation, timeout, asynchronous I/O, or allocation
  limit beyond the requested bound and existing Buffer overflow checks.
- The source-additive SDK change updates source digest, public-interface hash, package instance,
  generated locks, dependent identities, and whole-package cache keys.
- Portable paths, Linux public linkage, vectored I/O, and a cross-target file contract remain
  separate decisions.

## Evidence

- `internal/sdkstdlib/sources/std/fs.fy` owns the public API and typed error accessors.
- `internal/cstdlib/stage0_windows_file_buffer.c` owns the private cross-component commit boundary.
- `internal/cstdlibcontract/` owns dependency closure and deterministic C source order.
- `examples/std_fs_quickstart/src/main.fy` executes successful append, short read, EOF, and file
  failure through generated Fairy.
- `cmd/fairyc-windows-host-smoke/read_append_failure_probe.go` executes the forced allocation
  failure and cleanup contract.
